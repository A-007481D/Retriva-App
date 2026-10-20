package ytdlp

import (
	"context"
	"testing"
)

func TestCanHandle(t *testing.T) {
	r := New("", "")

	tests := []struct {
		url  string
		want bool
	}{
		{"https://t.me/europa_press/613", true},
		{"https://t.me/c/1234567890/100", true},
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", true},
		{"http://example.com/video.mp4", true},
		{"not-a-url", false},
		{"ftp://example.com/file", false},
	}

	for _, tc := range tests {
		got := r.CanHandle(tc.url)
		if got != tc.want {
			t.Errorf("CanHandle(%q) = %v; want %v", tc.url, got, tc.want)
		}
	}
}

func TestResolveInvalidURL(t *testing.T) {
	r := New("", "")
	_, err := r.Resolve(context.Background(), "invalid-url")
	if err == nil {
		t.Errorf("expected error for invalid url, got nil")
	}
}
