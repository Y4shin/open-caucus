package routing

// RouteConfig represents the entire YAML configuration
type RouteConfig struct {
	Version          string             `yaml:"version"`
	MiddlewareGroups []MiddlewareGroup  `yaml:"middleware_groups"`
	Routes           []Route            `yaml:"routes"`
}

// MiddlewareGroup defines middleware applied to path prefixes
type MiddlewareGroup struct {
	Prefix     string   `yaml:"prefix"`
	Middleware []string `yaml:"middleware"`
}

// Route represents a single route with multiple HTTP methods
type Route struct {
	Path    string        `yaml:"path"`
	Methods []RouteMethod `yaml:"methods"`
}

// RouteMethod represents a specific HTTP verb handler for a route
type RouteMethod struct {
	Verb       string   `yaml:"verb"`
	Handler    string   `yaml:"handler"`
	Template   Template `yaml:"template"`
	Middleware []string `yaml:"middleware,omitempty"`
	SSE        bool     `yaml:"sse,omitempty"`
}

// Template specifies which templ template to use
type Template struct {
	Package   string `yaml:"package"`
	Type      string `yaml:"type"`
	InputType string `yaml:"input_type"`
}
