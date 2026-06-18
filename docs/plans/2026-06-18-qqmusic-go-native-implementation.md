# QQ Music Go-Native Client — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Eliminate the Python FastAPI sidecar by implementing QQ Music API calls natively in Go, reducing to a single Go binary.

**Architecture:** Create a new `internal/qqmusic/` package that makes direct HTTP calls to `https://u.y.qq.com/cgi-bin/musicu.fcg` (plain JSON POST, no signing). Rewrite `internal/tool/tme_search.go` to call this package instead of the HTTP sidecar. Delete the Python sidecar directory and update Docker/configuration.

**Tech Stack:** Go 1.25, `net/http`, `encoding/json`, standard library only (no external music dependencies).

**Design Doc:** `docs/plans/2026-06-18-qqmusic-go-native-design.md`

---

### Task 0: Verify Prerequisites

**Step 1: Confirm current tests pass**

Run: `go test ./... -count=1`
Expected: All existing tests pass (establishes baseline)

**Step 2: Check go.mod for GPL dependencies**

Run: `grep -i "gpl\|agpl\|qqmusic\|music-lib" go.sum go.mod`
Expected: No matches (confirm no GPL deps exist yet)

### Task 1: Create qqmusic package skeleton with types

**Files:**
- Create: `internal/qqmusic/types.go`
- Create: `internal/qqmusic/client.go`

**Step 1: Write types.go**

```go
// internal/qqmusic/types.go
package qqmusic

// Song represents a QQ Music search result.
type Song struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Artists         []string `json:"artists"`
	Album           string   `json:"album,omitempty"`
	DurationSeconds int      `json:"duration_seconds,omitempty"`
	ArtworkURL      string   `json:"artwork_url,omitempty"`
}

// Credential holds QQ Music login state, stored in ~/.musio/credentials/qqmusic.json
type Credential struct {
	OpenID       string `json:"openid"`
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
	ExpiredAt    int64  `json:"expired_at"`
	MusicID      int64  `json:"musicid"`
	MusicKey     string `json:"musickey"`
	UnionID      string `json:"unionid"`
	EncryptUin   string `json:"encrypt_uin"`
	LoginType    int    `json:"login_type"`
	StrMusicID   string `json:"str_musicid"`
	RefreshKey   string `json:"refresh_key"`
}
```

**Step 2: Write client.go**

```go
// internal/qqmusic/client.go
package qqmusic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	qqMusicAPIURL    = "https://u.y.qq.com/cgi-bin/musicu.fcg"
	defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
)

// Client makes direct HTTP calls to QQ Music's API.
type Client struct {
	httpClient *http.Client
	credential *Credential
}

// NewClient creates a Client with optional credential loading.
func NewClient(credentialPath string) (*Client, error) {
	c := &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}

	if credentialPath != "" {
		cred, err := loadCredential(credentialPath)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("load credential: %w", err)
		}
		c.credential = cred
	}

	return c, nil
}

// HasCredential reports whether a valid credential is loaded.
func (c *Client) HasCredential() bool {
	return c.credential != nil && c.credential.MusicKey != ""
}

func loadCredential(path string) (*Credential, error) {
	expanded, err := expandPath(path)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(expanded)
	if err != nil {
		return nil, err
	}

	var cred Credential
	if err := json.Unmarshal(data, &cred); err != nil {
		return nil, fmt.Errorf("parse credential: %w", err)
	}
	return &cred, nil
}

func expandPath(path string) (string, error) {
	if len(path) >= 2 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("home dir: %w", err)
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}
```

**Step 3: Verify compilation**

Run: `go build ./internal/qqmusic/`
Expected: No errors

**Step 4: Commit**

```bash
git add internal/qqmusic/types.go internal/qqmusic/client.go
git commit -m "feat(qqmusic): add types and Client skeleton

Introduce internal/qqmusic/ package with Song and Credential types.
Client struct handles credential loading from JSON files."
```

---

### Task 2: Implement search endpoint

**Files:**
- Create: `internal/qqmusic/search.go`
- Create: `internal/qqmusic/search_test.go`

**Step 1: Write the search.go implementation**

```go
// internal/qqmusic/search.go
package qqmusic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const searchReferer = "https://y.qq.com/"

// Search searches QQ Music for songs matching keyword.
// Returns up to `limit` results (capped at 50).
func (c *Client) Search(ctx context.Context, keyword string, limit int) ([]Song, error) {
	if limit < 1 {
		limit = 5
	}
	if limit > 50 {
		limit = 50
	}

	payload := map[string]any{
		"music.search.SearchCgiService": map[string]any{
			"method": "DoSearchForQQMusicDesktop",
			"module": "music.search.SearchCgiService",
			"param": map[string]any{
				"query":        keyword,
				"num_per_page": limit,
				"page_num":     1,
				"search_type":  0, // 0=song, 1=singer, 2=album, 3=playlist
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal search payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, qqMusicAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create search request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", searchReferer)
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("search returned %d: %s", resp.StatusCode, string(body))
	}

	return parseSearchResponse(resp.Body, limit)
}

func parseSearchResponse(r io.Reader, limit int) ([]Song, error) {
	var raw struct {
		Code int `json:"code"`
		SearchService struct {
			Code int `json:"code"`
			Data struct {
				Body struct {
					Song struct {
						List []struct {
							Mid      string `json:"mid"`
							Name     string `json:"name"`
							Title    string `json:"title"`
							Album    struct {
								Name string `json:"name"`
								Mid  string `json:"mid"`
							} `json:"album"`
							Singer []struct {
								Name string `json:"name"`
							} `json:"singer"`
							Interval int `json:"interval"`
						} `json:"list"`
					} `json:"song"`
				} `json:"body"`
			} `json:"data"`
		} `json:"music.search.SearchCgiService"`
	}

	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return nil, fmt.Errorf("parse search response: %w", err)
	}

	items := raw.SearchService.Data.Body.Song.List
	if len(items) == 0 {
		return []Song{}, nil
	}

	songs := make([]Song, 0, limit)
	for i, item := range items {
		if i >= limit {
			break
		}
		if item.Mid == "" {
			continue
		}

		title := item.Title
		if title == "" {
			title = item.Name
		}

		artists := make([]string, 0, len(item.Singer))
		for _, s := range item.Singer {
			if name := strings.TrimSpace(s.Name); name != "" {
				artists = append(artists, name)
			}
		}

		artworkURL := ""
		if item.Album.Mid != "" {
			artworkURL = fmt.Sprintf("https://y.gtimg.cn/music/photo_new/T002R300x300M000%s.jpg", item.Album.Mid)
		}

		songs = append(songs, Song{
			ID:              fmt.Sprintf("qqmusic:%s", item.Mid),
			Title:           title,
			Artists:         artists,
			Album:           item.Album.Name,
			DurationSeconds: item.Interval,
			ArtworkURL:      artworkURL,
		})
	}

	return songs, nil
}
```

**Step 2: Write search_test.go with mock HTTP server**

```go
// internal/qqmusic/search_test.go
package qqmusic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearch_Success(t *testing.T) {
	// Mock QQ Music API response
	mockResp := map[string]any{
		"code": 0,
		"music.search.SearchCgiService": map[string]any{
			"code": 0,
			"data": map[string]any{
				"body": map[string]any{
					"song": map[string]any{
						"list": []map[string]any{
							{
								"mid":      "003IGhQO0JdnuC",
								"title":    "晴天",
								"name":     "晴天",
								"album":    map[string]any{"name": "叶惠美", "mid": "0024bjiL2aocxT"},
								"singer":   []map[string]any{{"name": "周杰伦"}},
								"interval": 269,
							},
						},
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	// Override the API URL for testing
	origURL := qqMusicAPIURL
	qqMusicAPIURL = server.URL
	defer func() { qqMusicAPIURL = origURL }()

	client := &Client{httpClient: server.Client()}
	songs, err := client.Search(context.Background(), "晴天", 5)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if len(songs) != 1 {
		t.Fatalf("expected 1 song, got %d", len(songs))
	}

	song := songs[0]
	if song.ID != "qqmusic:003IGhQO0JdnuC" {
		t.Errorf("expected ID qqmusic:003IGhQO0JdnuC, got %s", song.ID)
	}
	if song.Title != "晴天" {
		t.Errorf("expected Title 晴天, got %s", song.Title)
	}
	if len(song.Artists) != 1 || song.Artists[0] != "周杰伦" {
		t.Errorf("expected Artists [周杰伦], got %v", song.Artists)
	}
	if song.Album != "叶惠美" {
		t.Errorf("expected Album 叶惠美, got %s", song.Album)
	}
	if song.DurationSeconds != 269 {
		t.Errorf("expected DurationSeconds 269, got %d", song.DurationSeconds)
	}
	expectedArtwork := "https://y.gtimg.cn/music/photo_new/T002R300x300M0000024bjiL2aocxT.jpg"
	if song.ArtworkURL != expectedArtwork {
		t.Errorf("expected ArtworkURL %s, got %s", expectedArtwork, song.ArtworkURL)
	}
}

func TestSearch_EmptyResults(t *testing.T) {
	mockResp := map[string]any{
		"code": 0,
		"music.search.SearchCgiService": map[string]any{
			"code": 0,
			"data": map[string]any{
				"body": map[string]any{
					"song": map[string]any{
						"list": []map[string]any{},
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	origURL := qqMusicAPIURL
	qqMusicAPIURL = server.URL
	defer func() { qqMusicAPIURL = origURL }()

	client := &Client{httpClient: server.Client()}
	songs, err := client.Search(context.Background(), "xyznonexistent123", 10)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if len(songs) != 0 {
		t.Errorf("expected 0 songs, got %d", len(songs))
	}
}

func TestSearch_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	origURL := qqMusicAPIURL
	qqMusicAPIURL = server.URL
	defer func() { qqMusicAPIURL = origURL }()

	client := &Client{httpClient: server.Client()}
	_, err := client.Search(context.Background(), "test", 5)
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}
```

**Step 3: Run tests to verify**

Run: `go test ./internal/qqmusic/ -v -run TestSearch -count=1`
Expected: 3 tests PASS

**Step 4: Commit**

```bash
git add internal/qqmusic/search.go internal/qqmusic/search_test.go
git commit -m "feat(qqmusic): implement Search endpoint with tests

Direct HTTP POST to u.y.qq.com/cgi-bin/musicu.fcg using
DoSearchForQQMusicDesktop method. Parses nested JSON response
into []Song structs. Three test cases cover success, empty
results, and HTTP errors."
```

---

### Task 3: Rewrite tme_search.go to use qqmusic.Client

**Files:**
- Modify: `internal/tool/tme_search.go`
- Modify: `cmd/server/main.go` (or wherever the tool is initialized)

**Step 1: Find where TMESearchSongs is instantiated**

Run: `grep -rn "NewTMESearchSongs\|TMESearchSongs" internal/ cmd/`
Expected: Find the initialization point (likely in `cmd/server/main.go` or `internal/agent/`)

**Step 2: Rewrite tme_search.go**

Replace the entire file content:

```go
// internal/tool/tme_search.go
package tool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yourorg/music_searchrecom_agent/internal/qqmusic"
)

// TMESearchSongs searches QQ Music via native Go HTTP client.
type TMESearchSongs struct {
	client *qqmusic.Client
}

// NewTMESearchSongs creates a TMESearchSongs with a qqmusic client.
func NewTMESearchSongs(client *qqmusic.Client) *TMESearchSongs {
	return &TMESearchSongs{client: client}
}

func (t *TMESearchSongs) Name() string {
	return "search_songs"
}

func (t *TMESearchSongs) Description() string {
	return "Search for songs on QQ Music by keyword. Returns song id, title, artists, album, duration, and artwork URL."
}

func (t *TMESearchSongs) Execute(ctx context.Context, args map[string]any) (ToolResult, error) {
	keyword := ""
	if kw, ok := args["keyword"]; ok {
		keyword = fmt.Sprintf("%v", kw)
	}
	if q, ok := args["query"]; ok && keyword == "" {
		keyword = fmt.Sprintf("%v", q)
	}
	if keyword == "" {
		return ToolResult{}, fmt.Errorf("keyword is required")
	}

	limit := 5
	if l, ok := args["limit"]; ok {
		switch v := l.(type) {
		case float64:
			limit = int(v)
		case int:
			limit = v
		}
	}

	songs, err := t.client.Search(ctx, keyword, limit)
	if err != nil {
		return ToolResult{}, fmt.Errorf("qqmusic search: %w", err)
	}

	data, err := json.Marshal(songs)
	if err != nil {
		return ToolResult{}, fmt.Errorf("marshal songs: %w", err)
	}

	return ToolResult{Data: string(data)}, nil
}

func (t *TMESearchSongs) IsAvailable(ctx context.Context) bool {
	// Quick connectivity check: search for a single known term
	_, err := t.client.Search(ctx, "test", 1)
	return err == nil
}
```

**Step 3: Update initialization code**

Find the line that creates `NewTMESearchSongs(sidecarURL)` and replace with:

```go
qqmusicClient, err := qqmusic.NewClient(cfg.QQMusic.CredentialPath)
if err != nil {
    log.Fatalf("failed to create qqmusic client: %v", err)
}
searchTool := tool.NewTMESearchSongs(qqmusicClient)
```

Add a `QQMusic` config struct if one doesn't exist:

```go
// In internal/config/config.go, add:
type QQMusicConfig struct {
    CredentialPath string `toml:"credential_path"`
}
```

Update `config.toml`:

```toml
[providers.qqmusic]
credential_path = "~/.musio/credentials/qqmusic.json"
```

**Step 4: Verify compilation**

Run: `go build ./...`
Expected: No errors

**Step 5: Run existing tests**

Run: `go test ./internal/tool/... -v -count=1`
Expected: Any existing tool tests pass (or update mocks if needed)

**Step 6: Commit**

```bash
git add internal/tool/tme_search.go internal/config/config.go config.toml cmd/server/main.go
git commit -m "refactor(tool): replace Python sidecar HTTP call with native qqmusic.Client

TMESearchSongs now calls qqmusic.Client.Search() directly instead of
HTTP GET to the Python FastAPI sidecar. Removes sidecar URL dependency
and HTTP client from the tool struct."
```

---

### Task 4: Implement song detail endpoint

**Files:**
- Create: `internal/qqmusic/song.go`
- Create: `internal/qqmusic/song_test.go`

**Step 1: Write song.go**

```go
// internal/qqmusic/song.go
package qqmusic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// SongDetail adds source URL to Song.
type SongDetail struct {
	Song
	SourceURL string `json:"source_url,omitempty"`
}

// SongURL holds a playback URL.
type SongURL struct {
	SongID           string `json:"song_id"`
	URL              string `json:"url,omitempty"`
	ExpiresInSeconds int    `json:"expires_in_seconds,omitempty"`
}

// GetSongDetail fetches full song metadata by qqmusic ID.
func (c *Client) GetSongDetail(ctx context.Context, songID string) (*SongDetail, error) {
	mid := extractMID(songID)

	payload := map[string]any{
		"songinfo": map[string]any{
			"method": "get_song_detail_yqq",
			"module": "music.pf_song_detail_svr",
			"param": map[string]any{
				"song_mid": mid,
			},
		},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, qqMusicAPIURL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", searchReferer)
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("song detail request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("song detail returned %d: %s", resp.StatusCode, string(body))
	}

	return parseSongDetailResponse(resp.Body, songID)
}

// extractMID strips the "qqmusic:" prefix from an ID.
func extractMID(id string) string {
	const prefix = "qqmusic:"
	if len(id) > len(prefix) && id[:len(prefix)] == prefix {
		return id[len(prefix):]
	}
	return id
}

func parseSongDetailResponse(r io.Reader, songID string) (*SongDetail, error) {
	var raw struct {
		SongInfo struct {
			Data struct {
				TrackInfo struct {
					Mid      string `json:"mid"`
					Name     string `json:"name"`
					Title    string `json:"title"`
					Album    struct {
						Name string `json:"name"`
						Mid  string `json:"mid"`
					} `json:"album"`
					Singer []struct {
						Name string `json:"name"`
					} `json:"singer"`
					Interval int `json:"interval"`
				} `json:"track_info"`
			} `json:"data"`
		} `json:"songinfo"`
	}
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return nil, fmt.Errorf("parse song detail: %w", err)
	}

	info := raw.SongInfo.Data.TrackInfo
	mid := info.Mid
	if mid == "" {
		mid = extractMID(songID)
	}
	if info.Name == "" && info.Title == "" {
		return nil, fmt.Errorf("song not found: %s", songID)
	}

	title := info.Title
	if title == "" {
		title = info.Name
	}

	artists := make([]string, 0, len(info.Singer))
	for _, s := range info.Singer {
		if s.Name != "" {
			artists = append(artists, s.Name)
		}
	}

	artworkURL := ""
	if info.Album.Mid != "" {
		artworkURL = fmt.Sprintf("https://y.gtimg.cn/music/photo_new/T002R300x300M000%s.jpg", info.Album.Mid)
	}

	return &SongDetail{
		Song: Song{
			ID:              fmt.Sprintf("qqmusic:%s", mid),
			Title:           title,
			Artists:         artists,
			Album:           info.Album.Name,
			DurationSeconds: info.Interval,
			ArtworkURL:      artworkURL,
		},
		SourceURL: fmt.Sprintf("https://y.qq.com/n/ryqq/songDetail/%s", mid),
	}, nil
}
```

**Step 2: Write song_test.go**

```go
// internal/qqmusic/song_test.go
package qqmusic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetSongDetail_Success(t *testing.T) {
	mockResp := map[string]any{
		"songinfo": map[string]any{
			"data": map[string]any{
				"track_info": map[string]any{
					"mid":   "003IGhQO0JdnuC",
					"title": "晴天",
					"name":  "晴天",
					"album": map[string]any{"name": "叶惠美", "mid": "0024bjiL2aocxT"},
					"singer": []map[string]any{
						{"name": "周杰伦"},
					},
					"interval": 269,
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	origURL := qqMusicAPIURL
	qqMusicAPIURL = server.URL
	defer func() { qqMusicAPIURL = origURL }()

	client := &Client{httpClient: server.Client()}
	detail, err := client.GetSongDetail(context.Background(), "qqmusic:003IGhQO0JdnuC")
	if err != nil {
		t.Fatalf("GetSongDetail() error: %v", err)
	}

	if detail.ID != "qqmusic:003IGhQO0JdnuC" {
		t.Errorf("expected ID qqmusic:003IGhQO0JdnuC, got %s", detail.ID)
	}
	if detail.SourceURL != "https://y.qq.com/n/ryqq/songDetail/003IGhQO0JdnuC" {
		t.Errorf("unexpected SourceURL: %s", detail.SourceURL)
	}
}

func TestExtractMID(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"qqmusic:003IGhQO0JdnuC", "003IGhQO0JdnuC"},
		{"003IGhQO0JdnuC", "003IGhQO0JdnuC"},
		{"", ""},
	}
	for _, tt := range tests {
		got := extractMID(tt.input)
		if got != tt.expected {
			t.Errorf("extractMID(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
```

**Step 3: Run tests**

Run: `go test ./internal/qqmusic/ -v -run "TestGetSongDetail|TestExtractMID" -count=1`
Expected: All PASS

**Step 4: Commit**

```bash
git add internal/qqmusic/song.go internal/qqmusic/song_test.go
git commit -m "feat(qqmusic): implement GetSongDetail endpoint"
```

---

### Task 5: Cleanup — remove Python sidecar

**Files:**
- Delete: `musio/providers/qqmusic-python-sidecar/` (entire directory)
- Modify: `Dockerfile` (remove Python deps)
- Modify: `docker-compose.yml` (remove sidecar service)
- Modify: `config.toml` (remove `sidecar_base_url`)
- Modify: `Makefile` (remove sidecar targets if any)

**Step 1: Delete the sidecar directory**

Run: `rm -rf musio/providers/qqmusic-python-sidecar/`
Expected: Directory removed

**Step 2: Update Dockerfile**

Remove any Python/FastAPI related stages or dependencies. Keep only the Go build.

**Step 3: Update config.toml**

```toml
# Remove or comment out:
# [providers.qqmusic]
# sidecar_base_url = "http://127.0.0.1:18767"

# Replace with:
[providers.qqmusic]
credential_path = "~/.musio/credentials/qqmusic.json"
```

**Step 4: Verify full build**

Run: `make build` (or `go build ./cmd/server/`)
Expected: Successful build with no Python references

**Step 5: Commit**

```bash
git add -A
git commit -m "chore: remove Python sidecar, update Docker/config

Delete musio/providers/qqmusic-python-sidecar/ entirely.
Update Dockerfile to single-stage Go build.
Replace sidecar_base_url config with credential_path."
```

---

### Task 6: Integration verification

**Step 1: Run full test suite**

Run: `go test ./... -count=1 -v`
Expected: All tests pass, no regressions

**Step 2: Manual smoke test** (requires credential file)

If a valid `~/.musio/credentials/qqmusic.json` exists, run:

```go
// Quick one-off test in a _test.go or main:
func TestRealSearch(t *testing.T) {
    if os.Getenv("QQMUSIC_INTEGRATION") != "1" {
        t.Skip("set QQMUSIC_INTEGRATION=1 to run")
    }
    client, _ := qqmusic.NewClient("~/.musio/credentials/qqmusic.json")
    songs, err := client.Search(context.Background(), "周杰伦", 5)
    if err != nil {
        t.Fatal(err)
    }
    if len(songs) == 0 {
        t.Fatal("no results")
    }
    t.Logf("Found %d songs: %+v", len(songs), songs[0])
}
```

Run: `QQMUSIC_INTEGRATION=1 go test ./internal/qqmusic/ -v -run TestRealSearch -count=1`

**Step 3: Verify no GPL dependencies**

Run: `grep -ri "gpl\|agpl" go.sum go.mod`
Expected: No matches

**Step 4: Final commit (if any changes)**

---

### Task 7: Documentation

**Step 1: Update README.md**

Remove references to Python sidecar, update architecture diagram.

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: update README for Go-native QQ Music client"
```

---

## Risk Mitigation Checklist

- [ ] Search returns same format as sidecar (`Song` struct with same JSON keys)
- [ ] `IsAvailable()` health check works (lightweight search)
- [ ] No Python or qqmusic-api-python references remain in codebase
- [ ] Docker image builds without Python layer
- [ ] Credential file path is configurable
- [ ] Error messages are clear when credential is missing
