package routing

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseConfig reads and parses a YAML route configuration file
func ParseConfig(filename string) (*RouteConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config RouteConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// validateConfig performs validation on the parsed configuration
func validateConfig(config *RouteConfig) error {
	if config.Version == "" {
		return fmt.Errorf("version is required")
	}

	// Validate routes
	seenPaths := make(map[string]map[string]bool) // path -> verb -> exists
	for i, route := range config.Routes {
		if route.Path == "" {
			return fmt.Errorf("route[%d]: path is required", i)
		}

		if !strings.HasPrefix(route.Path, "/") {
			return fmt.Errorf("route[%d]: path must start with /", i)
		}

		if len(route.Methods) == 0 {
			return fmt.Errorf("route[%d]: at least one method is required", i)
		}

		if seenPaths[route.Path] == nil {
			seenPaths[route.Path] = make(map[string]bool)
		}

		for j, method := range route.Methods {
			if method.Verb == "" {
				return fmt.Errorf("route[%d].method[%d]: verb is required", i, j)
			}

			verb := strings.ToUpper(method.Verb)
			if seenPaths[route.Path][verb] {
				return fmt.Errorf("route[%d]: duplicate handler for %s %s", i, verb, route.Path)
			}
			seenPaths[route.Path][verb] = true

			if method.Handler == "" {
				return fmt.Errorf("route[%d].method[%d]: handler is required", i, j)
			}

			if method.Template.Package == "" {
				return fmt.Errorf("route[%d].method[%d]: template package is required", i, j)
			}

			if method.Template.Type == "" {
				return fmt.Errorf("route[%d].method[%d]: template type is required", i, j)
			}

			if method.Template.InputType == "" {
				return fmt.Errorf("route[%d].method[%d]: template input_type is required", i, j)
			}
		}
	}

	return nil
}

// ExtractPathParams extracts parameter names from a path like /posts/{id}/comments/{commentId}
func ExtractPathParams(path string) []string {
	re := regexp.MustCompile(`\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(path, -1)

	var params []string
	for _, match := range matches {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}

	return params
}

// ToPascalCase converts a string to PascalCase (e.g., "user_id" -> "UserId")
func ToPascalCase(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-'
	})

	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, "")
}

// ToCamelCase converts a string to camelCase (e.g., "user_id" -> "userId")
func ToCamelCase(s string) string {
	pascal := ToPascalCase(s)
	if len(pascal) == 0 {
		return pascal
	}
	return strings.ToLower(pascal[:1]) + pascal[1:]
}
