package model

// SpeakerEntry represents one entry in the speakers list for an agenda point.
type SpeakerEntry struct {
	ID            int64
	AgendaPointID int64
	AttendeeID    int64
	AttendeeName  string
	Type          string // "regular" or "ropm"
	Status        string // "WAITING", "SPEAKING", "DONE", "WITHDRAWN"
}
