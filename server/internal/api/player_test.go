package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/music-agent/music-agent/internal/tme"
)

func TestGetPlayURL_Success(t *testing.T) {
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

	c := tme.NewClient()
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
	if resp["url"] != "https://isure.stream.qqmusic.qq.com/C400001abc.m4a" {
		t.Errorf("unexpected url: %v", resp["url"])
	}
	if resp["source"] != "cdn" {
		t.Errorf("expected source cdn, got %v", resp["source"])
	}
}

func TestGetPlayURL_NoURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer srv.Close()

	c := tme.NewClient()
	c.SetBaseURL(srv.URL)
	ph := NewPlayerHandler(c, tme.NewCredentialStore())

	req := httptest.NewRequest("GET", "/api/player/url/qqmusic:empty", nil)
	w := httptest.NewRecorder()
	ph.HandleGetPlayURL(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["source"] != "unavailable" {
		t.Errorf("expected source unavailable, got %v", resp["source"])
	}
}

func TestGetLyrics_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"code": 0,
			"req_0": {
				"code": 0,
				"data": {
					"lyric": "WzAwOjAwLjAwXeWkqeawr+eahOWwveWktOaYr+mjjuaygAo=",
					"trans": ""
				}
			}
		}`))
	}))
	defer srv.Close()

	c := tme.NewClient()
	c.SetBaseURL(srv.URL)
	ph := NewPlayerHandler(c, tme.NewCredentialStore())

	req := httptest.NewRequest("GET", "/api/player/lyrics/qqmusic%3A001abc", nil)
	w := httptest.NewRecorder()
	ph.HandleGetLyrics(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	// The lyrics will be decoded base64, so we just verify it's valid JSON
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
}

func TestGetPlayURL_MissingSongID(t *testing.T) {
	ph := NewPlayerHandler(tme.NewClient(), tme.NewCredentialStore())
	req := httptest.NewRequest("GET", "/api/player/url/", nil)
	w := httptest.NewRecorder()
	ph.HandleGetPlayURL(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetPlayURL_TMEError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	c := tme.NewClient()
	c.SetBaseURL(srv.URL)
	ph := NewPlayerHandler(c, tme.NewCredentialStore())

	req := httptest.NewRequest("GET", "/api/player/url/qqmusic:bad", nil)
	w := httptest.NewRecorder()
	ph.HandleGetPlayURL(w, req)

	if w.Code != 500 {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
