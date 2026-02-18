package model

import "time"

// Meeting represents a committee meeting
type Meeting struct {
	ID          int64
	Name        string
	Description string
	SignupOpen  bool
	CreatedAt   time.Time
}
