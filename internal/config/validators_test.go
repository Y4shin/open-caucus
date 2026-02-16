package config

import (
	"strings"
	"testing"
)

func TestParseTagCall(t *testing.T) {
	tests := []struct {
		name         string
		tagValue     string
		expectedFunc string
		expectedArgs *string
	}{
		{
			name:         "simple_no_args",
			tagValue:     "invert",
			expectedFunc: "invert",
			expectedArgs: nil,
		},
		{
			name:         "with_args",
			tagValue:     "enforceRangeInt(1:65535)",
			expectedFunc: "enforceRangeInt",
			expectedArgs: stringPtr("1:65535"),
		},
		{
			name:         "multiple_args",
			tagValue:     "oneOf(development,staging,production)",
			expectedFunc: "oneOf",
			expectedArgs: stringPtr("development,staging,production"),
		},
		{
			name:         "with_spaces",
			tagValue:     "enforceMin( 1 )",
			expectedFunc: "enforceMin",
			expectedArgs: stringPtr("1 "),
		},
		{
			name:         "empty_args",
			tagValue:     "func()",
			expectedFunc: "func",
			expectedArgs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			funcName, args := parseTagCall(tt.tagValue)

			if funcName != tt.expectedFunc {
				t.Errorf("funcName: got %q, want %q", funcName, tt.expectedFunc)
			}

			if tt.expectedArgs == nil {
				if args != nil {
					t.Errorf("args: got %q, want nil", *args)
				}
			} else {
				if args == nil {
					t.Errorf("args: got nil, want %q", *tt.expectedArgs)
				} else if *args != *tt.expectedArgs {
					t.Errorf("args: got %q, want %q", *args, *tt.expectedArgs)
				}
			}
		})
	}
}

func TestEnforceRangeInt(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		args      *string
		shouldErr bool
	}{
		{
			name:      "valid_in_range",
			value:     50,
			args:      stringPtr("1:100"),
			shouldErr: false,
		},
		{
			name:      "valid_at_min",
			value:     1,
			args:      stringPtr("1:100"),
			shouldErr: false,
		},
		{
			name:      "valid_at_max",
			value:     100,
			args:      stringPtr("1:100"),
			shouldErr: false,
		},
		{
			name:      "invalid_below_min",
			value:     0,
			args:      stringPtr("1:100"),
			shouldErr: true,
		},
		{
			name:      "invalid_above_max",
			value:     101,
			args:      stringPtr("1:100"),
			shouldErr: true,
		},
		{
			name:      "int32_valid",
			value:     int32(50),
			args:      stringPtr("1:100"),
			shouldErr: false,
		},
		{
			name:      "int64_valid",
			value:     int64(50),
			args:      stringPtr("1:100"),
			shouldErr: false,
		},
		{
			name:      "uint16_valid",
			value:     uint16(8080),
			args:      stringPtr("1:65535"),
			shouldErr: false,
		},
		{
			name:      "port_range_valid",
			value:     8080,
			args:      stringPtr("1:65535"),
			shouldErr: false,
		},
		{
			name:      "wrong_type",
			value:     "not_an_int",
			args:      stringPtr("1:100"),
			shouldErr: true,
		},
		{
			name:      "missing_args",
			value:     50,
			args:      nil,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := enforceRangeInt(tt.value, tt.args)
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestEnforceMin(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		args      *string
		shouldErr bool
	}{
		{
			name:      "valid_above_min",
			value:     10,
			args:      stringPtr("1"),
			shouldErr: false,
		},
		{
			name:      "valid_at_min",
			value:     1,
			args:      stringPtr("1"),
			shouldErr: false,
		},
		{
			name:      "invalid_below_min",
			value:     0,
			args:      stringPtr("1"),
			shouldErr: true,
		},
		{
			name:      "int32_valid",
			value:     int32(10),
			args:      stringPtr("1"),
			shouldErr: false,
		},
		{
			name:      "int64_valid",
			value:     int64(10),
			args:      stringPtr("1"),
			shouldErr: false,
		},
		{
			name:      "wrong_type",
			value:     "not_an_int",
			args:      stringPtr("1"),
			shouldErr: true,
		},
		{
			name:      "missing_args",
			value:     10,
			args:      nil,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := enforceMin(tt.value, tt.args)
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestOneOf(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		args      *string
		shouldErr bool
	}{
		{
			name:      "valid_first_option",
			value:     "development",
			args:      stringPtr("development,staging,production"),
			shouldErr: false,
		},
		{
			name:      "valid_middle_option",
			value:     "staging",
			args:      stringPtr("development,staging,production"),
			shouldErr: false,
		},
		{
			name:      "valid_last_option",
			value:     "production",
			args:      stringPtr("development,staging,production"),
			shouldErr: false,
		},
		{
			name:      "invalid_not_in_list",
			value:     "test",
			args:      stringPtr("development,staging,production"),
			shouldErr: true,
		},
		{
			name:      "valid_with_spaces",
			value:     "info",
			args:      stringPtr("debug, info, warn, error"),
			shouldErr: false,
		},
		{
			name:      "log_level_debug",
			value:     "debug",
			args:      stringPtr("debug,info,warn,error"),
			shouldErr: false,
		},
		{
			name:      "log_format_json",
			value:     "json",
			args:      stringPtr("json,text"),
			shouldErr: false,
		},
		{
			name:      "log_format_text",
			value:     "text",
			args:      stringPtr("json,text"),
			shouldErr: false,
		},
		{
			name:      "invalid_log_format",
			value:     "xml",
			args:      stringPtr("json,text"),
			shouldErr: true,
		},
		{
			name:      "wrong_type",
			value:     123,
			args:      stringPtr("development,staging,production"),
			shouldErr: true,
		},
		{
			name:      "missing_args",
			value:     "development",
			args:      nil,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := oneOf(tt.value, tt.args)
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	type TestConfig struct {
		IntField    int    `validate:"enforceRangeInt(1:100)"`
		MinField    int    `validate:"enforceMin(1)"`
		StringField string `validate:"oneOf(option1,option2,option3)"`
	}

	tests := []struct {
		name      string
		config    TestConfig
		shouldErr bool
		errField  string
	}{
		{
			name: "all_valid",
			config: TestConfig{
				IntField:    50,
				MinField:    10,
				StringField: "option2",
			},
			shouldErr: false,
		},
		{
			name: "invalid_int",
			config: TestConfig{
				IntField:    200,
				MinField:    10,
				StringField: "option2",
			},
			shouldErr: true,
			errField:  "IntField",
		},
		{
			name: "invalid_min",
			config: TestConfig{
				IntField:    50,
				MinField:    0,
				StringField: "option2",
			},
			shouldErr: true,
			errField:  "MinField",
		},
		{
			name: "invalid_string",
			config: TestConfig{
				IntField:    50,
				MinField:    10,
				StringField: "invalid",
			},
			shouldErr: true,
			errField:  "StringField",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(&tt.config)
			if tt.shouldErr {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errField != "" && !strings.Contains(err.Error(), tt.errField) {
					t.Errorf("expected error to mention field %q, got: %v", tt.errField, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateConfig_NestedStructs(t *testing.T) {
	type NestedConfig struct {
		Value int `validate:"enforceMin(1)"`
	}

	type TestConfig struct {
		TopLevel int          `validate:"enforceMin(1)"`
		Nested   NestedConfig
	}

	tests := []struct {
		name      string
		config    TestConfig
		shouldErr bool
	}{
		{
			name: "all_valid",
			config: TestConfig{
				TopLevel: 10,
				Nested:   NestedConfig{Value: 5},
			},
			shouldErr: false,
		},
		{
			name: "invalid_top_level",
			config: TestConfig{
				TopLevel: 0,
				Nested:   NestedConfig{Value: 5},
			},
			shouldErr: true,
		},
		{
			name: "invalid_nested",
			config: TestConfig{
				TopLevel: 10,
				Nested:   NestedConfig{Value: 0},
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(&tt.config)
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateConfig_SkipsFieldsWithoutTag(t *testing.T) {
	type TestConfig struct {
		WithTag    int `validate:"enforceMin(1)"`
		WithoutTag int
	}

	config := TestConfig{
		WithTag:    10,
		WithoutTag: 0, // Would fail if validated, but should be skipped
	}

	err := ValidateConfig(&config)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateConfig_UnknownValidator(t *testing.T) {
	type TestConfig struct {
		Field int `validate:"unknownValidator(arg)"`
	}

	config := TestConfig{Field: 10}
	err := ValidateConfig(&config)
	if err == nil {
		t.Error("expected error for unknown validator")
	}
	if !strings.Contains(err.Error(), "unknown validator") {
		t.Errorf("expected error to mention unknown validator, got: %v", err)
	}
}

func TestValidateConfig_RealApplicationConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    ApplicationConfig
		shouldErr bool
		errField  string
	}{
		{
			name: "all_valid",
			config: ApplicationConfig{
				ServiceName: "test-service",
				Environment: "development",
				Port:        8080,
				Host:        "0.0.0.0",
				LogLevel:    "info",
				LogFormat:   "json",
			},
			shouldErr: false,
		},
		{
			name: "invalid_environment",
			config: ApplicationConfig{
				ServiceName: "test-service",
				Environment: "invalid",
				Port:        8080,
				Host:        "0.0.0.0",
				LogLevel:    "info",
				LogFormat:   "json",
			},
			shouldErr: true,
			errField:  "Environment",
		},
		{
			name: "invalid_port_below",
			config: ApplicationConfig{
				ServiceName: "test-service",
				Environment: "development",
				Port:        0,
				Host:        "0.0.0.0",
				LogLevel:    "info",
				LogFormat:   "json",
			},
			shouldErr: true,
			errField:  "Port",
		},
		{
			name: "invalid_port_above",
			config: ApplicationConfig{
				ServiceName: "test-service",
				Environment: "development",
				Port:        99999,
				Host:        "0.0.0.0",
				LogLevel:    "info",
				LogFormat:   "json",
			},
			shouldErr: true,
			errField:  "Port",
		},
		{
			name: "invalid_log_level",
			config: ApplicationConfig{
				ServiceName: "test-service",
				Environment: "development",
				Port:        8080,
				Host:        "0.0.0.0",
				LogLevel:    "trace",
				LogFormat:   "json",
			},
			shouldErr: true,
			errField:  "LogLevel",
		},
		{
			name: "invalid_log_format",
			config: ApplicationConfig{
				ServiceName: "test-service",
				Environment: "development",
				Port:        8080,
				Host:        "0.0.0.0",
				LogLevel:    "info",
				LogFormat:   "xml",
			},
			shouldErr: true,
			errField:  "LogFormat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(&tt.config)
			if tt.shouldErr {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errField != "" && !strings.Contains(err.Error(), tt.errField) {
					t.Errorf("expected error to mention field %q, got: %v", tt.errField, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
