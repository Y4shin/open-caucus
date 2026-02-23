package model

import "time"

// Attendee represents a meeting attendee (can be a user or guest)
type Attendee struct {
	ID             int64
	MeetingID      int64
	AttendeeNumber int64
	UserID         *int64 // nil for guests, set for registered users
	FullName       string
	Secret         string // Used for guest authentication
	IsChair        bool   // Whether this attendee is chairing the meeting
	Quoted         bool   // Whether the attendee uses quoted speech in protocols
	CreatedAt      time.Time
}
