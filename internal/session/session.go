package session

import (
	"context"
	"time"
)

// SessionType represents the type of session
type SessionType string

const (
	SessionTypeAccount SessionType = "account"
	SessionTypeGuest   SessionType = "guest"
)

// SessionData contains information about an authenticated session.
// This is the lightweight version used in request context.
// Identity fields (IsAdmin, CurrentUser, CurrentAttendee) are populated
// per-request by the session and committee_access middleware via DB lookups.
type SessionData struct {
	SessionType SessionType

	// Raw IDs from the session row
	AccountID  *int64
	AttendeeID *int64

	// IsAdmin is populated by sessionMiddleware via GetAccountByID
	IsAdmin bool

	ExpiresAt time.Time
}

// CurrentUser is the request-scoped user identity view exposed via context.
type CurrentUser struct {
	UserID        int64
	CommitteeSlug string
	Username      string
	Role          string
	Quoted        bool
}

// CurrentAttendee is the request-scoped attendee identity view exposed via context.
type CurrentAttendee struct {
	AttendeeID int64
	MeetingID  int64
	FullName   string
	IsChair    bool
	Quoted     bool
}

// IsExpired checks if the session has expired
func (s *SessionData) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsAccountSession returns true if this is an account session
func (s *SessionData) IsAccountSession() bool {
	return s.SessionType == SessionTypeAccount
}

// IsGuestSession returns true if this is a guest session
func (s *SessionData) IsGuestSession() bool {
	return s.SessionType == SessionTypeGuest
}

// Context keys for storing session data in request context
type contextKey int

const sessionContextKey contextKey = 0

type namedContextKey string

const (
	currentUserContextKey     namedContextKey = "current-user"
	currentAttendeeContextKey namedContextKey = "current-attendee"
)

// WithSession adds session data to the context
func WithSession(ctx context.Context, session *SessionData) context.Context {
	return context.WithValue(ctx, sessionContextKey, session)
}

// GetSession retrieves session data from the context
func GetSession(ctx context.Context) (*SessionData, bool) {
	session, ok := ctx.Value(sessionContextKey).(*SessionData)
	return session, ok
}

// WithCurrentUser adds the normalized current user identity to context.
func WithCurrentUser(ctx context.Context, user *CurrentUser) context.Context {
	return context.WithValue(ctx, currentUserContextKey, user)
}

// GetCurrentUser retrieves the normalized current user identity from context.
func GetCurrentUser(ctx context.Context) (*CurrentUser, bool) {
	user, ok := ctx.Value(currentUserContextKey).(*CurrentUser)
	return user, ok
}

// WithCurrentAttendee adds the normalized current attendee identity to context.
func WithCurrentAttendee(ctx context.Context, attendee *CurrentAttendee) context.Context {
	return context.WithValue(ctx, currentAttendeeContextKey, attendee)
}

// GetCurrentAttendee retrieves the normalized current attendee identity from context.
func GetCurrentAttendee(ctx context.Context) (*CurrentAttendee, bool) {
	attendee, ok := ctx.Value(currentAttendeeContextKey).(*CurrentAttendee)
	return attendee, ok
}

// IsAdminAuthenticated checks if the context has an authenticated admin account session.
func IsAdminAuthenticated(ctx context.Context) bool {
	sd, ok := GetSession(ctx)
	return ok && sd.IsAdmin
}
