// Package ytdlp implements a resolver using the yt-dlp binary to extract media URLs.
package ytdlp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

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
// Since yt-dlp handles hundreds of sites, we let it try anything that is a valid URL
// but we prioritize it after the DirectResolver.
func (r *Resolver) CanHandle(rawURL string) bool {
	_, err := security.ValidateURL(context.Background(), rawURL)
	return err == nil
}

// Resolve shells out to yt-dlp to extract the direct URL and metadata.
func (r *Resolver) Resolve(ctx context.Context, rawURL string) (*resolver.MediaInfo, error) {
	_, err := security.ValidateURL(ctx, rawURL)
	if err != nil {
		return nil, err
	}

	args := []string{
		"--dump-json",
		"-f", "b",
		"--no-playlist",
		"--no-warnings",
	}

	if r.cookies != "" {
		args = append(args, "--cookies", r.cookies)
	}

	args = append(args, rawURL)

	cmd := exec.CommandContext(ctx, r.binPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("yt-dlp error: %w\nstderr: %s", err, stderr.String())
	}

	var data struct {
		URL      string  `json:"url"`
		Ext      string  `json:"ext"`
		Title    string  `json:"title"`
		Filesize float64 `json:"filesize"`
		Vcodec   string  `json:"vcodec"`
		Acodec   string  `json:"acodec"`
		Id       string  `json:"id"`
	}

	if err := json.NewDecoder(bytes.NewReader(stdout.Bytes())).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to parse yt-dlp json: %w", err)
	}

	if data.URL == "" {
		return nil, fmt.Errorf("yt-dlp did not return a valid download URL")
	}

	// Sanitize output
	title := data.Title
	if title == "" {
		title = data.Id
	}
	if title == "" {
		title = "downloaded_media"
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
	} else if ext == "jpg" || ext == "png" {
		mimeType = "image/" + ext
	} else if ext == "mp3" || ext == "m4a" {
		mimeType = "audio/" + ext
	}

	// Verify the extracted URL is safe
	if err := security.CheckSSRF(ctx, data.URL); err != nil {
		return nil, fmt.Errorf("SSRF check failed on yt-dlp URL: %w", err)
	}

	return &resolver.MediaInfo{
		URL:       data.URL,
		MIMEType:  mimeType,
		Filename:  filename,
		SizeBytes: size,
	}, nil
}
