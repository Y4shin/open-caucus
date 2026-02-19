package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// DirStorage stores blobs as files inside a base directory on the local filesystem.
// The storagePath returned by Store is the filename (not the full path), so the
// base directory can be changed without invalidating stored paths.
type DirStorage struct {
	baseDir string
}

// NewDirStorage creates a DirStorage rooted at baseDir, creating the directory
// if it does not already exist.
func NewDirStorage(baseDir string) (*DirStorage, error) {
	if err := os.MkdirAll(baseDir, 0o750); err != nil {
		return nil, fmt.Errorf("create storage directory %q: %w", baseDir, err)
	}
	return &DirStorage{baseDir: baseDir}, nil
}

// Store writes the content of data to a new file in the base directory.
// The filename is a UUID to avoid collisions; the original filename is not used
// in the path.
func (s *DirStorage) Store(filename, _ string, data io.Reader) (storagePath string, sizeBytes int64, err error) {
	id := uuid.NewString()
	ext := filepath.Ext(filename)
	name := id + ext

	fullPath := filepath.Join(s.baseDir, name)
	f, err := os.Create(fullPath) //nolint:gosec // path is constructed from baseDir + UUID
	if err != nil {
		return "", 0, fmt.Errorf("create blob file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close blob file: %w", cerr)
		}
	}()

	n, err := io.Copy(f, data)
	if err != nil {
		return "", 0, fmt.Errorf("write blob file: %w", err)
	}
	return name, n, nil
}

// Open returns a ReadCloser for the blob identified by storagePath (the filename
// returned by Store).
func (s *DirStorage) Open(storagePath string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.baseDir, storagePath)
	f, err := os.Open(fullPath) //nolint:gosec // path constructed from baseDir + stored filename
	if err != nil {
		return nil, fmt.Errorf("open blob file %q: %w", storagePath, err)
	}
	return f, nil
}

// Delete removes the blob identified by storagePath from the base directory.
func (s *DirStorage) Delete(storagePath string) error {
	fullPath := filepath.Join(s.baseDir, storagePath)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete blob file %q: %w", storagePath, err)
	}
	return nil
}
