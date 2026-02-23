package middleware

import (
	"net/http"

	"github.com/Y4shin/conference-tool/internal/session"
)

// adminRequired middleware blocks requests that don't have an admin session.
// Redirects unauthenticated requests to the admin login page.
func (r *Registry) adminRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		sd, ok := session.GetSession(req.Context())
		if !ok || !sd.IsAdminSession() {
			http.Redirect(w, req, "/admin/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, req)
	})
}
