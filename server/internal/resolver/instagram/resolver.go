// Package instagram implements a native resolver for Instagram Posts, Carousels, Stories, and Highlights.
package instagram

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
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

var (
	highlightRegex = regexp.MustCompile(`/(?:stories/)?highlights/(\d+)`)
	postRegex      = regexp.MustCompile(`/(?:p|reel|reels|tv)/([A-Za-z0-9_-]+)`)
	userStoryRegex = regexp.MustCompile(`/stories/([A-Za-z0-9_.]+)(?:/(\d+))?/?`)
)

type targetType int

const (
	targetHighlight targetType = iota
	targetPost
	targetStory
)

type igTarget struct {
	Type     targetType
	ID       string // highlightID, mediaID, or storyID
	Username string // for user stories
	StoryID  string // specific story ID if present in URL
}

const igAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

func shortcodeToMediaID(code string) (string, error) {
	if code == "" {
		return "", fmt.Errorf("empty shortcode")
	}
	var id big.Int
	base := big.NewInt(64)
	for _, char := range code {
		idx := strings.IndexRune(igAlphabet, char)
		if idx == -1 {
			return "", fmt.Errorf("invalid shortcode character: %c", char)
		}
		id.Mul(&id, base)
		id.Add(&id, big.NewInt(int64(idx)))
	}
	return id.String(), nil
}

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
	MediaType     int              `json:"media_type"` // 1 = Photo, 2 = Video, 8 = Carousel
	TakenAt       int64            `json:"taken_at"`
	VideoDuration float64          `json:"video_duration"`
	ImageVersions struct {
		Candidates []igCandidate `json:"candidates"`
	} `json:"image_versions2"`
	VideoVersions []igVideoVersion `json:"video_versions"`
	User          struct {
		PK       json.Number `json:"pk"`
		ID       string      `json:"id"`
		Username string      `json:"username"`
		FullName string      `json:"full_name"`
	} `json:"user"`
	Caption struct {
		Text string `json:"text"`
	} `json:"caption"`
	CarouselMedia []igItem `json:"carousel_media"`
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

// Resolver handles Instagram URLs (posts, carousels, reels, stories, highlights) directly using the Instagram mobile API.
type Resolver struct {
	cookiesPath string
	apiBaseURL  string
}

// New creates an Instagram resolver with optional Netscape cookies path.
func New(cookiesPath string) *Resolver {
	return &Resolver{
		cookiesPath: cookiesPath,
	}
}

func (r *Resolver) base() string {
	if r.apiBaseURL != "" {
		return r.apiBaseURL
	}
	return "https://i.instagram.com"
}

// CanHandle returns true if the URL points to an Instagram post, carousel, reel, story, or highlight.
func (r *Resolver) CanHandle(rawURL string) bool {
	if !strings.Contains(rawURL, "instagram.com") {
		return false
	}
	return highlightRegex.MatchString(rawURL) ||
		postRegex.MatchString(rawURL) ||
		userStoryRegex.MatchString(rawURL) ||
		strings.Contains(rawURL, "/stories/") ||
		strings.Contains(rawURL, "/highlights/")
}

func (r *Resolver) extractTarget(rawURL string) (*igTarget, error) {
	if m := highlightRegex.FindStringSubmatch(rawURL); len(m) >= 2 {
		return &igTarget{Type: targetHighlight, ID: m[1]}, nil
	}
	if m := postRegex.FindStringSubmatch(rawURL); len(m) >= 2 {
		mediaID, err := shortcodeToMediaID(m[1])
		if err != nil {
			return nil, err
		}
		return &igTarget{Type: targetPost, ID: mediaID}, nil
	}
	if m := userStoryRegex.FindStringSubmatch(rawURL); len(m) >= 2 {
		user := m[1]
		var storyID string
		if len(m) >= 3 && m[2] != "" {
			storyID = m[2]
		}
		if user == "highlights" {
			if storyID != "" {
				return &igTarget{Type: targetHighlight, ID: storyID}, nil
			}
			return nil, fmt.Errorf("missing highlight ID")
		}
		return &igTarget{
			Type:     targetStory,
			Username: user,
			StoryID:  storyID,
			ID:       storyID,
		}, nil
	}
	return nil, fmt.Errorf("could not identify Instagram media, story, or highlight from URL")
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
					if strings.Contains(domain, "instagram.com") {
						if uWww, err := url.Parse("https://www.instagram.com"); err == nil {
							jar.SetCookies(uWww, []*http.Cookie{c})
						}
						if uI, err := url.Parse("https://i.instagram.com"); err == nil {
							jar.SetCookies(uI, []*http.Cookie{c})
						}
					}
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

	apiURL := fmt.Sprintf("%s/api/v1/feed/reels_media/?reel_ids=highlight:%s", r.base(), highlightID)
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

func (r *Resolver) fetchMediaInfo(ctx context.Context, mediaID string) (*igItem, error) {
	client, err := r.createHTTPClient()
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	apiURL := fmt.Sprintf("%s/api/v1/media/%s/info/", r.base(), mediaID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("X-IG-App-ID", "936619743392459")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("instagram media api request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("instagram media api returned status %d", resp.StatusCode)
	}

	var data struct {
		Status string   `json:"status"`
		Items  []igItem `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode instagram media response: %w", err)
	}

	if len(data.Items) == 0 {
		return nil, fmt.Errorf("media %s not found in response", mediaID)
	}

	return &data.Items[0], nil
}

// Inspect fetches metadata for highlights, user stories, or multi-item posts/carousels.
func (r *Resolver) Inspect(ctx context.Context, rawURL string) (*resolver.PlaylistInfo, error) {
	if _, err := security.ValidateURL(ctx, rawURL); err != nil {
		return nil, err
	}

	target, err := r.extractTarget(rawURL)
	if err != nil {
		return nil, err
	}

	switch target.Type {
	case targetHighlight:
		return r.inspectHighlight(ctx, target.ID)
	case targetStory:
		return r.inspectStory(ctx, target.Username, target.StoryID)
	case targetPost:
		return r.inspectMedia(ctx, target.ID)
	}

	return nil, fmt.Errorf("unsupported Instagram target type")
}

func (r *Resolver) inspectHighlight(ctx context.Context, highlightID string) (*resolver.PlaylistInfo, error) {
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

func (r *Resolver) inspectStory(ctx context.Context, username, storyID string) (*resolver.PlaylistInfo, error) {
	reel, err := r.fetchUserReel(ctx, username, storyID)
	if err != nil {
		return nil, err
	}

	defaultUsername := reel.User.Username
	if defaultUsername == "" {
		defaultUsername = username
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

		title := fmt.Sprintf("Story %d by @%s", i+1, user)

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

	return &resolver.PlaylistInfo{
		Title: fmt.Sprintf("Stories by @%s", defaultUsername),
		Count: len(items),
		Items: items,
	}, nil
}

func (r *Resolver) fetchUserReel(ctx context.Context, username, storyID string) (*igReel, error) {
	client, err := r.createHTTPClient()
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	userID := ""
	// 1. If we have a storyID, try to query media info
	if storyID != "" {
		if item, err := r.fetchMediaInfo(ctx, storyID); err == nil && item != nil {
			if string(item.User.PK) != "" {
				userID = string(item.User.PK)
			} else if item.User.Username != "" {
				username = item.User.Username
			}
		}
	}

	// 2. If userID still not resolved, query profile info
	if userID == "" && username != "" {
		if uid, err := r.fetchUserIDByUsername(ctx, client, username); err == nil && uid != "" {
			userID = uid
		}
	}

	if userID == "" && storyID != "" {
		userID = storyID
	}

	if userID == "" {
		return nil, fmt.Errorf("could not determine user ID for @%s (please ensure cookies.txt has a valid sessionid)", username)
	}

	apiURL := fmt.Sprintf("%s/api/v1/feed/reels_media/?reel_ids=%s", r.base(), userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("X-IG-App-ID", "936619743392459")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("instagram reels request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("Instagram authentication required: please provide a valid sessionid cookie in cookies.txt to access stories")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("instagram reels api returned status %d", resp.StatusCode)
	}

	var data igFeedResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode instagram response: %w", err)
	}

	if reel, ok := data.Reels[userID]; ok {
		return &reel, nil
	}

	for _, reel := range data.Reels {
		return &reel, nil
	}

	return nil, fmt.Errorf("no active stories currently found for @%s", username)
}

func (r *Resolver) fetchUserIDByUsername(ctx context.Context, client *http.Client, username string) (string, error) {
	apiURL := fmt.Sprintf("%s/api/v1/users/web_profile_info/?username=%s", r.base(), username)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("X-IG-App-ID", "936619743392459")
	req.Header.Set("Referer", fmt.Sprintf("https://www.instagram.com/%s/", username))

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("web_profile_info returned status %d", resp.StatusCode)
	}

	var data struct {
		Data struct {
			User struct {
				ID string `json:"id"`
				PK string `json:"pk"`
			} `json:"user"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	if data.Data.User.ID != "" {
		return data.Data.User.ID, nil
	}
	if data.Data.User.PK != "" {
		return data.Data.User.PK, nil
	}

	return "", fmt.Errorf("user ID not found in profile info")
}

func (r *Resolver) inspectMedia(ctx context.Context, mediaID string) (*resolver.PlaylistInfo, error) {
	item, err := r.fetchMediaInfo(ctx, mediaID)
	if err != nil {
		return nil, err
	}

	username := item.User.Username
	if username == "" {
		username = item.User.FullName
	}
	if username == "" {
		username = "instagram_user"
	}

	// Check if this is a carousel (multi-slide post)
	if item.MediaType == 8 || len(item.CarouselMedia) > 0 {
		var items []resolver.PlaylistItem
		for i, child := range item.CarouselMedia {
			childType := "video"
			if child.MediaType == 1 {
				childType = "photo"
			}

			thumb := ""
			if len(child.ImageVersions.Candidates) > 0 {
				thumb = child.ImageVersions.Candidates[0].URL
			}

			pkStr := string(child.PK)
			if pkStr == "" {
				pkStr = child.Code
			}
			if pkStr == "" {
				pkStr = child.ID
			}

			items = append(items, resolver.PlaylistItem{
				Index:     i + 1,
				ID:        pkStr,
				Type:      childType,
				Title:     fmt.Sprintf("%s %d by %s", strings.Title(childType), i+1, username),
				Duration:  child.VideoDuration,
				Timestamp: child.TakenAt,
				Thumbnail: thumb,
			})
		}

		postTitle := item.Caption.Text
		if postTitle == "" {
			postTitle = fmt.Sprintf("Post by %s", username)
		} else if len(postTitle) > 60 {
			postTitle = postTitle[:57] + "..."
		}

		return &resolver.PlaylistInfo{
			Title: postTitle,
			Count: len(items),
			Items: items,
		}, nil
	}

	// Single item (photo, video, or reel)
	itemType := "video"
	if item.MediaType == 1 {
		itemType = "photo"
	}

	thumb := ""
	if len(item.ImageVersions.Candidates) > 0 {
		thumb = item.ImageVersions.Candidates[0].URL
	}

	pkStr := string(item.PK)
	if pkStr == "" {
		pkStr = item.Code
	}

	singleTitle := item.Caption.Text
	if singleTitle == "" {
		singleTitle = fmt.Sprintf("%s by %s", strings.Title(itemType), username)
	} else if len(singleTitle) > 60 {
		singleTitle = singleTitle[:57] + "..."
	}

	return &resolver.PlaylistInfo{
		Title: singleTitle,
		Count: 1,
		Items: []resolver.PlaylistItem{
			{
				Index:     1,
				ID:        pkStr,
				Type:      itemType,
				Title:     singleTitle,
				Duration:  item.VideoDuration,
				Timestamp: item.TakenAt,
				Thumbnail: thumb,
			},
		},
	}, nil
}

// Resolve extracts the direct download URL for Instagram items (highlight stories, user stories, posts, carousel slides, reels).
func (r *Resolver) Resolve(ctx context.Context, rawURL string) (*resolver.MediaInfo, error) {
	parsedURL, err := security.ValidateURL(ctx, rawURL)
	if err != nil {
		return nil, err
	}

	target, err := r.extractTarget(rawURL)
	if err != nil {
		return nil, err
	}

	switch target.Type {
	case targetHighlight:
		return r.resolveHighlight(ctx, parsedURL, target.ID)
	case targetStory:
		return r.resolveStory(ctx, parsedURL, target.Username, target.StoryID)
	case targetPost:
		return r.resolveMedia(ctx, parsedURL, target.ID)
	}

	return nil, fmt.Errorf("unsupported Instagram target type")
}

func selectItem(items []igItem, itemParam string) (*igItem, int) {
	if len(items) == 0 {
		return nil, 0
	}
	if itemParam != "" {
		for i := range items {
			it := &items[i]
			if string(it.PK) == itemParam || it.Code == itemParam || it.ID == itemParam || strings.HasPrefix(it.ID, itemParam+"_") {
				return it, i + 1
			}
		}
		if idx, err := strconv.Atoi(itemParam); err == nil && idx >= 1 && idx <= len(items) {
			return &items[idx-1], idx
		}
	}
	return &items[0], 1
}

func (r *Resolver) resolveStory(ctx context.Context, parsedURL *url.URL, username, storyID string) (*resolver.MediaInfo, error) {
	reel, err := r.fetchUserReel(ctx, username, storyID)
	if err != nil {
		return nil, err
	}

	if len(reel.Items) == 0 {
		return nil, fmt.Errorf("no active stories found for @%s", username)
	}

	itemParam := ""
	if parsedURL != nil {
		q := parsedURL.Query()
		if p := q.Get("item"); p != "" {
			itemParam = p
		} else if p := q.Get("story_id"); p != "" {
			itemParam = p
		} else if p := q.Get("id"); p != "" {
			itemParam = p
		} else if p := q.Get("index"); p != "" {
			itemParam = p
		}
	}
	if itemParam == "" && storyID != "" {
		itemParam = storyID
	}

	selectedItem, _ := selectItem(reel.Items, itemParam)

	user := selectedItem.User.Username
	if user == "" {
		user = username
	}
	if user == "" {
		user = "instagram_user"
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
			return nil, fmt.Errorf("no image candidates found for photo story")
		}
		downloadURL = selectedItem.ImageVersions.Candidates[0].URL
		mimeType = "image/jpeg"
		filename = fmt.Sprintf("Story by %s [%s].jpg", user, itemPK)
	} else { // Video
		if len(selectedItem.VideoVersions) == 0 {
			return nil, fmt.Errorf("no video versions found for video story")
		}
		downloadURL = selectedItem.VideoVersions[0].URL
		mimeType = "video/mp4"
		filename = fmt.Sprintf("Story by %s [%s].mp4", user, itemPK)
	}

	filename = security.SanitizeFilename(filename)

	return &resolver.MediaInfo{
		URL:       downloadURL,
		MIMEType:  mimeType,
		Filename:  filename,
		SizeBytes: -1,
	}, nil
}

func (r *Resolver) resolveHighlight(ctx context.Context, parsedURL *url.URL, highlightID string) (*resolver.MediaInfo, error) {
	reel, err := r.fetchReel(ctx, highlightID)
	if err != nil {
		return nil, err
	}

	if len(reel.Items) == 0 {
		return nil, fmt.Errorf("no items found in highlight")
	}

	itemParam := ""
	if parsedURL != nil {
		q := parsedURL.Query()
		if p := q.Get("item"); p != "" {
			itemParam = p
		} else if p := q.Get("story_id"); p != "" {
			itemParam = p
		} else if p := q.Get("id"); p != "" {
			itemParam = p
		} else if p := q.Get("index"); p != "" {
			itemParam = p
		}
	}

	selectedItem, _ := selectItem(reel.Items, itemParam)

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

func (r *Resolver) resolveMedia(ctx context.Context, parsedURL *url.URL, mediaID string) (*resolver.MediaInfo, error) {
	item, err := r.fetchMediaInfo(ctx, mediaID)
	if err != nil {
		return nil, err
	}

	itemParam := ""
	if parsedURL != nil {
		q := parsedURL.Query()
		if p := q.Get("item"); p != "" {
			itemParam = p
		} else if p := q.Get("img_index"); p != "" {
			itemParam = p
		} else if p := q.Get("index"); p != "" {
			itemParam = p
		} else if p := q.Get("id"); p != "" {
			itemParam = p
		} else if p := q.Get("pk"); p != "" {
			itemParam = p
		}
	}

	username := item.User.Username
	if username == "" {
		username = item.User.FullName
	}
	if username == "" {
		username = "instagram_user"
	}

	// Carousel post handling
	if item.MediaType == 8 || len(item.CarouselMedia) > 0 {
		if len(item.CarouselMedia) == 0 {
			return nil, fmt.Errorf("carousel post has no child media items")
		}

		selectedChild, slideNumber := selectItem(item.CarouselMedia, itemParam)

		childPK := string(selectedChild.PK)
		if childPK == "" {
			childPK = selectedChild.Code
		}
		if childPK == "" {
			childPK = selectedChild.ID
		}

		var downloadURL string
		var mimeType string
		var filename string

		if selectedChild.MediaType == 1 { // Photo
			if len(selectedChild.ImageVersions.Candidates) == 0 {
				return nil, fmt.Errorf("no image candidates found for carousel photo item")
			}
			downloadURL = selectedChild.ImageVersions.Candidates[0].URL
			mimeType = "image/jpeg"
			filename = fmt.Sprintf("Photo %d by %s [%s].jpg", slideNumber, username, childPK)
		} else { // Video
			if len(selectedChild.VideoVersions) == 0 {
				return nil, fmt.Errorf("no video versions found for carousel video item")
			}
			downloadURL = selectedChild.VideoVersions[0].URL
			mimeType = "video/mp4"
			filename = fmt.Sprintf("Video %d by %s [%s].mp4", slideNumber, username, childPK)
		}

		filename = security.SanitizeFilename(filename)
		return &resolver.MediaInfo{
			URL:       downloadURL,
			MIMEType:  mimeType,
			Filename:  filename,
			SizeBytes: -1,
		}, nil
	}

	// Single photo or video / reel
	itemCode := item.Code
	if itemCode == "" {
		itemCode = string(item.PK)
	}

	var downloadURL string
	var mimeType string
	var filename string

	if item.MediaType == 1 { // Photo
		if len(item.ImageVersions.Candidates) == 0 {
			return nil, fmt.Errorf("no image candidates found for photo post")
		}
		downloadURL = item.ImageVersions.Candidates[0].URL
		mimeType = "image/jpeg"
		filename = fmt.Sprintf("Photo by %s [%s].jpg", username, itemCode)
	} else { // Video
		if len(item.VideoVersions) == 0 {
			return nil, fmt.Errorf("no video versions found for video post")
		}
		downloadURL = item.VideoVersions[0].URL
		mimeType = "video/mp4"
		filename = fmt.Sprintf("Video by %s [%s].mp4", username, itemCode)
	}

	filename = security.SanitizeFilename(filename)
	return &resolver.MediaInfo{
		URL:       downloadURL,
		MIMEType:  mimeType,
		Filename:  filename,
		SizeBytes: -1,
	}, nil
}
