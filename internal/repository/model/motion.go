package model

import "time"

// Motion represents a motion document attached to an agenda point.
// Votes are either all set or all nil (enforced by a DB CHECK constraint).
type Motion struct {
	ID             int64
	AgendaPointID  int64
	BlobID         int64
	Title          string
	VotesFor       *int64
	VotesAgainst   *int64
	VotesAbstained *int64
	VotesEligible  *int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// HasVotes reports whether vote tallies have been recorded for this motion.
func (m *Motion) HasVotes() bool {
	return m.VotesFor != nil
}
