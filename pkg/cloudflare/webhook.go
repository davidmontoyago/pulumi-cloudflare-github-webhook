package cloudflare

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	namer "github.com/davidmontoyago/commodity-namer"
	"github.com/pulumi/pulumi-cloudflare/sdk/v6/go/cloudflare"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

//go:embed scripts/base/webhook.js
var webhookBaseScript []byte

// CloudflareGithubWebhook represents the Cloudflare Worker infrastructure
type CloudflareGithubWebhook struct {
	namer.Namer

	pulumi.ResourceState

	// Worker represents the Cloudflare Worker
	Worker *cloudflare.WorkersScript

	// WorkerURL is the URL where the worker is accessible
	WorkerURL pulumi.StringOutput

	// WorkerDomainURL is the zone domain URL where the worker is accessible
	domainURL  string
	workerPath string
}

type WebhookEnvVar struct {
	Name pulumi.StringInput
	Type pulumi.StringInput
	Text pulumi.StringInput
}

// NewCloudflareWebhookStack creates a new Cloudflare Worker and Worker Script
func NewCloudflareWebhookStack(ctx *pulumi.Context, name string, envVars []WebhookEnvVar, config *Config) (*CloudflareGithubWebhook, error) {
	component := &CloudflareGithubWebhook{
		Namer: namer.New(name),

		domainURL:  config.WorkerDomainURL,
		workerPath: config.WorkerPath,
	}

	err := ctx.RegisterComponentResource("custom:webhook:CloudflareWebhookStack", name, component)
	if err != nil {
		return nil, fmt.Errorf("failed to register component resource: %w", err)
	}

	// load the handler script and prepend it to the base webhook script
	postAuthHandlerScript, err := os.ReadFile(config.HandlerScriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read post auth handler script file: %w", err)
	}

	// webhookBaseScript is embedded at compile time via //go:embed directive
	workerScript := fmt.Sprintf("%s\n%s", string(postAuthHandlerScript), string(webhookBaseScript))

	sha256 := sha256.Sum256([]byte(workerScript))
	sha256String := hex.EncodeToString(sha256[:])

	// Build bindings from envVars
	bindings := make(cloudflare.WorkersScriptBindingArray, len(envVars))
	for i, envVar := range envVars {
		bindings[i] = &cloudflare.WorkersScriptBindingArgs{
			Name: envVar.Name,
			Text: envVar.Text,
			Type: envVar.Type,
		}
	}

	// Create the Cloudflare Worker Script
	scriptName := component.NewResourceName("webhook", "worker", 63)
	worker, err := cloudflare.NewWorkersScript(ctx, scriptName, &cloudflare.WorkersScriptArgs{
		AccountId:     pulumi.String(config.AccountID),
		ScriptName:    pulumi.String(scriptName),
		MainModule:    pulumi.String("webhook.js"),
		Content:       pulumi.String(workerScript),
		ContentSha256: pulumi.String(sha256String),
		Observability: &cloudflare.WorkersScriptObservabilityArgs{
			Enabled: pulumi.Bool(false),
		},
		Bindings: bindings,
	}, pulumi.Parent(component))
	if err != nil {
		return nil, fmt.Errorf("failed to create cloudflare worker: %w", err)
	}

	component.Worker = worker

	route, err := cloudflare.NewWorkersRoute(ctx, component.NewResourceName("webhook", "route", 63), &cloudflare.WorkersRouteArgs{
		ZoneId:  pulumi.String(config.ZoneID),
		Pattern: pulumi.String(fmt.Sprintf("%s%s*", component.domainURL, component.workerPath)),
		Script:  pulumi.String(scriptName),
	}, pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{worker}))
	if err != nil {
		return nil, fmt.Errorf("failed to create cloudflare workers route: %w", err)
	}

	domainParts := strings.Split(component.domainURL, ".")
	rootDomain := domainParts[1:]

	recordResourceName := component.NewResourceName("webhook", "dns", 63)
	record, err := cloudflare.NewDnsRecord(ctx, recordResourceName, &cloudflare.DnsRecordArgs{
		ZoneId:  route.ZoneId,
		Name:    pulumi.String(component.domainURL),
		Content: pulumi.String(strings.Join(rootDomain, ".")),
		Type:    pulumi.String("CNAME"),
		Proxied: pulumi.Bool(true), // Orange-clouded
		Ttl:     pulumi.Float64(1), // Automatic TTL when proxied
	}, pulumi.Parent(component))
	if err != nil {
		return nil, fmt.Errorf("failed to create workers DNS record: %w", err)
	}

	component.WorkerURL = pulumi.Sprintf("https://%s%s", record.Name, component.workerPath)

	return component, nil
}
