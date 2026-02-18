package routing

type MiddlewareRegistry struct {
}

const middlewareRegistry string = `// MiddlewareRegistry provides middleware functions by name
type MiddlewareRegistry interface {
	Get(name string) func(http.Handler) http.Handler
}`

func GetMiddlewareRegistry(config *RouteConfig) MiddlewareRegistry {
	return MiddlewareRegistry{}
}
