package middleware

import (
	"net/http"

	"github.com/Y4shin/conference-tool/internal/session"
)

// adminSession middleware extracts admin session cookie and validates it
// This middleware does not block requests - it only adds admin auth to context if valid
func (r *Registry) adminSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Try to read admin session cookie
		cookie, err := req.Cookie("admin_session")
		if err == nil && cookie.Value != "" {
			// Validate admin session
			if r.AdminSessionManager.ValidateAdminSession(cookie.Value) {
				// Mark context as admin authenticated
				ctx := session.WithAdminAuth(req.Context())
				req = req.WithContext(ctx)
			}
		}

		// Always continue to next handler
		next.ServeHTTP(w, req)
	})
}

// adminRequired middleware blocks requests that don't have admin authentication
// Redirects unauthenticated requests to the admin login page
func (r *Registry) adminRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Check if admin authenticated
		if !session.IsAdminAuthenticated(req.Context()) {
			// Not authenticated - redirect to admin login
			http.Redirect(w, req, "/admin/login", http.StatusSeeOther)
			return
		}

		// Admin authenticated - continue
		next.ServeHTTP(w, req)
	})
}
