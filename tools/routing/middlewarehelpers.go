package routing

type MiddlewareHelpers struct {
}

const middlewareHelpers string = `// middlewareGroup defines middleware applied to path prefixes
type middlewareGroup struct {
	prefix     string
	middleware []string
}

// getMiddlewareForPath returns all middleware that should apply to a path
// by matching against all prefix groups
func getMiddlewareForPath(path string, groups []middlewareGroup) []string {
	var middleware []string

	// Apply middleware from all matching prefix groups
	for _, group := range groups {
		if strings.HasPrefix(path, group.prefix) {
			middleware = append(middleware, group.middleware...)
		}
	}

	return middleware
}`

func GetMiddlewareHelpers(config *RouteConfig) MiddlewareHelpers {
	return MiddlewareHelpers{}
}
