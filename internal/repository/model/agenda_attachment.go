package model

import "time"

// AgendaAttachment links a binary blob to an agenda point, with an optional label.
type AgendaAttachment struct {
	ID            int64
	AgendaPointID int64
	BlobID        int64
	Label         *string
	CreatedAt     time.Time
}
