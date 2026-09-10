package vault

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/A-007481D/retriva/server/internal/media"
	"github.com/A-007481D/retriva/server/internal/storage"
	"io"
)

type mockMediaRepo struct {
	expired []*media.Item
	updated []*media.Item
}

func (m *mockMediaRepo) Create(ctx context.Context, item *media.Item) error { return nil }
func (m *mockMediaRepo) Get(ctx context.Context, id string) (*media.Item, error) { return nil, nil }
func (m *mockMediaRepo) List(ctx context.Context, filter media.ListFilter) ([]*media.Item, error) { return nil, nil }
func (m *mockMediaRepo) Update(ctx context.Context, item *media.Item) error {
	m.updated = append(m.updated, item)
	return nil
}
func (m *mockMediaRepo) Delete(ctx context.Context, id string) error { return nil }
func (m *mockMediaRepo) ListExpired(ctx context.Context) ([]*media.Item, error) {
	return m.expired, nil
}
func (m *mockMediaRepo) Count(ctx context.Context, filter media.ListFilter) (int, error) { return 0, nil }

type mockStorage struct {
	deleted []string
	err     error
}

func (m *mockStorage) Put(ctx context.Context, key string, r io.Reader, size int64) error { return nil }
func (m *mockStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) { return nil, nil }
func (m *mockStorage) Exists(ctx context.Context, key string) (bool, error) { return false, nil }
func (m *mockStorage) Delete(ctx context.Context, key string) error {
	m.deleted = append(m.deleted, key)
	if m.err != nil {
		return m.err
	}
	return nil
}
func (m *mockStorage) Ping(ctx context.Context) error { return nil }

func TestVaultService_Cleanup(t *testing.T) {
	repo := &mockMediaRepo{
		expired: []*media.Item{
			{ID: "m1", Status: media.StatusActive, StorageKey: "key1"},
			{ID: "m2", Status: media.StatusExpired, StorageKey: ""}, // already expired
			{ID: "m3", Status: media.StatusActive, StorageKey: "key3"},
		},
	}
	st := &mockStorage{}

	svc := NewService(repo, st, slog.Default())
	svc.cleanup(context.Background())

	if len(st.deleted) != 2 {
		t.Errorf("expected 2 deleted files, got %d", len(st.deleted))
	}
	if len(repo.updated) != 2 {
		t.Errorf("expected 2 updated media items, got %d", len(repo.updated))
	}
	
	if repo.updated[0].Status != media.StatusExpired || repo.updated[0].StorageKey != "" {
		t.Errorf("expected m1 to be expired and have empty storage key")
	}
}

func TestVaultService_Cleanup_DeleteErrorNotFound(t *testing.T) {
	repo := &mockMediaRepo{
		expired: []*media.Item{
			{ID: "m1", Status: media.StatusActive, StorageKey: "key1"},
		},
	}
	st := &mockStorage{err: storage.ErrNotFound}

	svc := NewService(repo, st, slog.Default())
	svc.cleanup(context.Background())

	if len(repo.updated) != 1 {
		t.Errorf("expected media to be updated even if file not found")
	}
}

func TestVaultService_Cleanup_DeleteError(t *testing.T) {
	repo := &mockMediaRepo{
		expired: []*media.Item{
			{ID: "m1", Status: media.StatusActive, StorageKey: "key1"},
		},
	}
	st := &mockStorage{err: errors.New("IO error")}

	svc := NewService(repo, st, slog.Default())
	svc.cleanup(context.Background())

	if len(repo.updated) != 0 {
		t.Errorf("expected media NOT to be updated on random storage error")
	}
}

func TestVaultService_StartStop(t *testing.T) {
	repo := &mockMediaRepo{}
	st := &mockStorage{}

	svc := NewService(repo, st, slog.Default())
	svc.Start(1 * time.Millisecond)
	
	time.Sleep(5 * time.Millisecond)
	svc.Stop()
}
