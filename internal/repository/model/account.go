package model

import "time"

// Account represents a sitewide identity.
type Account struct {
	ID         int64
	Username   string
	FullName   string
	Email      string // from OIDC email claim; used for notifications
	AuthMethod string
	IsAdmin    bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
