package model

import "time"

// User represents a committee membership record. A member can be linked to a
// sitewide account (AccountID set) or identified by email only (AccountID nil).
type User struct {
	ID            int64
	AccountID     *int64  // nil for email-only members
	CommitteeID   int64
	Email         *string // nil for account-based members
	Username      string  // Denormalized from accounts table (empty for email-only members)
	CommitteeSlug string  // Populated when joined with committees table (read-only)
	FullName      string
	Quoted        bool    // Whether the user uses quoted speech
	Role          string  // 'chairperson' or 'member'
	InviteSecret  *string // Personalized login token for email-only members
	OAuthManaged  bool    // True when membership is synchronized by OAuth group rules
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
