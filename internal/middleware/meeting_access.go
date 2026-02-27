package middleware

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/Y4shin/conference-tool/internal/session"
)

// meetingAccess validates meeting-level access and, when possible, populates
// CurrentAttendee in context.
//
// For account sessions: looks up the attendee by CurrentUser.UserID + meetingID from the URL.
//   - Found → populates CurrentAttendee in context and continues.
//   - Not found → continues without CurrentAttendee (downstream middleware/handler decides).
//
// For guest sessions: validates that CurrentAttendee.MeetingID matches the URL meeting ID.
//   - Mismatch → 403 Forbidden.
func (r *Registry) meetingAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		sd, ok := session.GetSession(req.Context())
		if !ok || sd.IsExpired() {
			http.Redirect(w, req, "/", http.StatusSeeOther)
			return
		}

		slug, meetingIDStr, ok := extractSlugAndMeetingID(req.URL.Path)
		if !ok {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if sd.IsAccountSession() {
			if cu, cuOK := session.GetCurrentUser(req.Context()); cuOK {
				if cu.Role == "member" {
					committee, err := r.Repository.GetCommitteeBySlug(req.Context(), slug)
					if err != nil || committee.CurrentMeetingID == nil || *committee.CurrentMeetingID != meetingID {
						slog.Warn("member meeting access denied: meeting not active", "slug", slug, "meeting_id", meetingID)
						http.Error(w, "Forbidden", http.StatusForbidden)
						return
					}
				}

				attendee, err := r.Repository.GetAttendeeByUserIDAndMeetingID(req.Context(), cu.UserID, meetingID)
				if err == nil {
					ctx := session.WithCurrentAttendee(req.Context(), &session.CurrentAttendee{
						AttendeeID: attendee.ID,
						MeetingID:  attendee.MeetingID,
						FullName:   attendee.FullName,
						IsChair:    attendee.IsChair,
						Quoted:     attendee.Quoted,
					})
					req = req.WithContext(ctx)
				}
				// No attendee found: continue without CurrentAttendee.
				// Downstream middleware (moderate_access, attendee_required)
				// or the handler itself decides whether to allow or reject.
			}
		}

		if sd.IsGuestSession() {
			ca, ok := session.GetCurrentAttendee(req.Context())
			if !ok || ca.MeetingID != meetingID {
				slog.Warn("meeting access denied for guest session", "meeting_id", meetingID, "path", req.URL.Path)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, req)
	})
}

// extractSlugAndMeetingID parses /committee/{slug}/meeting/{meetingID}/... from the path.
func extractSlugAndMeetingID(path string) (slug, meetingIDStr string, ok bool) {
	// Strip leading slash and split
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	// Expect: committee / {slug} / meeting / {id} / ...
	if len(parts) < 4 || parts[0] != "committee" || parts[2] != "meeting" {
		return "", "", false
	}
	return parts[1], parts[3], true
}
