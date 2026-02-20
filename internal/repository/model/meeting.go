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
	ProtocolWriterID             *int64
	GenderQuotationEnabled       bool   // default true
	FirstSpeakerQuotationEnabled bool   // default true
	ModeratorID                  *int64 // nil if not set
	CreatedAt                    time.Time
}
