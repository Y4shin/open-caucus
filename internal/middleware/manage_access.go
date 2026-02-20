package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Y4shin/conference-tool/internal/session"
)

// manageAccess allows meeting-manage endpoints for:
// 1) logged-in committee users with role "chairperson"
// 2) attendee sessions that are chairpersons of the same meeting
func (r *Registry) manageAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		sd, ok := session.GetSession(req.Context())
		if !ok || sd.IsExpired() {
			http.Redirect(w, req, "/", http.StatusSeeOther)
			return
		}

		if sd.IsUserSession() {
			if sd.Role != nil && *sd.Role == "chairperson" {
				next.ServeHTTP(w, req)
				return
			}
			http.Error(w, "Forbidden: chairperson role required", http.StatusForbidden)
			return
		}

		if !sd.IsAttendeeSession() || sd.MeetingID == nil || sd.IsChair == nil || !*sd.IsChair {
			http.Error(w, "Forbidden: meeting chair attendee required", http.StatusForbidden)
			return
		}
		if sd.AttendeeID == nil || r.Repository == nil {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		pathParts := strings.Split(strings.TrimPrefix(req.URL.Path, "/"), "/")
		if len(pathParts) < 4 || pathParts[0] != "committee" || pathParts[2] != "meeting" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		meetingID, err := strconv.ParseInt(pathParts[3], 10, 64)
		if err != nil || meetingID != *sd.MeetingID {
			http.Error(w, "Forbidden: meeting mismatch", http.StatusForbidden)
			return
		}

		attendee, err := r.Repository.GetAttendeeByID(req.Context(), *sd.AttendeeID)
		if err != nil || attendee.MeetingID != meetingID || !attendee.IsChair {
			http.Error(w, "Forbidden: meeting chair attendee required", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, req)
	})
}
