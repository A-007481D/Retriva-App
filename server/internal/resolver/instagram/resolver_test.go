package instagram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/A-007481D/retriva/server/internal/security"
)

func init() {
	security.DisableSSRFCheck = true
}

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

func TestCreateHTTPClientCookies(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test_cookies_*.txt")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	cookieContent := `# Netscape HTTP Cookie File
.instagram.com	TRUE	/	TRUE	1808093035	sessionid	testsession123
.instagram.com	TRUE	/	TRUE	1808093035	ds_user_id	123456789
`
	if _, err := tempFile.Write([]byte(cookieContent)); err != nil {
		t.Fatalf("write cookie: %v", err)
	}
	tempFile.Close()

	r := New(tempFile.Name())
	client, err := r.createHTTPClient()
	if err != nil {
		t.Fatalf("createHTTPClient error: %v", err)
	}
	u, _ := url.Parse("https://i.instagram.com/api/v1/feed/reels_media/")
	cookies := client.Jar.Cookies(u)
	if len(cookies) != 2 {
		t.Errorf("expected 2 cookies, got %d", len(cookies))
	}
}

func TestExtractTarget(t *testing.T) {
	r := New("")

	target, err := r.extractTarget("https://www.instagram.com/stories/highlights/18122897002451585/")
	if err != nil || target.Type != targetHighlight || target.ID != "18122897002451585" {
		t.Fatalf("unexpected extractTarget for highlight: %v, %v", target, err)
	}

	target, err = r.extractTarget("https://www.instagram.com/p/DaWHkQpid02/?img_index=2")
	if err != nil || target.Type != targetPost || target.ID != "3933364602658217270" {
		t.Fatalf("unexpected extractTarget for post: %v, %v", target, err)
	}

	target, err = r.extractTarget("https://www.instagram.com/stories/someone/3982929260775922792/")
	if err != nil || target.Type != targetStory || target.Username != "someone" || target.StoryID != "3982929260775922792" {
		t.Fatalf("unexpected extractTarget for story with ID: %v, %v", target, err)
	}

	target, err = r.extractTarget("https://www.instagram.com/stories/omnia7086/")
	if err != nil || target.Type != targetStory || target.Username != "omnia7086" || target.StoryID != "" {
		t.Fatalf("unexpected extractTarget for story without ID: %v, %v", target, err)
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

func TestInspectAndResolveStory(t *testing.T) {
	mockReelsResponse := map[string]any{
		"status": "ok",
		"reels": map[string]any{
			"3422436823": map[string]any{
				"id":    "3422436823",
				"title": "",
				"user": map[string]any{
					"username": "omnia7086",
				},
				"items": []map[string]any{
					{
						"pk":             "4012173398310500441",
						"id":             "4012173398310500441",
						"code":           "storyCode1",
						"media_type":     1, // Photo
						"taken_at":       1700000000,
						"video_duration": 0.0,
						"image_versions2": map[string]any{
							"candidates": []map[string]any{
								{
									"url":    "https://cdn.example.com/story1.jpg",
									"width":  1080,
									"height": 1920,
								},
							},
						},
						"user": map[string]any{
							"username": "omnia7086",
						},
					},
					{
						"pk":             "4012173398310500443",
						"id":             "4012173398310500443",
						"code":           "storyCode2",
						"media_type":     2, // Video
						"taken_at":       1700000010,
						"video_duration": 15.0,
						"video_versions": []map[string]any{
							{
								"url":    "https://cdn.example.com/story2.mp4",
								"width":  1080,
								"height": 1920,
							},
						},
						"image_versions2": map[string]any{
							"candidates": []map[string]any{
								{
									"url":    "https://cdn.example.com/story2_thumb.jpg",
									"width":  1080,
									"height": 1920,
								},
							},
						},
						"user": map[string]any{
							"username": "omnia7086",
						},
					},
				},
			},
		},
	}

	mockProfileResponse := map[string]any{
		"data": map[string]any{
			"user": map[string]any{
				"id":        "3422436823",
				"username":  "omnia7086",
				"full_name": "Omnia",
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if req.URL.Path == "/api/v1/users/web_profile_info/" {
			json.NewEncoder(w).Encode(mockProfileResponse)
			return
		}
		if req.URL.Path == "/api/v1/feed/reels_media/" {
			json.NewEncoder(w).Encode(mockReelsResponse)
			return
		}
		http.NotFound(w, req)
	}))
	defer srv.Close()

	r := New("")
	r.apiBaseURL = srv.URL

	ctx := context.Background()

	// 1. Test Inspect without story ID
	info, err := r.Inspect(ctx, "https://www.instagram.com/stories/omnia7086/")
	if err != nil {
		t.Fatalf("Inspect story failed: %v", err)
	}

	if info.Count != 2 {
		t.Errorf("expected count 2, got %d", info.Count)
	}
	if info.Title != "Stories by @omnia7086" {
		t.Errorf("expected title 'Stories by @omnia7086', got %q", info.Title)
	}
	if len(info.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(info.Items))
	}
	if info.Items[0].Type != "photo" || info.Items[1].Type != "video" {
		t.Errorf("expected photo and video, got %s and %s", info.Items[0].Type, info.Items[1].Type)
	}

	// 2. Test Inspect with story ID
	info2, err := r.Inspect(ctx, "https://www.instagram.com/stories/omnia7086/4012173398310500443/")
	if err != nil {
		t.Fatalf("Inspect story with ID failed: %v", err)
	}
	if info2.Count != 2 {
		t.Errorf("expected count 2, got %d", info2.Count)
	}

	// 3. Test Resolve specific story ID
	media, err := r.Resolve(ctx, "https://www.instagram.com/stories/omnia7086/4012173398310500443/")
	if err != nil {
		t.Fatalf("Resolve story with ID failed: %v", err)
	}
	if media.URL != "https://cdn.example.com/story2.mp4" {
		t.Errorf("expected story2 video URL, got %q", media.URL)
	}
	if media.MIMEType != "video/mp4" {
		t.Errorf("expected video/mp4, got %q", media.MIMEType)
	}

	// 4. Test Resolve story item 1
	media1, err := r.Resolve(ctx, "https://www.instagram.com/stories/omnia7086/?item=1")
	if err != nil {
		t.Fatalf("Resolve story with item=1 failed: %v", err)
	}
	if media1.URL != "https://cdn.example.com/story1.jpg" {
		t.Errorf("expected story1 photo URL, got %q", media1.URL)
	}
	if media1.MIMEType != "image/jpeg" {
		t.Errorf("expected image/jpeg, got %q", media1.MIMEType)
	}

	// 5. Test Resolve story with item=<storyID> (large numeric ID as passed by multi-item selection)
	mediaByStoryID, err := r.Resolve(ctx, "https://www.instagram.com/stories/omnia7086/?item=4012173398310500443")
	if err != nil {
		t.Fatalf("Resolve story with item=4012173398310500443 failed: %v", err)
	}
	if mediaByStoryID.URL != "https://cdn.example.com/story2.mp4" {
		t.Errorf("expected story2 video URL, got %q", mediaByStoryID.URL)
	}
}

func TestInspectAndResolveHighlightGraphQL(t *testing.T) {
	mockGQLResponse := map[string]any{
		"data": map[string]any{
			"reels_media": []map[string]any{
				{
					"__typename": "GraphHighlightReel",
					"id":         "18008730106751263",
					"owner": map[string]any{
						"id":       "3422436823",
						"username": "omnia7086",
					},
					"items": []map[string]any{
						{
							"id":                 "4016792134560039347",
							"is_video":           true,
							"video_duration":     15.0,
							"taken_at_timestamp": 1729000000,
							"display_url":        "https://cdn.example.com/thumb1.jpg",
							"video_resources": []map[string]any{
								{
									"src":           "https://cdn.example.com/video1.mp4",
									"config_width":  720,
									"config_height": 1280,
								},
							},
						},
						{
							"id":                 "4016791798243980558",
							"is_video":           false,
							"taken_at_timestamp": 1729000100,
							"display_url":        "https://cdn.example.com/photo2.jpg",
						},
					},
				},
			},
		},
		"status": "ok",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockGQLResponse)
	}))
	defer srv.Close()

	r := New("")
	r.apiBaseURL = srv.URL
	ctx := context.Background()

	// 1. Inspect Highlight
	info, err := r.Inspect(ctx, "https://www.instagram.com/stories/highlights/18008730106751263/")
	if err != nil {
		t.Fatalf("Inspect highlight failed: %v", err)
	}

	if info.Count != 2 {
		t.Fatalf("expected count 2, got %d", info.Count)
	}
	if info.Title != "Highlight by omnia7086" {
		t.Errorf("expected title 'Highlight by omnia7086', got %q", info.Title)
	}
	if info.Items[0].Type != "video" || info.Items[0].ID != "4016792134560039347" {
		t.Errorf("item 0 mismatch: %+v", info.Items[0])
	}
	if info.Items[1].Type != "photo" || info.Items[1].ID != "4016791798243980558" {
		t.Errorf("item 1 mismatch: %+v", info.Items[1])
	}

	// 2. Resolve Highlight (default item 1)
	media1, err := r.Resolve(ctx, "https://www.instagram.com/stories/highlights/18008730106751263/")
	if err != nil {
		t.Fatalf("Resolve highlight failed: %v", err)
	}
	if media1.URL != "https://cdn.example.com/video1.mp4" {
		t.Errorf("expected video1 URL, got %q", media1.URL)
	}
	if media1.MIMEType != "video/mp4" {
		t.Errorf("expected video/mp4, got %q", media1.MIMEType)
	}

	// 3. Resolve Highlight with specific item ID
	media2, err := r.Resolve(ctx, "https://www.instagram.com/stories/highlights/18008730106751263/?item=4016791798243980558")
	if err != nil {
		t.Fatalf("Resolve highlight with item failed: %v", err)
	}
	if media2.URL != "https://cdn.example.com/photo2.jpg" {
		t.Errorf("expected photo2 URL, got %q", media2.URL)
	}
	if media2.MIMEType != "image/jpeg" {
		t.Errorf("expected image/jpeg, got %q", media2.MIMEType)
	}
}
