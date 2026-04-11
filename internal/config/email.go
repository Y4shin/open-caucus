package config

// EmailConfig holds SMTP email sending settings.
type EmailConfig struct {
	// Enabled controls whether email sending is active.
	// Env var: EMAIL_ENABLED
	Enabled bool `mapstructure:"enabled" env:"EMAIL_ENABLED" default:"false" env_doc:"Enable email sending via SMTP" env_default:"false"`

	// SMTPHost is the SMTP server hostname.
	// Env var: EMAIL_SMTP_HOST
	SMTPHost string `mapstructure:"smtp_host" env:"EMAIL_SMTP_HOST" default:"" env_doc:"SMTP server hostname" env_default:""`

	// SMTPPort is the SMTP server port. Default: 587 (STARTTLS).
	// Env var: EMAIL_SMTP_PORT
	SMTPPort int `mapstructure:"smtp_port" env:"EMAIL_SMTP_PORT" default:"587" env_doc:"SMTP server port" env_default:"587"`

	// Username is the SMTP authentication username.
	// Env var: EMAIL_USERNAME
	Username string `mapstructure:"username" env:"EMAIL_USERNAME" default:"" env_doc:"SMTP auth username" env_default:""`

	// Password is the SMTP authentication password.
	// Env var: EMAIL_PASSWORD
	Password string `mapstructure:"password" env:"EMAIL_PASSWORD" default:"" env_doc:"SMTP auth password" env_default:""`

	// FromAddress is the sender email address.
	// Env var: EMAIL_FROM_ADDRESS
	FromAddress string `mapstructure:"from_address" env:"EMAIL_FROM_ADDRESS" default:"" env_doc:"Sender email address" env_default:""`

	// FromName is the sender display name.
	// Env var: EMAIL_FROM_NAME
	FromName string `mapstructure:"from_name" env:"EMAIL_FROM_NAME" default:"Open Caucus" env_doc:"Sender display name" env_default:"Open Caucus"`
}
