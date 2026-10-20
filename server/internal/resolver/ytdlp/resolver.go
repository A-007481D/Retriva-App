package ytdlp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"strings"

	"github.com/A-007481D/retriva/server/internal/resolver"
	"github.com/A-007481D/retriva/server/internal/security"
)

// Resolver uses yt-dlp to resolve complex media URLs.
type Resolver struct {
	binPath string
	cookies string
}

// New creates a new YtDlpResolver.
// binPath is the path to the yt-dlp executable.
// cookies is an optional path to a Netscape cookies.txt file.
func New(binPath, cookies string) *Resolver {
	if binPath == "" {
		binPath = "yt-dlp"
	}
	return &Resolver{
		binPath: binPath,
		cookies: cookies,
	}
}

// CanHandle returns true if yt-dlp can handle the URL.
// Since yt-dlp handles hundreds of sites, we let it try anything that is a valid HTTP(S) URL
// but we prioritize it after the DirectResolver.
func (r *Resolver) CanHandle(rawURL string) bool {
	u, err := url.ParseRequestURI(rawURL)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// Resolve shells out to yt-dlp to extract the direct URL and metadata.
func (r *Resolver) Resolve(ctx context.Context, rawURL string) (*resolver.MediaInfo, error) {
	parsedURL, err := security.ValidateURL(ctx, rawURL)
	if err != nil {
		return nil, err
	}

	var playlistItem string
	var storyID string
	cleanURL := rawURL

	if parsedURL != nil {
		q := parsedURL.Query()
		if val := q.Get("item"); val != "" {
			playlistItem = val
			q.Del("item")
		} else if val := q.Get("index"); val != "" {
			playlistItem = val
			q.Del("index")
		} else if val := q.Get("playlist_items"); val != "" {
			playlistItem = val
			q.Del("playlist_items")
		}

		if val := q.Get("story_id"); val != "" {
			storyID = val
			q.Del("story_id")
		} else if val := q.Get("story_media_id"); val != "" {
			storyID = val
			q.Del("story_media_id")
		}

		parsedURL.RawQuery = q.Encode()
		parsedURL.Fragment = ""
		cleanURL = parsedURL.String()
	}

	args := []string{
		"--dump-json",
		"-f", "b",
		"--no-warnings",
	}

	if playlistItem != "" {
		args = append(args, "--playlist-items", playlistItem)
	} else if storyID != "" {
		args = append(args, "--match-filter", fmt.Sprintf("id = %s", storyID))
	} else {
		args = append(args, "--no-playlist")
	}

	if r.cookies != "" {
		args = append(args, "--cookies", r.cookies)
	}

	args = append(args, cleanURL)

	cmd := exec.CommandContext(ctx, r.binPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("yt-dlp error: %w\nstderr: %s", err, stderr.String())
	}

	var data struct {
		URL       string  `json:"url"`
		Ext       string  `json:"ext"`
		Title     string  `json:"title"`
		Fulltitle string  `json:"fulltitle"`
		Filesize  float64 `json:"filesize"`
		Vcodec    string  `json:"vcodec"`
		Acodec    string  `json:"acodec"`
		Id        string  `json:"id"`
	}

	scanner := bufio.NewScanner(bytes.NewReader(stdout.Bytes()))
	// 10MB buffer for large JSON outputs
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	found := false
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if err := json.Unmarshal(line, &data); err == nil && data.URL != "" {
			found = true
			break
		}
	}

	if !found || data.URL == "" {
		return nil, fmt.Errorf("yt-dlp did not return a valid download URL")
	}

	// Sanitize output
	title := data.Title
	if title == "" {
		title = data.Fulltitle
	}
	if title == "" {
		title = data.Id
	}
	if title == "" {
		title = "downloaded_media"
	}
	if data.Id != "" && !strings.Contains(title, data.Id) {
		title = fmt.Sprintf("%s [%s]", title, data.Id)
	}

	ext := data.Ext
	if ext == "" {
		if data.Vcodec != "none" {
			ext = "mp4"
		} else {
			ext = "mp3"
		}
	}
	
	filename := security.SanitizeFilename(fmt.Sprintf("%s.%s", title, ext))

	size := int64(data.Filesize)
	if size == 0 {
		size = -1
	}

	// Determine generic mimetype based on extension
	mimeType := "application/octet-stream"
	if ext == "mp4" || ext == "webm" {
		mimeType = "video/" + ext
	} else if ext == "jpg" || ext == "jpeg" || ext == "png" {
		mimeType = "image/" + ext
	} else if ext == "mp3" || ext == "m4a" {
		mimeType = "audio/" + ext
	}

	// Note: We intentionally skip re-validating the CDN URL here.
	// The source URL was already SSRF-checked at the top of this function.
	// The CDN URL is extracted by yt-dlp from that trusted source and
	// re-checking it via DNS causes timeouts for CDN hostnames (e.g. fbcdn.net).

	return &resolver.MediaInfo{
		URL:       data.URL,
		MIMEType:  mimeType,
		Filename:  filename,
		SizeBytes: size,
	}, nil
}

// Inspect extracts playlist/highlight metadata without downloading media.
func (r *Resolver) Inspect(ctx context.Context, rawURL string) (*resolver.PlaylistInfo, error) {
	parsedURL, err := security.ValidateURL(ctx, rawURL)
	if err != nil {
		return nil, err
	}

	cleanURL := rawURL
	if parsedURL != nil {
		q := parsedURL.Query()
		q.Del("item")
		q.Del("index")
		q.Del("playlist_items")
		q.Del("story_id")
		q.Del("story_media_id")
		parsedURL.RawQuery = q.Encode()
		parsedURL.Fragment = ""
		cleanURL = parsedURL.String()
	}

	args := []string{
		"--flat-playlist",
		"--dump-json",
		"--no-warnings",
	}

	if r.cookies != "" {
		args = append(args, "--cookies", r.cookies)
	}

	args = append(args, cleanURL)

	cmd := exec.CommandContext(ctx, r.binPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("yt-dlp inspect error: %w\nstderr: %s", err, stderr.String())
	}

	var items []resolver.PlaylistItem
	scanner := bufio.NewScanner(&stdout)
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	playlistTitle := ""
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry struct {
			PlaylistIndex int     `json:"playlist_index"`
			ID            string  `json:"id"`
			Title         string  `json:"title"`
			Fulltitle     string  `json:"fulltitle"`
			Duration      float64 `json:"duration"`
			Timestamp     int64   `json:"timestamp"`
			Thumbnail     string  `json:"thumbnail"`
			PlaylistTitle string  `json:"playlist_title"`
			Playlist      string  `json:"playlist"`
		}
		if err := json.Unmarshal(line, &entry); err == nil {
			t := entry.Title
			if t == "" {
				t = entry.Fulltitle
			}
			idx := entry.PlaylistIndex
			if idx == 0 {
				idx = len(items) + 1
			}
			if playlistTitle == "" {
				if entry.PlaylistTitle != "" {
					playlistTitle = entry.PlaylistTitle
				} else if entry.Playlist != "" {
					playlistTitle = entry.Playlist
				}
			}
			items = append(items, resolver.PlaylistItem{
				Index:     idx,
				ID:        entry.ID,
				Title:     t,
				Duration:  entry.Duration,
				Timestamp: entry.Timestamp,
				Thumbnail: entry.Thumbnail,
			})
		}
	}

	if playlistTitle == "" && len(items) > 0 {
		playlistTitle = items[0].Title
	}

	if len(items) == 0 {
		// Fallback: If yt-dlp flat-playlist yielded no items, resolve the single media
		mediaInfo, err := r.Resolve(ctx, rawURL)
		if err == nil && mediaInfo != nil {
			return &resolver.PlaylistInfo{
				Title: mediaInfo.Filename,
				Count: 1,
				Items: []resolver.PlaylistItem{
					{
						Index: 1,
						ID:    "1",
						Title: mediaInfo.Filename,
					},
				},
			}, nil
		}
	}

	return &resolver.PlaylistInfo{
		Title: playlistTitle,
		Count: len(items),
		Items: items,
	}, nil
}
