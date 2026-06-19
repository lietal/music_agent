# TME Go Client — Replace Python Sidecar

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the Python QQ Music sidecar with a native Go implementation (`internal/tme/`), removing the Python/httpx/qqmusic-api-python dependency.

**Architecture:** A single Go package that calls `https://u.y.qq.com/cgi-bin/musicu.fcg` directly via `net/http`. No signing, no encryption (except 3DES+zlib for lyrics). All TME features exposed as clean Go functions. The `internal/tool/tme_search.go` is updated to use this internal client instead of HTTP-to-sidecar.

**Tech Stack:** Go stdlib (`net/http`, `encoding/json`, `crypto/des`, `compress/zlib`)

---

### Task 1: Core TME Client — HTTP agent for musicu.fcg

**Files:**
- Create: `internal/tme/client.go`
- Create: `internal/tme/models.go`

**Step 1: Define request/response models**

```go
// models.go
package tme

type CommParams struct {
    Ct      int    `json:"ct"`
    CV      int64  `json:"cv,omitempty"`
    V       int64  `json:"v,omitempty"`
    QQ      string `json:"qq,omitempty"`
    Authst  string `json:"authst,omitempty"`
    QIMEI36 string `json:"QIMEI36,omitempty"`
}

type MusicuRequest struct {
    Comm CommParams                  `json:"comm"`
    Req  map[string]MusicuSubRequest `json:"-"`
}

type MusicuSubRequest struct {
    Module string         `json:"module"`
    Method string         `json:"method"`
    Param  map[string]any `json:"param"`
}

type MusicuResponse struct {
    Code int64                       `json:"code"`
    Req  map[string]MusicuSubResponse `json:"-"`
}

type MusicuSubResponse struct {
    Code int64          `json:"code"`
    Data map[string]any `json:"data"`
}
```

**Step 2: Create Client with Call method**

```go
// client.go
type Client struct {
    baseURL    string
    httpClient *http.Client
    comm       CommParams
}

func NewClient() *Client {
    return &Client{
        baseURL:    "https://u.y.qq.com/cgi-bin/musicu.fcg",
        httpClient: &http.Client{Timeout: 10 * time.Second},
        comm: CommParams{Ct: 11, CV: 14090008, V: 14090008},
    }
}

func (c *Client) Call(ctx context.Context, reqs map[string]MusicuSubRequest) (*MusicuResponse, error)
```

**Step 3: Implement Call — build merged JSON, POST, parse response**

The key trick: musicu.fcg expects keys like `"req_0"`, `"req_1"` flattened into the JSON root, not nested. Build the full JSON struct dynamically using `map[string]any`.

**Step 4: Write test with httptest mock server**

---

### Task 2: Search Implementation

**Files:**
- Create: `internal/tme/search.go`

**Step 1: SearchSongs function**

```go
type Song struct {
    ID             string   `json:"id"`
    Title          string   `json:"title"`
    Artists        []string `json:"artists"`
    Album          string   `json:"album"`
    DurationSeconds int     `json:"duration_seconds"`
    ArtworkURL     string   `json:"artwork_url"`
}

func (c *Client) SearchSongs(ctx context.Context, keyword string, limit int) ([]Song, error)
```

Module: `music.search.SearchCgiService`, Method: `DoSearchForQQMusicMobile`

Parse response: `body.item_song[]` → extract `mid` (for id), `name` (title), `singer[].name` (artists), `album.name`, `interval` (duration), album `pmid` for artwork URL.

---

### Task 3: Song Detail, Lyrics, URLs

**Files:**
- Create: `internal/tme/song.go`
- Create: `internal/tme/lyrics.go`

**Step 1: SongDetail**
Module: `music.pf_song_detail_svr`, Method: `get_song_detail_yqq`

**Step 2: Lyrics with 3DES+zlib decryption**

Hardcoded key: `!@#)(*$%123ZXC!@!@#)(NHL` (24 bytes)
1. `hex.DecodeString(encryptedLyric)` → bytes
2. 3DES-ECB decrypt → bytes
3. `zlib.NewReader` → decompress → UTF-8 text

**Step 3: SongURL (playback)**
Module: `music.vkey.GetVkey`, Method: `UrlGetVkey`

---

### Task 4: Playlists, Charts, Comments, Artists, Albums

**Files:**
- Create: `internal/tme/playlist.go`
- Create: `internal/tme/chart.go`  
- Create: `internal/tme/comment.go`
- Create: `internal/tme/artist.go`
- Create: `internal/tme/album.go`

---

### Task 5: Wire into tme_search.go

**Files:**
- Modify: `internal/tool/tme_search.go`

Replace HTTP-to-sidecar with direct `tme.Client.SearchSongs()` calls. Remove sidecar URL config.

**Files:**
- Modify: `cmd/server/main.go`
- Modify: `config.toml`

Remove sidecar URL config. `tme_search.go` no longer needs it.

---

### Task 6: Remove Python sidecar dependency

**Files:**
- Modify: `internal/config/config.go` — remove ProviderConfig
- Modify: `config.toml` — remove [providers.qqmusic]
- Clean up: remove references to sidecar in docs

**Step: Stop the Python sidecar process**
```bash
kill $(lsof -i :18767 -t)
```
