package routing

type RouterStruct struct {
}

const routerStruct string = `// Router wraps the handler and provides route registration
type Router struct {
	handler    Handler
	middleware MiddlewareRegistry
	mux        *http.ServeMux
}

func NewRouter(handler Handler, middleware MiddlewareRegistry) *Router {
	return &Router{
		handler:    handler,
		middleware: middleware,
		mux:        http.NewServeMux(),
	}
}`

func GetRouterStruct(config *RouteConfig) RouterStruct {
	return RouterStruct{}
}
