package model

import "time"

// Meeting represents a committee meeting
type Meeting struct {
	ID                   int64
	Name                 string
	Description          string
	SignupOpen           bool
	CurrentAgendaPointID *int64
	ProtocolWriterID     *int64
	CreatedAt            time.Time
}
