// Package media defines the MediaItem model and its repository interface.
package media

import (
	"context"
	"time"
)

// Status values for a media item.
const (
	StatusPending = "pending" // download in progress
	StatusActive  = "active"  // available in vault
	StatusExpired = "expired" // server copy deleted, metadata retained
	StatusDeleted = "deleted" // soft-deleted by user
)

// SourceType identifies which resolver produced the media.
const (
	SourceDirect    = "direct"
	SourceInstagram = "instagram"
	SourceFacebook  = "facebook"
	SourceYouTube   = "youtube"
	SourceGeneric   = "generic"
)

// Item is the normalised internal model for a downloaded media object.
// StorageKey and MediaHash are intentionally hidden from JSON to avoid
// exposing filesystem paths or content hashes to clients.
type Item struct {
	ID         string    `json:"id"`
	OwnerID    string    `json:"ownerId"`
	SourceURL  string    `json:"sourceUrl"`
	SourceType string    `json:"sourceType"`
	Filename   string    `json:"filename"`
	MIMEType   string    `json:"mimeType"`
	SizeBytes  *int64    `json:"sizeBytes,omitempty"`
	DurationS  *int      `json:"durationSeconds,omitempty"`
	Width      *int      `json:"width,omitempty"`
	Height     *int      `json:"height,omitempty"`
	StorageKey string    `json:"-"`
	ThumbKey   string    `json:"thumbKey,omitempty"`
	Status     string    `json:"status"`
	MediaHash  string    `json:"-"`
	CreatedAt  time.Time `json:"createdAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

// IsExpired reports whether the server-side copy has passed its expiry time.
func (i *Item) IsExpired() bool {
	return time.Now().UTC().After(i.ExpiresAt)
}

// ListFilter controls which items are returned by Repository.List.
type ListFilter struct {
	OwnerID string
	Status  []string // empty = all statuses
	Limit   int      // 0 = no limit
	Offset  int
}

// Repository is the data access interface for media items.
// V1 implementation: SQLiteRepository (see sqlite.go).
// Future implementation: PostgresRepository.
type Repository interface {
	Create(ctx context.Context, item *Item) error
	Get(ctx context.Context, id string) (*Item, error)
	List(ctx context.Context, filter ListFilter) ([]*Item, error)
	Update(ctx context.Context, item *Item) error
	Delete(ctx context.Context, id string) error
	ListExpired(ctx context.Context) ([]*Item, error)
	Count(ctx context.Context, filter ListFilter) (int, error)
}
