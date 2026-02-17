package session

import (
	"context"
	"time"
)

// SessionType represents the type of session
type SessionType string

const (
	SessionTypeUser     SessionType = "user"
	SessionTypeAttendee SessionType = "attendee"
)

// SessionData contains information about an authenticated session
// This is the lightweight version used in request context
type SessionData struct {
	SessionType SessionType

	// For user sessions
	UserID        *int64
	CommitteeSlug *string
	Username      *string
	Role          *string

	// For attendee (guest) sessions
	AttendeeID *int64
	MeetingID  *int64
	FullName   *string
	IsChair    *bool

	ExpiresAt time.Time
}

// IsExpired checks if the session has expired
func (s *SessionData) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsUserSession returns true if this is a user session
func (s *SessionData) IsUserSession() bool {
	return s.SessionType == SessionTypeUser
}

// IsAttendeeSession returns true if this is an attendee/guest session
func (s *SessionData) IsAttendeeSession() bool {
	return s.SessionType == SessionTypeAttendee
}

// Context keys for storing session data in request context
type contextKey int

const sessionContextKey contextKey = 0

// WithSession adds session data to the context
func WithSession(ctx context.Context, session *SessionData) context.Context {
	return context.WithValue(ctx, sessionContextKey, session)
}

// GetSession retrieves session data from the context
func GetSession(ctx context.Context) (*SessionData, bool) {
	session, ok := ctx.Value(sessionContextKey).(*SessionData)
	return session, ok
}
