// Package storage defines the interface for blob storage in Retriva.
package storage

import (
	"context"
	"io"
)

// Storage abstracts the underlying blob storage (Filesystem, S3, etc.).
type Storage interface {
	// Put streams data from r into storage under the given key.
	// If size is unknown, pass -1 (some implementations may not support this, but filesystem does).
	Put(ctx context.Context, key string, r io.Reader, size int64) error

	// Get retrieves a reader for the given key. It is the caller's responsibility to close it.
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete removes the file at the given key. It may return an error if it fails,
	// but deleting a non-existent file is generally a no-op or returns nil.
	Delete(ctx context.Context, key string) error

	// Exists returns true if the key exists in storage.
	Exists(ctx context.Context, key string) (bool, error)

	// Ping checks if the storage is available and writable.
	Ping(ctx context.Context) error
}

// Common errors.
var (
	ErrNotFound = stringError("storage: key not found")
)

type stringError string

func (e stringError) Error() string { return string(e) }
