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
	storyRegex     = regexp.MustCompile(`/stories/[^/]+/(\d+)`)
)

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
		Username string `json:"username"`
		FullName string `json:"full_name"`
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
		storyRegex.MatchString(rawURL) ||
		strings.Contains(rawURL, "/stories/highlights/") ||
		strings.Contains(rawURL, "/highlights/")
}

func (r *Resolver) extractTarget(rawURL string) (isHighlight bool, id string, err error) {
	if m := highlightRegex.FindStringSubmatch(rawURL); len(m) >= 2 {
		return true, m[1], nil
	}
	if m := postRegex.FindStringSubmatch(rawURL); len(m) >= 2 {
		mediaID, err := shortcodeToMediaID(m[1])
		if err != nil {
			return false, "", err
		}
		return false, mediaID, nil
	}
	if m := storyRegex.FindStringSubmatch(rawURL); len(m) >= 2 {
		return false, m[1], nil
	}
	return false, "", fmt.Errorf("could not identify Instagram media or highlight ID from URL")
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

// Inspect fetches metadata for highlights or multi-item posts/carousels.
func (r *Resolver) Inspect(ctx context.Context, rawURL string) (*resolver.PlaylistInfo, error) {
	if _, err := security.ValidateURL(ctx, rawURL); err != nil {
		return nil, err
	}

	isHighlight, targetID, err := r.extractTarget(rawURL)
	if err != nil {
		return nil, err
	}

	if isHighlight {
		return r.inspectHighlight(ctx, targetID)
	}

	return r.inspectMedia(ctx, targetID)
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

// Resolve extracts the direct download URL for Instagram items (highlight stories, posts, carousel slides, reels).
func (r *Resolver) Resolve(ctx context.Context, rawURL string) (*resolver.MediaInfo, error) {
	parsedURL, err := security.ValidateURL(ctx, rawURL)
	if err != nil {
		return nil, err
	}

	isHighlight, targetID, err := r.extractTarget(rawURL)
	if err != nil {
		return nil, err
	}

	if isHighlight {
		return r.resolveHighlight(ctx, parsedURL, targetID)
	}

	return r.resolveMedia(ctx, parsedURL, targetID)
}

func (r *Resolver) resolveHighlight(ctx context.Context, parsedURL *url.URL, highlightID string) (*resolver.MediaInfo, error) {
	reel, err := r.fetchReel(ctx, highlightID)
	if err != nil {
		return nil, err
	}

	if len(reel.Items) == 0 {
		return nil, fmt.Errorf("no items found in highlight")
	}

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

func (r *Resolver) resolveMedia(ctx context.Context, parsedURL *url.URL, mediaID string) (*resolver.MediaInfo, error) {
	item, err := r.fetchMediaInfo(ctx, mediaID)
	if err != nil {
		return nil, err
	}

	targetIndex := -1
	targetID := ""
	if parsedURL != nil {
		q := parsedURL.Query()
		if idxVal := q.Get("img_index"); idxVal != "" {
			if idx, err := strconv.Atoi(idxVal); err == nil {
				targetIndex = idx
			}
		} else if itemVal := q.Get("item"); itemVal != "" {
			if idx, err := strconv.Atoi(itemVal); err == nil {
				targetIndex = idx
			} else {
				targetID = itemVal
			}
		} else if idxVal := q.Get("index"); idxVal != "" {
			if idx, err := strconv.Atoi(idxVal); err == nil {
				targetIndex = idx
			}
		} else if idVal := q.Get("id"); idVal != "" {
			targetID = idVal
		} else if idVal := q.Get("pk"); idVal != "" {
			targetID = idVal
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

		var selectedChild *igItem
		slideNumber := 1

		if targetIndex > 0 && targetIndex <= len(item.CarouselMedia) {
			selectedChild = &item.CarouselMedia[targetIndex-1]
			slideNumber = targetIndex
		} else if targetID != "" {
			for i := range item.CarouselMedia {
				c := &item.CarouselMedia[i]
				if string(c.PK) == targetID || c.Code == targetID || c.ID == targetID {
					selectedChild = c
					slideNumber = i + 1
					break
				}
			}
		}

		if selectedChild == nil {
			selectedChild = &item.CarouselMedia[0]
			slideNumber = 1
		}

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
