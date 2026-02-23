package model

import "time"

// SessionType represents the type of session
type SessionType string

const (
	SessionTypeAccount SessionType = "account"
	SessionTypeGuest   SessionType = "guest"
)

// Session represents an authentication session
type Session struct {
	SessionID   string
	SessionType SessionType

	// For account sessions (logged-in users)
	AccountID *int64

	// For guest sessions (attendees without an account)
	AttendeeID *int64

	CreatedAt time.Time
	ExpiresAt time.Time
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsAccountSession returns true if this is an account session
func (s *Session) IsAccountSession() bool {
	return s.SessionType == SessionTypeAccount
}

// IsGuestSession returns true if this is a guest session
func (s *Session) IsGuestSession() bool {
	return s.SessionType == SessionTypeGuest
}
