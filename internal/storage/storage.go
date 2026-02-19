package storage

import "io"

// Service is the interface for storing, retrieving, and deleting binary blobs.
// The storagePath returned by Store is an opaque key that can be persisted in
// the database and later passed back to Open or Delete.
type Service interface {
	// Store saves the content from data and returns a storagePath that can be
	// used to retrieve or delete the blob later, along with the number of bytes
	// written.
	Store(filename, contentType string, data io.Reader) (storagePath string, sizeBytes int64, err error)

	// Open returns a ReadCloser for the blob identified by storagePath.
	// The caller is responsible for closing the returned reader.
	Open(storagePath string) (io.ReadCloser, error)

	// Delete permanently removes the blob identified by storagePath.
	Delete(storagePath string) error
}
