package downloader

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/A-007481D/retriva/server/internal/resolver"
	"github.com/A-007481D/retriva/server/internal/security"
	"github.com/A-007481D/retriva/server/internal/storage/filesystem"
)

func init() {
	security.DisableSSRFCheck = true
}

func TestHTTPDownloader_Download(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("hello world")) // 11 bytes
	}))
	defer ts.Close()

	store, _ := filesystem.New(t.TempDir())
	d := NewHTTPDownloader(store, 5*time.Second, 1024, 5)

	info := &resolver.MediaInfo{
		URL:      ts.URL,
		Filename: "test.txt",
	}

	var progressCalled bool
	res, err := d.Download(context.Background(), info, func(w, tot int64) {
		progressCalled = true
	})

	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	if !progressCalled {
		t.Error("Progress callback was never called")
	}

	if res.SizeBytes != 11 {
		t.Errorf("expected 11 bytes, got %d", res.SizeBytes)
	}

	// b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9 is SHA256 of "hello world"
	if res.MediaHash != "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9" {
		t.Errorf("unexpected hash: %s", res.MediaHash)
	}

	exists, _ := store.Exists(context.Background(), res.StorageKey)
	if !exists {
		t.Error("file not found in storage")
	}
}

func TestHTTPDownloader_SizeLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Flush headers so we send chunked response, avoiding automatic Content-Length
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		w.Write([]byte(strings.Repeat("a", 2048)))
	}))
	defer ts.Close()

	store, _ := filesystem.New(t.TempDir())
	d := NewHTTPDownloader(store, 5*time.Second, 1024, 5)

	info := &resolver.MediaInfo{URL: ts.URL}

	_, err := d.Download(context.Background(), info, nil)
	if err == nil || !strings.Contains(err.Error(), "size limit exceeded") {
		t.Errorf("expected size limit error, got %v", err)
	}
}

func TestHTTPDownloader_ContentLengthHeaderExceeds(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Announce a huge file, but don't actually send it yet
		w.Header().Set("Content-Length", "2048")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	store, _ := filesystem.New(t.TempDir())
	d := NewHTTPDownloader(store, 5*time.Second, 1024, 5)

	info := &resolver.MediaInfo{URL: ts.URL}

	_, err := d.Download(context.Background(), info, nil)
	if err == nil || !strings.Contains(err.Error(), "file too large") {
		t.Errorf("expected early size limit error, got %v", err)
	}
}
