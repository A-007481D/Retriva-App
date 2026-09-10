// Package filesystem implements the storage.Storage interface using the local filesystem.
package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/A-007481D/retriva/server/internal/storage"
)

// Storage implements the storage.Storage interface.
type Storage struct {
	baseDir string
}

// New creates a new filesystem Storage rooted at baseDir.
// It ensures that the base directory exists.
func New(baseDir string) (*Storage, error) {
	absDir, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("absolute path for storage dir: %w", err)
	}
	if err := os.MkdirAll(absDir, 0755); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}
	return &Storage{baseDir: absDir}, nil
}

func (s *Storage) getPath(key string) string {
	return filepath.Join(s.baseDir, filepath.FromSlash(key))
}

// Put writes data from r to the given key.
// It uses a temporary file and an atomic rename to prevent partial writes.
func (s *Storage) Put(ctx context.Context, key string, r io.Reader, size int64) error {
	fullPath := s.getPath(key)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir storage path: %w", err)
	}

	// Write to a temporary file in the same directory to ensure rename is atomic
	// (same filesystem mount).
	tmpFile := fullPath + fmt.Sprintf(".tmp.%d", time.Now().UnixNano())
	f, err := os.OpenFile(tmpFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	cleanup := func() {
		f.Close()
		os.Remove(tmpFile)
	}

	// Write data
	written, err := io.Copy(f, r)
	if err != nil {
		cleanup()
		return fmt.Errorf("write data: %w", err)
	}
	if size >= 0 && written != size {
		cleanup()
		return fmt.Errorf("wrote %d bytes, expected %d", written, size)
	}

	if err := f.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpFile, fullPath); err != nil {
		os.Remove(tmpFile) // clean up on rename failure
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}

// Get opens the file for reading. Returns storage.ErrNotFound if it doesn't exist.
func (s *Storage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	fullPath := s.getPath(key)
	f, err := os.Open(fullPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("open file: %w", err)
	}
	return f, nil
}

// Delete removes the file at the given key. It ignores os.ErrNotExist.
func (s *Storage) Delete(ctx context.Context, key string) error {
	fullPath := s.getPath(key)
	err := os.Remove(fullPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove file: %w", err)
	}
	return nil
}

// Exists checks if the file exists.
func (s *Storage) Exists(ctx context.Context, key string) (bool, error) {
	fullPath := s.getPath(key)
	_, err := os.Stat(fullPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat file: %w", err)
	}
	return true, nil
}

// Ping verifies that the storage directory is writable.
func (s *Storage) Ping(ctx context.Context) error {
	tmpFile := filepath.Join(s.baseDir, ".ping")
	f, err := os.OpenFile(tmpFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("ping write failed: %w", err)
	}
	f.Close()
	os.Remove(tmpFile)
	return nil
}
