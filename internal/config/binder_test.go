package config

import (
	"reflect"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestBindConfigToViper_DefaultValues(t *testing.T) {
	type TestConfig struct {
		StringField   string        `mapstructure:"string_field" default:"test_default"`
		IntField      int           `mapstructure:"int_field" default:"42"`
		BoolField     bool          `mapstructure:"bool_field" default:"true"`
		Uint16Field   uint16        `mapstructure:"uint16_field" default:"8080"`
		Float64Field  float64       `mapstructure:"float64_field" default:"3.14"`
		DurationField time.Duration `mapstructure:"duration_field" default:"5m"`
	}

	v := viper.New()
	cfg := &TestConfig{}

	err := BindConfigToViper(v, cfg, "test")
	if err != nil {
		t.Fatalf("BindConfigToViper failed: %v", err)
	}

	// Test string default
	if got := v.GetString("test.string_field"); got != "test_default" {
		t.Errorf("string_field default: got %q, want %q", got, "test_default")
	}

	// Test int default
	if got := v.GetInt("test.int_field"); got != 42 {
		t.Errorf("int_field default: got %d, want 42", got)
	}

	// Test bool default
	if got := v.GetBool("test.bool_field"); got != true {
		t.Errorf("bool_field default: got %v, want true", got)
	}

	// Test uint16 default
	if got := v.GetUint16("test.uint16_field"); got != 8080 {
		t.Errorf("uint16_field default: got %d, want 8080", got)
	}

	// Test float64 default
	if got := v.GetFloat64("test.float64_field"); got != 3.14 {
		t.Errorf("float64_field default: got %f, want 3.14", got)
	}

	// Test duration default
	if got := v.GetDuration("test.duration_field"); got != 5*time.Minute {
		t.Errorf("duration_field default: got %v, want 5m", got)
	}
}

func TestBindConfigToViper_EnvBinding(t *testing.T) {
	type TestConfig struct {
		ServiceName string `mapstructure:"service_name" env:"TEST_SERVICE_NAME" default:"default-service"`
		Port        uint16 `mapstructure:"port" env:"TEST_PORT" default:"8080"`
	}

	snapshot := SetTestEnv(map[string]*string{
		"TEST_SERVICE_NAME": stringPtr("my-service"),
		"TEST_PORT":         stringPtr("9000"),
	})
	defer snapshot.Restore()

	v := viper.New()
	v.AutomaticEnv()
	cfg := &TestConfig{}

	err := BindConfigToViper(v, cfg, "test")
	if err != nil {
		t.Fatalf("BindConfigToViper failed: %v", err)
	}

	// Test env var overrides default
	if got := v.GetString("test.service_name"); got != "my-service" {
		t.Errorf("service_name from env: got %q, want %q", got, "my-service")
	}

	if got := v.GetUint16("test.port"); got != 9000 {
		t.Errorf("port from env: got %d, want 9000", got)
	}
}

func TestBindConfigToViper_NestedStructs(t *testing.T) {
	type NestedConfig struct {
		NestedField string `mapstructure:"nested_field" default:"nested_value"`
	}

	type TestConfig struct {
		TopLevel string       `mapstructure:"top_level" default:"top_value"`
		Nested   NestedConfig `mapstructure:"nested"`
	}

	v := viper.New()
	cfg := &TestConfig{}

	err := BindConfigToViper(v, cfg, "test")
	if err != nil {
		t.Fatalf("BindConfigToViper failed: %v", err)
	}

	// Test top-level field
	if got := v.GetString("test.top_level"); got != "top_value" {
		t.Errorf("top_level default: got %q, want %q", got, "top_value")
	}

	// Test nested field
	if got := v.GetString("test.nested.nested_field"); got != "nested_value" {
		t.Errorf("nested.nested_field default: got %q, want %q", got, "nested_value")
	}
}

func TestBindConfigToViper_SkipsFieldsWithoutMapstructure(t *testing.T) {
	type TestConfig struct {
		WithTag    string `mapstructure:"with_tag" default:"has_tag"`
		WithoutTag string `default:"no_tag"`
	}

	v := viper.New()
	cfg := &TestConfig{}

	err := BindConfigToViper(v, cfg, "test")
	if err != nil {
		t.Fatalf("BindConfigToViper failed: %v", err)
	}

	// Field with mapstructure tag should be bound
	if got := v.GetString("test.with_tag"); got != "has_tag" {
		t.Errorf("with_tag default: got %q, want %q", got, "has_tag")
	}

	// Field without mapstructure tag should be skipped
	if v.IsSet("test.without_tag") {
		t.Errorf("without_tag should not be bound, but was")
	}
}

func TestBindConfigToViper_NoDefaultNoEnv(t *testing.T) {
	type TestConfig struct {
		OptionalField string `mapstructure:"optional_field"`
	}

	v := viper.New()
	cfg := &TestConfig{}

	err := BindConfigToViper(v, cfg, "test")
	if err != nil {
		t.Fatalf("BindConfigToViper failed: %v", err)
	}

	// Field should be registered but have zero value
	if v.IsSet("test.optional_field") {
		t.Errorf("optional_field should not have a default value")
	}
}

func TestParseDefault_StringTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		typ      string
		expected interface{}
	}{
		{"string", "hello", "string", "hello"},
		{"bool_true", "true", "bool", true},
		{"bool_false", "false", "bool", false},
		{"int", "123", "int", int(123)},
		{"int32", "456", "int32", int32(456)},
		{"uint16", "8080", "uint16", uint16(8080)},
		{"float64", "3.14", "float64", 3.14},
		{"duration_sec", "30s", "duration", 30 * time.Second},
		{"duration_min", "5m", "duration", 5 * time.Minute},
		{"duration_hour", "2h", "duration", 2 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result interface{}
			var err error

			switch tt.typ {
			case "string":
				result, err = parseDefault(tt.input, stringType())
			case "bool":
				result, err = parseDefault(tt.input, boolType())
			case "int":
				result, err = parseDefault(tt.input, intType())
			case "int32":
				result, err = parseDefault(tt.input, int32Type())
			case "uint16":
				result, err = parseDefault(tt.input, uint16Type())
			case "float64":
				result, err = parseDefault(tt.input, float64Type())
			case "duration":
				result, err = parseDefault(tt.input, durationType())
			default:
				t.Fatalf("unknown type: %s", tt.typ)
			}

			if err != nil {
				t.Fatalf("parseDefault failed: %v", err)
			}

			if result != tt.expected {
				t.Errorf("parseDefault: got %v (type %T), want %v (type %T)", result, result, tt.expected, tt.expected)
			}
		})
	}
}

func TestParseDefault_InvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		input string
		typ   string
	}{
		{"invalid_int", "not_a_number", "int"},
		{"invalid_float", "not_float", "float64"},
		{"invalid_duration", "invalid", "duration"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error

			switch tt.typ {
			case "int":
				_, err = parseDefault(tt.input, intType())
			case "float64":
				_, err = parseDefault(tt.input, float64Type())
			case "duration":
				_, err = parseDefault(tt.input, durationType())
			}

			if err == nil {
				t.Errorf("parseDefault should have failed for invalid input %q", tt.input)
			}
		})
	}
}

func TestBindConfigToViper_RealApplicationConfig(t *testing.T) {
	snapshot := SetTestEnv(map[string]*string{
		"SERVICE_NAME": nil,
		"ENVIRONMENT":  nil,
		"PORT":         nil,
		"HOST":         nil,
		"LOG_LEVEL":    nil,
		"LOG_FORMAT":   nil,
	})
	defer snapshot.Restore()

	v := viper.New()
	v.AutomaticEnv()
	cfg := &ApplicationConfig{}

	err := BindConfigToViper(v, cfg, "application")
	if err != nil {
		t.Fatalf("BindConfigToViper failed: %v", err)
	}

	// Test all defaults are set correctly
	tests := []struct {
		key      string
		expected interface{}
	}{
		{"application.service_name", "conference-tool"},
		{"application.environment", "development"},
		{"application.port", 8080},
		{"application.host", "0.0.0.0"},
		{"application.log_level", "info"},
		{"application.log_format", "json"},
	}

	for _, tt := range tests {
		switch expected := tt.expected.(type) {
		case string:
			if got := v.GetString(tt.key); got != expected {
				t.Errorf("%s: got %q, want %q", tt.key, got, expected)
			}
		case int:
			if got := v.GetInt(tt.key); got != expected {
				t.Errorf("%s: got %d, want %d", tt.key, got, expected)
			}
		}
	}
}

// Helper functions to get reflect.Type values
func stringType() reflect.Type {
	var s string
	return reflect.TypeOf(s)
}

func boolType() reflect.Type {
	var b bool
	return reflect.TypeOf(b)
}

func intType() reflect.Type {
	var i int
	return reflect.TypeOf(i)
}

func int32Type() reflect.Type {
	var i int32
	return reflect.TypeOf(i)
}

func uint16Type() reflect.Type {
	var u uint16
	return reflect.TypeOf(u)
}

func float64Type() reflect.Type {
	var f float64
	return reflect.TypeOf(f)
}

func durationType() reflect.Type {
	var d time.Duration
	return reflect.TypeOf(d)
}
