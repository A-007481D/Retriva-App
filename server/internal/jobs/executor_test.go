package jobs

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/A-007481D/retriva/server/internal/downloader"
	"github.com/A-007481D/retriva/server/internal/history"
	"github.com/A-007481D/retriva/server/internal/media"
	"github.com/A-007481D/retriva/server/internal/resolver"
)

type mockJobsRepo struct {
	Job *Job
}

func (m *mockJobsRepo) Create(ctx context.Context, job *Job) error       { return nil }
func (m *mockJobsRepo) Get(ctx context.Context, id string) (*Job, error) { return m.Job, nil }
func (m *mockJobsRepo) Update(ctx context.Context, job *Job) error       { m.Job = job; return nil }
func (m *mockJobsRepo) ListPending(ctx context.Context) ([]*Job, error)  { return nil, nil }
func (m *mockJobsRepo) Cancel(ctx context.Context, id string) error      { return nil }

type mockMediaRepo struct {
	Item *media.Item
}

func (m *mockMediaRepo) Create(ctx context.Context, item *media.Item) error { return nil }
func (m *mockMediaRepo) Get(ctx context.Context, id string) (*media.Item, error) {
	if m.Item == nil {
		return nil, errors.New("not found")
	}
	return m.Item, nil
}
func (m *mockMediaRepo) List(ctx context.Context, filter media.ListFilter) ([]*media.Item, error) {
	return nil, nil
}
func (m *mockMediaRepo) Update(ctx context.Context, item *media.Item) error {
	m.Item = item
	return nil
}
func (m *mockMediaRepo) Delete(ctx context.Context, id string) error               { return nil }
func (m *mockMediaRepo) ListExpired(ctx context.Context) ([]*media.Item, error)    { return nil, nil }
func (m *mockMediaRepo) Count(ctx context.Context, filter media.ListFilter) (int, error) { return 0, nil }

type mockHistoryRepo struct {
	called int
}

func (m *mockHistoryRepo) Create(ctx context.Context, record *history.Record) error {
	m.called++
	return nil
}
func (m *mockHistoryRepo) List(ctx context.Context, filter history.ListFilter) ([]*history.Record, error) {
	return nil, nil
}
func (m *mockHistoryRepo) Count(ctx context.Context, ownerID string) (int, error) { return 0, nil }

type mockDownloader struct {
	err error
}

func (m *mockDownloader) Download(ctx context.Context, info *resolver.MediaInfo, progress downloader.ProgressFn) (*downloader.DownloadResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	progress(50, 100)
	progress(100, 100)
	return &downloader.DownloadResult{
		StorageKey: "test-key",
		MediaHash:  "test-hash",
		SizeBytes:  100,
	}, nil
}

type mockResolverHandler struct{}
func (h *mockResolverHandler) CanHandle(rawURL string) bool { return true }
func (h *mockResolverHandler) Resolve(ctx context.Context, rawURL string) (*resolver.MediaInfo, error) {
	return &resolver.MediaInfo{
		URL:       "http://resolved.com/video.mp4",
		MIMEType:  "video/mp4",
		Filename:  "video.mp4",
		SizeBytes: 100,
	}, nil
}

func TestExecutor_Success(t *testing.T) {
	jRepo := &mockJobsRepo{}
	mRepo := &mockMediaRepo{
		Item: &media.Item{ID: "m1", SourceURL: "http://example.com/video.mp4"},
	}
	hRepo := &mockHistoryRepo{}
	dl := &mockDownloader{}
	res := resolver.NewRegistry(&mockResolverHandler{})

	exec := NewExecutor(res, dl, jRepo, mRepo, hRepo, slog.Default())

	mediaID := "m1"
	job := &Job{
		ID:      "j1",
		MediaID: &mediaID,
		Status:  StatusQueued,
	}

	err := exec.Execute(context.Background(), job)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if job.Status != StatusCompleted {
		t.Errorf("expected completed, got %s", job.Status)
	}
	if job.Progress != 100.0 {
		t.Errorf("expected 100%% progress, got %f", job.Progress)
	}
	if mRepo.Item.Status != media.StatusActive {
		t.Errorf("expected media active, got %s", mRepo.Item.Status)
	}
	if hRepo.called != 1 {
		t.Errorf("expected history record created")
	}
}

func TestExecutor_NoMedia(t *testing.T) {
	jRepo := &mockJobsRepo{}
	mRepo := &mockMediaRepo{}
	hRepo := &mockHistoryRepo{}
	dl := &mockDownloader{}
	res := resolver.NewRegistry(&mockResolverHandler{})

	exec := NewExecutor(res, dl, jRepo, mRepo, hRepo, slog.Default())

	job := &Job{ID: "j1", Status: StatusQueued}
	err := exec.Execute(context.Background(), job)
	if err == nil {
		t.Fatal("expected error")
	}

	if job.Status != StatusFailed {
		t.Errorf("expected failed, got %s", job.Status)
	}
}
