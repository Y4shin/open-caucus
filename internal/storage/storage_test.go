package storage_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/Y4shin/open-caucus/internal/storage"
)

// runContractTests exercises every Service method against the given implementation.
// Both DirStorage and MemStorage are expected to pass the same contract.
func runContractTests(t *testing.T, svc storage.Service) {
	t.Helper()

	t.Run("StoreAndOpen", func(t *testing.T) {
		want := "hello storage"
		path, size, err := svc.Store("note.txt", "text/plain", strings.NewReader(want))
		if err != nil {
			t.Fatalf("Store: %v", err)
		}
		if path == "" {
			t.Fatal("Store returned empty storagePath")
		}
		if size != int64(len(want)) {
			t.Errorf("Store size = %d, want %d", size, len(want))
		}

		rc, err := svc.Open(path)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer rc.Close()

		got, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		if string(got) != want {
			t.Errorf("content = %q, want %q", got, want)
		}
	})

	t.Run("StorePreservesExtension", func(t *testing.T) {
		path, _, err := svc.Store("document.pdf", "application/pdf", strings.NewReader("data"))
		if err != nil {
			t.Fatalf("Store: %v", err)
		}
		if ext := filepath.Ext(path); ext != ".pdf" {
			t.Errorf("storagePath extension = %q, want .pdf", ext)
		}
	})

	t.Run("MultipleStoresAreIndependent", func(t *testing.T) {
		path1, _, err := svc.Store("a.txt", "text/plain", strings.NewReader("content-A"))
		if err != nil {
			t.Fatalf("Store 1: %v", err)
		}
		path2, _, err := svc.Store("b.txt", "text/plain", strings.NewReader("content-B"))
		if err != nil {
			t.Fatalf("Store 2: %v", err)
		}
		if path1 == path2 {
			t.Fatal("two Store calls returned the same storagePath")
		}

		check := func(path, want string) {
			rc, err := svc.Open(path)
			if err != nil {
				t.Fatalf("Open %q: %v", path, err)
			}
			defer rc.Close()
			got, _ := io.ReadAll(rc)
			if string(got) != want {
				t.Errorf("Open %q = %q, want %q", path, got, want)
			}
		}
		check(path1, "content-A")
		check(path2, "content-B")
	})

	t.Run("DeleteRemovesBlob", func(t *testing.T) {
		path, _, err := svc.Store("del.txt", "text/plain", strings.NewReader("bye"))
		if err != nil {
			t.Fatalf("Store: %v", err)
		}
		if err := svc.Delete(path); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if _, err := svc.Open(path); err == nil {
			t.Error("Open after Delete should return an error")
		}
	})

	t.Run("DeleteNonExistentIsNoop", func(t *testing.T) {
		if err := svc.Delete("does-not-exist.bin"); err != nil {
			t.Errorf("Delete of non-existent path should not error, got: %v", err)
		}
	})

	t.Run("OpenNonExistentReturnsError", func(t *testing.T) {
		_, err := svc.Open("ghost.txt")
		if err == nil {
			t.Error("Open of non-existent path should return an error")
		}
	})

	t.Run("EmptyContent", func(t *testing.T) {
		path, size, err := svc.Store("empty.bin", "application/octet-stream", strings.NewReader(""))
		if err != nil {
			t.Fatalf("Store empty: %v", err)
		}
		if size != 0 {
			t.Errorf("size = %d, want 0", size)
		}
		rc, err := svc.Open(path)
		if err != nil {
			t.Fatalf("Open empty: %v", err)
		}
		defer rc.Close()
		data, _ := io.ReadAll(rc)
		if len(data) != 0 {
			t.Errorf("content of empty blob = %q, want empty", data)
		}
	})
}

func TestDirStorage(t *testing.T) {
	dir := t.TempDir()
	svc, err := storage.NewDirStorage(dir)
	if err != nil {
		t.Fatalf("NewDirStorage: %v", err)
	}
	runContractTests(t, svc)
}

func TestDirStorage_CreatesDirectoryIfMissing(t *testing.T) {
	base := t.TempDir()
	nested := filepath.Join(base, "a", "b", "c")

	svc, err := storage.NewDirStorage(nested)
	if err != nil {
		t.Fatalf("NewDirStorage with nested path: %v", err)
	}

	// Writing through the service confirms the directory was really created.
	path, _, err := svc.Store("test.txt", "text/plain", strings.NewReader("hi"))
	if err != nil {
		t.Fatalf("Store after directory creation: %v", err)
	}
	if _, err := os.Stat(filepath.Join(nested, path)); errors.Is(err, os.ErrNotExist) {
		t.Error("file was not created in the nested directory")
	}
}

func TestDirStorage_StoredFileExistsOnDisk(t *testing.T) {
	dir := t.TempDir()
	svc, err := storage.NewDirStorage(dir)
	if err != nil {
		t.Fatalf("NewDirStorage: %v", err)
	}

	path, _, err := svc.Store("img.png", "image/png", strings.NewReader("fakeimage"))
	if err != nil {
		t.Fatalf("Store: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
		t.Errorf("expected file on disk at %q: %v", path, err)
	}
}

func TestMemStorage(t *testing.T) {
	runContractTests(t, storage.NewMemStorage())
}

func TestMemStorage_ConcurrentAccess(t *testing.T) {
	svc := storage.NewMemStorage()
	const goroutines = 20

	var wg sync.WaitGroup
	paths := make([]string, goroutines)
	var mu sync.Mutex

	// Concurrently store blobs.
	for i := range goroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			p, _, err := svc.Store("file.txt", "text/plain", strings.NewReader("data"))
			if err != nil {
				t.Errorf("concurrent Store %d: %v", i, err)
				return
			}
			mu.Lock()
			paths[i] = p
			mu.Unlock()
		}(i)
	}
	wg.Wait()

	// All paths must be unique.
	seen := make(map[string]bool, goroutines)
	for _, p := range paths {
		if seen[p] {
			t.Errorf("duplicate storagePath: %q", p)
		}
		seen[p] = true
	}

	// Concurrently open and delete.
	for _, p := range paths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			rc, err := svc.Open(p)
			if err != nil {
				t.Errorf("concurrent Open %q: %v", p, err)
				return
			}
			rc.Close()
			if err := svc.Delete(p); err != nil {
				t.Errorf("concurrent Delete %q: %v", p, err)
			}
		}(p)
	}
	wg.Wait()
}
