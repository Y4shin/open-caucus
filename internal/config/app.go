package config

// ApplicationConfig holds basic web application settings
type ApplicationConfig struct {
	// Service identity
	ServiceName string `mapstructure:"service_name" env:"SERVICE_NAME" default:"conference-tool" env_doc:"Service name for identification" env_default:"conference-tool"`
	Environment string `mapstructure:"environment" env:"ENVIRONMENT" default:"development" env_doc:"Environment (development, staging, production)" validate:"oneOf(development,staging,production)" env_default:"development"`

	// HTTP Server
	Port int    `mapstructure:"port" env:"PORT" default:"8080" env_doc:"HTTP server port" validate:"enforceRangeInt(1:65535)" env_default:"8080"`
	Host string `mapstructure:"host" env:"HOST" default:"0.0.0.0" env_doc:"HTTP server bind address" env_default:"0.0.0.0"`

	// Logging
	LogLevel  string `mapstructure:"log_level" env:"LOG_LEVEL" default:"info" env_doc:"Log level (debug, info, warn, error)" validate:"oneOf(debug,info,warn,error)" env_default:"info"`
	LogFormat string `mapstructure:"log_format" env:"LOG_FORMAT" default:"json" env_doc:"Log format (json, text)" validate:"oneOf(json,text)" env_default:"json"`
}
