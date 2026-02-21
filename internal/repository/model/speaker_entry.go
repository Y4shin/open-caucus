package model

import "time"

// SpeakerEntry represents one entry in the speakers list for an agenda point.
type SpeakerEntry struct {
	ID            int64
	AgendaPointID int64
	AttendeeID    int64
	AttendeeName  string
	Type          string // "regular" or "ropm"
	Status        string // "WAITING", "SPEAKING", "DONE"
	GenderQuoted  bool   // snapshot: was gender quotation applied at request time?
	FirstSpeaker  bool   // snapshot: was this the attendee's first time on this agenda point?
	Priority      bool   // manually promoted by chairperson
	OrderPosition int64  // computed ordering index (meaningful only for WAITING)
	StartOfSpeech *time.Time
	DurationSeconds int64
}
