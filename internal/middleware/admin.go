package middleware

import (
	"net/http"

	"github.com/Y4shin/conference-tool/internal/session"
)

// adminRequired middleware blocks requests that don't have an admin account session.
// Redirects unauthenticated or non-admin requests to the admin login page.
func (r *Registry) adminRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		sd, ok := session.GetSession(req.Context())
		if !ok || sd.IsExpired() || !sd.IsAdmin {
			http.Redirect(w, req, "/admin/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, req)
	})
}
