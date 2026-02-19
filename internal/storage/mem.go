package storage

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
)

// MemStorage stores blobs in memory. It is intended for use in tests where
// filesystem cleanup would be inconvenient.
type MemStorage struct {
	mu   sync.Mutex
	data map[string][]byte
}

// NewMemStorage returns an empty in-memory storage.
func NewMemStorage() *MemStorage {
	return &MemStorage{data: make(map[string][]byte)}
}

// Store reads all bytes from data and stores them under a generated key that
// preserves the original file extension.
func (s *MemStorage) Store(filename string, _ string, data io.Reader) (storagePath string, sizeBytes int64, err error) {
	buf, err := io.ReadAll(data)
	if err != nil {
		return "", 0, fmt.Errorf("read blob data: %w", err)
	}
	key := uuid.NewString() + filepath.Ext(filename)
	s.mu.Lock()
	s.data[key] = buf
	s.mu.Unlock()
	return key, int64(len(buf)), nil
}

// Open returns a ReadCloser backed by an in-memory copy of the stored bytes.
func (s *MemStorage) Open(storagePath string) (io.ReadCloser, error) {
	s.mu.Lock()
	buf, ok := s.data[storagePath]
	s.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("blob not found: %q", storagePath)
	}
	// Return a copy so concurrent reads don't share a single reader position.
	cp := make([]byte, len(buf))
	copy(cp, buf)
	return io.NopCloser(bytes.NewReader(cp)), nil
}

// Delete removes the blob identified by storagePath. It is a no-op if the blob
// does not exist.
func (s *MemStorage) Delete(storagePath string) error {
	s.mu.Lock()
	delete(s.data, storagePath)
	s.mu.Unlock()
	return nil
}
