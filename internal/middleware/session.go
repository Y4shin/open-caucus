package middleware

import (
	"net/http"

	"github.com/Y4shin/conference-tool/internal/session"
)

// sessionMiddleware extracts session data from cookies and adds it to the request context
// This middleware does not block requests - it only adds session data if present and valid
func (r *Registry) sessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Try to read session cookie
		cookie, err := req.Cookie("session_id")
		if err == nil && cookie.Value != "" {
			// Validate and load session
			sessionData, err := r.SessionManager.GetSession(req.Context(), cookie.Value)
			if err == nil && !sessionData.IsExpired() {
				// Add session to context
				ctx := session.WithSession(req.Context(), sessionData)
				req = req.WithContext(ctx)
			}
		}

		// Always continue to next handler (even if no session)
		next.ServeHTTP(w, req)
	})
}
