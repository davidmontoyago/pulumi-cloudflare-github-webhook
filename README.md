# pulumi-cloudflare-github-webhook


[![Develop](https://github.com/davidmontoyago/pulumi-cloudflare-github-webhook/actions/workflows/develop.yaml/badge.svg)](https://github.com/davidmontoyago/pulumi-cloudflare-github-webhook/actions/workflows/develop.yaml) [![Go Coverage](https://raw.githubusercontent.com/wiki/davidmontoyago/pulumi-cloudflare-github-webhook/coverage.svg)](https://raw.githack.com/wiki/davidmontoyago/pulumi-cloudflare-github-webhook/coverage.html) [![Go Reference](https://pkg.go.dev/badge/github.com/davidmontoyago/pulumi-cloudflare-github-webhook.svg)](https://pkg.go.dev/github.com/davidmontoyago/pulumi-cloudflare-github-webhook)

Deploy a Github webhook as a cloudflare worker.

## Features
- GitHub webhook secret validation (HMAC-SHA256)
- Bring your own JS handler function
- Worker routing & DNS record
- Automatic script composition (base webhook + your handler)
- Set worker environment variables for secrets and config

## Pre-requisites
- A Cloudflare account and env var `CLOUDFLARE_API_TOKEN` set
- A domain registered with Cloudflare
- Pulumi & Go

## Getting Started

```bash
go get github.com/davidmontoyago/pulumi-cloudflare-github-webhook
```

### Basic Usage

```go
package main

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/davidmontoyago/pulumi-cloudflare-github-webhook/pkg/cloudflare"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Load configuration from environment variables
		cfg, err := cloudflare.LoadConfig()
		if err != nil {
			return err
		}

		// Define environment variables for the worker
		envVars := []cloudflare.WebhookEnvVar{
			{
				Name: pulumi.String("GITHUB_WEBHOOK_SECRET"),
				Type: pulumi.String("secret_text"),
				Text: pulumi.String(cfg.GithubWebhookSecret),
			},
		}

		// Create the webhook
		webhook, err := cloudflare.NewCloudflareWebhookStack(ctx, cfg.ResourcePrefix, envVars, cfg)
		if err != nil {
			return err
		}

		// Set this webhook URL on the GitHub repo
		ctx.Export("worker_url", webhook.WorkerURL)

		return nil
	})
}
```

### Handler Function

JS handler file must implement the `handle` function:

```javascript
async function handle(githubEvent, payload, env)
```

**Parameters:**
- `githubEvent` (string): The GitHub event type (e.g., "push", "pull_request", "workflow_job")
- `payload` (object): The deserialized JSON payload from GitHub
- `env` (object): Env vars configured in the worker

**Returns:**
- An object that will be serialized and returned to GitHub

**Example Handler:**

```javascript
async function handle(githubEvent, payload, env) {
    console.log(`Received GitHub event: ${githubEvent}`);

    // Handle push events
    if (githubEvent === "push") {
        const branch = payload.ref.replace('refs/heads/', '');
        const commits = payload.commits.length;
        const pusher = payload.pusher.name;

        console.log(`Push to ${branch} by ${pusher}: ${commits} commit(s)`);

        // Your custom logic here
        // Example: trigger a deployment, send a notification, etc.

        return {
            status: "success",
            message: `Processed ${commits} commit(s) on ${branch}`,
            event: githubEvent
        };
    }

    // Handle pull request events
    if (githubEvent === "pull_request") {
        const action = payload.action;
        const prNumber = payload.pull_request.number;
        const prTitle = payload.pull_request.title;

        console.log(`Pull request #${prNumber} ${action}: ${prTitle}`);

        return {
            status: "success",
            message: `Processed PR #${prNumber} (${action})`,
            event: githubEvent
        };
    }

    // Default handler for other events
    return {
        status: "received",
        message: `Event ${githubEvent} acknowledged`,
        event: githubEvent
    };
}
```

## Configuration

Set the following environment variables (see [example/env-config/env.example](example/env-config/env.example)):

| Variable                     | Required | Description                                                             |
| ---------------------------- | -------- | ----------------------------------------------------------------------- |
| `CLOUDFLARE_API_TOKEN`       | Yes      | Cloudflare API token with Workers Scripts:Edit and DNS:Edit permissions |
| `CLOUDFLARE_ACCOUNT_ID`      | Yes      | Your Cloudflare account ID                                              |
| `CLOUDFLARE_ZONE_ID`         | Yes      | Cloudflare zone ID for your domain                                      |
| `WORKER_HANDLER_SCRIPT_PATH` | Yes      | Path to your JavaScript handler file                                    |
| `GITHUB_WEBHOOK_SECRET`      | Yes      | Secret key for GitHub webhook signature verification                    |
| `WORKER_DOMAIN_URL`          | No       | Worker domain URL (default: "workers.path2prod.dev")                    |
| `WORKER_PATH`                | No       | Worker path (default: "/webhook/v1/fetch")                              |
| `RESOURCE_PREFIX`            | No       | Resource prefix for Cloudflare resources (default: "ci-webhook")        |

### How it Works

1. GitHub sends a webhook POST request with an HMAC-SHA256 signature
2. The Cloudflare Worker validates the signature using your webhook secret
3. If valid, the worker extracts the event type and payload
4. Your custom `handle` function is executed with the event data
5. The response from your handler is returned to GitHub

## GitHub Webhook Setup

1. Deploy your webhook worker using Pulumi
2. Note the exported `worker_url` from Pulumi
3. In your GitHub repository, go to: Settings → Webhooks → Add webhook
4. Configure:
   - **Payload URL**: Your worker URL (from step 2)
   - **Content type**: `application/json`
   - **Secret**: The same value as `GITHUB_WEBHOOK_SECRET`
   - **Events**: Select the events you want to handle
5. Save the webhook

## Example

See the [example](example/) directory for a complete working example.

## Security

- All webhook payloads are verified using HMAC-SHA256 signatures
- Requests without valid signatures are rejected with 403 Forbidden
- Secrets are stored as Cloudflare Worker environment variables
- Only POST requests are accepted

## License

See [LICENSE](LICENSE) file for details.
- Set env vars for the worker
