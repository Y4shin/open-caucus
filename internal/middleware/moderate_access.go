package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Y4shin/conference-tool/internal/session"
)

// moderateAccess allows meeting-moderate endpoints for:
// 1) logged-in committee users with role "chairperson"
// 2) attendees (populated by meeting_access) that are meeting chairpersons
// 3) attendees that are the designated meeting moderator
func (r *Registry) moderateAccess(next http.Handler) http.Handler {
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

		// Attendee context (populated by meeting_access for both session types)
		ca, caOK := session.GetCurrentAttendee(req.Context())
		if !caOK {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Chair attendees are always allowed
		if ca.IsChair {
			next.ServeHTTP(w, req)
			return
		}

		// Extract meeting ID from URL to look up the designated moderator
		pathParts := strings.Split(strings.TrimPrefix(req.URL.Path, "/"), "/")
		if len(pathParts) < 4 || pathParts[0] != "committee" || pathParts[2] != "meeting" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		meetingID, err := strconv.ParseInt(pathParts[3], 10, 64)
		if err != nil {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		meeting, err := r.Repository.GetMeetingByID(req.Context(), meetingID)
		if err != nil {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		if meeting.ModeratorID != nil && *meeting.ModeratorID == ca.AttendeeID {
			next.ServeHTTP(w, req)
			return
		}

		http.Error(w, "Forbidden: moderator access required", http.StatusForbidden)
	})
}
