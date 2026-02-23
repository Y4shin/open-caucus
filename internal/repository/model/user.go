package model

import "time"

// User represents a committee membership record linking an account to a committee.
type User struct {
	ID          int64
	AccountID   int64
	CommitteeID int64
	Username    string // Denormalized from the accounts table (read-only)
	FullName    string
	Quoted      bool   // Whether the user uses quoted speech in protocols
	Role        string // 'chairperson' or 'member'
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
