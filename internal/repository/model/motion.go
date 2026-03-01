package model

import "time"

// Motion represents a motion document attached to an agenda point.
type Motion struct {
	ID            int64
	AgendaPointID int64
	BlobID        int64
	Title         string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
