package direct

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/A-007481D/retriva/server/internal/security"
)

func init() {
	security.DisableSSRFCheck = true
}

func TestDirectResolver_CanHandle(t *testing.T) {
	r := New()

	tests := []struct {
		url  string
		want bool
	}{
		{"https://example.com/video.mp4", true},
		{"https://example.com/video.MP4", true},
		{"https://example.com/image.jpg", true},
		{"https://example.com/stream.m3u8", true},
		{"https://example.com/page.html", false},
		{"ftp://example.com/video.mp4", false},
	}

	for _, tc := range tests {
		got := r.CanHandle(tc.url)
		if got != tc.want {
			t.Errorf("CanHandle(%q) = %v; want %v", tc.url, got, tc.want)
		}
	}
}

func TestDirectResolver_Resolve(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("expected HEAD request, got %s", r.Method)
		}
		if r.URL.Path == "/video.mp4" {
			w.Header().Set("Content-Type", "video/mp4")
			w.Header().Set("Content-Length", "12345")
			w.WriteHeader(http.StatusOK)
		} else if r.URL.Path == "/page.html" {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	r := New()
	ctx := context.Background()

	// 1. Success case
	info, err := r.Resolve(ctx, ts.URL+"/video.mp4")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if info.MIMEType != "video/mp4" {
		t.Errorf("expected video/mp4, got %s", info.MIMEType)
	}
	if info.SizeBytes != 12345 {
		t.Errorf("expected 12345, got %d", info.SizeBytes)
	}
	if info.Filename != "video.mp4" {
		t.Errorf("expected video.mp4, got %s", info.Filename)
	}

	// 2. Unsupported Content-Type case
	_, err = r.Resolve(ctx, ts.URL+"/page.html")
	if err == nil {
		t.Fatal("expected error for text/html")
	}

	// 3. Not Found case
	_, err = r.Resolve(ctx, ts.URL+"/missing.mp4")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}
