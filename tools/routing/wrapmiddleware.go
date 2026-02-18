package routing

type WrapMiddleware struct {
}

const wrapMiddleware string = `// wrapMiddleware chains middleware in the correct order
// prefixMiddleware comes from matching prefixes, routeMiddleware from the route definition
func (rt *Router) wrapMiddleware(
	handler http.HandlerFunc,
	prefixMiddleware []string,
	routeMiddleware []string,
	sse bool,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set SSE headers if needed
		if sse {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
		}

		h := http.Handler(handler)

		// Combine middleware: prefix middleware first, then route-specific
		allMiddleware := append(prefixMiddleware, routeMiddleware...)

		// Apply middleware in reverse order so they execute in declaration order
		for i := len(allMiddleware) - 1; i >= 0; i-- {
			middleware := rt.middleware.Get(allMiddleware[i])
			if middleware != nil {
				h = middleware(h)
			}
		}

		h.ServeHTTP(w, r)
	}
}`

func GetWrapMiddleware(config *RouteConfig) WrapMiddleware {
	return WrapMiddleware{}
}
