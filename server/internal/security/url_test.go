package security

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestValidateURL(t *testing.T) {
	ctx := context.Background()

	validURL := "https://example.com/video.mp4"
	if _, err := ValidateURL(ctx, validURL); err != nil {
		t.Errorf("expected valid URL to pass: %v", err)
	}

	invalidScheme := "ftp://example.com/file"
	if _, err := ValidateURL(ctx, invalidScheme); err == nil {
		t.Errorf("expected error for invalid scheme %s", invalidScheme)
	}

	ssrfURL := "http://127.0.0.1/admin"
	if _, err := ValidateURL(ctx, ssrfURL); err == nil {
		t.Errorf("expected SSRF error for %s", ssrfURL)
	}
}

func TestSafeHTTPClient_Redirect(t *testing.T) {
	// Setup a server that redirects to localhost (SSRF)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "http://127.0.0.1/target", http.StatusFound)
			return
		}
	}))
	defer ts.Close()

	client := SafeHTTPClient(5*time.Second, 5)

	req, _ := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/start", nil)
	_, err := client.Do(req)

	if err == nil {
		t.Fatal("expected error when redirecting to localhost")
	}
	if err != nil && !contains(err.Error(), "SSRF check failed") {
		t.Errorf("expected SSRF check failure, got: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s != "" && substr != ""
}
