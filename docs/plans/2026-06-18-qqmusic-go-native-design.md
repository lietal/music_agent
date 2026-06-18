# QQ Music Go-Native Client — Design Document

**Date**: 2026-06-18
**Status**: Proposed
**Author**: AI-assisted architectural evaluation

---

## Problem

The Go backend (`cmd/server/main.go`) calls a Python FastAPI sidecar at `http://127.0.0.1:18767` for all QQ Music operations (search, song detail, playback URL, lyrics, auth). The sidecar depends on `qqmusic-api-python` (GPL-3.0). This creates:

- Two runtime dependencies (Go + Python 3.11+)
- Docker complexity (multi-stage, Python venv)
- An extra HTTP hop per request (~5-10ms)
- Licensing concern (GPL-3.0 transitive dependency)

## Goal

Eliminate the Python sidecar by implementing QQ Music API calls natively in Go, reducing the stack to a single Go binary.

## Research Findings

### The QQ Music API Is Plain HTTP JSON

Contrary to assumptions, the QQ Music API does NOT require complex signing, encryption, or protocol-level magic for basic operations. Evidence:

1. **Your sidecar uses `enable_sign=False`** — requests are sent without cryptographic signing:
   ```python
   # musio/providers/qqmusic-python-sidecar/app/qqmusic_client.py, line 1070
   async with Client(
       credential=credential or self._credential(),
       device_path=device_path,
       enable_sign=False,  # <-- Sign is explicitly disabled
   ) as client:
   ```

2. **The API endpoint is a single JSON POST gateway** — all operations go through:
   - `POST https://u.y.qq.com/cgi-bin/musicu.fcg`
   - Body: `{"module.name": {"method": "...", "module": "...", "param": {...}}}`

3. **A Go implementation already exists** — `guohuiyuan/music-lib/qq` (AGPL-3.0) uses the exact same endpoints, proving Go can do this natively.

### What the Python Sidecar Actually Does (1500 lines total)

| Component | Lines | Portability to Go |
|---|---|---|
| Auth (QR login) | 430 | Trivial — plain HTTP, hash33 is 5 lines |
| QQMusicClient (API calls) | 1488 | Straightforward — HTTP POST + JSON parsing |
| Response parsing (field extraction) | ~600 | Standard Go struct unmarshaling |
| Cover URL construction | ~100 | String manipulation |
| Lyrics decryption | ~50 | AES/DES via `crypto/aes` |
| Schema types | 174 | Go structs |

### API Endpoints Used

| Operation | Endpoint | Module.Method |
|---|---|---|
| Search | `https://u.y.qq.com/cgi-bin/musicu.fcg` | `music.search.SearchCgiService.DoSearchForQQMusicDesktop` |
| Song Detail | `https://u.y.qq.com/cgi-bin/musicu.fcg` | `music.pf_song_detail_svr.get_song_detail_yqq` |
| Playback URL | `https://u.y.qq.com/cgi-bin/musicu.fcg` | `vkey.GetVkeyServer.CgiGetVkey` |
| Lyrics | `https://c.y.qq.com/lyric/fcgi-bin/fcg_query_lyric_new.fcg` | (GET, requires Referer header) |
| QR Login | `https://ssl.ptlogin2.qq.com/ptqrshow` | (QR generation) |
| | `https://ssl.ptlogin2.qq.com/ptqrlogin` | (Poll login status) |
| | `https://graph.qq.com/oauth2.0/authorize` | (OAuth exchange) |

## Design

### Architecture

**Before:**
```
Go backend → HTTP → Python FastAPI → qqmusic-api-python → QQ Music API
```

**After:**
```
Go backend → internal/qqmusic/ → QQ Music API (direct HTTP)
```

### Package Structure

```
internal/
├── qqmusic/                   # NEW: Native QQ Music client
│   ├── client.go              # Client struct, credential mgmt, HTTP client
│   ├── search.go              # Search endpoint (P0)
│   ├── song.go                # Song detail + playback URL (P0-P1)
│   ├── lyrics.go              # Lyrics fetch + decrypt (P2)
│   ├── auth.go                # QR code login flow (P2, deferrable)
│   ├── types.go               # Go structs: Song, Lyrics, Credential, etc.
│   ├── decrypt.go             # AES/DES lyrics decryption
│   └── client_test.go         # Integration tests with mock HTTP server
├── tool/
│   └── tme_search.go          # MODIFY: call qqmusic.Search() directly
```

### Core Types

```go
// internal/qqmusic/types.go

type Credential struct {
    OpenID      string `json:"openid"`
    RefreshToken string `json:"refresh_token"`
    AccessToken string `json:"access_token"`
    ExpiredAt   int64  `json:"expired_at"`
    MusicID     int64  `json:"musicid"`
    MusicKey    string `json:"musickey"`
    UnionID     string `json:"unionid"`
    EncryptUin  string `json:"encrypt_uin"`
    LoginType   int    `json:"login_type"`
}

type Song struct {
    ID              string   `json:"id"`               // "qqmusic:<songmid>"
    Title           string   `json:"title"`
    Artists         []string `json:"artists"`
    Album           string   `json:"album,omitempty"`
    DurationSeconds int      `json:"duration_seconds,omitempty"`
    ArtworkURL      string   `json:"artwork_url,omitempty"`
}

type Client struct {
    httpClient *http.Client
    credential *Credential
}
```

### Search Implementation (Core P0 Feature)

```go
// internal/qqmusic/search.go

const qqMusicAPIURL = "https://u.y.qq.com/cgi-bin/musicu.fcg"

func (c *Client) Search(ctx context.Context, keyword string, limit int) ([]Song, error) {
    payload := map[string]any{
        "music.search.SearchCgiService": map[string]any{
            "method": "DoSearchForQQMusicDesktop",
            "module": "music.search.SearchCgiService",
            "param": map[string]any{
                "query":        keyword,
                "num_per_page": limit,
                "page_num":     1,
                "search_type":  0,
            },
        },
    }

    body, _ := json.Marshal(payload)
    req, _ := http.NewRequestWithContext(ctx, "POST", qqMusicAPIURL, bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Referer", "https://y.qq.com/")

    resp, err := c.httpClient.Do(req)
    // ... handle response, parse body.song.list[]
    return songs, nil
}
```

### Modified tme_search.go

```go
// internal/tool/tme_search.go (simplified)

type TMESearchSongs struct {
    client *qqmusic.Client
}

func (t *TMESearchSongs) Execute(ctx context.Context, args map[string]any) (ToolResult, error) {
    keyword := args["keyword"].(string)
    limit := 5
    // ...

    songs, err := t.client.Search(ctx, keyword, limit)  // <-- Direct Go call
    if err != nil {
        return ToolResult{}, err
    }

    data, _ := json.Marshal(songs)
    return ToolResult{Data: string(data)}, nil
}
```

### Auth Flow (Deferrable)

The QR code login flow maps to 5 HTTP calls. The critical algorithm:

```go
func hash33(s string, seed int) int {
    result := seed
    for _, c := range s {
        result = (result << 5) + result + int(c)
    }
    return result & 0x7FFFFFFF
}
```

Credential storage: `~/.musio/credentials/qqmusic.json` (same path as sidecar for compatibility).

### Error Handling

```go
var (
    ErrNotLoggedIn  = errors.New("qqmusic: no credential found at ~/.musio/credentials/qqmusic.json")
    ErrLoginExpired = errors.New("qqmusic: credential expired, re-login required")
    ErrRateLimited  = errors.New("qqmusic: rate limited by upstream")
)

type APIError struct {
    Code    int
    Message string
}

func (e *APIError) Error() string {
    return fmt.Sprintf("qqmusic api error %d: %s", e.Code, e.Message)
}
```

## Alternatives Considered

| Approach | Effort | Risk | License |
|---|---|---|---|
| **A: Keep sidecar** (status quo) | 0 | Python dep, GPL-3.0 | Concerns remain |
| **B: Use music-lib** (Go lib) | ~1 week | AGPL-3.0 dependency | AGPL-3.0 |
| **C: Thin Go adapter** (recommended) | ~1-2 weeks | API change maintenance | **Clean (MIT/Apache)** |

### Why Approach C over B

`guohuiyuan/music-lib` is AGPL-3.0 licensed. For most production services, this is problematic. The API surface you need is small enough (3-5 endpoints) that writing a thin adapter is less code than understanding music-lib's full codebase.

### Why Approach C over A

- Single binary deployment (one Dockerfile, one runtime)
- No GPL-3.0 concern
- Lower latency (no HTTP hop)
- Simpler debugging (single process)

## Implementation Plan Summary

1. **Create `internal/qqmusic/` package** with types and HTTP client
2. **Implement `Search()`** — match current sidecar output format
3. **Implement `SongDetail()`** — fetch by songmid
4. **Implement `SongURL()`** — fetch playback URL
5. **Rewrite `tme_search.go`** to call `qqmusic.Client` directly
6. **Remove sidecar** — delete `musio/providers/qqmusic-python-sidecar/`
7. **Update config** — remove `[providers.qqmusic]` section, add credential path
8. **Update Dockerfile** — single-stage Go build, remove Python
9. **Add tests** — mock HTTP server for search/song/url endpoints

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| QQ Music API changes | Monitor; Go adapter is small enough to update quickly |
| Lyrics decryption complexity | Start without lyrics support; add later if needed |
| Auth session management | Start without auth (credential file from sidecar); add QR flow later |
| Rate limiting | Add exponential backoff + circuit breaker pattern |

## Success Criteria

- [ ] `search_songs` tool returns identical results to current Python sidecar
- [ ] Single Go binary Docker image (no Python layer)
- [ ] All existing integration tests pass
- [ ] No GPL-3.0 dependencies in `go.mod`
- [ ] `musio/providers/qqmusic-python-sidecar/` directory deleted
