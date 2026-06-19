package api

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/music-agent/music-agent/internal/tme"
)

func TestGetQRCode_ReturnsDataURL(t *testing.T) {
	h := NewLoginHandler(tme.NewClient(), tme.NewCredentialStore(), []byte("test"), nil)

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
	if resp["qrcode_url"] == nil || resp["qrcode_url"] == "" {
		t.Error("qrcode_url should not be empty")
	}
	if resp["key"] == nil || resp["key"] == "" {
		t.Error("key should not be empty")
	}
	// Verify it's a data URL
	url := resp["qrcode_url"].(string)
	if len(url) < 50 {
		t.Errorf("qrcode_url too short: %d chars", len(url))
	}
}

func TestCheckQRStatus_Pending(t *testing.T) {
	if os.Getenv("E2E_TEST") == "" {
		t.Skip("skipping: requires E2E_TEST=1 (calls ptlogin2.qq.com)")
	}
	h := NewLoginHandler(tme.NewClient(), tme.NewCredentialStore(), []byte("test"), nil)

	req := httptest.NewRequest("GET", "/api/qqmusic/login/status/invalid_key", nil)
	w := httptest.NewRecorder()
	h.HandleCheckQRStatus(w, req)

	// Should return 200 with status "pending" or "expired" for invalid key
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	status := resp["status"].(string)
	if status != "pending" && status != "expired" {
		t.Errorf("expected pending or expired, got %s", status)
	}
}

func TestCheckQRStatus_MissingKey(t *testing.T) {
	h := NewLoginHandler(tme.NewClient(), tme.NewCredentialStore(), []byte("test"), nil)

	req := httptest.NewRequest("GET", "/api/qqmusic/login/status/", nil)
	w := httptest.NewRecorder()
	h.HandleCheckQRStatus(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for empty key, got %d", w.Code)
	}
}

func TestGetStatus_NotLoggedIn(t *testing.T) {
	creds := tme.NewCredentialStore()
	h := NewLoginHandler(tme.NewClient(), creds, []byte("test"), nil)

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

func TestGetStatus_LoggedIn(t *testing.T) {
	creds := tme.NewCredentialStore()
	creds.Set("123", "key")
	h := NewLoginHandler(tme.NewClient(), creds, []byte("test"), nil)

	req := httptest.NewRequest("GET", "/api/qqmusic/login/status", nil)
	w := httptest.NewRecorder()
	h.HandleGetStatus(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["logged_in"] != true {
		t.Errorf("expected logged_in true, got %v", resp["logged_in"])
	}
}
