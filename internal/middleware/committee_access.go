package middleware

import (
	"net/http"
	"strings"

	"github.com/Y4shin/conference-tool/internal/session"
)

// committeeAccess middleware ensures the user can only access their own committee's data
// Compares the committee slug in the URL with the user's session committee
func (r *Registry) committeeAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Get session from context (auth middleware guarantees this exists)
		sessionData, ok := session.GetSession(req.Context())
		if !ok {
			// Should never happen if auth middleware is applied first
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Extract slug from URL path
		// Expected format: /committee/{slug} or /committee/{slug}/...
		pathParts := strings.Split(strings.TrimPrefix(req.URL.Path, "/"), "/")
		if len(pathParts) < 2 || pathParts[0] != "committee" {
			// Not a committee path - this middleware shouldn't be applied here
			next.ServeHTTP(w, req)
			return
		}

		slugFromPath := pathParts[1]

		// Compare with session committee slug (only for user sessions)
		if sessionData.IsUserSession() {
			if sessionData.CommitteeSlug == nil || slugFromPath != *sessionData.CommitteeSlug {
				// User trying to access a different committee
				http.Error(w, "Forbidden: You don't have access to this committee", http.StatusForbidden)
				return
			}
		}
		// Note: Attendee sessions will be validated differently (by meeting_id)

		// Access granted - continue
		next.ServeHTTP(w, req)
	})
}
