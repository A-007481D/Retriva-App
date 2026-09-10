package downloader

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"path"
	"time"

	"github.com/A-007481D/retriva/server/internal/resolver"
	"github.com/A-007481D/retriva/server/internal/security"
	"github.com/A-007481D/retriva/server/internal/storage"
	"github.com/oklog/ulid/v2"
)

// HTTPDownloader implements Downloader using standard HTTP GET requests.
type HTTPDownloader struct {
	client       *http.Client
	storage      storage.Storage
	maxSizeBytes int64
}

// NewHTTPDownloader creates a new HTTP downloader.
func NewHTTPDownloader(store storage.Storage, timeout time.Duration, maxSizeBytes int64, maxRedirects int) *HTTPDownloader {
	return &HTTPDownloader{
		client:       security.SafeHTTPClient(timeout, maxRedirects),
		storage:      store,
		maxSizeBytes: maxSizeBytes,
	}
}

// Download fetches the file and streams it directly to storage, calculating the SHA-256 hash.
func (d *HTTPDownloader) Download(ctx context.Context, info *resolver.MediaInfo, progress ProgressFn) (*DownloadResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, info.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; RetrivaBot/1.0)")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if resp.ContentLength > d.maxSizeBytes {
		return nil, fmt.Errorf("file too large: %d > %d", resp.ContentLength, d.maxSizeBytes)
	}

	// Generate storage key: media/PREFIX/ULID.ext
	id := ulid.Make().String()
	prefix := id[:4]
	ext := path.Ext(info.Filename)
	storageKey := fmt.Sprintf("media/%s/%s%s", prefix, id, ext)

	hasher := sha256.New()
	
	// Create a reader that tees to hasher, enforces size limit, and reports progress
	reader := &progressReader{
		r:          io.TeeReader(resp.Body, hasher),
		limit:      d.maxSizeBytes,
		total:      resp.ContentLength,
		progressFn: progress,
	}

	err = d.storage.Put(ctx, storageKey, reader, resp.ContentLength)
	if err != nil {
		// Clean up partial file
		_ = d.storage.Delete(context.Background(), storageKey)
		return nil, fmt.Errorf("storage put failed: %w", err)
	}

	if reader.limitExceeded {
		_ = d.storage.Delete(context.Background(), storageKey)
		return nil, fmt.Errorf("file exceeds maximum allowed size of %d bytes", d.maxSizeBytes)
	}

	hashHex := hex.EncodeToString(hasher.Sum(nil))

	return &DownloadResult{
		StorageKey: storageKey,
		MediaHash:  hashHex,
		SizeBytes:  reader.written,
	}, nil
}

type progressReader struct {
	r             io.Reader
	limit         int64
	total         int64
	written       int64
	progressFn    ProgressFn
	lastReport    time.Time
	limitExceeded bool
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.r.Read(p)
	pr.written += int64(n)

	if pr.written > pr.limit {
		pr.limitExceeded = true
		return n, fmt.Errorf("size limit exceeded")
	}

	if pr.progressFn != nil && n > 0 {
		now := time.Now()
		// Report progress at most every 500ms to avoid spamming the DB/channel
		if now.Sub(pr.lastReport) >= 500*time.Millisecond {
			pr.progressFn(pr.written, pr.total)
			pr.lastReport = now
		}
	}

	return n, err
}
