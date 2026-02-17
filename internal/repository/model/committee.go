package model

import "time"

// Committee represents a committee in the system
type Committee struct {
	ID               int64
	Name             string
	Slug             string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	CurrentMeetingID *int64 // nil if no current meeting
}
