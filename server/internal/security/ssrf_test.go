package security

import (
	"context"
	"testing"
)

func TestCheckSSRF_BlockedHostnames(t *testing.T) {
	tests := []string{
		"http://localhost",
		"https://localhost:8080/foo",
		"http://169.254.169.254/latest/meta-data/",
		"http://metadata.google.internal",
	}

	for _, urlStr := range tests {
		err := CheckSSRF(context.Background(), urlStr)
		if err == nil {
			t.Errorf("expected error for blocked hostname: %s", urlStr)
		}
	}
}

func TestCheckSSRF_BlockedIPs(t *testing.T) {
	// These rely on local DNS or parsing. Since we test raw IPs in the URL,
	// net.LookupIP will just parse them directly without a network call.
	tests := []string{
		"http://127.0.0.1",
		"http://10.1.2.3",
		"http://192.168.1.100",
		"http://172.16.0.1",
		"http://[::1]",
		"http://[fc00::1]",
	}

	for _, urlStr := range tests {
		err := CheckSSRF(context.Background(), urlStr)
		if err == nil {
			t.Errorf("expected error for blocked IP: %s", urlStr)
		}
	}
}

func TestCheckSSRF_Allowed(t *testing.T) {
	// We rely on real DNS here. These should resolve to public IPs.
	tests := []string{
		"https://example.com",
		"https://google.com",
	}

	for _, urlStr := range tests {
		err := CheckSSRF(context.Background(), urlStr)
		if err != nil {
			t.Errorf("unexpected error for allowed URL %s: %v", urlStr, err)
		}
	}
}

func TestCheckSSRF_InvalidURL(t *testing.T) {
	tests := []string{
		"://missing-scheme",
		"not-a-url",
	}

	for _, urlStr := range tests {
		err := CheckSSRF(context.Background(), urlStr)
		if err == nil {
			t.Errorf("expected error for invalid URL: %s", urlStr)
		}
	}
}
