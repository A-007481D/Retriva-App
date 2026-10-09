// Package instagram implements a native resolver for Instagram Highlights and Stories.
package instagram

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/A-007481D/retriva/server/internal/resolver"
	"github.com/A-007481D/retriva/server/internal/security"
)

var highlightRegex = regexp.MustCompile(`/stories/highlights/(\d+)`)

type igCandidate struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type igVideoVersion struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type igItem struct {
	PK            json.Number      `json:"pk"`
	ID            string           `json:"id"`
	Code          string           `json:"code"`
	MediaType     int              `json:"media_type"` // 1 = Photo, 2 = Video
	TakenAt       int64            `json:"taken_at"`
	VideoDuration float64          `json:"video_duration"`
	ImageVersions struct {
		Candidates []igCandidate `json:"candidates"`
	} `json:"image_versions2"`
	VideoVersions []igVideoVersion `json:"video_versions"`
	User          struct {
		Username string `json:"username"`
		FullName string `json:"full_name"`
	} `json:"user"`
}

type igReel struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	User  struct {
		Username string `json:"username"`
		FullName string `json:"full_name"`
	} `json:"user"`
	Items []igItem `json:"items"`
}

type igFeedResponse struct {
	Status string            `json:"status"`
	Reels  map[string]igReel `json:"reels"`
}

// Resolver handles Instagram Highlight URLs directly using the Instagram mobile API.
type Resolver struct {
	cookiesPath string
}

// New creates an Instagram resolver with optional Netscape cookies path.
func New(cookiesPath string) *Resolver {
	return &Resolver{
		cookiesPath: cookiesPath,
	}
}

// CanHandle returns true if the URL points to an Instagram Highlight.
func (r *Resolver) CanHandle(rawURL string) bool {
	return strings.Contains(rawURL, "instagram.com/stories/highlights/") ||
		(strings.Contains(rawURL, "instagram.com") && strings.Contains(rawURL, "/highlights/"))
}

func (r *Resolver) extractHighlightID(rawURL string) (string, error) {
	m := highlightRegex.FindStringSubmatch(rawURL)
	if len(m) >= 2 {
		return m[1], nil
	}
	return "", fmt.Errorf("could not extract highlight id from URL")
}

func (r *Resolver) createHTTPClient() (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	if r.cookiesPath != "" {
		if f, err := os.Open(r.cookiesPath); err == nil {
			defer f.Close()
			sc := bufio.NewScanner(f)
			for sc.Scan() {
				line := strings.TrimSpace(sc.Text())
				if strings.HasPrefix(line, "#HttpOnly_") {
					line = line[len("#HttpOnly_"):]
				} else if strings.HasPrefix(line, "#") || line == "" {
					continue
				}
				parts := strings.Split(line, "\t")
				if len(parts) < 7 {
					continue
				}
				domain := parts[0]
				expiresUnix, _ := strconv.ParseInt(parts[4], 10, 64)
				val := strings.Trim(parts[6], "\"")
				c := &http.Cookie{
					Name:   parts[5],
					Value:  val,
					Path:   parts[2],
					Domain: domain,
					Secure: strings.EqualFold(parts[3], "TRUE"),
				}
				if expiresUnix > 0 {
					c.Expires = time.Unix(expiresUnix, 0)
				}
				u, err := url.Parse("https://" + strings.TrimPrefix(domain, "."))
				if err == nil {
					jar.SetCookies(u, []*http.Cookie{c})
				}
			}
		}
	}

	return &http.Client{
		Jar:     jar,
		Timeout: 20 * time.Second,
	}, nil
}

func (r *Resolver) fetchReel(ctx context.Context, highlightID string) (*igReel, error) {
	client, err := r.createHTTPClient()
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	apiURL := fmt.Sprintf("https://i.instagram.com/api/v1/feed/reels_media/?reel_ids=highlight:%s", highlightID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("X-IG-App-ID", "936619743392459")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("instagram api request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("instagram api returned status %d", resp.StatusCode)
	}

	var data igFeedResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode instagram response: %w", err)
	}

	reelKey := "highlight:" + highlightID
	reel, ok := data.Reels[reelKey]
	if !ok {
		return nil, fmt.Errorf("highlight %s not found in response", highlightID)
	}

	return &reel, nil
}

// Inspect fetches metadata for all stories in the highlight (both photos and videos).
func (r *Resolver) Inspect(ctx context.Context, rawURL string) (*resolver.PlaylistInfo, error) {
	if _, err := security.ValidateURL(ctx, rawURL); err != nil {
		return nil, err
	}

	highlightID, err := r.extractHighlightID(rawURL)
	if err != nil {
		return nil, err
	}

	reel, err := r.fetchReel(ctx, highlightID)
	if err != nil {
		return nil, err
	}

	defaultUsername := reel.User.Username
	if defaultUsername == "" {
		defaultUsername = reel.User.FullName
	}
	if defaultUsername == "" {
		defaultUsername = "instagram_user"
	}

	var items []resolver.PlaylistItem
	for i, it := range reel.Items {
		user := it.User.Username
		if user == "" {
			user = defaultUsername
		}

		itemType := "video"
		if it.MediaType == 1 {
			itemType = "photo"
		}

		title := fmt.Sprintf("%s by %s", strings.Title(itemType), user)

		thumb := ""
		if len(it.ImageVersions.Candidates) > 0 {
			thumb = it.ImageVersions.Candidates[0].URL
		}

		pkStr := string(it.PK)
		if pkStr == "" {
			pkStr = it.Code
		}
		if pkStr == "" {
			pkStr = it.ID
		}

		items = append(items, resolver.PlaylistItem{
			Index:     i + 1,
			ID:        pkStr,
			Type:      itemType,
			Title:     title,
			Duration:  it.VideoDuration,
			Timestamp: it.TakenAt,
			Thumbnail: thumb,
		})
	}

	playlistTitle := reel.Title
	if playlistTitle == "" || playlistTitle == "ً" {
		playlistTitle = fmt.Sprintf("Highlight by %s", defaultUsername)
	}

	return &resolver.PlaylistInfo{
		Title: playlistTitle,
		Count: len(items),
		Items: items,
	}, nil
}

// Resolve extracts the direct download URL for a specific highlight item (photo or video).
func (r *Resolver) Resolve(ctx context.Context, rawURL string) (*resolver.MediaInfo, error) {
	parsedURL, err := security.ValidateURL(ctx, rawURL)
	if err != nil {
		return nil, err
	}

	highlightID, err := r.extractHighlightID(rawURL)
	if err != nil {
		return nil, err
	}

	reel, err := r.fetchReel(ctx, highlightID)
	if err != nil {
		return nil, err
	}

	if len(reel.Items) == 0 {
		return nil, fmt.Errorf("no items found in highlight")
	}

	// Parse item selection from query
	targetIndex := -1
	targetID := ""
	if parsedURL != nil {
		q := parsedURL.Query()
		if itemParam := q.Get("item"); itemParam != "" {
			if idx, err := strconv.Atoi(itemParam); err == nil {
				targetIndex = idx
			} else {
				targetID = itemParam
			}
		} else if idxParam := q.Get("index"); idxParam != "" {
			if idx, err := strconv.Atoi(idxParam); err == nil {
				targetIndex = idx
			}
		} else if idParam := q.Get("story_id"); idParam != "" {
			targetID = idParam
		} else if idParam := q.Get("id"); idParam != "" {
			targetID = idParam
		}
	}

	var selectedItem *igItem
	if targetIndex > 0 && targetIndex <= len(reel.Items) {
		selectedItem = &reel.Items[targetIndex-1]
	} else if targetID != "" {
		for i := range reel.Items {
			it := &reel.Items[i]
			if string(it.PK) == targetID || it.Code == targetID || it.ID == targetID {
				selectedItem = it
				break
			}
		}
	}

	// Fallback to first item if none matched or specified
	if selectedItem == nil {
		selectedItem = &reel.Items[0]
	}

	username := selectedItem.User.Username
	if username == "" {
		username = reel.User.Username
	}
	if username == "" {
		username = "instagram_user"
	}

	itemPK := string(selectedItem.PK)
	if itemPK == "" {
		itemPK = selectedItem.Code
	}

	var downloadURL string
	var mimeType string
	var filename string

	if selectedItem.MediaType == 1 { // Photo
		if len(selectedItem.ImageVersions.Candidates) == 0 {
			return nil, fmt.Errorf("no image candidates found for photo item")
		}
		downloadURL = selectedItem.ImageVersions.Candidates[0].URL
		mimeType = "image/jpeg"
		filename = fmt.Sprintf("Photo by %s [%s].jpg", username, itemPK)
	} else { // Video
		if len(selectedItem.VideoVersions) == 0 {
			return nil, fmt.Errorf("no video versions found for video item")
		}
		downloadURL = selectedItem.VideoVersions[0].URL
		mimeType = "video/mp4"
		filename = fmt.Sprintf("Video by %s [%s].mp4", username, itemPK)
	}

	filename = security.SanitizeFilename(filename)

	return &resolver.MediaInfo{
		URL:       downloadURL,
		MIMEType:  mimeType,
		Filename:  filename,
		SizeBytes: -1,
	}, nil
}
