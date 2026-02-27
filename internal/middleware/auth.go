package middleware

import (
	"log/slog"
	"net/http"

	"github.com/Y4shin/conference-tool/internal/session"
)

// authRequired middleware blocks requests that don't have a valid session
// Redirects unauthenticated users to the login page
func (r *Registry) authRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Get session from context
		sessionData, ok := session.GetSession(req.Context())
		if !ok || sessionData.IsExpired() {
			slog.Debug("unauthenticated request redirected to login", "path", req.URL.Path)
			http.Redirect(w, req, "/", http.StatusSeeOther)
			return
		}

		// Session is valid - continue
		next.ServeHTTP(w, req)
	})
}
