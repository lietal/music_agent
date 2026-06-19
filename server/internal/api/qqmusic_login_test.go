package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/music-agent/music-agent/internal/tme"
)

func TestGetQRCode_Success(t *testing.T) {
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

	c := tme.NewClient()
	c.SetBaseURL(srv.URL)
	creds := tme.NewCredentialStore()
	h := NewLoginHandler(c, creds)

	req := httptest.NewRequest("POST", "/api/qqmusic/login/qrcode", nil)
	w := httptest.NewRecorder()
	h.HandleGetQRCode(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["qrcode_url"] != "https://qrcode.example.com/abc" {
		t.Errorf("got qrcode_url %v", resp["qrcode_url"])
	}
	if resp["key"] != "key_abc123" {
		t.Errorf("got key %v", resp["key"])
	}
}

func TestCheckQRStatus_Confirmed(t *testing.T) {
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
					"nickname": "test_user"
				}
			}
		}`))
	}))
	defer srv.Close()

	c := tme.NewClient()
	c.SetBaseURL(srv.URL)
	creds := tme.NewCredentialStore()
	h := NewLoginHandler(c, creds)

	req := httptest.NewRequest("GET", "/api/qqmusic/login/status/key_abc123", nil)
	w := httptest.NewRecorder()
	h.HandleCheckQRStatus(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["status"] != "confirmed" {
		t.Errorf("got status %v", resp["status"])
	}
	if resp["user_name"] != "test_user" {
		t.Errorf("got user_name %v", resp["user_name"])
	}
	if !creds.IsLoggedIn() {
		t.Error("expected creds to be set")
	}
	mid, mk := creds.Get()
	if mid != "12345" {
		t.Errorf("got musicid %q", mid)
	}
	if mk != "test_key_abc" {
		t.Errorf("got musickey %q", mk)
	}
}

func TestGetStatus_NotLoggedIn(t *testing.T) {
	creds := tme.NewCredentialStore()
	h := NewLoginHandler(tme.NewClient(), creds)

	req := httptest.NewRequest("GET", "/api/qqmusic/login/status", nil)
	w := httptest.NewRecorder()
	h.HandleGetStatus(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["logged_in"] != false {
		t.Errorf("got logged_in %v", resp["logged_in"])
	}
}
