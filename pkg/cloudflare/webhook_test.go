package cloudflare_test

import (
	"log"
	"os"
	"testing"

	"github.com/pulumi/pulumi-cloudflare/sdk/v6/go/cloudflare"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	webhook "github.com/davidmontoyago/pulumi-cloudflare-github-webhook/pkg/cloudflare"
)

const (
	testAccountID       = "test-cloudflare-account-id-123"
	testZoneID          = "test-zone-id-123"
	testWorkerDomainURL = "workers.example.com"
	testWorkerPath      = "/webhook/v1/fetch"
)

func TestNewCloudflareWebhookStack_HappyPath(t *testing.T) {
	t.Parallel()

	// Create a temporary handler script for testing
	tempHandlerScript, err := os.CreateTemp("", "test-handler-*.js")
	require.NoError(t, err, "Failed to create temp handler script")
	defer func() {
		err := os.Remove(tempHandlerScript.Name())
		log.Default().Printf("Failed to remove temp handler script: %v", err)
	}()

	handlerScriptContent := `
async function handle(githubEvent, payload, env) {
	console.log("Test handler called");
	return { success: true };
}
`
	_, err = tempHandlerScript.WriteString(handlerScriptContent)
	require.NoError(t, err, "Failed to write handler script")
	err = tempHandlerScript.Close()
	require.NoError(t, err, "Failed to close handler script")

	err = pulumi.RunErr(func(ctx *pulumi.Context) error {
		// Setup test config
		config := &webhook.Config{
			AccountID:         testAccountID,
			ZoneID:            testZoneID,
			HandlerScriptPath: tempHandlerScript.Name(),
			WorkerDomainURL:   testWorkerDomainURL,
			WorkerPath:        testWorkerPath,
		}

		// Setup test environment variables
		envVars := []webhook.WebhookEnvVar{
			{
				Name: pulumi.String("GITHUB_WEBHOOK_SECRET"),
				Type: pulumi.String("secret_text"),
				Text: pulumi.String("test-webhook-secret-123"),
			},
			{
				Name: pulumi.String("GCP_CREDENTIALS_JSON"),
				Type: pulumi.String("secret_text"),
				Text: pulumi.String(`{"type":"service_account"}`),
			},
		}

		// Create the webhook stack
		webhookStack, err := webhook.NewCloudflareWebhookStack(ctx, "test-webhook", envVars, config)
		require.NoError(t, err, "Failed to create webhook stack")
		require.NotNil(t, webhookStack, "Webhook stack should not be nil")

		// Verify the Worker was created
		require.NotNil(t, webhookStack.Worker, "Worker should not be nil")

		// Verify worker account ID
		accountIDCh := make(chan string, 1)
		defer close(accountIDCh)
		webhookStack.Worker.AccountId.ApplyT(func(accountID string) error {
			accountIDCh <- accountID
			return nil
		})
		assert.Equal(t, testAccountID, <-accountIDCh, "Worker account ID should match")

		// Verify worker main module
		mainModuleCh := make(chan string, 1)
		defer close(mainModuleCh)
		webhookStack.Worker.MainModule.ApplyT(func(mainModule *string) error {
			if mainModule != nil {
				mainModuleCh <- *mainModule
			}
			return nil
		})
		assert.Equal(t, "webhook.js", <-mainModuleCh, "Worker main module should be webhook.js")

		// Verify worker content contains both handler and base script
		contentCh := make(chan string, 1)
		defer close(contentCh)
		webhookStack.Worker.Content.ApplyT(func(content *string) error {
			if content != nil {
				contentCh <- *content
			}
			return nil
		})
		workerContent := <-contentCh
		assert.Contains(t, workerContent, "Test handler called", "Worker content should contain handler script")

		// Verify worker bindings
		bindingsCh := make(chan int, 1)
		defer close(bindingsCh)
		webhookStack.Worker.Bindings.ApplyT(func(bindings []cloudflare.WorkersScriptBinding) error {
			bindingsCh <- len(bindings)
			return nil
		})
		assert.Equal(t, 2, <-bindingsCh, "Worker should have 2 bindings")

		// Verify WorkerURL is constructed correctly
		workerURLCh := make(chan string, 1)
		defer close(workerURLCh)
		webhookStack.WorkerURL.ApplyT(func(url string) error {
			workerURLCh <- url
			return nil
		})
		expectedURL := "https://" + testWorkerDomainURL + testWorkerPath
		assert.Equal(t, expectedURL, <-workerURLCh, "Worker URL should match expected format")

		return nil
	}, pulumi.WithMocks("project", "stack", &webhookMocks{}))

	if err != nil {
		t.Fatalf("Pulumi WithMocks failed: %v", err)
	}
}

type webhookMocks struct{}

func (m *webhookMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	outputs := map[string]interface{}{}
	for k, v := range args.Inputs {
		outputs[string(k)] = v
	}

	// Mock resource outputs for each resource type
	switch args.TypeToken {
	case "cloudflare:index/workersScript:WorkersScript":
		outputs["accountId"] = args.Inputs["accountId"]
		outputs["scriptName"] = args.Inputs["scriptName"]
		// MainModule is StringPtrOutput, so keep as pointer-compatible value
		if mainModule, ok := args.Inputs["mainModule"]; ok {
			outputs["mainModule"] = mainModule
		}
		// Content is StringPtrOutput, so keep as pointer-compatible value
		if content, ok := args.Inputs["content"]; ok {
			outputs["content"] = content
		}
		if contentSha256, ok := args.Inputs["contentSha256"]; ok {
			outputs["contentSha256"] = contentSha256
		}
		outputs["bindings"] = args.Inputs["bindings"]

	case "cloudflare:index/workersRoute:WorkersRoute":
		outputs["zoneId"] = testZoneID
		outputs["pattern"] = args.Inputs["pattern"]
		outputs["script"] = args.Inputs["script"]

	case "cloudflare:index/dnsRecord:DnsRecord":
		outputs["zoneId"] = testZoneID
		outputs["name"] = testWorkerDomainURL
		outputs["content"] = args.Inputs["content"]
		outputs["type"] = args.Inputs["type"]
		outputs["proxied"] = args.Inputs["proxied"]
		outputs["ttl"] = args.Inputs["ttl"]
	}

	return args.Name + "_id", resource.NewPropertyMapFromMap(outputs), nil
}

func (m *webhookMocks) Call(_ pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return resource.PropertyMap{}, nil
}
