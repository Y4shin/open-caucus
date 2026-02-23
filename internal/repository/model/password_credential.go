package model

import "time"

// PasswordCredential holds the bcrypt hash for an account that authenticates with a password.
type PasswordCredential struct {
	ID           int64
	AccountID    int64
	AuthMethod   string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
