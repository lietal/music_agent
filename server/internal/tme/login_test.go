package tme

import (
	"context"
	"net/http"
	"net/http/httptest"
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
					"nickname": "test_user"
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
	if status.MusicKey != "test_key_abc" {
		t.Errorf("got musickey %q", status.MusicKey)
	}
	if status.UserName != "test_user" {
		t.Errorf("got username %q", status.UserName)
	}
}

func TestCheckQRCodeStatus_Pending(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"code": 0,
			"req_0": {
				"code": 0,
				"data": {
					"status": 1
				}
			}
		}`))
	}))
	defer srv.Close()

	c := NewClient()
	c.SetBaseURL(srv.URL)

	status, err := c.CheckQRCodeStatus(context.Background(), "key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Status != "pending" {
		t.Errorf("expected pending, got %s", status.Status)
	}
}
