package filesystem

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/A-007481D/retriva/server/internal/storage"
)

func TestFilesystemStorage_PutAndGet(t *testing.T) {
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	key := "test/foo/bar.txt"
	content := []byte("hello world")

	err = s.Put(ctx, key, bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	rc, err := s.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("Get: got %q, want %q", got, content)
	}
}

func TestFilesystemStorage_Exists(t *testing.T) {
	s, _ := New(t.TempDir())
	ctx := context.Background()

	key := "exists.txt"
	s.Put(ctx, key, strings.NewReader(""), 0)

	ok, err := s.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists error: %v", err)
	}
	if !ok {
		t.Error("Exists returned false for existing file")
	}

	ok, err = s.Exists(ctx, "nope.txt")
	if err != nil {
		t.Fatalf("Exists error for non-existent file: %v", err)
	}
	if ok {
		t.Error("Exists returned true for non-existent file")
	}
}

func TestFilesystemStorage_Delete(t *testing.T) {
	s, _ := New(t.TempDir())
	ctx := context.Background()

	key := "delete.txt"
	s.Put(ctx, key, strings.NewReader("hi"), 2)

	if err := s.Delete(ctx, key); err != nil {
		t.Fatalf("Delete error: %v", err)
	}

	ok, _ := s.Exists(ctx, key)
	if ok {
		t.Error("File still exists after Delete")
	}

	// Delete non-existent should not fail
	if err := s.Delete(ctx, key); err != nil {
		t.Errorf("Delete non-existent error: %v", err)
	}
}

func TestFilesystemStorage_GetNotFound(t *testing.T) {
	s, _ := New(t.TempDir())
	ctx := context.Background()

	_, err := s.Get(ctx, "missing.txt")
	if err != storage.ErrNotFound {
		t.Errorf("Get missing file: got %v, want %v", err, storage.ErrNotFound)
	}
}

func TestFilesystemStorage_ConcurrentWrites(t *testing.T) {
	s, _ := New(t.TempDir())
	ctx := context.Background()
	key := "concurrent.txt"

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			s.Put(ctx, key, bytes.NewReader([]byte{byte(id)}), 1)
		}(i)
	}
	wg.Wait()

	// Should not crash and file should contain one of the values.
	ok, _ := s.Exists(ctx, key)
	if !ok {
		t.Error("File should exist after concurrent writes")
	}
}

func TestFilesystemStorage_Ping(t *testing.T) {
	s, _ := New(t.TempDir())
	ctx := context.Background()

	if err := s.Ping(ctx); err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}
