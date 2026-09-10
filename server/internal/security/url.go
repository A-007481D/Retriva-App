package security

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// ValidateURL parses the URL, checks its scheme, and verifies it passes SSRF checks.
func ValidateURL(ctx context.Context, rawURL string) (*url.URL, error) {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	if u.Scheme != "https" && u.Scheme != "http" {
		return nil, fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}

	if err := CheckSSRF(ctx, rawURL); err != nil {
		return nil, fmt.Errorf("SSRF check failed: %w", err)
	}

	return u, nil
}

// SafeHTTPClient creates an http.Client configured to prevent SSRF, limit redirects,
// and enforce timeouts. It re-validates the URL on each redirect hop.
func SafeHTTPClient(timeout time.Duration, maxRedirects int) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("stopped after %d redirects", maxRedirects)
			}
			if req.URL.Scheme != "https" && req.URL.Scheme != "http" {
				return fmt.Errorf("unsupported redirect scheme: %s", req.URL.Scheme)
			}

			// Re-evaluate SSRF for the redirect target.
			if err := CheckSSRF(req.Context(), req.URL.String()); err != nil {
				return fmt.Errorf("SSRF check failed on redirect: %w", err)
			}
			return nil
		},
	}
}
