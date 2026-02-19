package middleware

import (
	"net/http"

	"github.com/Y4shin/conference-tool/internal/session"
)

// attendeeSession loads the session from the cookie and adds it to context,
// identically to sessionMiddleware. It is provided as a named middleware so
// that routes requiring attendee context can be annotated clearly in routes.yaml.
func (r *Registry) attendeeSession(next http.Handler) http.Handler {
	return r.sessionMiddleware(next)
}

// attendeeRequired blocks requests that do not carry a valid, non-expired
// attendee session. Non-attendee requests receive a 403 Forbidden response.
func (r *Registry) attendeeRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		sessionData, ok := session.GetSession(req.Context())
		if !ok || sessionData.IsExpired() || !sessionData.IsAttendeeSession() {
			http.Error(w, "Forbidden: attendee session required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, req)
	})
}
