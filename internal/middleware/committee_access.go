package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/Y4shin/conference-tool/internal/session"
)

// committeeAccess middleware ensures the user can access the requested committee.
// For account sessions: looks up the user's membership by account_id + slug and
// populates a full CurrentUser in context. Returns 403 if no membership found.
// For guest sessions: passes through unchanged (meeting_access validates the meeting).
func (r *Registry) committeeAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		sd, ok := session.GetSession(req.Context())
		if !ok {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Extract slug from URL path: /committee/{slug}/...
		pathParts := strings.Split(strings.TrimPrefix(req.URL.Path, "/"), "/")
		if len(pathParts) < 2 || pathParts[0] != "committee" {
			next.ServeHTTP(w, req)
			return
		}
		slugFromPath := pathParts[1]

		if sd.IsAccountSession() && sd.AccountID != nil {
			membership, err := r.Repository.GetUserMembershipByAccountIDAndSlug(req.Context(), *sd.AccountID, slugFromPath)
			if err != nil {
				slog.Warn("committee access denied: no membership", "account_id", *sd.AccountID, "slug", slugFromPath)
				http.Error(w, "Forbidden: no committee membership", http.StatusForbidden)
				return
			}
			ctx := session.WithCurrentUser(req.Context(), &session.CurrentUser{
				UserID:        membership.ID,
				Username:      membership.Username,
				CommitteeSlug: membership.CommitteeSlug,
				Role:          membership.Role,
				Quoted:        membership.Quoted,
			})
			req = req.WithContext(ctx)
		}
		// Guest sessions pass through; meeting_access validates the attendee's meeting

		next.ServeHTTP(w, req)
	})
}
