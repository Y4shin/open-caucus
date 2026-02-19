package model

import "time"

// BinaryBlob holds metadata for an uploaded file. The actual bytes are stored
// on the filesystem (or in-memory during tests) at StoragePath.
type BinaryBlob struct {
	ID          int64
	Filename    string
	ContentType string
	SizeBytes   int64
	StoragePath string
	CreatedAt   time.Time
}
