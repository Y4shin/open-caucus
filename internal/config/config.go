package config

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	Application *ApplicationConfig `mapstructure:"application"`
	Database    *DatabaseConfig    `mapstructure:"database"`
	Auth        *AuthConfig        `mapstructure:"auth"`
	Webhook     *WebhookConfig     `mapstructure:"webhook"`
}

// ConfigGroup identifies config subsections
type ConfigGroup string

const (
	ApplicationGroup ConfigGroup = "application"
	DatabaseGroup    ConfigGroup = "database"
	AuthGroup        ConfigGroup = "auth"
	WebhookGroup     ConfigGroup = "webhook"
)

// LoadConfig loads full configuration
func LoadConfig() (*Config, error) {
	return LoadConfigSelective([]ConfigGroup{ApplicationGroup, DatabaseGroup, AuthGroup, WebhookGroup})
}

// stringToStringMapHook decodes comma-separated key=value strings into map[string]string
// This allows environment variables like "key1=val1,key2=val2" to be unmarshaled into map fields
func stringToStringMapHook() mapstructure.DecodeHookFuncType {
	return func(from reflect.Type, to reflect.Type, data any) (any, error) {
		// Only process string -> map[string]string conversions
		if from.Kind() != reflect.String {
			return data, nil
		}
		if to.Kind() != reflect.Map || to.Key().Kind() != reflect.String || to.Elem().Kind() != reflect.String {
			return data, nil
		}

		s := strings.TrimSpace(data.(string))
		if s == "" {
			return map[string]string{}, nil
		}

		out := make(map[string]string)
		pairs := strings.Split(s, ",")
		for _, p := range pairs {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			k, v, ok := strings.Cut(p, "=")
			if !ok {
				return nil, fmt.Errorf("invalid pair %q (expected k=v)", p)
			}
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			if k == "" {
				return nil, fmt.Errorf("empty key in %q", p)
			}
			out[k] = v
		}
		return out, nil
	}
}

// LoadConfigSelective loads configuration from environment variables and config files
func LoadConfigSelective(groups []ConfigGroup) (*Config, error) {
	v := viper.New()

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Create empty config struct
	cfg := &Config{}

	// Bind config groups using reflection
	for _, group := range groups {
		switch group {
		case ApplicationGroup:
			cfg.Application = &ApplicationConfig{}
			if err := BindConfigToViper(v, cfg.Application, string(ApplicationGroup)); err != nil {
				return nil, fmt.Errorf("failed to bind application config: %w", err)
			}
		case DatabaseGroup:
			cfg.Database = &DatabaseConfig{}
			if err := BindConfigToViper(v, cfg.Database, string(DatabaseGroup)); err != nil {
				return nil, fmt.Errorf("failed to bind database config: %w", err)
			}
		case AuthGroup:
			cfg.Auth = &AuthConfig{}
			if err := BindConfigToViper(v, cfg.Auth, string(AuthGroup)); err != nil {
				return nil, fmt.Errorf("failed to bind auth config: %w", err)
			}
		case WebhookGroup:
			cfg.Webhook = &WebhookConfig{}
			if err := BindConfigToViper(v, cfg.Webhook, string(WebhookGroup)); err != nil {
				return nil, fmt.Errorf("failed to bind webhook config: %w", err)
			}
		default:
			return nil, fmt.Errorf("unknown config group: %s", group)
		}
	}

	// Try to read from config file if present (optional)
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	_ = v.ReadInConfig() // Ignore error if file doesn't exist

	// Unmarshal into config struct using custom decoder with hooks
	decCfg := &mapstructure.DecoderConfig{
		Result:  &cfg,
		TagName: "mapstructure",
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			stringToStringMapHook(),                     // Handle string -> map[string]string
			mapstructure.StringToTimeDurationHookFunc(), // Handle time.Duration
			mapstructure.StringToSliceHookFunc(","),     // Handle string -> slice
		),
		WeaklyTypedInput: true, // Allow type conversions (string -> int, string -> bool, etc.)
	}

	dec, err := mapstructure.NewDecoder(decCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}
	if err := dec.Decode(v.AllSettings()); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Apply transforms to each config group
	for _, group := range groups {
		var groupCfg interface{}
		switch group {
		case ApplicationGroup:
			groupCfg = cfg.Application
		case DatabaseGroup:
			groupCfg = cfg.Database
		case AuthGroup:
			groupCfg = cfg.Auth
		case WebhookGroup:
			groupCfg = cfg.Webhook
		}

		if groupCfg != nil {
			if err := ApplyTransforms(groupCfg); err != nil {
				return nil, fmt.Errorf("failed to apply transforms for %s: %w", group, err)
			}
		}
	}

	// Validate each config group
	for _, group := range groups {
		var groupCfg interface{}
		switch group {
		case ApplicationGroup:
			groupCfg = cfg.Application
		case DatabaseGroup:
			groupCfg = cfg.Database
		case AuthGroup:
			groupCfg = cfg.Auth
		case WebhookGroup:
			groupCfg = cfg.Webhook
		}

		if groupCfg != nil {
			if err := ValidateConfig(groupCfg); err != nil {
				return nil, fmt.Errorf("validation failed for %s: %w", group, err)
			}
		}
	}

	if cfg.Auth != nil {
		if !cfg.Auth.PasswordEnabled && !cfg.Auth.OAuthEnabled {
			return nil, fmt.Errorf("validation failed for auth: at least one auth provider must be enabled")
		}
		if cfg.Auth.OAuthEnabled {
			if cfg.Auth.OAuthIssuerURL == "" {
				return nil, fmt.Errorf("validation failed for auth: oauth_issuer_url is required when OAuth is enabled")
			}
			if cfg.Auth.OAuthClientID == "" {
				return nil, fmt.Errorf("validation failed for auth: oauth_client_id is required when OAuth is enabled")
			}
			if cfg.Auth.OAuthClientSecret == "" {
				return nil, fmt.Errorf("validation failed for auth: oauth_client_secret is required when OAuth is enabled")
			}
			if cfg.Auth.OAuthRedirectURL == "" {
				return nil, fmt.Errorf("validation failed for auth: oauth_redirect_url is required when OAuth is enabled")
			}
		}
	}

	return cfg, nil
}
