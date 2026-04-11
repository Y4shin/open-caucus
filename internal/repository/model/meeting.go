package model

import "time"

// Meeting represents a committee meeting
type Meeting struct {
	ID                           int64
	Name                         string
	Description                  string
	Secret                       string
	SignupOpen                   bool
	CurrentAgendaPointID         *int64
	QuotationOrder []string // ordered list of enabled quotation types, e.g. ["gender", "first_speaker"]
	ModeratorID                  *int64 // nil if not set
	Version                      int64
	CreatedAt                    time.Time
	StartAt                      *time.Time // optional, always UTC
	EndAt                        *time.Time // optional, always UTC
}
