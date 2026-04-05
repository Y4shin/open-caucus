package config

// WebhookConfig holds outbound webhook settings.
type WebhookConfig struct {
	// URLs is the list of endpoints to POST events to.
	// Env var: WEBHOOK_URLS (comma-separated)
	// Empty → webhooks disabled.
	URLs []string `mapstructure:"urls" env:"WEBHOOK_URLS" default:"" env_doc:"Comma-separated list of URLs to POST meeting events to" env_default:""`

	// Secret is sent as the X-Webhook-Secret header on every request.
	// Env var: WEBHOOK_SECRET
	// Optional — omit header when empty.
	Secret string `mapstructure:"secret" env:"WEBHOOK_SECRET" default:"" env_doc:"Shared secret sent as X-Webhook-Secret header" env_default:""`

	// TimeoutSeconds is the per-request HTTP timeout. Default: 10.
	// Env var: WEBHOOK_TIMEOUT_SECONDS
	TimeoutSeconds int `mapstructure:"timeout_seconds" env:"WEBHOOK_TIMEOUT_SECONDS" default:"10" env_doc:"Per-request timeout in seconds" env_default:"10"`
}
