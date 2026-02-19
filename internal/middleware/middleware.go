package middleware

import (
	"log"
	"net/http"

	"github.com/Y4shin/conference-tool/internal/session"
)

// Registry provides middleware functions by name
type Registry struct {
	SessionManager      *session.Manager
	AdminSessionManager *session.AdminSessionManager
}

// NewRegistry creates a new middleware registry
func NewRegistry(sessionManager *session.Manager, adminSessionManager *session.AdminSessionManager) *Registry {
	return &Registry{
		SessionManager:      sessionManager,
		AdminSessionManager: adminSessionManager,
	}
}

// Get returns the middleware function for the given name
func (r *Registry) Get(name string) func(http.Handler) http.Handler {
	switch name {
	case "session":
		return r.sessionMiddleware
	case "auth":
		return r.authRequired
	case "committee_access":
		return r.committeeAccess
	case "admin_session":
		return r.adminSession
	case "admin_required":
		return r.adminRequired
	case "attendee_session":
		return r.attendeeSession
	case "attendee_required":
		return r.attendeeRequired
	default:
		return r.defaultLogger(name)
	}
}

// defaultLogger returns a simple logging middleware
func (r *Registry) defaultLogger(name string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			log.Printf("[%s] %s %s", name, req.Method, req.URL.Path)
			next.ServeHTTP(w, req)
		})
	}
}
