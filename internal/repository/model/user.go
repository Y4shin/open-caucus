package model

import "time"

// User represents a committee member user
type User struct {
	ID           int64
	CommitteeID  int64
	Username     string
	PasswordHash string
	FullName     string
	Quoted       bool   // Whether the user uses quoted speech in protocols
	Role         string // 'chairperson' or 'member'
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
