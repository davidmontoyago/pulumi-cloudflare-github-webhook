// Package main demonstrates how to use the webhook component.
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
			// Add any additional environment variables your handler needs
			// {
			//     Name: pulumi.String("MY_CUSTOM_VAR"),
			//     Type: pulumi.String("secret_text"),
			//     Text: pulumi.String("my-value"),
			// },
		}

		// Create the Github webhook worker
		webhook, err := cloudflare.NewCloudflareWebhookStack(ctx, cfg.ResourcePrefix, envVars, cfg)
		if err != nil {
			return err
		}

		// Export key outputs
		ctx.Export("worker_url", webhook.WorkerURL)
		ctx.Export("worker_script_id", webhook.Worker.ID())

		return nil
	})
}
