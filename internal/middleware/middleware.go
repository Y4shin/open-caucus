package middleware

import (
	"log"
	"net/http"
)

// Registry is a dummy middleware registry that returns
// a simple logging wrapper for any requested middleware.
type Registry struct{}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) Get(name string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			log.Printf("[%s] %s %s", name, req.Method, req.URL.Path)
			next.ServeHTTP(w, req)
		})
	}
}
