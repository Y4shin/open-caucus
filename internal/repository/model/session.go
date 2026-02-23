package model

import "time"

// SessionType represents the type of session
type SessionType string

const (
	SessionTypeUser     SessionType = "user"
	SessionTypeAttendee SessionType = "attendee"
	SessionTypeAdmin    SessionType = "admin"
)

// Session represents an authentication session
type Session struct {
	SessionID   string
	SessionType SessionType

	// For user sessions
	UserID        *int64
	CommitteeSlug *string
	Username      *string
	Role          *string
	Quoted        *bool

	// For attendee sessions
	AttendeeID *int64
	MeetingID  *int64
	FullName   *string
	IsChair    *bool

	CreatedAt time.Time
	ExpiresAt time.Time
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsUserSession returns true if this is a user session
func (s *Session) IsUserSession() bool {
	return s.SessionType == SessionTypeUser
}

// IsAttendeeSession returns true if this is an attendee session
func (s *Session) IsAttendeeSession() bool {
	return s.SessionType == SessionTypeAttendee
}
