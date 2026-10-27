// Package instagram implements a native resolver for Instagram Posts, Carousels, Stories, and Highlights.
package instagram

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/A-007481D/retriva/server/internal/resolver"
	"github.com/A-007481D/retriva/server/internal/security"
)

var (
	highlightRegex = regexp.MustCompile(`/(?:stories/)?highlights/(\d+)`)
	postRegex      = regexp.MustCompile(`/(?:p|reel|reels|tv)/([A-Za-z0-9_-]+)`)
	userStoryRegex = regexp.MustCompile(`/stories/([A-Za-z0-9_.]+)(?:/(\d+))?/?`)
	profileIDRegex = regexp.MustCompile(`"(?:profile_id|page_id)":\s*"(?:profilePage_)?(\d+)"`)
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
		ID       string `json:"id"`
		Username string `json:"username"`
		FullName string `json:"full_name"`
	} `json:"user"`
	Items []igItem `json:"items"`
}

type igFeedResponse struct {
	Status string            `json:"status"`
	Reels  map[string]igReel `json:"reels"`
}

type cachedReel struct {
	reel      *igReel
	expiresAt time.Time
}

type igGQLReelsMediaResponse struct {
	Data struct {
		ReelsMedia []struct {
			Typename        string `json:"__typename"`
			ID              string `json:"id"`
			LatestReelMedia int64  `json:"latest_reel_media"`
			Owner           struct {
				ID            string `json:"id"`
				Username      string `json:"username"`
				ProfilePicURL string `json:"profile_pic_url"`
			} `json:"owner"`
			Items []struct {
				Typename         string  `json:"__typename"`
				ID               string  `json:"id"`
				IsVideo          bool    `json:"is_video"`
				TakenAtTimestamp int64   `json:"taken_at_timestamp"`
				VideoDuration    float64 `json:"video_duration"`
				DisplayURL       string  `json:"display_url"`
				DisplayResources []struct {
					Src          string `json:"src"`
					ConfigWidth  int    `json:"config_width"`
					ConfigHeight int    `json:"config_height"`
				} `json:"display_resources"`
				VideoResources []struct {
					Src          string `json:"src"`
					ConfigWidth  int    `json:"config_width"`
					ConfigHeight int    `json:"config_height"`
					MimeType     string `json:"mime_type"`
				} `json:"video_resources"`
				Owner struct {
					ID       string `json:"id"`
					Username string `json:"username"`
				} `json:"owner"`
			} `json:"items"`
		} `json:"reels_media"`
	} `json:"data"`
	Status string `json:"status"`
}

// Resolver handles Instagram URLs (posts, carousels, reels, stories, highlights) directly using the Instagram mobile API.
type Resolver struct {
	cookiesPath string
	apiBaseURL  string
	cacheMu     sync.RWMutex
	reelCache   map[string]cachedReel
}

// New creates an Instagram resolver with optional Netscape cookies path.
func New(cookiesPath string) *Resolver {
	return &Resolver{
		cookiesPath: cookiesPath,
		reelCache:   make(map[string]cachedReel),
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

func (r *Resolver) fetchGraphQLReelsMedia(ctx context.Context, reelIDs, highlightReelIDs []string) (*igReel, error) {
	client, err := r.createHTTPClient()
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	if reelIDs == nil {
		reelIDs = []string{}
	}
	if highlightReelIDs == nil {
		highlightReelIDs = []string{}
	}

	variables := map[string]any{
		"reel_ids":             reelIDs,
		"tag_names":            []string{},
		"location_ids":         []string{},
		"highlight_reel_ids":   highlightReelIDs,
		"precomposed_overlay": false,
	}
	varBytes, err := json.Marshal(variables)
	if err != nil {
		return nil, fmt.Errorf("marshal variables: %w", err)
	}

	queryHash := "45246d3fe16ccc6577e0bd297a5db1ab"
	baseURL := "https://www.instagram.com"
	if r.apiBaseURL != "" {
		baseURL = r.apiBaseURL
	}
	gqlURL := fmt.Sprintf("%s/graphql/query/?query_hash=%s&variables=%s", baseURL, queryHash, url.QueryEscape(string(varBytes)))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, gqlURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create gql request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("X-IG-App-ID", "936619743392459")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", "https://www.instagram.com/")

	if client.Jar != nil {
		if u, err := url.Parse("https://www.instagram.com"); err == nil {
			for _, c := range client.Jar.Cookies(u) {
				if c.Name == "csrftoken" && c.Value != "" {
					req.Header.Set("X-CSRFToken", c.Value)
					break
				}
			}
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("graphql request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("graphql returned status %d", resp.StatusCode)
	}

	var data igGQLReelsMediaResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode graphql response: %w", err)
	}

	if len(data.Data.ReelsMedia) == 0 {
		return nil, fmt.Errorf("empty reels_media returned by graphql")
	}

	gReel := data.Data.ReelsMedia[0]
	username := gReel.Owner.Username
	if username == "" {
		username = "instagram_user"
	}

	reel := &igReel{
		ID:    gReel.ID,
		Title: fmt.Sprintf("Highlight by %s", username),
	}
	reel.User.ID = gReel.Owner.ID
	reel.User.Username = gReel.Owner.Username

	for _, gItem := range gReel.Items {
		item := igItem{
			PK:            json.Number(gItem.ID),
			ID:            gItem.ID,
			Code:          gItem.ID,
			TakenAt:       gItem.TakenAtTimestamp,
			VideoDuration: gItem.VideoDuration,
		}
		item.User.ID = gItem.Owner.ID
		item.User.Username = gItem.Owner.Username
		if item.User.Username == "" {
			item.User.Username = gReel.Owner.Username
		}

		if gItem.IsVideo {
			item.MediaType = 2 // Video
			for _, vr := range gItem.VideoResources {
				if vr.Src != "" {
					item.VideoVersions = append(item.VideoVersions, igVideoVersion{
						URL:    vr.Src,
						Width:  vr.ConfigWidth,
						Height: vr.ConfigHeight,
					})
				}
			}
			sort.Slice(item.VideoVersions, func(i, j int) bool {
				return item.VideoVersions[i].Width*item.VideoVersions[i].Height > item.VideoVersions[j].Width*item.VideoVersions[j].Height
			})
		} else {
			item.MediaType = 1 // Photo
		}

		if gItem.DisplayURL != "" {
			item.ImageVersions.Candidates = append(item.ImageVersions.Candidates, igCandidate{
				URL: gItem.DisplayURL,
			})
		}
		for _, dr := range gItem.DisplayResources {
			if dr.Src != "" {
				item.ImageVersions.Candidates = append(item.ImageVersions.Candidates, igCandidate{
					URL:    dr.Src,
					Width:  dr.ConfigWidth,
					Height: dr.ConfigHeight,
				})
			}
		}
		sort.Slice(item.ImageVersions.Candidates, func(i, j int) bool {
			return item.ImageVersions.Candidates[i].Width*item.ImageVersions.Candidates[i].Height > item.ImageVersions.Candidates[j].Width*item.ImageVersions.Candidates[j].Height
		})

		reel.Items = append(reel.Items, item)
	}

	// Sort stories in chronological order (first story added appears first)
	sort.SliceStable(reel.Items, func(i, j int) bool {
		return reel.Items[i].TakenAt < reel.Items[j].TakenAt
	})

	return reel, nil
}

func (r *Resolver) fetchReel(ctx context.Context, highlightID string) (*igReel, error) {
	// 1. Check in-memory cache
	r.cacheMu.RLock()
	if c, ok := r.reelCache[highlightID]; ok && time.Now().Before(c.expiresAt) {
		r.cacheMu.RUnlock()
		return c.reel, nil
	}
	r.cacheMu.RUnlock()

	// 2. Primary: Fetch via Instagram GraphQL (query_hash: 45246d3fe16ccc6577e0bd297a5db1ab)
	if reel, err := r.fetchGraphQLReelsMedia(ctx, nil, []string{highlightID}); err == nil && len(reel.Items) > 0 {
		r.cacheMu.Lock()
		if r.reelCache == nil {
			r.reelCache = make(map[string]cachedReel)
		}
		r.reelCache[highlightID] = cachedReel{
			reel:      reel,
			expiresAt: time.Now().Add(10 * time.Minute),
		}
		r.cacheMu.Unlock()
		return reel, nil
	}

	// 3. Fallback: Fetch via mobile API feed/reels_media endpoint
	client, err := r.createHTTPClient()
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	apiURL := fmt.Sprintf("%s/api/v1/feed/reels_media/?reel_ids=highlight:%s", r.base(), highlightID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "Instagram 278.0.0.19.115 Android (33/13; 420dpi; 1080x2400; Google/google; Pixel 7; cheetah; cheetah; en_US; 467268872)")
	req.Header.Set("X-IG-App-ID", "936619743392459")

	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var data igFeedResponse
			if err := json.NewDecoder(resp.Body).Decode(&data); err == nil {
				reelKey := "highlight:" + highlightID
				if reel, ok := data.Reels[reelKey]; ok && len(reel.Items) > 0 {
					r.cacheMu.Lock()
					if r.reelCache == nil {
						r.reelCache = make(map[string]cachedReel)
					}
					r.reelCache[highlightID] = cachedReel{
						reel:      &reel,
						expiresAt: time.Now().Add(10 * time.Minute),
					}
					r.cacheMu.Unlock()
					return &reel, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("highlight %s not found or has been deleted/removed (empty reel)", highlightID)
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

	req.Header.Set("User-Agent", "Instagram 278.0.0.19.115 Android (33/13; 420dpi; 1080x2400; Google/google; Pixel 7; cheetah; cheetah; en_US; 467268872)")
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

	// Ensure chronological order (first story added appears first)
	sort.SliceStable(reel.Items, func(i, j int) bool {
		return reel.Items[i].TakenAt < reel.Items[j].TakenAt
	})

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
		capType := "Video"
		if it.MediaType == 1 {
			itemType = "photo"
			capType = "Photo"
		}

		title := fmt.Sprintf("%s %d by @%s", capType, i+1, user)

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

	// Ensure chronological order (first story added appears first)
	sort.SliceStable(reel.Items, func(i, j int) bool {
		return reel.Items[i].TakenAt < reel.Items[j].TakenAt
	})

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
	// 1. If we have a storyID, try to query media info directly
	if storyID != "" {
		if item, err := r.fetchMediaInfo(ctx, storyID); err == nil && item != nil {
			reel := &igReel{
				ID: storyID,
				Items: []igItem{*item},
			}
			reel.User.Username = item.User.Username
			reel.User.FullName = item.User.FullName
			return reel, nil
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

	// 3. Try GraphQL reels_media query first
	if reel, err := r.fetchGraphQLReelsMedia(ctx, []string{userID}, nil); err == nil && len(reel.Items) > 0 {
		return reel, nil
	}

	apiURL := fmt.Sprintf("%s/api/v1/feed/reels_media/?reel_ids=%s", r.base(), userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "Instagram 278.0.0.19.115 Android (33/13; 420dpi; 1080x2400; Google/google; Pixel 7; cheetah; cheetah; en_US; 467268872)")
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

	if len(data.Reels) == 0 {
		return nil, fmt.Errorf("Instagram returned empty story reel for @%s (Meta is rate-limiting automated requests or stories have expired)", username)
	}

	return nil, fmt.Errorf("@%s has no active stories right now (stories expire after 24 hours)", username)
}

func (r *Resolver) fetchUserIDByUsername(ctx context.Context, client *http.Client, username string) (string, error) {
	// 1. Try public profile HTML scraping using Facebook crawler User-Agent
	// Instagram always server-renders full meta tags and profile IDs for crawlers without requiring login.
	profileURL := fmt.Sprintf("https://www.instagram.com/%s/", username)
	reqHTML, err := http.NewRequestWithContext(ctx, http.MethodGet, profileURL, nil)
	if err == nil {
		reqHTML.Header.Set("User-Agent", "facebookexternalhit/1.1")
		reqHTML.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		unauthClient := &http.Client{Timeout: 10 * time.Second}
		if respHTML, err := unauthClient.Do(reqHTML); err == nil {
			defer respHTML.Body.Close()
			if respHTML.StatusCode == http.StatusOK {
				bodyBytes, _ := io.ReadAll(io.LimitReader(respHTML.Body, 2*1024*1024))
				if m := profileIDRegex.FindSubmatch(bodyBytes); len(m) >= 2 {
					return string(m[1]), nil
				}
			}
		}
	}

	// 2. Fallback to web_profile_info endpoint with authenticated client
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
	if selectedItem == nil && itemParam != "" {
		if it, err := r.fetchMediaInfo(ctx, itemParam); err == nil && it != nil {
			selectedItem = it
		}
	}
	if selectedItem == nil {
		return nil, fmt.Errorf("item %s not found in highlight", itemParam)
	}

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

	// Ensure chronological order matches inspect
	sort.SliceStable(reel.Items, func(i, j int) bool {
		return reel.Items[i].TakenAt < reel.Items[j].TakenAt
	})

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

	selectedItem, itemIndex := selectItem(reel.Items, itemParam)
	if selectedItem == nil && itemParam != "" {
		if it, err := r.fetchMediaInfo(ctx, itemParam); err == nil && it != nil {
			selectedItem = it
		}
	}
	if selectedItem == nil {
		return nil, fmt.Errorf("item %s not found in highlight", itemParam)
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

	highlightTitle := reel.Title
	if highlightTitle == "" || highlightTitle == "ً" {
		highlightTitle = fmt.Sprintf("Highlight by %s", username)
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
		if len(reel.Items) > 1 {
			filename = fmt.Sprintf("%s - %d of %d [%s].jpg", highlightTitle, itemIndex, len(reel.Items), itemPK)
		} else {
			filename = fmt.Sprintf("%s [%s].jpg", highlightTitle, itemPK)
		}
	} else { // Video
		if len(selectedItem.VideoVersions) == 0 {
			return nil, fmt.Errorf("no video versions found for video item")
		}
		downloadURL = selectedItem.VideoVersions[0].URL
		mimeType = "video/mp4"
		if len(reel.Items) > 1 {
			filename = fmt.Sprintf("%s - %d of %d [%s].mp4", highlightTitle, itemIndex, len(reel.Items), itemPK)
		} else {
			filename = fmt.Sprintf("%s [%s].mp4", highlightTitle, itemPK)
		}
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
