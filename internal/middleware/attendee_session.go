package middleware

import (
	"net/http"

	"github.com/Y4shin/open-caucus/internal/session"
)

// attendeeSession loads the session from the cookie and adds it to context,
// identically to sessionMiddleware. It is provided as a named middleware so
// that routes requiring attendee context can be annotated clearly in routes.yaml.
func (r *Registry) attendeeSession(next http.Handler) http.Handler {
	return r.sessionMiddleware(next)
}

// attendeeRequired blocks requests that do not have a CurrentAttendee in context.
// This is satisfied by both guest sessions (set in sessionMiddleware) and account
// sessions with an attendee record (set by meeting_access).
func (r *Registry) attendeeRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if _, ok := session.GetCurrentAttendee(req.Context()); !ok {
			http.Error(w, "Forbidden: attendee context required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, req)
	})
}
