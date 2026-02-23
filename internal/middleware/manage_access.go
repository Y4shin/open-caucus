package middleware

import (
	"net/http"

	"github.com/Y4shin/conference-tool/internal/session"
)

// manageAccess allows meeting-manage endpoints for:
// 1) logged-in committee users with role "chairperson"
// 2) attendees (populated by meeting_access) that are meeting chairpersons
func (r *Registry) manageAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		sd, ok := session.GetSession(req.Context())
		if !ok || sd.IsExpired() {
			http.Redirect(w, req, "/", http.StatusSeeOther)
			return
		}

		// Chairperson role from committee membership (account session)
		if cu, ok := session.GetCurrentUser(req.Context()); ok && cu.Role == "chairperson" {
			next.ServeHTTP(w, req)
			return
		}

		// Chair attendee (populated by meeting_access for both account and guest sessions)
		if ca, ok := session.GetCurrentAttendee(req.Context()); ok && ca.IsChair {
			next.ServeHTTP(w, req)
			return
		}

		http.Error(w, "Forbidden: meeting chair required", http.StatusForbidden)
	})
}
