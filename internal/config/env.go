package config

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// EnvFieldInfo holds metadata for a single environment variable
type EnvFieldInfo struct {
	EnvVar        string // "SERVICE_NAME"
	ConfigKey     string // "application.service_name"
	ConfigGroup   string // "application"
	DefaultValue  string // "conference-tool"
	ExampleValue  string // From env_default or default tag
	Documentation string // From env_doc tag
	Required      bool   // true if no default tag
	FieldType     string // "string", "int", "duration", "map[string]string"
	Transform     string // "invert", "ms_to_sec", ""
	Validation    string // "oneOf(a,b)", "enforceMin(1)", ""
}

// EnvValidationError represents a single validation error
type EnvValidationError struct {
	EnvVar      string // "SERVICE_NAME"
	ConfigGroup string // "application"
	Issue       string // "missing", "invalid_format", "constraint_violation"
	Details     string // Detailed error message
}

// CollectEnvFields uses reflection to extract environment variable metadata from config groups
// Returns deduplicated list (first occurrence wins)
func CollectEnvFields(groups []ConfigGroup) ([]EnvFieldInfo, error) {
	var fields []EnvFieldInfo
	seen := make(map[string]bool) // Track env vars we've already seen

	// Map groups to their config structs
	for _, group := range groups {
		var cfg interface{}
		switch group {
		case ApplicationGroup:
			cfg = &ApplicationConfig{}
		default:
			return nil, fmt.Errorf("unknown config group: %s", group)
		}

		// Collect fields from this config struct
		groupFields, err := collectFieldsFromStruct(cfg, string(group), "")
		if err != nil {
			return nil, fmt.Errorf("failed to collect fields from %s: %w", group, err)
		}

		// Add to result, deduplicating by env var name (first occurrence wins)
		for _, field := range groupFields {
			if !seen[field.EnvVar] {
				fields = append(fields, field)
				seen[field.EnvVar] = true
			}
		}
	}

	return fields, nil
}

// collectFieldsFromStruct recursively walks a struct and extracts EnvFieldInfo for each field
func collectFieldsFromStruct(cfg interface{}, groupName string, parentKey string) ([]EnvFieldInfo, error) {
	var fields []EnvFieldInfo

	val := reflect.ValueOf(cfg)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Get struct tags
		mapstructureKey := field.Tag.Get("mapstructure")
		envVar := field.Tag.Get("env")
		defaultValue, hasDefault := field.Tag.Lookup("default")
		envDefault := field.Tag.Get("env_default")
		envDoc := field.Tag.Get("env_doc")
		transform := field.Tag.Get("transform")
		validation := field.Tag.Get("validate")

		// Skip fields without env tag
		if envVar == "" {
			// Recursively handle nested structs
			if field.Type.Kind() == reflect.Struct {
				nestedKey := mapstructureKey
				if parentKey != "" {
					nestedKey = parentKey + "." + mapstructureKey
				}
				nestedFields, err := collectFieldsFromStruct(fieldVal.Addr().Interface(), groupName, nestedKey)
				if err != nil {
					return nil, err
				}
				fields = append(fields, nestedFields...)
			}
			continue
		}

		// Build config key
		configKey := mapstructureKey
		if parentKey != "" {
			configKey = parentKey + "." + mapstructureKey
		}

		// Determine example value (env_default or default)
		exampleValue := envDefault
		if exampleValue == "" {
			exampleValue = defaultValue
		}

		// Detect field type
		fieldType := detectFieldType(field)

		// Build documentation with auto-generated hints
		documentation := envDoc
		if documentation != "" {
			// Add format hints for special types
			if fieldType == "map[string]string" {
				documentation += " (Format: key1=value1,key2=value2)"
			} else if fieldType == "duration" {
				documentation += " (Duration format: 1h30m, 5m, etc.)"
			}

			// Add transform hints
			if transform != "" {
				documentation += fmt.Sprintf(" (Transformed: %s)", describeTransform(transform))
			}

			// Add validation hints
			if validation != "" {
				documentation += fmt.Sprintf(" (Constraints: %s)", describeValidation(validation))
			}
		}

		fields = append(fields, EnvFieldInfo{
			EnvVar:        envVar,
			ConfigKey:     configKey,
			ConfigGroup:   groupName,
			DefaultValue:  defaultValue,
			ExampleValue:  exampleValue,
			Documentation: documentation,
			Required:      !hasDefault, // Field is required if no default tag exists
			FieldType:     fieldType,
			Transform:     transform,
			Validation:    validation,
		})
	}

	return fields, nil
}

// detectFieldType returns a string representation of the field type
func detectFieldType(field reflect.StructField) string {
	t := field.Type
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Check if it's a time.Duration
		if t.String() == "time.Duration" {
			return "duration"
		}
		return "int"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "uint"
	case reflect.Float32, reflect.Float64:
		return "float"
	case reflect.Bool:
		return "bool"
	case reflect.Map:
		if t.Key().Kind() == reflect.String && t.Elem().Kind() == reflect.String {
			return "map[string]string"
		}
		return "map"
	default:
		return t.String()
	}
}

// describeTransform returns a human-readable description of a transform
func describeTransform(transform string) string {
	funcName, _ := parseTagCall(transform)
	switch funcName {
	case "invert":
		return "boolean value is inverted"
	case "ms_to_sec":
		return "milliseconds converted to seconds"
	default:
		return funcName
	}
}

// describeValidation returns a human-readable description of validation constraints
func describeValidation(validation string) string {
	funcName, args := parseTagCall(validation)
	switch funcName {
	case "oneOf":
		if args != nil {
			return fmt.Sprintf("must be one of: %s", *args)
		}
		return "must match allowed values"
	case "enforceRangeFloat", "enforceRangeInt":
		if args != nil {
			return fmt.Sprintf("must be in range %s", *args)
		}
		return "must be within range"
	case "enforceMin":
		if args != nil {
			return fmt.Sprintf("must be >= %s", *args)
		}
		return "must meet minimum value"
	default:
		return validation
	}
}

// GenerateExampleEnv generates .env.example file contents with documentation and examples
// Returns the formatted string content that can be written to a file
func GenerateExampleEnv(groups []ConfigGroup) (string, error) {
	// Collect all environment fields
	fields, err := CollectEnvFields(groups)
	if err != nil {
		return "", fmt.Errorf("failed to collect env fields: %w", err)
	}

	// Group fields by config group
	groupedFields := make(map[string][]EnvFieldInfo)
	for _, field := range fields {
		groupedFields[field.ConfigGroup] = append(groupedFields[field.ConfigGroup], field)
	}

	// Sort fields within each group alphabetically by env var
	for group := range groupedFields {
		sort.Slice(groupedFields[group], func(i, j int) bool {
			return groupedFields[group][i].EnvVar < groupedFields[group][j].EnvVar
		})
	}

	// Build output
	var output strings.Builder

	// Process groups in defined order
	groupOrder := []string{"application"}
	for _, groupName := range groupOrder {
		fields, ok := groupedFields[groupName]
		if !ok || len(fields) == 0 {
			continue
		}

		// Write section header
		output.WriteString(formatEnvSection(groupName))
		output.WriteString("\n")

		// Write fields
		for _, field := range fields {
			output.WriteString(formatEnvField(field))
			output.WriteString("\n")
		}
	}

	return output.String(), nil
}

// formatEnvSection formats a section header for a config group
func formatEnvSection(groupName string) string {
	// Convert group name to title case
	title := strings.ReplaceAll(groupName, "_", " ")
	title = strings.Title(title) + " Configuration"

	separator := strings.Repeat("=", 60)

	return fmt.Sprintf("# %s\n# %s\n# %s\n", separator, title, separator)
}

// formatEnvField formats a single environment variable with documentation
func formatEnvField(field EnvFieldInfo) string {
	var lines []string

	// Add documentation comment if present
	if field.Documentation != "" {
		lines = append(lines, fmt.Sprintf("# %s", field.Documentation))
	}

	// Format the variable assignment
	if field.Required {
		// Required fields are commented out with example value if available
		if field.ExampleValue != "" {
			lines = append(lines, fmt.Sprintf("# %s=%s  # Required", field.EnvVar, field.ExampleValue))
		} else {
			lines = append(lines, fmt.Sprintf("# %s=  # Required", field.EnvVar))
		}
	} else if field.ExampleValue != "" {
		// Optional fields with example values
		lines = append(lines, fmt.Sprintf("%s=%s", field.EnvVar, field.ExampleValue))
	} else {
		// Optional fields without example values (shouldn't happen but handle it)
		lines = append(lines, fmt.Sprintf("# %s=", field.EnvVar))
	}

	return strings.Join(lines, "\n")
}

// ValidateEnv validates a .env file against config requirements
// Process:
//  1. Snapshot current environment for all config-related env vars
//  2. Clear those env vars from current process
//  3. Load .env file into process environment
//  4. Validate loaded values (presence, format, constraints)
//  5. Clear .env file vars from process
//  6. Restore original environment from snapshot
//
// Returns all validation errors found (does NOT stop at first error)
func ValidateEnv(groups []ConfigGroup, envFilePath string) ([]EnvValidationError, error) {
	// Collect all environment fields
	fields, err := CollectEnvFields(groups)
	if err != nil {
		return nil, fmt.Errorf("failed to collect env fields: %w", err)
	}

	// Extract list of env var names
	envVars := make([]string, len(fields))
	for i, field := range fields {
		envVars[i] = field.EnvVar
	}

	// 1. Snapshot current environment
	snapshot := snapshotEnv(envVars)
	defer func() {
		// 6. Always restore environment on exit (defer ensures this happens)
		restoreEnv(snapshot, envVars)
	}()

	// 2. Clear environment
	clearEnv(envVars)

	// 3. Load .env file
	if err := loadEnvFile(envFilePath); err != nil {
		return nil, fmt.Errorf("failed to load .env file: %w", err)
	}

	// 4. Validate all fields
	var errors []EnvValidationError

	for _, field := range fields {
		value := os.Getenv(field.EnvVar)

		// Level 1: Presence check
		if field.Required && value == "" {
			errors = append(errors, EnvValidationError{
				EnvVar:      field.EnvVar,
				ConfigGroup: field.ConfigGroup,
				Issue:       "missing",
				Details:     "required field is not set",
			})
			continue // Skip further validation if missing
		}

		// Skip further validation if optional and not set
		if value == "" {
			continue
		}

		// Level 2: Format validation
		if err := validateFieldFormat(field.EnvVar, value, field.FieldType); err != nil {
			errors = append(errors, EnvValidationError{
				EnvVar:      field.EnvVar,
				ConfigGroup: field.ConfigGroup,
				Issue:       "invalid_format",
				Details:     err.Error(),
			})
			continue // Skip constraint validation if format is invalid
		}

		// Level 3: Constraint validation (if validation tag present)
		if field.Validation != "" {
			parsedValue, err := parseValue(value, field.FieldType)
			if err != nil {
				// This shouldn't happen since we just validated format, but handle it
				errors = append(errors, EnvValidationError{
					EnvVar:      field.EnvVar,
					ConfigGroup: field.ConfigGroup,
					Issue:       "invalid_format",
					Details:     err.Error(),
				})
				continue
			}

			if err := applyValidator(parsedValue, field.Validation); err != nil {
				errors = append(errors, EnvValidationError{
					EnvVar:      field.EnvVar,
					ConfigGroup: field.ConfigGroup,
					Issue:       "constraint_violation",
					Details:     err.Error(),
				})
			}
		}
	}

	// 5. Clear .env file vars (happens in defer before restore)
	clearEnv(envVars)

	return errors, nil
}

// snapshotEnv captures the current values of specified env vars
func snapshotEnv(envVars []string) map[string]string {
	snapshot := make(map[string]string)
	for _, envVar := range envVars {
		value, exists := os.LookupEnv(envVar)
		if exists {
			snapshot[envVar] = value
		}
	}
	return snapshot
}

// restoreEnv restores env vars from a snapshot
func restoreEnv(snapshot map[string]string, allVars []string) {
	// First clear all vars
	for _, envVar := range allVars {
		os.Unsetenv(envVar)
	}

	// Then restore values from snapshot
	for envVar, value := range snapshot {
		os.Setenv(envVar, value)
	}
}

// clearEnv clears the specified env vars from the process
func clearEnv(envVars []string) {
	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}
}

// loadEnvFile loads a .env file into the process environment
func loadEnvFile(filePath string) error {
	return godotenv.Load(filePath)
}

// validateFieldFormat validates that a value can be parsed to the expected type
func validateFieldFormat(envVar, value, fieldType string) error {
	switch fieldType {
	case "string":
		return nil // Strings are always valid
	case "int":
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			return fmt.Errorf("invalid integer: %w", err)
		}
	case "uint":
		if _, err := strconv.ParseUint(value, 10, 64); err != nil {
			return fmt.Errorf("invalid unsigned integer: %w", err)
		}
	case "float":
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return fmt.Errorf("invalid float: %w", err)
		}
	case "bool":
		if _, err := strconv.ParseBool(value); err != nil {
			return fmt.Errorf("invalid boolean: %w", err)
		}
	case "duration":
		if _, err := time.ParseDuration(value); err != nil {
			return fmt.Errorf("invalid duration: %w", err)
		}
	case "map[string]string":
		// Basic validation: should contain at least one '=' if not empty
		if value != "" && !strings.Contains(value, "=") {
			return fmt.Errorf("invalid map format: expected key=value pairs")
		}
	}
	return nil
}

// parseValue parses a string value to the appropriate type
func parseValue(value, fieldType string) (interface{}, error) {
	switch fieldType {
	case "string":
		return value, nil
	case "int":
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, err
		}
		return int(i), nil
	case "uint":
		return strconv.ParseUint(value, 10, 64)
	case "float":
		return strconv.ParseFloat(value, 64)
	case "bool":
		return strconv.ParseBool(value)
	case "duration":
		return time.ParseDuration(value)
	case "map[string]string":
		// Return as string for now - validation will be done by constraint validators
		return value, nil
	default:
		return value, nil
	}
}

// applyValidator applies a validation constraint to a value
func applyValidator(value interface{}, validatorTag string) error {
	funcName, args := parseTagCall(validatorTag)

	validatorFunc, exists := ValidatorRegistry[funcName]
	if !exists {
		return fmt.Errorf("unknown validator: %s", funcName)
	}

	return validatorFunc(value, args)
}
