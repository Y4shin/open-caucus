package config

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	Path string `mapstructure:"path" env:"DATABASE_PATH" default:"conference.db" env_doc:"Path to the SQLite database file" env_default:"conference.db"`
}
