package config

import (
	"fmt"
	"reflect"
	"strings"
)

// TransformFunc applies a transformation to a config value with optional arguments
// args is nil if no arguments, otherwise points to the raw argument string
type TransformFunc func(value interface{}, args *string) (interface{}, error)

// TransformRegistry maps transform names to functions
// Initially empty - transforms can be added as needed
var TransformRegistry = map[string]TransformFunc{
	// Empty initially - transforms like "invert" or "ms_to_sec" can be added here as needed
}

// parseTagCallTransform parses a tag value like "funcName(args)"
// Returns function name and pointer to args string (or nil if no args)
func parseTagCallTransform(tagValue string) (funcName string, args *string) {
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

// ApplyTransforms applies registered transforms based on struct tags
func ApplyTransforms(cfg interface{}) error {
	val := reflect.ValueOf(cfg)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		if !fieldVal.CanSet() {
			continue
		}

		// Apply transform if present
		transformTag := field.Tag.Get("transform")
		if transformTag != "" {
			// Parse function name and arguments
			funcName, args := parseTagCallTransform(transformTag)

			transformFunc, exists := TransformRegistry[funcName]
			if !exists {
				return fmt.Errorf("unknown transform: %s", funcName)
			}

			transformed, err := transformFunc(fieldVal.Interface(), args)
			if err != nil {
				return fmt.Errorf("transform %s failed on field %s: %w", funcName, field.Name, err)
			}

			fieldVal.Set(reflect.ValueOf(transformed))
		}

		// Recursively apply to nested structs (regardless of transform tag)
		if field.Type.Kind() == reflect.Struct {
			if err := ApplyTransforms(fieldVal.Addr().Interface()); err != nil {
				return err
			}
		}
	}

	return nil
}
