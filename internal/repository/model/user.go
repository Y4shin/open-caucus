package model

import "time"

// User represents a committee membership record linking an account to a committee.
type User struct {
	ID            int64
	AccountID     int64
	CommitteeID   int64
	Username      string // Denormalized from the accounts table (read-only)
	CommitteeSlug string // Populated when joined with committees table (read-only)
	FullName      string
	Quoted        bool   // Whether the user uses quoted speech
	Role          string // 'chairperson' or 'member'
	OAuthManaged  bool   // True when membership is synchronized by OAuth group rules
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
