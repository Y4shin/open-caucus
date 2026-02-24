package model

import "time"

// Account represents a sitewide identity.
type Account struct {
	ID         int64
	Username   string
	FullName   string
	AuthMethod string
	IsAdmin    bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
