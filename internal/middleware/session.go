package middleware

import (
	"net/http"

	"github.com/Y4shin/conference-tool/internal/session"
)

// sessionMiddleware extracts session data from cookies and adds it to the request context.
// For account sessions: fetches the Account from DB to populate IsAdmin and a partial CurrentUser.
// For guest sessions: fetches the Attendee from DB to populate CurrentAttendee.
// This middleware does not block requests — it only adds session data if present and valid.
func (r *Registry) sessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		cookie, err := req.Cookie("session_id")
		if err != nil || cookie.Value == "" {
			next.ServeHTTP(w, req)
			return
		}

		sessionData, err := r.SessionManager.GetSession(req.Context(), cookie.Value)
		if err != nil || sessionData.IsExpired() {
			next.ServeHTTP(w, req)
			return
		}

		ctx := session.WithSession(req.Context(), sessionData)

		if sessionData.IsAccountSession() && sessionData.AccountID != nil {
			account, err := r.Repository.GetAccountByID(ctx, *sessionData.AccountID)
			if err == nil {
				sessionData.IsAdmin = account.IsAdmin
				ctx = session.WithCurrentUser(ctx, &session.CurrentUser{
					Username: account.Username,
					// CommitteeSlug, Role, Quoted, UserID filled by committee_access
				})
			}
		}

		if sessionData.IsGuestSession() && sessionData.AttendeeID != nil {
			attendee, err := r.Repository.GetAttendeeByID(ctx, *sessionData.AttendeeID)
			if err == nil {
				ctx = session.WithCurrentAttendee(ctx, &session.CurrentAttendee{
					AttendeeID: attendee.ID,
					MeetingID:  attendee.MeetingID,
					FullName:   attendee.FullName,
					IsChair:    attendee.IsChair,
					Quoted:     attendee.Quoted,
				})
			}
		}

		req = req.WithContext(ctx)
		next.ServeHTTP(w, req)
	})
}
