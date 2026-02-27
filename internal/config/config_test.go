package config

import (
	"reflect"
	"testing"
)

func TestStringToStringMapHook(t *testing.T) {
	hook := stringToStringMapHook()

	tests := []struct {
		name      string
		input     string
		expected  map[string]string
		shouldErr bool
	}{
		{
			name:  "simple_key_value_pairs",
			input: "key1=value1,key2=value2",
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			shouldErr: false,
		},
		{
			name:  "with_spaces",
			input: "key1 = value1 , key2 = value2",
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			shouldErr: false,
		},
		{
			name:      "empty_string",
			input:     "",
			expected:  map[string]string{},
			shouldErr: false,
		},
		{
			name:      "whitespace_only",
			input:     "   ",
			expected:  map[string]string{},
			shouldErr: false,
		},
		{
			name:  "single_pair",
			input: "key=value",
			expected: map[string]string{
				"key": "value",
			},
			shouldErr: false,
		},
		{
			name:  "value_with_equals",
			input: "Authorization=Bearer token=abc123",
			expected: map[string]string{
				"Authorization": "Bearer token=abc123",
			},
			shouldErr: false,
		},
		{
			name:  "skip_empty_pairs",
			input: "key1=value1,,key2=value2",
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			shouldErr: false,
		},
		{
			name:      "invalid_pair_no_equals",
			input:     "key1=value1,invalid_pair,key2=value2",
			expected:  nil,
			shouldErr: true,
		},
		{
			name:      "empty_key",
			input:     "=value",
			expected:  nil,
			shouldErr: true,
		},
		{
			name:  "empty_value",
			input: "key=",
			expected: map[string]string{
				"key": "",
			},
			shouldErr: false,
		},
		{
			name:  "complex_values",
			input: "service.namespace=production,deployment.environment=us-east-1",
			expected: map[string]string{
				"service.namespace":      "production",
				"deployment.environment": "us-east-1",
			},
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the hook function
			result, err := hook(
				reflect.TypeOf(""),                  // from type: string
				reflect.TypeOf(map[string]string{}), // to type: map[string]string
				tt.input,
			)

			if tt.shouldErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			resultMap, ok := result.(map[string]string)
			if !ok {
				t.Fatalf("result is not map[string]string, got %T", result)
			}

			if len(resultMap) != len(tt.expected) {
				t.Errorf("map length: got %d, want %d", len(resultMap), len(tt.expected))
			}

			for key, expectedValue := range tt.expected {
				if gotValue, ok := resultMap[key]; !ok {
					t.Errorf("missing key %q", key)
				} else if gotValue != expectedValue {
					t.Errorf("key %q: got %q, want %q", key, gotValue, expectedValue)
				}
			}
		})
	}
}

func TestStringToStringMapHook_NonStringInput(t *testing.T) {
	hook := stringToStringMapHook()

	// Should pass through non-string types unchanged
	input := 123
	result, err := hook(
		reflect.TypeOf(input),
		reflect.TypeOf(map[string]string{}),
		input,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != input {
		t.Errorf("expected input to pass through unchanged, got %v", result)
	}
}

func TestStringToStringMapHook_NonMapTarget(t *testing.T) {
	hook := stringToStringMapHook()

	// Should pass through if target is not map[string]string
	input := "key=value"
	result, err := hook(
		reflect.TypeOf(input),
		reflect.TypeOf(""), // target is string, not map
		input,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != input {
		t.Errorf("expected input to pass through unchanged, got %v", result)
	}
}

func TestStringToStringMapHook_WrongMapType(t *testing.T) {
	hook := stringToStringMapHook()

	// Should pass through if target is map but not map[string]string
	input := "key=value"
	result, err := hook(
		reflect.TypeOf(input),
		reflect.TypeOf(map[string]int{}), // wrong value type
		input,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != input {
		t.Errorf("expected input to pass through unchanged, got %v", result)
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	// Clear any env vars that might affect the test
	snapshot := SetTestEnv(map[string]*string{
		"SERVICE_NAME":           nil,
		"ENVIRONMENT":            nil,
		"PORT":                   nil,
		"HOST":                   nil,
		"AUTH_PASSWORD_ENABLED":  nil,
		"AUTH_OAUTH_ENABLED":     nil,
		"OAUTH_ISSUER_URL":       nil,
		"OAUTH_CLIENT_ID":        nil,
		"OAUTH_CLIENT_SECRET":    nil,
		"OAUTH_REDIRECT_URL":     nil,
		"OAUTH_PROVISIONING_MODE": nil,
	})
	defer snapshot.Restore()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Application == nil {
		t.Fatal("Application config is nil")
	}

	// Check defaults
	if cfg.Application.ServiceName != "conference-tool" {
		t.Errorf("ServiceName: got %q, want %q", cfg.Application.ServiceName, "conference-tool")
	}
	if cfg.Application.Environment != "development" {
		t.Errorf("Environment: got %q, want %q", cfg.Application.Environment, "development")
	}
	if cfg.Application.Port != 8080 {
		t.Errorf("Port: got %d, want 8080", cfg.Application.Port)
	}
	if cfg.Application.Host != "0.0.0.0" {
		t.Errorf("Host: got %q, want %q", cfg.Application.Host, "0.0.0.0")
	}
	if cfg.Auth == nil {
		t.Fatal("Auth config is nil")
	}
	if !cfg.Auth.PasswordEnabled {
		t.Errorf("PasswordEnabled: got %v, want true", cfg.Auth.PasswordEnabled)
	}
	if cfg.Auth.OAuthEnabled {
		t.Errorf("OAuthEnabled: got %v, want false", cfg.Auth.OAuthEnabled)
	}
}

func TestLoadConfig_EnvOverrides(t *testing.T) {
	snapshot := SetTestEnv(map[string]*string{
		"SERVICE_NAME":          stringPtr("my-service"),
		"ENVIRONMENT":           stringPtr("production"),
		"PORT":                  stringPtr("9000"),
		"HOST":                  stringPtr("127.0.0.1"),
		"AUTH_PASSWORD_ENABLED": stringPtr("true"),
		"AUTH_OAUTH_ENABLED":    stringPtr("false"),
	})
	defer snapshot.Restore()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Application == nil {
		t.Fatal("Application config is nil")
	}

	// Check env var overrides
	if cfg.Application.ServiceName != "my-service" {
		t.Errorf("ServiceName: got %q, want %q", cfg.Application.ServiceName, "my-service")
	}
	if cfg.Application.Environment != "production" {
		t.Errorf("Environment: got %q, want %q", cfg.Application.Environment, "production")
	}
	if cfg.Application.Port != 9000 {
		t.Errorf("Port: got %d, want 9000", cfg.Application.Port)
	}
	if cfg.Application.Host != "127.0.0.1" {
		t.Errorf("Host: got %q, want %q", cfg.Application.Host, "127.0.0.1")
	}
}

func TestLoadConfig_ValidationFailures(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]*string
		wantErr string
	}{
		{
			name: "invalid_environment",
			envVars: map[string]*string{
				"ENVIRONMENT": stringPtr("invalid"),
			},
			wantErr: "validation failed",
		},
		{
			name: "port_below_range",
			envVars: map[string]*string{
				"PORT": stringPtr("0"),
			},
			wantErr: "validation failed",
		},
		{
			name: "port_above_range",
			envVars: map[string]*string{
				"PORT": stringPtr("99999"),
			},
			wantErr: "validation failed",
		},
		{
			name: "no_auth_provider_enabled",
			envVars: map[string]*string{
				"AUTH_PASSWORD_ENABLED": stringPtr("false"),
				"AUTH_OAUTH_ENABLED":    stringPtr("false"),
			},
			wantErr: "at least one auth provider must be enabled",
		},
		{
			name: "oauth_enabled_missing_required_values",
			envVars: map[string]*string{
				"AUTH_PASSWORD_ENABLED": stringPtr("false"),
				"AUTH_OAUTH_ENABLED":    stringPtr("true"),
				"OAUTH_ISSUER_URL":      stringPtr(""),
				"OAUTH_CLIENT_ID":       stringPtr(""),
				"OAUTH_CLIENT_SECRET":   stringPtr(""),
				"OAUTH_REDIRECT_URL":    stringPtr(""),
			},
			wantErr: "oauth_issuer_url is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envVars := map[string]*string{
				"AUTH_PASSWORD_ENABLED":  nil,
				"AUTH_OAUTH_ENABLED":     nil,
				"OAUTH_ISSUER_URL":       nil,
				"OAUTH_CLIENT_ID":        nil,
				"OAUTH_CLIENT_SECRET":    nil,
				"OAUTH_REDIRECT_URL":     nil,
				"OAUTH_PROVISIONING_MODE": nil,
			}
			for k, v := range tt.envVars {
				envVars[k] = v
			}
			snapshot := SetTestEnv(envVars)
			defer snapshot.Restore()

			_, err := LoadConfig()
			if err == nil {
				t.Error("expected error but got none")
			} else if tt.wantErr != "" && !contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
