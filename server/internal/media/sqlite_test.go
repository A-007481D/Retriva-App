package media

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/A-007481D/retriva/server/internal/database"
	"github.com/oklog/ulid/v2"
)

func openTestDB(t *testing.T) *database.DB {
	t.Helper()
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	db, err := database.Open(filepath.Join(dir, "test.db"), logger)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func newTestItem() *Item {
	size := int64(1024 * 1024)
	return &Item{
		ID:         ulid.Make().String(),
		OwnerID:    "local",
		SourceURL:  "https://example.com/video.mp4",
		SourceType: SourceDirect,
		Filename:   "video.mp4",
		MIMEType:   "video/mp4",
		SizeBytes:  &size,
		StorageKey: "media/AB/ABCDEF.mp4",
		Status:     StatusActive,
		CreatedAt:  time.Now().UTC().Truncate(time.Second),
		ExpiresAt:  time.Now().UTC().Add(7 * 24 * time.Hour).Truncate(time.Second),
	}
}

func TestSQLiteRepository_CreateAndGet(t *testing.T) {
	db := openTestDB(t)
	repo := NewSQLiteRepository(db.DB)
	ctx := context.Background()

	item := newTestItem()
	if err := repo.Create(ctx, item); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.Get(ctx, item.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if got.ID != item.ID {
		t.Errorf("ID mismatch: got %q, want %q", got.ID, item.ID)
	}
	if got.Filename != item.Filename {
		t.Errorf("Filename mismatch: got %q, want %q", got.Filename, item.Filename)
	}
	if got.Status != item.Status {
		t.Errorf("Status mismatch: got %q, want %q", got.Status, item.Status)
	}
}

func TestSQLiteRepository_Get_NotFound(t *testing.T) {
	db := openTestDB(t)
	repo := NewSQLiteRepository(db.DB)

	got, err := repo.Get(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("Get: unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for missing item, got %+v", got)
	}
}

func TestSQLiteRepository_Update(t *testing.T) {
	db := openTestDB(t)
	repo := NewSQLiteRepository(db.DB)
	ctx := context.Background()

	item := newTestItem()
	if err := repo.Create(ctx, item); err != nil {
		t.Fatalf("Create: %v", err)
	}

	item.Status = StatusExpired
	item.Filename = "updated.mp4"
	if err := repo.Update(ctx, item); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := repo.Get(ctx, item.ID)
	if got.Status != StatusExpired {
		t.Errorf("Status not updated: got %q", got.Status)
	}
	if got.Filename != "updated.mp4" {
		t.Errorf("Filename not updated: got %q", got.Filename)
	}
}

func TestSQLiteRepository_Delete(t *testing.T) {
	db := openTestDB(t)
	repo := NewSQLiteRepository(db.DB)
	ctx := context.Background()

	item := newTestItem()
	repo.Create(ctx, item)
	repo.Delete(ctx, item.ID)

	got, _ := repo.Get(ctx, item.ID)
	if got == nil {
		t.Fatal("expected item to exist after soft-delete")
	}
	if got.Status != StatusDeleted {
		t.Errorf("expected status=deleted, got %q", got.Status)
	}
}

func TestSQLiteRepository_List(t *testing.T) {
	db := openTestDB(t)
	repo := NewSQLiteRepository(db.DB)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		repo.Create(ctx, newTestItem())
	}

	items, err := repo.List(ctx, ListFilter{OwnerID: "local", Status: []string{StatusActive}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 5 {
		t.Errorf("expected 5 items, got %d", len(items))
	}
}

func TestSQLiteRepository_ListExpired(t *testing.T) {
	db := openTestDB(t)
	repo := NewSQLiteRepository(db.DB)
	ctx := context.Background()

	// One expired, one not.
	expired := newTestItem()
	expired.ExpiresAt = time.Now().UTC().Add(-1 * time.Hour)
	repo.Create(ctx, expired)

	active := newTestItem()
	active.ExpiresAt = time.Now().UTC().Add(7 * 24 * time.Hour)
	repo.Create(ctx, active)

	items, err := repo.ListExpired(ctx)
	if err != nil {
		t.Fatalf("ListExpired: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 expired item, got %d", len(items))
	}
	if items[0].ID != expired.ID {
		t.Errorf("wrong item returned as expired")
	}
}
