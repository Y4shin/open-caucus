package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// ValidatorFunc validates a config value with optional arguments
// args is nil if no arguments, otherwise points to the raw argument string
type ValidatorFunc func(value interface{}, args *string) error

// ValidatorRegistry maps validator names to functions
var ValidatorRegistry = map[string]ValidatorFunc{
	"enforceRangeInt": enforceRangeInt,
	"enforceMin":      enforceMin,
	"oneOf":           oneOf,
}

// enforceRangeInt validates an int is within range
// Usage: validate:"enforceRangeInt(1:65535)"
func enforceRangeInt(v interface{}, args *string) error {
	if args == nil {
		return fmt.Errorf("enforceRangeInt requires argument (min:max)")
	}

	// Handle different int types
	var i int64
	switch val := v.(type) {
	case int:
		i = int64(val)
	case int32:
		i = int64(val)
	case int64:
		i = val
	case uint16:
		i = int64(val)
	default:
		return fmt.Errorf("enforceRangeInt expects int type, got %T", v)
	}

	parts := strings.Split(*args, ":")
	if len(parts) != 2 {
		return fmt.Errorf("enforceRangeInt argument must be min:max format")
	}

	min, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid min value: %w", err)
	}

	max, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid max value: %w", err)
	}

	if i < min || i > max {
		return fmt.Errorf("value %d must be between %d and %d", i, min, max)
	}

	return nil
}

// enforceMin validates value is at least minimum
// Usage: validate:"enforceMin(1)"
func enforceMin(v interface{}, args *string) error {
	if args == nil {
		return fmt.Errorf("enforceMin requires argument")
	}

	var i int64
	switch val := v.(type) {
	case int:
		i = int64(val)
	case int32:
		i = int64(val)
	case int64:
		i = val
	default:
		return fmt.Errorf("enforceMin expects int type, got %T", v)
	}

	min, err := strconv.ParseInt(*args, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid min value: %w", err)
	}

	if i < min {
		return fmt.Errorf("value %d must be at least %d", i, min)
	}

	return nil
}

// oneOf validates string is one of allowed values
// Usage: validate:"oneOf(console,otlp,multi,none)"
func oneOf(v interface{}, args *string) error {
	if args == nil {
		return fmt.Errorf("oneOf requires argument")
	}

	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("oneOf expects string, got %T", v)
	}

	allowed := strings.Split(*args, ",")
	for _, allowedVal := range allowed {
		if s == strings.TrimSpace(allowedVal) {
			return nil
		}
	}

	return fmt.Errorf("value %q must be one of: %s", s, *args)
}

// parseTagCall parses a tag value like "funcName(args)"
// Returns function name and pointer to args string (or nil if no args)
func parseTagCall(tagValue string) (funcName string, args *string) {
	if !strings.Contains(tagValue, "(") {
		return tagValue, nil
	}

	parts := strings.SplitN(tagValue, "(", 2)
	funcName = strings.TrimSpace(parts[0])

	if len(parts) == 2 {
		argsStr := strings.TrimSuffix(strings.TrimSpace(parts[1]), ")")
		if argsStr != "" {
			return funcName, &argsStr
		}
	}

	return funcName, nil
}

// ValidateConfig validates all fields with validate tags
func ValidateConfig(cfg interface{}) error {
	val := reflect.ValueOf(cfg)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Validate if tag is present
		validateTag := field.Tag.Get("validate")
		if validateTag != "" {
			// Parse function name and arguments
			funcName, args := parseTagCall(validateTag)

			validatorFunc, exists := ValidatorRegistry[funcName]
			if !exists {
				return fmt.Errorf("unknown validator: %s", funcName)
			}

			if err := validatorFunc(fieldVal.Interface(), args); err != nil {
				return fmt.Errorf("validation failed for field %s: %w", field.Name, err)
			}
		}

		// Recursively validate nested structs (regardless of validate tag)
		if field.Type.Kind() == reflect.Struct {
			if err := ValidateConfig(fieldVal.Addr().Interface()); err != nil {
				return err
			}
		}
	}

	return nil
}
