package model

// AgendaPoint represents a single item on a meeting's agenda.
type AgendaPoint struct {
	ID                           int64
	MeetingID                    int64
	ParentID                     *int64
	Position                     int64
	Title                        string
	Protocol                     string
	CurrentSpeakerID             *int64
	GenderQuotationEnabled       *bool  // nil = inherit from meeting
	FirstSpeakerQuotationEnabled *bool  // nil = inherit from meeting
	ModeratorID                  *int64 // nil if not set
}
