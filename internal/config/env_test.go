package config

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCollectEnvFields_ApplicationGroup(t *testing.T) {
	fields, err := CollectEnvFields([]ConfigGroup{ApplicationGroup})
	if err != nil {
		t.Fatalf("CollectEnvFields failed: %v", err)
	}

	// ApplicationConfig has 5 fields
	expectedCount := 7
	if len(fields) != expectedCount {
		t.Errorf("Expected %d fields, got %d", expectedCount, len(fields))
	}

	// Verify all key fields exist
	envVars := make(map[string]bool)
	for _, field := range fields {
		envVars[field.EnvVar] = true
	}

	requiredEnvVars := []string{
		"SERVICE_NAME",
		"ENVIRONMENT",
		"PORT",
		"HOST",
		"ADMIN_KEY",
		"SESSION_SECRET",
		"SESSION_EXPIRATION",
	}

	for _, envVar := range requiredEnvVars {
		if !envVars[envVar] {
			t.Errorf("Expected env var %s not found in collected fields", envVar)
		}
	}
}

func TestCollectEnvFields_FieldTypes(t *testing.T) {
	fields, err := CollectEnvFields([]ConfigGroup{ApplicationGroup})
	if err != nil {
		t.Fatalf("CollectEnvFields failed: %v", err)
	}

	tests := []struct {
		envVar       string
		expectedType string
	}{
		{"SERVICE_NAME", "string"},
		{"ENVIRONMENT", "string"},
		{"PORT", "int"},
		{"HOST", "string"},
		{"ADMIN_KEY", "string"},
	}

	for _, tt := range tests {
		found := false
		for _, field := range fields {
			if field.EnvVar == tt.envVar {
				found = true
				if field.FieldType != tt.expectedType {
					t.Errorf("%s: expected type %s, got %s", tt.envVar, tt.expectedType, field.FieldType)
				}
				break
			}
		}

		if !found {
			t.Errorf("Expected field %s not found", tt.envVar)
		}
	}
}

func TestCollectEnvFields_RequiredVsOptional(t *testing.T) {
	fields, err := CollectEnvFields([]ConfigGroup{ApplicationGroup})
	if err != nil {
		t.Fatalf("CollectEnvFields failed: %v", err)
	}

	tests := []struct {
		envVar   string
		required bool
	}{
		{"SERVICE_NAME", false}, // has default
		{"ENVIRONMENT", false},  // has default
		{"PORT", false},         // has default
		{"HOST", false},         // has default
		{"ADMIN_KEY", false},    // has default
	}

	for _, tt := range tests {
		found := false
		for _, field := range fields {
			if field.EnvVar == tt.envVar {
				found = true
				if field.Required != tt.required {
					t.Errorf("%s: expected Required=%v, got %v", tt.envVar, tt.required, field.Required)
				}
				break
			}
		}

		if !found {
			t.Errorf("Expected field %s not found", tt.envVar)
		}
	}
}

func TestCollectEnvFields_ValidationHints(t *testing.T) {
	fields, err := CollectEnvFields([]ConfigGroup{ApplicationGroup})
	if err != nil {
		t.Fatalf("CollectEnvFields failed: %v", err)
	}

	tests := []struct {
		envVar          string
		shouldHaveHints bool
		hintContains    string
	}{
		{
			envVar:          "ENVIRONMENT",
			shouldHaveHints: true,
			hintContains:    "Constraints:",
		},
		{
			envVar:          "PORT",
			shouldHaveHints: true,
			hintContains:    "Constraints:",
		},
	}

	for _, tt := range tests {
		found := false
		for _, field := range fields {
			if field.EnvVar == tt.envVar {
				found = true

				if tt.shouldHaveHints && !strings.Contains(field.Documentation, tt.hintContains) {
					t.Errorf("%s: expected documentation to contain %q, got: %s", tt.envVar, tt.hintContains, field.Documentation)
				}

				break
			}
		}

		if !found {
			t.Errorf("Expected field %s not found", tt.envVar)
		}
	}
}

func TestGenerateExampleEnv(t *testing.T) {
	content, err := GenerateExampleEnv([]ConfigGroup{ApplicationGroup})
	if err != nil {
		t.Fatalf("GenerateExampleEnv failed: %v", err)
	}

	// Check for section header
	if !strings.Contains(content, "# Application Configuration") {
		t.Error("Expected Application Configuration section header")
	}

	// Check for optional fields with defaults
	if !strings.Contains(content, "SERVICE_NAME=conference-tool") {
		t.Error("Expected SERVICE_NAME with default value")
	}
	if !strings.Contains(content, "ENVIRONMENT=development") {
		t.Error("Expected ENVIRONMENT with default value")
	}
	if !strings.Contains(content, "PORT=8080") {
		t.Error("Expected PORT with default value")
	}
	if !strings.Contains(content, "HOST=0.0.0.0") {
		t.Error("Expected HOST with default value")
	}
	if !strings.Contains(content, "ADMIN_KEY=changeme") {
		t.Error("Expected ADMIN_KEY with default value")
	}

	// Check for documentation comments
	if !strings.Contains(content, "# Service name for identification") {
		t.Error("Expected documentation for SERVICE_NAME")
	}
	if !strings.Contains(content, "# HTTP server port") {
		t.Error("Expected documentation for PORT")
	}

	// Check for validation hints
	if !strings.Contains(content, "Constraints:") {
		t.Error("Expected validation constraints in documentation")
	}
}

func TestValidateEnv_AllValid(t *testing.T) {
	// Create a temporary .env file with all valid values
	tmpFile := t.TempDir() + "/.env.test"
	content := `
SERVICE_NAME=test-service
ENVIRONMENT=production
PORT=9000
HOST=127.0.0.1
ADMIN_KEY=test-admin-key
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	errors, err := ValidateEnv([]ConfigGroup{ApplicationGroup}, tmpFile)
	if err != nil {
		t.Fatalf("ValidateEnv failed: %v", err)
	}

	if len(errors) > 0 {
		t.Errorf("Expected no validation errors, got %d: %+v", len(errors), errors)
	}
}

func TestValidateEnv_InvalidFormats(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "invalid_port_format",
			content: "PORT=not_a_number\n",
			wantErr: "invalid integer",
		},
		{
			name:    "invalid_port_range",
			content: "PORT=99999\n",
			wantErr: "constraint_violation",
		},
		{
			name:    "invalid_environment",
			content: "ENVIRONMENT=invalid\n",
			wantErr: "constraint_violation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := t.TempDir() + "/.env.test"
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			errors, err := ValidateEnv([]ConfigGroup{ApplicationGroup}, tmpFile)
			if err != nil {
				t.Fatalf("ValidateEnv failed: %v", err)
			}

			if len(errors) == 0 {
				t.Error("Expected validation errors but got none")
			}

			found := false
			for _, e := range errors {
				if strings.Contains(e.Issue, tt.wantErr) || strings.Contains(e.Details, tt.wantErr) {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected error containing %q, got errors: %+v", tt.wantErr, errors)
			}
		})
	}
}

func TestValidateEnv_SnapshotRestore(t *testing.T) {
	// Set up an initial environment
	snapshot := SetTestEnv(map[string]*string{
		"SERVICE_NAME": stringPtr("original-value"),
		"PORT":         stringPtr("8888"),
	})
	defer snapshot.Restore()

	// Create a temporary .env file with different values
	tmpFile := t.TempDir() + "/.env.test"
	content := `
SERVICE_NAME=test-service
PORT=9000
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Run validation
	_, err := ValidateEnv([]ConfigGroup{ApplicationGroup}, tmpFile)
	if err != nil {
		t.Fatalf("ValidateEnv failed: %v", err)
	}

	// Verify original environment is restored
	if got := os.Getenv("SERVICE_NAME"); got != "original-value" {
		t.Errorf("SERVICE_NAME: got %q, want %q (original value not restored)", got, "original-value")
	}
	if got := os.Getenv("PORT"); got != "8888" {
		t.Errorf("PORT: got %q, want %q (original value not restored)", got, "8888")
	}
}

func TestDetectFieldType(t *testing.T) {
	type TestStruct struct {
		StringField   string
		IntField      int
		Int32Field    int32
		BoolField     bool
		Float64Field  float64
		DurationField time.Duration
	}

	tests := []struct {
		fieldName    string
		expectedType string
	}{
		{"StringField", "string"},
		{"IntField", "int"},
		{"Int32Field", "int"},
		{"BoolField", "bool"},
		{"Float64Field", "float"},
	}

	val := reflect.TypeOf(TestStruct{})
	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			field, _ := val.FieldByName(tt.fieldName)
			got := detectFieldType(field)
			if got != tt.expectedType {
				t.Errorf("detectFieldType(%s): got %q, want %q", tt.fieldName, got, tt.expectedType)
			}
		})
	}
}

func TestDescribeValidation(t *testing.T) {
	tests := []struct {
		validation string
		expected   string
	}{
		{"oneOf(a,b,c)", "must be one of: a,b,c"},
		{"enforceRangeInt(1:100)", "must be in range 1:100"},
		{"enforceMin(1)", "must be >= 1"},
	}

	for _, tt := range tests {
		t.Run(tt.validation, func(t *testing.T) {
			got := describeValidation(tt.validation)
			if !strings.Contains(got, tt.expected) {
				t.Errorf("describeValidation(%q): got %q, want substring %q", tt.validation, got, tt.expected)
			}
		})
	}
}

func TestFormatEnvSection(t *testing.T) {
	result := formatEnvSection("application")

	if !strings.Contains(result, "Application Configuration") {
		t.Error("Expected section title 'Application Configuration'")
	}

	if !strings.Contains(result, "====") {
		t.Error("Expected separator line")
	}
}

func TestFormatEnvField(t *testing.T) {
	tests := []struct {
		name     string
		field    EnvFieldInfo
		expected []string
	}{
		{
			name: "optional_with_default",
			field: EnvFieldInfo{
				EnvVar:        "PORT",
				Documentation: "HTTP server port",
				Required:      false,
				ExampleValue:  "8080",
			},
			expected: []string{"# HTTP server port", "PORT=8080"},
		},
		{
			name: "required_field",
			field: EnvFieldInfo{
				EnvVar:        "API_KEY",
				Documentation: "API key for authentication",
				Required:      true,
				ExampleValue:  "your-key-here",
			},
			expected: []string{"# API key for authentication", "# API_KEY=your-key-here  # Required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatEnvField(tt.field)

			for _, exp := range tt.expected {
				if !strings.Contains(result, exp) {
					t.Errorf("formatEnvField: expected to contain %q, got: %s", exp, result)
				}
			}
		})
	}
}
