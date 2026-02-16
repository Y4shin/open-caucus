package config

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

// BindConfigToViper uses reflection to read struct tags and automatically:
// 1. Set defaults from `default` tags
// 2. Bind environment variables from `env` tags
// 3. Build the parent key path from mapstructure tags
func BindConfigToViper(v *viper.Viper, cfg interface{}, parentKey string) error {
	val := reflect.ValueOf(cfg)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)

		// Get tag values
		mapstructureKey := field.Tag.Get("mapstructure")
		envVar := field.Tag.Get("env")
		defaultVal := field.Tag.Get("default")

		if mapstructureKey == "" {
			continue // Skip fields without mapstructure tag
		}

		// Build full config key
		fullKey := fmt.Sprintf("%s.%s", parentKey, mapstructureKey)

		// Set default if provided
		if defaultVal != "" {
			parsedDefault, err := parseDefault(defaultVal, field.Type)
			if err != nil {
				return fmt.Errorf("failed to parse default for %s.%s: %w", parentKey, mapstructureKey, err)
			}
			v.SetDefault(fullKey, parsedDefault)
		}

		// Bind environment variable if provided
		if envVar != "" {
			if err := v.BindEnv(fullKey, envVar); err != nil {
				return fmt.Errorf("failed to bind env var %s to %s: %w", envVar, fullKey, err)
			}
		}

		// Recursively handle nested structs
		if field.Type.Kind() == reflect.Struct && field.Type != reflect.TypeOf(time.Duration(0)) {
			nestedVal := val.Field(i)
			if err := BindConfigToViper(v, nestedVal.Addr().Interface(), fullKey); err != nil {
				return err
			}
		}
	}

	return nil
}

// parseDefault converts string default to appropriate type
func parseDefault(s string, t reflect.Type) (interface{}, error) {
	switch t.Kind() {
	case reflect.String:
		return s, nil
	case reflect.Bool:
		return s == "true", nil
	case reflect.Int:
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, err
		}
		return int(i), nil
	case reflect.Int32:
		i, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return nil, err
		}
		return int32(i), nil
	case reflect.Int64:
		// Special handling for time.Duration
		if t == reflect.TypeOf(time.Duration(0)) {
			d, err := time.ParseDuration(s)
			if err != nil {
				return nil, err
			}
			return d, nil
		}
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, err
		}
		return i, nil
	case reflect.Uint16:
		u, err := strconv.ParseUint(s, 10, 16)
		if err != nil {
			return nil, err
		}
		return uint16(u), nil
	case reflect.Uint32:
		u, err := strconv.ParseUint(s, 10, 32)
		if err != nil {
			return nil, err
		}
		return uint32(u), nil
	case reflect.Float32:
		f, err := strconv.ParseFloat(s, 32)
		if err != nil {
			return nil, err
		}
		return float32(f), nil
	case reflect.Float64:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, err
		}
		return f, nil
	default:
		return s, nil
	}
}
