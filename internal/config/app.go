package config

// ApplicationConfig holds basic web application settings
type ApplicationConfig struct {
	// Service identity
	ServiceName string `mapstructure:"service_name" env:"SERVICE_NAME" default:"conference-tool" env_doc:"Service name for identification" env_default:"conference-tool"`
	Environment string `mapstructure:"environment" env:"ENVIRONMENT" default:"development" env_doc:"Environment (development, staging, production)" validate:"oneOf(development,staging,production)" env_default:"development"`

	// HTTP Server
	Port int    `mapstructure:"port" env:"PORT" default:"8080" env_doc:"HTTP server port" validate:"enforceRangeInt(1:65535)" env_default:"8080"`
	Host string `mapstructure:"host" env:"HOST" default:"0.0.0.0" env_doc:"HTTP server bind address" env_default:"0.0.0.0"`

	// Security
	AdminKey string `mapstructure:"admin_key" env:"ADMIN_KEY" default:"changeme" env_doc:"Site-wide administration API key" env_default:"changeme"`

	// Session Management
	SessionSecret     string `mapstructure:"session_secret" env:"SESSION_SECRET" default:"change-this-to-a-random-32-character-string" env_doc:"Secret key for signing session cookies (must be 32+ chars)" env_default:"change-this-to-a-random-32-character-string"`
	SessionExpiration int    `mapstructure:"session_expiration" env:"SESSION_EXPIRATION" default:"86400" env_doc:"Session expiration in seconds" env_default:"86400"`

	// Storage
	StorageDir string `mapstructure:"storage_dir" env:"STORAGE_DIR" default:"./uploads" env_doc:"Directory for uploaded binary blobs" env_default:"./uploads"`
}
