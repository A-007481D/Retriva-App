// Package direct implements a resolver for direct media URLs.
package direct

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/A-007481D/retriva/server/internal/resolver"
	"github.com/A-007481D/retriva/server/internal/security"
)

var allowedExtensions = []string{
	".mp4", ".mov", ".webm", ".mkv",
	".jpg", ".jpeg", ".png", ".webp", ".gif",
	".m3u8",
}

// Resolver handles direct links to media files.
type Resolver struct {
	client *http.Client
}

// New creates a new Direct URL Resolver.
func New() *Resolver {
	return &Resolver{
		client: security.SafeHTTPClient(10*time.Second, 5),
	}
}

// CanHandle returns true if the URL has a known media extension.
func (r *Resolver) CanHandle(rawURL string) bool {
	// Fast path: check extension
	u, err := security.ValidateURL(context.Background(), rawURL)
	if err != nil {
		return false
	}

	ext := strings.ToLower(path.Ext(u.Path))
	for _, a := range allowedExtensions {
		if ext == a {
			return true
		}
	}
	return false
}

// Resolve validates the URL, performs a HEAD request to verify the Content-Type,
// and extracts the content length and filename.
func (r *Resolver) Resolve(ctx context.Context, rawURL string) (*resolver.MediaInfo, error) {
	u, err := security.ValidateURL(ctx, rawURL)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Add a common User-Agent to avoid generic blocks
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; RetrivaBot/1.0)")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("head request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	mediaType, _, _ := mime.ParseMediaType(contentType)
	if mediaType == "" {
		// fallback to extension detection
		mediaType = mime.TypeByExtension(path.Ext(u.Path))
	}

	if !strings.HasPrefix(mediaType, "video/") && !strings.HasPrefix(mediaType, "image/") && mediaType != "application/vnd.apple.mpegurl" && mediaType != "application/x-mpegurl" {
		return nil, fmt.Errorf("unsupported content type: %s", mediaType)
	}

	size := resp.ContentLength // -1 if not present
	filename := path.Base(u.Path)
	if filename == "/" || filename == "." {
		filename = "downloaded_media"
	}
	filename = security.SanitizeFilename(filename)

	// ensure the final URL is safe
	finalURL := resp.Request.URL.String()
	if err := security.CheckSSRF(ctx, finalURL); err != nil {
		return nil, fmt.Errorf("SSRF check failed on final URL: %w", err)
	}

	return &resolver.MediaInfo{
		URL:       finalURL,
		MIMEType:  mediaType,
		Filename:  filename,
		SizeBytes: size,
	}, nil
}
