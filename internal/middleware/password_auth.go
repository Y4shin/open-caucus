package middleware

import "net/http"

// passwordAuthEnabled ensures password-login endpoints are disabled with 404 when configured off.
func (r *Registry) passwordAuthEnabled(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if !r.PasswordEnabled {
			http.NotFound(w, req)
			return
		}
		next.ServeHTTP(w, req)
	})
}

