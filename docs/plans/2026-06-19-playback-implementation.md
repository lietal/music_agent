# 播放功能 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add full music playback (bottom player bar, expandable panel, queue, QQ Music login, hybrid CDN/proxy streaming) to the music search agent.

**Architecture:** Go backend exposes song URL, audio stream proxy, lyrics, and QQ Music login endpoints. React frontend adds a `usePlayerStore` hook managing an HTML5 `<audio>` element with fallback from direct CDN to backend proxy. Player UI components (PlayerBar, PlayerPanel with NowPlaying/Lyrics/Queue tabs) integrate into ChatPage.

**Tech Stack:** Go 1.25 (chi router), React 19 + TypeScript, Tailwind CSS v4, HTML5 Audio API, QQ Music native API (existing `tme` package).

**Design Doc:** `docs/plans/2026-06-19-playback-design.md`

---

## Phase 1: Backend Foundation (parallelizable)

### Task 1: Credential Store (`server/internal/tme/credential_store.go`)

**Files:**
- Create: `server/internal/tme/credential_store.go`
- Test: `server/internal/tme/credential_store_test.go`

**Step 1: Write the test**

```go
// server/internal/tme/credential_store_test.go
package tme

import "testing"

func TestCredentialStore_SetGet(t *testing.T) {
    s := NewCredentialStore()
    if s.IsLoggedIn() {
        t.Error("should not be logged in initially")
    }
    s.Set("test_id", "test_key")
    if !s.IsLoggedIn() {
        t.Error("should be logged in after Set")
    }
    mid, mk := s.Get()
    if mid != "test_id" || mk != "test_key" {
        t.Errorf("got (%q, %q), want (test_id, test_key)", mid, mk)
    }
}

func TestCredentialStore_Clear(t *testing.T) {
    s := NewCredentialStore()
    s.Set("id", "key")
    s.Clear()
    if s.IsLoggedIn() {
        t.Error("should not be logged in after Clear")
    }
}
```

**Step 2: Run test to verify it fails**

```bash
cd server && go test ./internal/tme/ -run TestCredentialStore -v -count=1
```
Expected: FAIL — `undefined: NewCredentialStore`

**Step 3: Implement CredentialStore**

```go
// server/internal/tme/credential_store.go
package tme

import "sync"

type CredentialStore struct {
    musicid  string
    musickey string
    mu       sync.RWMutex
}

func NewCredentialStore() *CredentialStore {
    return &CredentialStore{}
}

func (s *CredentialStore) IsLoggedIn() bool {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.musicid != "" && s.musickey != ""
}

func (s *CredentialStore) Get() (musicid, musickey string) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.musicid, s.musickey
}

func (s *CredentialStore) Set(musicid, musickey string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.musicid = musicid
    s.musickey = musickey
}

func (s *CredentialStore) Clear() {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.musicid = ""
    s.musickey = ""
}
```

**Step 4: Run test to verify it passes**

```bash
cd server && go test ./internal/tme/ -run TestCredentialStore -v -count=1
```
Expected: PASS

**Step 5: Commit**

```bash
git add server/internal/tme/credential_store.go server/internal/tme/credential_store_test.go
git commit -m "feat(tme): add CredentialStore for QQ Music login credentials"
```

---

### Task 2: QQ Music Login API (`server/internal/tme/login.go`)

**Files:**
- Create: `server/internal/tme/login.go`
- Test: `server/internal/tme/login_test.go`

**Context:** QQ Music login uses QR code flow. The API endpoints involved:
- Get QR code: module `music.login.Qrcode`, method `GetQrcode`
- Check status: module `music.login.Qrcode`, method `CheckQrcode`
- (No separate credential exchange — credentials returned directly in `CheckQrcode` on confirmation)

**Step 1: Write the test — GetLoginQRCode**

```go
// server/internal/tme/login_test.go
package tme

import (
    "context"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
)

func TestGetLoginQRCode(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{
            "code": 0,
            "req_0": {
                "code": 0,
                "data": {
                    "qrcode_url": "https://qrcode.example.com/abc",
                    "qrcode_key": "key_abc123"
                }
            }
        }`))
    }))
    defer srv.Close()

    c := NewClient()
    c.SetBaseURL(srv.URL)
    ctx := context.Background()

    qr, err := c.GetLoginQRCode(ctx)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if qr.QrcodeURL != "https://qrcode.example.com/abc" {
        t.Errorf("got url %q", qr.QrcodeURL)
    }
    if qr.Key != "key_abc123" {
        t.Errorf("got key %q", qr.Key)
    }
}

func TestCheckQRCodeStatus_Confirmed(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{
            "code": 0,
            "req_0": {
                "code": 0,
                "data": {
                    "status": 3,
                    "musicid": 12345,
                    "musickey": "test_key_abc",
                    "nickname": "测试用户"
                }
            }
        }`))
    }))
    defer srv.Close()

    c := NewClient()
    c.SetBaseURL(srv.URL)
    ctx := context.Background()

    status, err := c.CheckQRCodeStatus(ctx, "key_abc123")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if status.Status != "confirmed" {
        t.Errorf("expected confirmed, got %s", status.Status)
    }
    if status.MusicID != "12345" {
        t.Errorf("got musicid %q", status.MusicID)
    }
    if !strings.HasPrefix(status.MusicKey, "test_key") {
        t.Errorf("got musickey %q", status.MusicKey)
    }
}
```

**Step 2: Run test to verify it fails**

```bash
cd server && go test ./internal/tme/ -run "TestGetLoginQRCode|TestCheckQRCodeStatus" -v -count=1
```
Expected: FAIL — `GetLoginQRCode not defined`

**Step 3: Implement login.go**

```go
// server/internal/tme/login.go
package tme

import (
    "context"
    "fmt"
)

type QRCode struct {
    QrcodeURL string `json:"qrcode_url"`
    Key       string `json:"key"`
}

type QRStatus struct {
    Status    string `json:"status"`
    MusicID   string `json:"music_id,omitempty"`
    MusicKey  string `json:"music_key,omitempty"`
    UserName  string `json:"user_name,omitempty"`
    AvatarURL string `json:"avatar_url,omitempty"`
}

func (c *Client) GetLoginQRCode(ctx context.Context) (*QRCode, error) {
    resp, err := c.Call(ctx, map[string]MusicuSubRequest{
        "req_0": {
            Module: "music.login.Qrcode",
            Method: "GetQrcode",
            Param:  map[string]any{},
        },
    })
    if err != nil {
        return nil, err
    }
    sub, ok := resp.Req["req_0"]
    if !ok || sub.Code != 0 {
        return nil, fmt.Errorf("get qrcode failed: code=%d", sub.Code)
    }
    return &QRCode{
        QrcodeURL: getString(sub.Data, "qrcode_url"),
        Key:       getString(sub.Data, "qrcode_key"),
    }, nil
}

func (c *Client) CheckQRCodeStatus(ctx context.Context, key string) (*QRStatus, error) {
    resp, err := c.Call(ctx, map[string]MusicuSubRequest{
        "req_0": {
            Module: "music.login.Qrcode",
            Method: "CheckQrcode",
            Param: map[string]any{
                "qrcode_key": key,
            },
        },
    })
    if err != nil {
        return nil, err
    }
    sub, ok := resp.Req["req_0"]
    if !ok || sub.Code != 0 {
        return nil, fmt.Errorf("check qrcode failed: code=%d", sub.Code)
    }

    statusInt := getInt(sub.Data, "status")
    statusMap := map[int]string{1: "pending", 2: "scanned", 3: "confirmed", 4: "expired"}
    status := statusMap[statusInt]
    if status == "" {
        status = "pending"
    }

    qr := &QRStatus{Status: status}
    if status == "confirmed" {
        qr.MusicID = fmt.Sprintf("%d", getInt(sub.Data, "musicid"))
        qr.MusicKey = getString(sub.Data, "musickey")
        qr.UserName = getString(sub.Data, "nickname")
        qr.AvatarURL = getString(sub.Data, "headurl")
    }
    return qr, nil
}
```

**Step 4: Run test to verify it passes**

```bash
cd server && go test ./internal/tme/ -run "TestGetLoginQRCode|TestCheckQRCodeStatus" -v -count=1
```
Expected: PASS

**Step 5: Commit**

```bash
git add server/internal/tme/login.go server/internal/tme/login_test.go
git commit -m "feat(tme): add QQ Music QR code login API"
```

---

### Task 3: Player API endpoints (`server/internal/api/player.go`)

**Files:**
- Create: `server/internal/api/player.go`
- Test: `server/internal/api/player_test.go`

**Step 1: Write the test — GetPlayURL**

```go
// server/internal/api/player_test.go
package api

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/music-agent/music-agent/internal/tme"
)

func TestGetPlayURL_Success(t *testing.T) {
    c := tme.NewClient()
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{
            "code": 0,
            "req_0": {
                "code": 0,
                "data": {
                    "midurlinfo": [{"purl": "C400001abc.m4a"}],
                    "expiration": 3600
                }
            }
        }`))
    }))
    defer srv.Close()
    c.SetBaseURL(srv.URL)

    ph := NewPlayerHandler(c, tme.NewCredentialStore())

    req := httptest.NewRequest("GET", "/api/player/url/qqmusic:001abc", nil)
    w := httptest.NewRecorder()
    ph.HandleGetPlayURL(w, req)

    if w.Code != 200 {
        t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
    }
    var resp map[string]any
    json.Unmarshal(w.Body.Bytes(), &resp)
    if resp["source"] != "cdn" {
        t.Errorf("expected source cdn, got %v", resp["source"])
    }
}

func TestGetLyrics_Success(t *testing.T) {
    c := tme.NewClient()
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{
            "code": 0,
            "req_0": {
                "code": 0,
                "data": {
                    "lyric": "W0xyaWNd",
                    "trans": ""
                }
            }
        }`))
    }))
    defer srv.Close()
    c.SetBaseURL(srv.URL)

    ph := NewPlayerHandler(c, tme.NewCredentialStore())

    req := httptest.NewRequest("GET", "/api/player/lyrics/qqmusic%3A001abc", nil)
    w := httptest.NewRecorder()
    ph.HandleGetLyrics(w, req)

    if w.Code != 200 {
        t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
    }
}
```

**Step 2: Run test to verify it fails**

```bash
cd server && go test ./internal/api/ -run "TestGetPlayURL|TestGetLyrics" -v -count=1
```
Expected: FAIL — `NewPlayerHandler not defined`

**Step 3: Implement player.go**

```go
// server/internal/api/player.go
package api

import (
    "encoding/json"
    "net/http"
    "strings"

    "github.com/music-agent/music-agent/internal/tme"
)

type PlayerHandler struct {
    client *tme.Client
    creds  *tme.CredentialStore
}

func NewPlayerHandler(client *tme.Client, creds *tme.CredentialStore) *PlayerHandler {
    return &PlayerHandler{client: client, creds: creds}
}

func (h *PlayerHandler) HandleGetPlayURL(w http.ResponseWriter, r *http.Request) {
    songID := strings.TrimPrefix(r.URL.Path, "/api/player/url/")
    if songID == "" {
        writeJSON(w, 400, map[string]any{"error": "song_id required"})
        return
    }

    if h.creds.IsLoggedIn() {
        mid, mk := h.creds.Get()
        h.client.SetCredential(mid, mk)
    }

    url, err := h.client.GetSongURL(r.Context(), songID)
    if err != nil {
        writeJSON(w, 500, map[string]any{"error": err.Error()})
        return
    }

    source := "cdn"
    if url.URL == "" {
        source = "unavailable"
    }

    writeJSON(w, 200, map[string]any{
        "song_id":            url.SongID,
        "url":                url.URL,
        "expires_in_seconds": url.ExpiresInSeconds,
        "source":             source,
    })
}

func (h *PlayerHandler) HandleGetLyrics(w http.ResponseWriter, r *http.Request) {
    songID := strings.TrimPrefix(r.URL.Path, "/api/player/lyrics/")
    if songID == "" {
        writeJSON(w, 400, map[string]any{"error": "song_id required"})
        return
    }

    lyrics, err := h.client.GetLyrics(r.Context(), songID)
    if err != nil {
        writeJSON(w, 500, map[string]any{"error": err.Error()})
        return
    }

    writeJSON(w, 200, lyrics)
}
```

**Step 4: Run test to verify it passes**

```bash
cd server && go test ./internal/api/ -run "TestGetPlayURL|TestGetLyrics" -v -count=1
```
Expected: PASS

**Step 5: Commit**

```bash
git add server/internal/api/player.go server/internal/api/player_test.go
git commit -m "feat(api): add player URL and lyrics endpoints"
```

---

### Task 4: QQ Music Login handlers (`server/internal/api/qqmusic_login.go`)

**Files:**
- Create: `server/internal/api/qqmusic_login.go`
- Test: `server/internal/api/qqmusic_login_test.go`

**Step 1: Write the test — LoginQRCode**

```go
// server/internal/api/qqmusic_login_test.go
package api

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/music-agent/music-agent/internal/tme"
)

func TestLoginQRCodeHandler(t *testing.T) {
    c := tme.NewClient()
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{
            "code": 0,
            "req_0": {
                "code": 0,
                "data": {
                    "qrcode_url": "https://qrcode.example.com/abc",
                    "qrcode_key": "key_123"
                }
            }
        }`))
    }))
    defer srv.Close()
    c.SetBaseURL(srv.URL)

    lh := NewLoginHandler(c, tme.NewCredentialStore())

    req := httptest.NewRequest("POST", "/api/qqmusic/login/qrcode", nil)
    w := httptest.NewRecorder()
    lh.HandleGetQRCode(w, req)

    if w.Code != 200 {
        t.Fatalf("expected 200, got %d", w.Code)
    }
    var resp map[string]any
    json.Unmarshal(w.Body.Bytes(), &resp)
    if resp["key"] != "key_123" {
        t.Errorf("got key %v", resp["key"])
    }
}

func TestLoginStatusHandler_NotLoggedIn(t *testing.T) {
    lh := NewLoginHandler(tme.NewClient(), tme.NewCredentialStore())

    req := httptest.NewRequest("GET", "/api/qqmusic/login/status", nil)
    w := httptest.NewRecorder()
    lh.HandleGetStatus(w, req)

    var resp map[string]any
    json.Unmarshal(w.Body.Bytes(), &resp)
    if resp["logged_in"] != false {
        t.Error("expected logged_in false")
    }
}
```

**Step 2: Run test to verify it fails**

```bash
cd server && go test ./internal/api/ -run "TestLoginQRCodeHandler|TestLoginStatusHandler" -v -count=1
```
Expected: FAIL — `NewLoginHandler not defined`

**Step 3: Implement qqmusic_login.go**

```go
// server/internal/api/qqmusic_login.go
package api

import (
    "net/http"
    "strings"

    "github.com/music-agent/music-agent/internal/tme"
)

type LoginHandler struct {
    client *tme.Client
    creds  *tme.CredentialStore
}

func NewLoginHandler(client *tme.Client, creds *tme.CredentialStore) *LoginHandler {
    return &LoginHandler{client: client, creds: creds}
}

func (h *LoginHandler) HandleGetQRCode(w http.ResponseWriter, r *http.Request) {
    qr, err := h.client.GetLoginQRCode(r.Context())
    if err != nil {
        writeJSON(w, 500, map[string]any{"error": err.Error()})
        return
    }
    writeJSON(w, 200, qr)
}

func (h *LoginHandler) HandleCheckQRStatus(w http.ResponseWriter, r *http.Request) {
    key := strings.TrimPrefix(r.URL.Path, "/api/qqmusic/login/status/")
    if key == "" || key == "/api/qqmusic/login/status" {
        writeJSON(w, 400, map[string]any{"error": "key required"})
        return
    }

    status, err := h.client.CheckQRCodeStatus(r.Context(), key)
    if err != nil {
        writeJSON(w, 500, map[string]any{"error": err.Error()})
        return
    }

    if status.Status == "confirmed" {
        h.creds.Set(status.MusicID, status.MusicKey)
    }

    writeJSON(w, 200, status)
}

func (h *LoginHandler) HandleGetStatus(w http.ResponseWriter, r *http.Request) {
    loggedIn := h.creds.IsLoggedIn()
    resp := map[string]any{"logged_in": loggedIn}
    if loggedIn {
        resp["user_name"] = h.creds.GetUserName()
    }
    writeJSON(w, 200, resp)
}
```

**Update CredentialStore** to support username:

```go
// Add to credential_store.go:
type CredentialStore struct {
    musicid  string
    musickey string
    userName string  // 🆕
    mu       sync.RWMutex
}

func (s *CredentialStore) SetUserInfo(musicid, musickey, userName string) { ... }
func (s *CredentialStore) GetUserName() string { ... }
```

**Step 4: Run test to verify it passes**

```bash
cd server && go test ./internal/api/ -run "TestLoginQRCodeHandler|TestLoginStatusHandler" -v -count=1
```
Expected: PASS

**Step 5: Commit**

```bash
git add server/internal/api/qqmusic_login.go server/internal/api/qqmusic_login_test.go server/internal/tme/credential_store.go
git commit -m "feat(api): add QQ Music login handlers"
```

---

### Task 5: Audio stream proxy (`server/internal/api/player_stream.go`)

**Files:**
- Create: `server/internal/api/player_stream.go`
- Test: `server/internal/api/player_stream_test.go`

**Step 1: Write the test**

```go
// server/internal/api/player_stream_test.go
package api

import (
    "io"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/music-agent/music-agent/internal/tme"
)

func TestStreamProxy_Success(t *testing.T) {
    audioBytes := []byte("fake-audio-data")
    // Start a mock CDN server
    cdnSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "audio/mp4")
        w.Write(audioBytes)
    }))
    defer cdnSrv.Close()

    // Start a mock TME API server
    tmeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{
            "code": 0,
            "req_0": {
                "code": 0,
                "data": {
                    "midurlinfo": [{"purl": "` + cdnSrv.URL + `/dummy.m4a"}],
                    "expiration": 3600
                }
            }
        }`))
    }))
    defer tmeSrv.Close()

    c := tme.NewClient()
    c.SetBaseURL(tmeSrv.URL)
    sh := NewStreamHandler(c)

    req := httptest.NewRequest("GET", "/api/player/stream/qqmusic:test", nil)
    w := httptest.NewRecorder()
    sh.HandleStream(w, req)

    if w.Code != 200 {
        t.Fatalf("expected 200, got %d", w.Code)
    }
    body, _ := io.ReadAll(w.Result().Body)
    if string(body) != string(audioBytes) {
        t.Errorf("body mismatch: got %q", string(body))
    }
    if w.Header().Get("Content-Type") != "audio/mp4" {
        t.Errorf("expected Content-Type audio/mp4, got %q", w.Header().Get("Content-Type"))
    }
}

func TestStreamProxy_NoURL(t *testing.T) {
    tmeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{
            "code": 0,
            "req_0": {
                "code": 0,
                "data": {
                    "midurlinfo": [],
                    "expiration": 0
                }
            }
        }`))
    }))
    defer tmeSrv.Close()

    c := tme.NewClient()
    c.SetBaseURL(tmeSrv.URL)
    sh := NewStreamHandler(c)

    req := httptest.NewRequest("GET", "/api/player/stream/qqmusic:test", nil)
    w := httptest.NewRecorder()
    sh.HandleStream(w, req)

    if w.Code != 404 {
        t.Errorf("expected 404 for no URL, got %d", w.Code)
    }
}
```

**Step 2: Run test to verify it fails**

```bash
cd server && go test ./internal/api/ -run "TestStreamProxy" -v -count=1
```
Expected: FAIL — `NewStreamHandler not defined`

**Step 3: Implement player_stream.go**

```go
// server/internal/api/player_stream.go
package api

import (
    "io"
    "net/http"
    "strings"

    "github.com/music-agent/music-agent/internal/tme"
)

type StreamHandler struct {
    client     *tme.Client
    httpClient *http.Client
}

func NewStreamHandler(client *tme.Client) *StreamHandler {
    return &StreamHandler{client: client, httpClient: &http.Client{}}
}

func (h *StreamHandler) HandleStream(w http.ResponseWriter, r *http.Request) {
    songID := strings.TrimPrefix(r.URL.Path, "/api/player/stream/")
    if songID == "" {
        http.Error(w, "song_id required", 400)
        return
    }

    urlResp, err := h.client.GetSongURL(r.Context(), songID)
    if err != nil || urlResp.URL == "" {
        http.Error(w, "no playable URL found", 404)
        return
    }

    req, err := http.NewRequestWithContext(r.Context(), "GET", urlResp.URL, nil)
    if err != nil {
        http.Error(w, "failed to create proxy request", 500)
        return
    }

    // Forward Range header for seeking
    if rangeHdr := r.Header.Get("Range"); rangeHdr != "" {
        req.Header.Set("Range", rangeHdr)
    }

    resp, err := h.httpClient.Do(req)
    if err != nil {
        http.Error(w, "failed to fetch audio", 502)
        return
    }
    defer resp.Body.Close()

    // Copy response headers
    if ct := resp.Header.Get("Content-Type"); ct != "" {
        w.Header().Set("Content-Type", ct)
    }
    if cr := resp.Header.Get("Content-Range"); cr != "" {
        w.Header().Set("Content-Range", cr)
    }
    if cl := resp.Header.Get("Content-Length"); cl != "" {
        w.Header().Set("Content-Length", cl)
    }
    w.Header().Set("Accept-Ranges", "bytes")
    w.Header().Set("Cache-Control", "public, max-age=3600")
    w.WriteHeader(resp.StatusCode)

    io.Copy(w, resp.Body)
}
```

**Step 4: Run test to verify it passes**

```bash
cd server && go test ./internal/api/ -run "TestStreamProxy" -v -count=1
```
Expected: PASS

**Step 5: Commit**

```bash
git add server/internal/api/player_stream.go server/internal/api/player_stream_test.go
git commit -m "feat(api): add audio stream proxy endpoint"
```

---

### Task 6: Router + main.go wiring

**Files:**
- Modify: `server/internal/api/router.go`
- Modify: `server/cmd/server/main.go`

**Step 1: Update router.go**

Add to `server/internal/api/router.go`, to the `SetupRoutes` function or a new `SetupPlayerRoutes`:

```go
// In router.go, add:
func SetupPlayerRoutes(r chi.Router, ph *PlayerHandler, sh *StreamHandler, lh *LoginHandler) {
    r.Get("/api/player/url/*", ph.HandleGetPlayURL)
    r.Get("/api/player/stream/*", sh.HandleStream)
    r.Get("/api/player/lyrics/*", ph.HandleGetLyrics)
    r.Post("/api/qqmusic/login/qrcode", lh.HandleGetQRCode)
    r.Get("/api/qqmusic/login/status/*", lh.HandleCheckQRStatus)
    r.Get("/api/qqmusic/login/status", lh.HandleGetStatus)
}
```

**Step 2: Update main.go**

In `server/cmd/server/main.go`, after creating the TME client:

```go
// Create credential store and login
credStore := tme.NewCredentialStore()
tmec := tme.NewClient()
if credStore.IsLoggedIn() {
    mid, mk := credStore.Get()
    tmec.SetCredential(mid, mk)
}

playerH := api.NewPlayerHandler(tmec, credStore)
streamH := api.NewStreamHandler(tmec)
loginH := api.NewLoginHandler(tmec, credStore)

api.SetupPlayerRoutes(r, playerH, streamH, loginH)
```

**Step 3: Run all API tests**

```bash
cd server && go test ./internal/api/ -v -count=1
```
Expected: All existing + new tests PASS

**Step 4: Build to verify**

```bash
cd server && go build ./...
```
Expected: Exit code 0

**Step 5: Commit**

```bash
git add server/internal/api/router.go server/cmd/server/main.go
git commit -m "feat: wire player and login routes into router and main"
```

---

## Phase 2: Frontend Foundation (parallel with Phase 1)

### Task 7: Update Types (`web/src/types.ts`)

**Files:**
- Modify: `web/src/types.ts`

**Step 1: Add new types**

Add after existing types:

```typescript
// Playback types
export type PlaybackMode = 'sequential' | 'repeat_one' | 'repeat_all' | 'shuffle';

export interface LyricLine {
  time: number;  // seconds
  text: string;
}

export interface PlayerState {
  currentSong: Song | null;
  queue: Song[];
  queueIndex: number;
  isPlaying: boolean;
  currentTime: number;
  duration: number;
  volume: number;
  playbackMode: PlaybackMode;
  urlSource: 'cdn' | 'proxy' | null;
  urlExpiresAt: number | null;
  lyrics: LyricLine[] | null;
  activeLyricIndex: number;
  panelOpen: boolean;
}

export interface SongURLResponse {
  song_id: string;
  url: string;
  expires_in_seconds: number;
  source: 'cdn' | 'unavailable';
}

export interface LyricsResponse {
  song_id: string;
  plain_text: string;
  synced_text: string;
}

export interface QRCodeResponse {
  qrcode_url: string;
  key: string;
}

export interface QRStatusResponse {
  status: 'pending' | 'scanned' | 'confirmed' | 'expired';
  music_id?: string;
  music_key?: string;
  user_name?: string;
  avatar_url?: string;
}

export interface LoginStatusResponse {
  logged_in: boolean;
  user_name?: string;
}
```

**Step 2: Run TypeScript check**

```bash
cd web && npx tsc --noEmit
```
Expected: Exit code 0

**Step 3: Commit**

```bash
git add web/src/types.ts
git commit -m "feat(web): add playback and login types"
```

---

### Task 8: Player API Client (`web/src/api/player.ts`)

**Files:**
- Create: `web/src/api/player.ts`
- Test: `web/src/api/player.test.ts`

**Step 1: Write the test**

```typescript
// web/src/api/player.test.ts
import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { http, HttpResponse } from 'msw';
import { server } from '../test-setup';
import { getPlayUrl, getLyrics, getStreamUrl } from './player';

describe('player API', () => {
  it('getPlayUrl returns song URL with expiry', async () => {
    server.use(
      http.get('/api/player/url/:songId', () =>
        HttpResponse.json({
          song_id: 'qqmusic:001abc',
          url: 'https://isure.stream.qqmusic.qq.com/C400001abc.m4a',
          expires_in_seconds: 3600,
          source: 'cdn',
        })
      )
    );
    const result = await getPlayUrl('qqmusic:001abc');
    expect(result.url).toBe('https://isure.stream.qqmusic.qq.com/C400001abc.m4a');
    expect(result.expires_in_seconds).toBe(3600);
    expect(result.source).toBe('cdn');
  });

  it('getLyrics returns lyrics data', async () => {
    server.use(
      http.get('/api/player/lyrics/:songId', () =>
        HttpResponse.json({
          song_id: 'qqmusic:001abc',
          plain_text: 'test lyrics',
          synced_text: '[00:00.00]test lyrics',
        })
      )
    );
    const result = await getLyrics('qqmusic:001abc');
    expect(result.plain_text).toBe('test lyrics');
    expect(result.synced_text).toBe('[00:00.00]test lyrics');
  });

  it('getStreamUrl returns proxy stream URL', () => {
    const url = getStreamUrl('qqmusic:001abc');
    expect(url).toContain('/api/player/stream/qqmusic%3A001abc');
  });
});
```

**Step 2: Run test to verify it fails**

```bash
cd web && npx vitest run src/api/player.test.ts
```
Expected: FAIL — module not found

**Step 3: Implement player.ts**

```typescript
// web/src/api/player.ts
import type { SongURLResponse, LyricsResponse, QRCodeResponse, QRStatusResponse, LoginStatusResponse } from '../types';

const BASE = '';

export async function getPlayUrl(songId: string): Promise<SongURLResponse> {
  const res = await fetch(`${BASE}/api/player/url/${encodeURIComponent(songId)}`);
  if (!res.ok) throw new Error(`Failed to get play URL: ${res.status}`);
  return res.json();
}

export async function getLyrics(songId: string): Promise<LyricsResponse> {
  const res = await fetch(`${BASE}/api/player/lyrics/${encodeURIComponent(songId)}`);
  if (!res.ok) throw new Error(`Failed to get lyrics: ${res.status}`);
  return res.json();
}

export function getStreamUrl(songId: string): string {
  return `${BASE}/api/player/stream/${encodeURIComponent(songId)}`;
}

export async function getLoginQRCode(): Promise<QRCodeResponse> {
  const res = await fetch(`${BASE}/api/qqmusic/login/qrcode`, { method: 'POST' });
  if (!res.ok) throw new Error(`Failed to get QR code: ${res.status}`);
  return res.json();
}

export async function checkQRStatus(key: string): Promise<QRStatusResponse> {
  const res = await fetch(`${BASE}/api/qqmusic/login/status/${encodeURIComponent(key)}`);
  if (!res.ok) throw new Error(`Failed to check QR status: ${res.status}`);
  return res.json();
}

export async function getLoginStatus(): Promise<LoginStatusResponse> {
  const res = await fetch(`${BASE}/api/qqmusic/login/status`);
  if (!res.ok) throw new Error(`Failed to get login status: ${res.status}`);
  return res.json();
}
```

**Step 4: Run test to verify it passes**

```bash
cd web && npx vitest run src/api/player.test.ts
```
Expected: PASS

**Step 5: Commit**

```bash
git add web/src/api/player.ts web/src/api/player.test.ts
git commit -m "feat(web): add player API client"
```

---

### Task 9: QQ Music Login Hook (`web/src/hooks/useQQMusicLogin.ts`)

**Files:**
- Create: `web/src/hooks/useQQMusicLogin.ts`
- Test: `web/src/hooks/useQQMusicLogin.test.ts`

**Step 1: Write the test**

```typescript
// web/src/hooks/useQQMusicLogin.test.ts
import { describe, it, expect, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { server } from '../test-setup';
import { useQQMusicLogin } from './useQQMusicLogin';

describe('useQQMusicLogin', () => {
  beforeEach(() => {
    server.use(
      http.post('/api/qqmusic/login/qrcode', () =>
        HttpResponse.json({ qrcode_url: 'https://qr.example.com/test', key: 'test_key_123' })
      ),
      http.get('/api/qqmusic/login/status/test_key_123', () =>
        HttpResponse.json({ status: 'confirmed', user_name: 'TestUser' })
      ),
      http.get('/api/qqmusic/login/status', () =>
        HttpResponse.json({ logged_in: false })
      ),
    );
  });

  it('starts with idle status', () => {
    const { result } = renderHook(() => useQQMusicLogin());
    expect(result.current.loginStatus).toBe('idle');
    expect(result.current.qrcodeUrl).toBeNull();
  });

  it('startLogin sets qrcodeUrl', async () => {
    const { result } = renderHook(() => useQQMusicLogin());
    await act(() => result.current.startLogin());
    expect(result.current.qrcodeUrl).toBe('https://qr.example.com/test');
    expect(result.current.loginStatus).toBe('pending_scan');
  });

  it('checkStatus confirms login', async () => {
    const { result } = renderHook(() => useQQMusicLogin());
    await act(() => result.current.startLogin());
    await act(() => result.current.checkStatus());
    await waitFor(() => {
      expect(result.current.loginStatus).toBe('confirmed');
      expect(result.current.userName).toBe('TestUser');
    });
  });
});
```

**Step 2: Run test to verify it fails**

```bash
cd web && npx vitest run src/hooks/useQQMusicLogin.test.ts
```
Expected: FAIL

**Step 3: Implement useQQMusicLogin.ts**

```typescript
// web/src/hooks/useQQMusicLogin.ts
import { useState, useCallback, useRef } from 'react';
import { getLoginQRCode, checkQRStatus, getLoginStatus } from '../api/player';

type LoginStatus = 'idle' | 'loading' | 'pending_scan' | 'scanned' | 'confirmed' | 'expired' | 'error';

export function useQQMusicLogin() {
  const [loginStatus, setLoginStatus] = useState<LoginStatus>('idle');
  const [qrcodeUrl, setQrcodeUrl] = useState<string | null>(null);
  const [userName, setUserName] = useState<string | null>(null);
  const qrKeyRef = useRef<string | null>(null);

  const startLogin = useCallback(async () => {
    setLoginStatus('loading');
    try {
      const qr = await getLoginQRCode();
      setQrcodeUrl(qr.qrcode_url);
      qrKeyRef.current = qr.key;
      setLoginStatus('pending_scan');
    } catch {
      setLoginStatus('error');
    }
  }, []);

  const checkStatus = useCallback(async () => {
    if (!qrKeyRef.current) return;
    try {
      const status = await checkQRStatus(qrKeyRef.current);
      if (status.status === 'confirmed') {
        setLoginStatus('confirmed');
        setUserName(status.user_name || null);
        qrKeyRef.current = null;
      } else if (status.status === 'expired') {
        setLoginStatus('expired');
        qrKeyRef.current = null;
      }
    } catch {
      // Polling errors are expected, don't change state
    }
  }, []);

  const logout = useCallback(async () => {
    setLoginStatus('idle');
    setQrcodeUrl(null);
    setUserName(null);
    qrKeyRef.current = null;
  }, []);

  return {
    loginStatus,
    qrcodeUrl,
    userName,
    isLoggedIn: loginStatus === 'confirmed',
    startLogin,
    checkStatus,
    logout,
  };
}
```

**Step 4: Run test to verify it passes**

```bash
cd web && npx vitest run src/hooks/useQQMusicLogin.test.ts
```
Expected: PASS

**Step 5: Commit**

```bash
git add web/src/hooks/useQQMusicLogin.ts web/src/hooks/useQQMusicLogin.test.ts
git commit -m "feat(web): add QQ Music login hook"
```

---

### Task 10: Player Store Hook (`web/src/hooks/usePlayerStore.ts`)

**Files:**
- Create: `web/src/hooks/usePlayerStore.ts`
- Test: `web/src/hooks/usePlayerStore.test.ts`

**Step 1: Write the test**

```typescript
// web/src/hooks/usePlayerStore.test.ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { usePlayerStore } from './usePlayerStore';

const MOCK_HTML5_AUDIO = { play: vi.fn(), pause: vi.fn(), currentTime: 0, src: '' };
// Note: html5 audio mocking is done via vi.stubGlobal in test-setup

const mockSong = { id: 'qqmusic:001', title: 'Test', artist: 'Artist' };

describe('usePlayerStore', () => {
  it('play sets current song and queue', () => {
    const { result } = renderHook(() => usePlayerStore());
    act(() => result.current.play(mockSong));
    expect(result.current.state.currentSong?.id).toBe('qqmusic:001');
    expect(result.current.state.queue).toHaveLength(1);
    expect(result.current.state.queue[0].id).toBe('qqmusic:001');
  });

  it('addToQueue appends song', () => {
    const { result } = renderHook(() => usePlayerStore());
    act(() => result.current.addToQueue(mockSong));
    act(() => result.current.addToQueue({ ...mockSong, id: 'qqmusic:002' }));
    expect(result.current.state.queue).toHaveLength(2);
  });

  it('removeFromQueue removes song', () => {
    const { result } = renderHook(() => usePlayerStore());
    act(() => result.current.play(mockSong));
    act(() => result.current.addToQueue({ ...mockSong, id: 'qqmusic:002' }));
    act(() => result.current.removeFromQueue(1));
    expect(result.current.state.queue).toHaveLength(1);
  });

  it('clearQueue clears all but keeps current', () => {
    const { result } = renderHook(() => usePlayerStore());
    act(() => result.current.play(mockSong));
    act(() => result.current.addToQueue({ ...mockSong, id: 'qqmusic:002' }));
    act(() => result.current.clearQueue());
    expect(result.current.state.queue).toHaveLength(0);
  });

  it('togglePanel opens and closes panel', () => {
    const { result } = renderHook(() => usePlayerStore());
    expect(result.current.state.panelOpen).toBe(false);
    act(() => result.current.togglePanel());
    expect(result.current.state.panelOpen).toBe(true);
    act(() => result.current.togglePanel());
    expect(result.current.state.panelOpen).toBe(false);
  });

  it('persists queue to localStorage', () => {
    localStorage.clear();
    const { result } = renderHook(() => usePlayerStore());
    act(() => result.current.play(mockSong));
    const stored = JSON.parse(localStorage.getItem('player_state') || '{}');
    expect(stored.queue).toHaveLength(1);
    expect(stored.queue[0].id).toBe('qqmusic:001');
  });
});
```

**Step 2: Run test to verify it fails**

```bash
cd web && npx vitest run src/hooks/usePlayerStore.test.ts
```
Expected: FAIL

**Step 3: Implement usePlayerStore.ts**

```typescript
// web/src/hooks/usePlayerStore.ts
import { useState, useCallback, useEffect, useRef } from 'react';
import type { Song, PlayerState, PlaybackMode } from '../types';
import { getPlayUrl, getStreamUrl, getLyrics } from '../api/player';

const STORAGE_KEY = 'player_state';
const URL_CACHE: Record<string, { url: string; expiresAt: number }> = {};

const DEFAULT_STATE: PlayerState = {
  currentSong: null, queue: [], queueIndex: 0,
  isPlaying: false, currentTime: 0, duration: 0, volume: 0.7,
  playbackMode: 'sequential', urlSource: null, urlExpiresAt: null,
  lyrics: null, activeLyricIndex: 0, panelOpen: false,
};

function loadState(): Partial<PlayerState> {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) return JSON.parse(raw);
  } catch {}
  return {};
}

function persistState(state: PlayerState) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({
      queue: state.queue, queueIndex: state.queueIndex,
      playbackMode: state.playbackMode, volume: state.volume,
    }));
  } catch {}
}

async function ensurePlayableURL(song: Song, audioEl: HTMLAudioElement | null): Promise<string> {
  const cache = URL_CACHE[song.id];
  if (cache && Date.now() < cache.expiresAt - 60000) return cache.url;

  try {
    const resp = await getPlayUrl(song.id);
    if (resp.url && resp.source === 'cdn') {
      URL_CACHE[song.id] = { url: resp.url, expiresAt: Date.now() + resp.expires_in_seconds * 1000 };
      return resp.url;
    }
  } catch {}

  return getStreamUrl(song.id);
}

function parseSyncedLyrics(synced: string): { time: number; text: string }[] {
  const lines = synced.split('\n');
  const result: { time: number; text: string }[] = [];
  const re = /\[(\d{2}):(\d{2})\.(\d{2,3})\](.*)/;
  for (const line of lines) {
    const m = line.match(re);
    if (m) {
      const mins = parseInt(m[1]), secs = parseInt(m[2]), ms = parseInt(m[3].padEnd(3, '0'));
      result.push({ time: mins * 60 + secs + ms / 1000, text: m[4].trim() });
    }
  }
  return result;
}

export function usePlayerStore() {
  const [state, setState] = useState<PlayerState>(() => ({ ...DEFAULT_STATE, ...loadState() }));
  const audioRef = useRef<HTMLAudioElement | null>(null);

  // Initialize audio element
  useEffect(() => {
    if (!audioRef.current) {
      audioRef.current = new Audio();
      audioRef.current.volume = state.volume;
    }
    return () => { audioRef.current?.pause(); };
  }, []);

  // Sync audio events
  useEffect(() => {
    const el = audioRef.current;
    if (!el) return;
    const onTime = () => setState(s => ({ ...s, currentTime: el.currentTime }));
    const onDur = () => setState(s => ({ ...s, duration: el.duration || 0 }));
    const onEnd = () => setState(s => ({ ...s, isPlaying: false }));
    el.addEventListener('timeupdate', onTime);
    el.addEventListener('durationchange', onDur);
    el.addEventListener('ended', onEnd);
    return () => {
      el.removeEventListener('timeupdate', onTime);
      el.removeEventListener('durationchange', onDur);
      el.removeEventListener('ended', onEnd);
    };
  }, []);

  // Persist
  useEffect(() => { persistState(state); }, [state.queue, state.queueIndex, state.playbackMode, state.volume]);

  const play = useCallback(async (song: Song) => {
    const url = await ensurePlayableURL(song, audioRef.current);
    if (!audioRef.current) return;
    audioRef.current.src = url;
    audioRef.current.play().catch(() => {});
    setState(s => ({
      ...s, currentSong: song, queue: [song, ...s.queue.filter(q => q.id !== song.id)],
      queueIndex: 0, isPlaying: true, urlSource: url.includes('/api/player/stream/') ? 'proxy' : 'cdn',
      lyrics: null, activeLyricIndex: 0,
    }));
    // Fetch lyrics
    getLyrics(song.id).then(l => {
      setState(s => ({ ...s, lyrics: parseSyncedLyrics(l.synced_text) }));
    }).catch(() => {});
  }, []);

  const togglePlay = useCallback(() => {
    if (!audioRef.current) return;
    if (state.isPlaying) audioRef.current.pause();
    else audioRef.current.play().catch(() => {});
    setState(s => ({ ...s, isPlaying: !s.isPlaying }));
  }, [state.isPlaying]);

  const next = useCallback(() => {
    setState(s => {
      const qi = s.queueIndex + 1;
      if (qi >= s.queue.length) return s;
      const nextSong = s.queue[qi];
      play(nextSong);
      return { ...s, queueIndex: qi };
    });
  }, [play]);

  const prev = useCallback(() => {
    setState(s => {
      const qi = Math.max(0, s.queueIndex - 1);
      const prevSong = s.queue[qi];
      play(prevSong);
      return { ...s, queueIndex: qi };
    });
  }, [play]);

  const seek = useCallback((seconds: number) => {
    if (audioRef.current) audioRef.current.currentTime = seconds;
    setState(s => ({ ...s, currentTime: seconds }));
  }, []);

  const setVolume = useCallback((v: number) => {
    if (audioRef.current) audioRef.current.volume = v;
    setState(s => ({ ...s, volume: v }));
  }, []);

  const addToQueue = useCallback((song: Song) => {
    setState(s => ({
      ...s, queue: [...s.queue.filter(q => q.id !== song.id), song],
    }));
  }, []);

  const removeFromQueue = useCallback((index: number) => {
    setState(s => {
      const q = [...s.queue]; q.splice(index, 1);
      return { ...s, queue: q };
    });
  }, []);

  const clearQueue = useCallback(() => {
    setState(s => ({ ...s, queue: [] }));
  }, []);

  const setPlaybackMode = useCallback((mode: PlaybackMode) => {
    setState(s => ({ ...s, playbackMode: mode }));
  }, []);

  const togglePanel = useCallback(() => {
    setState(s => ({ ...s, panelOpen: !s.panelOpen }));
  }, []);

  return {
    state,
    play, togglePlay, next, prev, seek, setVolume,
    addToQueue, removeFromQueue, clearQueue, setPlaybackMode,
    togglePanel,
  };
}
```

**Step 4: Run test to verify it passes**

```bash
cd web && npx vitest run src/hooks/usePlayerStore.test.ts
```
Expected: PASS

**Step 5: Commit**

```bash
git add web/src/hooks/usePlayerStore.ts
git commit -m "feat(web): add player store hook with hybrid URL fallback"
```

---

## Phase 3: Frontend Components (parallelizable)

### Task 11-14: Player UI Components (run in parallel)

These 4 components have no dependencies on each other and can be developed simultaneously:

#### Task 11: PlayerBar.tsx

**Files:** Create `web/src/components/PlayerBar.tsx`, test `web/src/components/PlayerBar.test.tsx`

Minimal bottom bar showing current song, progress, play/pause, next. Props: `{ state, onTogglePlay, onTogglePanel, onNext }` from `usePlayerStore`.

```tsx
export default function PlayerBar({ state, onTogglePlay, onTogglePanel, onNext }: PlayerBarProps) {
  if (!state.currentSong) return null;
  const progress = state.duration > 0 ? (state.currentTime / state.duration) * 100 : 0;
  const fmt = (s: number) => `${Math.floor(s/60)}:${String(Math.floor(s%60)).padStart(2,'0')}`;

  return (
    <div onClick={onTogglePanel} className="fixed bottom-0 left-0 right-0 bg-gray-900 border-t border-gray-700 p-2 cursor-pointer z-50">
      <div className="flex items-center gap-3 max-w-4xl mx-auto">
        <p className="text-sm text-gray-100 truncate w-40 text-left">{state.currentSong.title}</p>
        <p className="text-xs text-gray-400 truncate w-24 text-left">{state.currentSong.artist}</p>
        <div className="flex-1 mx-2">
          <div className="h-1 bg-gray-600 rounded"><div className="h-1 bg-green-500 rounded" style={{width:`${progress}%`}}/></div>
          <div className="flex justify-between text-xs text-gray-500 mt-0.5">
            <span>{fmt(state.currentTime)}</span><span>{fmt(state.duration)}</span>
          </div>
        </div>
        <button onClick={e => { e.stopPropagation(); onTogglePlay(); }} className="text-gray-300 hover:text-white p-1">
          {state.isPlaying ? <Pause size={18} /> : <Play size={18} />}
        </button>
        <button onClick={e => { e.stopPropagation(); onNext(); }} className="text-gray-300 hover:text-white p-1">
          <SkipForward size={18} />
        </button>
      </div>
    </div>
  );
}
```

#### Task 12: NowPlaying.tsx

**Files:** Create `web/src/components/NowPlaying.tsx`, test `web/src/components/NowPlaying.test.tsx`

Large album art, song info, playback controls (prev, play/pause, next, seek bar, volume, mode toggle).

#### Task 13: LyricsPanel.tsx

**Files:** Create `web/src/components/LyricsPanel.tsx`, test `web/src/components/LyricsPanel.test.tsx`

Scroll-synced lyrics display. Highlights current line based on `activeLyricIndex`.

#### Task 14: QueuePanel.tsx

**Files:** Create `web/src/components/QueuePanel.tsx`, test `web/src/components/QueuePanel.test.tsx`

Scrollable list of queued songs, with remove button per item. Click a song to play it.

**Step for each:** Write test → run to fail → implement → run to pass → commit.

---

### Task 15: PlayerPanel.tsx (combines NowPlaying + Lyrics + Queue)

**Files:** Create `web/src/components/PlayerPanel.tsx`, test `web/src/components/PlayerPanel.test.tsx`

Tab-based container. Props: `{ state, onTogglePanel, ...allCallbacks }`.

Tabs: "正在播放" | "歌词" | "队列" — renders NowPlaying / LyricsPanel / QueuePanel based on active tab.

---

### Task 16: SongCards modifications

**Files:** Modify `web/src/components/SongCards.tsx`, update test `web/src/components/SongCards.test.tsx`

Add props: `onPlay?: (song: Song) => void` and `onAddToQueue?: (song: Song) => void`.

Add a play icon (▶) on the right side of each card. Click card → `onPlay`. Click "+" button → `onAddToQueue`. Currently playing song gets a green border highlight (compare `currentSongId` prop).

---

## Phase 4: Integration

### Task 17: ChatPage Integration

**Files:** Modify `web/src/pages/ChatPage.tsx`, update test

- Import `usePlayerStore` hook
- Place `<PlayerBar>` at bottom (inside ChatPage, below the input)
- Place `<PlayerPanel>` above PlayerBar, conditionally rendered when `panelOpen`
- Pass `onPlay` and `onAddToQueue` callbacks down to `AgentMessageList` → `SongCards`
- Ensure layout shifts when player bar is visible (add `pb-14` to content area)

### Task 18: LoginPage Integration

**Files:** Modify `web/src/pages/LoginPage.tsx`, update test

- Import `useQQMusicLogin` hook
- Add QQ Music login section below existing WeChat login
- Show QR code image when login started
- Show status messages: "请用QQ音乐App扫描二维码" → "已扫码，确认中..." → "登录成功"
- Auto-poll every 2s during `pending_scan` status

---

## Phase 5: Tests & Coverage

### Task 19: Backend coverage check

```bash
cd server && go test ./internal/... -coverprofile=coverage.out -covermode=atomic -count=1 && go tool cover -func=coverage.out | grep total:
```
Expected: `total: ≥ 80.0%`

If any package drops below 80%, add tests until it meets threshold.

### Task 20: Frontend coverage check

```bash
cd web && npx vitest run --coverage
```
Expected: `All files ≥ 80%`

If any file drops below 80%, add tests until it meets threshold.

---

## Execution Order

```
Phase 1 (Backend, parallelizable):
  Task 1 (CredentialStore) ──┐
  Task 2 (Login API)        ├── can run together
  Task 3 (Player endpoints) ├── can run together
  Task 5 (Stream proxy)     ├── can run together
                             │
  Task 4 (Login handlers ── depends on 1,2) ──┐
  Task 6 (Router + main ─── depends on 1-5)   ├── sequential

Phase 2 (Frontend foundation, parallel with Phase 1):
  Task 7 (Types) ──► Task 8 (API client) ──► Task 9 (Login hook) + Task 10 (PlayerStore)

Phase 3 (Frontend components, parallel):
  Task 11 (PlayerBar) ──┐
  Task 12 (NowPlaying)  ├── parallel
  Task 13 (LyricsPanel) ├── parallel
  Task 14 (QueuePanel)  ├── parallel
                        │
  Task 15 (PlayerPanel ─ depends on 12,13,14)
  Task 16 (SongCards ─── depends on 10)

Phase 4 (Integration):
  Task 17 (ChatPage ── depends on 11,15,16)
  Task 18 (LoginPage ─ depends on 9)

Phase 5 (Coverage):
  Task 19 (Backend) ──┐
  Task 20 (Frontend)   ├── parallel
```

---

## Verification Checklist

- [ ] `make test-backend` passes with ≥80% coverage
- [ ] `cd web && npx vitest run --coverage` passes with ≥80% coverage
- [ ] `cd server && go build ./...` exit 0
- [ ] `cd web && npx tsc --noEmit` exit 0
- [ ] Manual: click song card → PlayerBar appears, plays audio
- [ ] Manual: open PlayerPanel → see lyrics + queue
- [ ] Manual: add songs to queue → correct order
- [ ] Manual: QQ Music QR login flow works
