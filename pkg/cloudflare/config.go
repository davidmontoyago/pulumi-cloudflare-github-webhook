package cloudflare

import (
	"fmt"
	"os"

	"github.com/kelseyhightower/envconfig"
)

// Config holds all the configuration from environment variables for Cloudflare
type Config struct {
	// Cloudflare Account ID
	AccountID string `envconfig:"CLOUDFLARE_ACCOUNT_ID" required:"true"`

	// Cloudflare Zone ID for the webhook route
	ZoneID string `envconfig:"CLOUDFLARE_ZONE_ID" required:"true"`

	// Path to the JS file with the "handle" function to run after webhook authentication and validation
	// JS file must implement function "async function handle(githubEvent, payload, env)"
	HandlerScriptPath string `envconfig:"WORKER_HANDLER_SCRIPT_PATH" required:"true"`

	// Github webhook secret key for payload signature verification
	GithubWebhookSecret string `envconfig:"GITHUB_WEBHOOK_SECRET" required:"true"`

	// Worker domain URL (e.g. workers.youdomain.dev)
	WorkerDomainURL string `envconfig:"WORKER_DOMAIN_URL" required:"false" default:"workers.path2prod.dev"`

	// Worker path (e.g. /webhook/v1)
	WorkerPath string `envconfig:"WORKER_PATH" required:"false" default:"/webhook/v1/fetch"`

	// Resource prefix for the webhook resources
	ResourcePrefix string `envconfig:"RESOURCE_PREFIX" default:"ci-webhook"`
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		return nil, fmt.Errorf("failed to load cloudflare config: %w", err)
	}

	// Validate that script path exists
	if _, err := os.Stat(config.HandlerScriptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("worker script path does not exist: %s", config.HandlerScriptPath)
	}

	return &config, nil
}
