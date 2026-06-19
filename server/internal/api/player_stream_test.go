package api

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/music-agent/music-agent/internal/tme"
)

func TestStreamProxy_Success(t *testing.T) {
	cdnSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write([]byte("fake-audio-bytes"))
	}))
	defer cdnSrv.Close()

	tmeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`{
			"code": 0,
			"req_0": {
				"code": 0,
				"data": {
					"midurlinfo": [{"purl": "%s"}],
					"expiration": 3600
				}
			}
		}`, cdnSrv.URL)))
	}))
	defer tmeSrv.Close()

	c := tme.NewClient()
	c.SetBaseURL(tmeSrv.URL)
	sh := NewStreamHandler(c)

	req := httptest.NewRequest("GET", "/api/player/stream/qqmusic:001abc", nil)
	w := httptest.NewRecorder()
	sh.HandleStream(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if body != "fake-audio-bytes" {
		t.Errorf("expected body 'fake-audio-bytes', got '%s'", body)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "audio/mpeg" {
		t.Errorf("expected Content-Type 'audio/mpeg', got '%s'", contentType)
	}

	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl != "public, max-age=3600" {
		t.Errorf("expected Cache-Control 'public, max-age=3600', got '%s'", cacheControl)
	}

	acceptRanges := w.Header().Get("Accept-Ranges")
	if acceptRanges != "bytes" {
		t.Errorf("expected Accept-Ranges 'bytes', got '%s'", acceptRanges)
	}
}

func TestStreamProxy_NotFound(t *testing.T) {
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

	req := httptest.NewRequest("GET", "/api/player/stream/qqmusic:empty", nil)
	w := httptest.NewRecorder()
	sh.HandleStream(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestStreamProxy_RangeHeader(t *testing.T) {
	cdrHeaders := make(chan http.Header, 1)

	cdnSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Header().Set("Content-Range", r.Header.Get("Range"))
		w.Write([]byte("partial-audio"))
		cdrHeaders <- r.Header
	}))
	defer cdnSrv.Close()

	tmeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`{
			"code": 0,
			"req_0": {
				"code": 0,
				"data": {
					"midurlinfo": [{"purl": "%s"}],
					"expiration": 3600
				}
			}
		}`, cdnSrv.URL)))
	}))
	defer tmeSrv.Close()

	c := tme.NewClient()
	c.SetBaseURL(tmeSrv.URL)
	sh := NewStreamHandler(c)

	req := httptest.NewRequest("GET", "/api/player/stream/qqmusic:002xyz", nil)
	req.Header.Set("Range", "bytes=0-99")
	w := httptest.NewRecorder()
	sh.HandleStream(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	forwardedRange := w.Header().Get("Content-Range")
	if forwardedRange != "bytes=0-99" {
		t.Errorf("expected Content-Range 'bytes=0-99', got '%s'", forwardedRange)
	}
}

func TestStreamProxy_CDRError(t *testing.T) {
	cdnSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer cdnSrv.Close()

	tmeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`{
			"code": 0,
			"req_0": {
				"code": 0,
				"data": {
					"midurlinfo": [{"purl": "%s"}],
					"expiration": 3600
				}
			}
		}`, cdnSrv.URL)))
	}))
	defer tmeSrv.Close()

	c := tme.NewClient()
	c.SetBaseURL(tmeSrv.URL)
	sh := NewStreamHandler(c)

	req := httptest.NewRequest("GET", "/api/player/stream/qqmusic:003err", nil)
	w := httptest.NewRecorder()
	sh.HandleStream(w, req)

	if w.Code != 502 {
		t.Fatalf("expected 502, got %d", w.Code)
	}
}

func TestStreamProxy_NoContentLength(t *testing.T) {
	cdnSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// chunked transfer — no Content-Length
		w.Header().Set("Content-Type", "audio/mpeg")
		io.WriteString(w, "streaming-data")
	}))
	defer cdnSrv.Close()

	tmeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`{
			"code": 0,
			"req_0": {
				"code": 0,
				"data": {
					"midurlinfo": [{"purl": "%s"}],
					"expiration": 3600
				}
			}
		}`, cdnSrv.URL)))
	}))
	defer tmeSrv.Close()

	c := tme.NewClient()
	c.SetBaseURL(tmeSrv.URL)
	sh := NewStreamHandler(c)

	req := httptest.NewRequest("GET", "/api/player/stream/qqmusic:004chunked", nil)
	w := httptest.NewRecorder()
	sh.HandleStream(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "streaming-data" {
		t.Errorf("expected 'streaming-data', got '%s'", w.Body.String())
	}
}
