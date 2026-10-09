package instagram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestShortcodeToMediaID(t *testing.T) {
	tests := []struct {
		shortcode string
		wantID    string
	}{
		{"DaWHkQpid02", "3933364602658217270"},
		{"B_1234", "2144824824"},
		{"B", "1"},
		{"BA", "64"},
	}

	for _, tt := range tests {
		got, err := shortcodeToMediaID(tt.shortcode)
		if err != nil {
			t.Fatalf("shortcodeToMediaID(%q) error: %v", tt.shortcode, err)
		}
		if got != tt.wantID {
			t.Errorf("shortcodeToMediaID(%q) = %q; want %q", tt.shortcode, got, tt.wantID)
		}
	}
}

func TestCanHandle(t *testing.T) {
	r := New("")
	testCases := []struct {
		url      string
		expected bool
	}{
		{"https://www.instagram.com/p/DaWHkQpid02/?img_index=2", true},
		{"https://www.instagram.com/p/DaWHkQpid02/", true},
		{"https://instagram.com/p/DaWHkQpid02", true},
		{"https://www.instagram.com/reel/DCXYZ1234/", true},
		{"https://www.instagram.com/reels/DCXYZ1234/", true},
		{"https://www.instagram.com/tv/DCXYZ1234/", true},
		{"https://www.instagram.com/stories/highlights/18122897002451585/", true},
		{"https://www.instagram.com/stories/username/3982929260775922792/", true},
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", false},
		{"https://twitter.com/user/status/123", false},
	}

	for _, tc := range testCases {
		if got := r.CanHandle(tc.url); got != tc.expected {
			t.Errorf("CanHandle(%q) = %v; want %v", tc.url, got, tc.expected)
		}
	}
}

func TestExtractTarget(t *testing.T) {
	r := New("")

	isHL, id, err := r.extractTarget("https://www.instagram.com/stories/highlights/18122897002451585/")
	if err != nil || !isHL || id != "18122897002451585" {
		t.Fatalf("unexpected extractTarget for highlight: %v, %v, %v", isHL, id, err)
	}

	isHL, id, err = r.extractTarget("https://www.instagram.com/p/DaWHkQpid02/?img_index=2")
	if err != nil || isHL || id != "3933364602658217270" {
		t.Fatalf("unexpected extractTarget for post: %v, %v, %v", isHL, id, err)
	}

	isHL, id, err = r.extractTarget("https://www.instagram.com/stories/someone/3982929260775922792/")
	if err != nil || isHL || id != "3982929260775922792" {
		t.Fatalf("unexpected extractTarget for story: %v, %v, %v", isHL, id, err)
	}
}

func TestInspectAndResolveCarousel(t *testing.T) {
	mockResponse := map[string]any{
		"status": "ok",
		"items": []map[string]any{
			{
				"pk":         "3933364602658217270",
				"id":         "3933364602658217270_4172144285",
				"code":       "DaWHkQpid02",
				"media_type": 8,
				"user": map[string]any{
					"username": "meldadel",
				},
				"caption": map[string]any{
					"text": "can you focus on me?",
				},
				"carousel_media": []map[string]any{
					{
						"pk":         "3933364417489830512",
						"id":         "3933364417489830512_4172144285",
						"media_type": 1,
						"image_versions2": map[string]any{
							"candidates": []map[string]any{
								{
									"url":    "https://cdn.example.com/slide1.jpg",
									"width":  1290,
									"height": 1720,
								},
							},
						},
					},
					{
						"pk":         "3933364420241425957",
						"id":         "3933364420241425957_4172144285",
						"media_type": 1,
						"image_versions2": map[string]any{
							"candidates": []map[string]any{
								{
									"url":    "https://cdn.example.com/slide2.jpg",
									"width":  1290,
									"height": 1720,
								},
							},
						},
					},
				},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockResponse)
	}))
	defer srv.Close()

	r := New("")
	r.apiBaseURL = srv.URL

	ctx := context.Background()

	// 1. Test Inspect
	info, err := r.Inspect(ctx, "https://www.instagram.com/p/DaWHkQpid02/")
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}

	if info.Count != 2 {
		t.Errorf("expected count 2, got %d", info.Count)
	}
	if info.Title != "can you focus on me?" {
		t.Errorf("expected title 'can you focus on me?', got %q", info.Title)
	}
	if len(info.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(info.Items))
	}
	if info.Items[0].Type != "photo" || info.Items[0].Thumbnail != "https://cdn.example.com/slide1.jpg" {
		t.Errorf("item 0 mismatch: %+v", info.Items[0])
	}
	if info.Items[1].Type != "photo" || info.Items[1].Thumbnail != "https://cdn.example.com/slide2.jpg" {
		t.Errorf("item 1 mismatch: %+v", info.Items[1])
	}

	// 2. Test Resolve with img_index=2
	media2, err := r.Resolve(ctx, "https://www.instagram.com/p/DaWHkQpid02/?img_index=2")
	if err != nil {
		t.Fatalf("Resolve with img_index=2 failed: %v", err)
	}
	if media2.URL != "https://cdn.example.com/slide2.jpg" {
		t.Errorf("expected slide2 URL, got %q", media2.URL)
	}
	if media2.MIMEType != "image/jpeg" {
		t.Errorf("expected image/jpeg, got %q", media2.MIMEType)
	}
	if media2.Filename != "Photo 2 by meldadel [3933364420241425957].jpg" {
		t.Errorf("expected filename 'Photo 2 by meldadel [3933364420241425957].jpg', got %q", media2.Filename)
	}

	// 3. Test Resolve with item=1
	media1, err := r.Resolve(ctx, "https://www.instagram.com/p/DaWHkQpid02/?item=1")
	if err != nil {
		t.Fatalf("Resolve with item=1 failed: %v", err)
	}
	if media1.URL != "https://cdn.example.com/slide1.jpg" {
		t.Errorf("expected slide1 URL, got %q", media1.URL)
	}
	if media1.Filename != "Photo 1 by meldadel [3933364417489830512].jpg" {
		t.Errorf("expected filename 'Photo 1 by meldadel [3933364417489830512].jpg', got %q", media1.Filename)
	}
}
