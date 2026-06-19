package tme

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHash33(t *testing.T) {
	result := hash33("test")
	if result <= 0 {
		t.Errorf("hash33 returned non-positive: %d", result)
	}
	if hash33("test") != result {
		t.Error("hash33 is not deterministic")
	}
}

func TestNormalizeUin(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"o12345", "12345"}, {"O67890", "67890"}, {"12345", "12345"}, {"", ""},
	}
	for _, tt := range tests {
		got := normalizeUin(tt.input)
		if got != tt.expected {
			t.Errorf("normalizeUin(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestSplitCSV(t *testing.T) {
	input := "'0','0','https://example.com','','','',''"
	parts := splitCSV(input)
	if len(parts) < 3 {
		t.Fatalf("splitCSV returned %d parts, want at least 3", len(parts))
	}
}

func TestExtractParam(t *testing.T) {
	url := "https://example.com?uin=12345&ptsigx=abc123"
	if extractParam(url, "uin") != "12345" {
		t.Error("extractParam failed for uin")
	}
	if extractParam(url, "missing") != "" {
		t.Error("extractParam should return empty for missing key")
	}
	if extractParam("", "uin") != "" {
		t.Error("extractParam should return empty for empty URL")
	}
}

func TestHash33WithSeed(t *testing.T) {
	result := hash33WithSeed("test", 5381)
	if result <= 0 {
		t.Error("hash33WithSeed returned non-positive")
	}
	if hash33WithSeed("test", 5381) != result {
		t.Error("hash33WithSeed not deterministic")
	}
}

func TestRandomUI(t *testing.T) {
	ui1 := randomUI()
	ui2 := randomUI()
	if ui1 == "" || ui2 == "" {
		t.Error("randomUI returned empty")
	}
	if ui1 == ui2 {
		t.Error("randomUI should generate different values")
	}
}

func mockLoginServer(t *testing.T, qrsig string, alwaysConfirmed bool) *httptest.Server {
	t.Helper()
	pollCount := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "ptqrshow"):
			w.Header().Set("Set-Cookie", fmt.Sprintf("qrsig=%s; Path=/; Secure", qrsig))
			w.Header().Set("Content-Type", "image/png")
			w.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})

		case strings.Contains(r.URL.Path, "ptqrlogin"):
			w.Header().Set("Content-Type", "text/html")
			pollCount++
			if alwaysConfirmed || pollCount > 1 {
				w.Write([]byte(`ptuiCB('0','0','https://ptlogin2.qq.com/redirect?uin=o123456&ptsigx=fake_sigx','','','','')`))
			} else {
				w.Write([]byte(`ptuiCB('66','0','','','','','')`))
			}

		case strings.Contains(r.URL.Path, "check_sig"):
			w.Header().Set("Set-Cookie", "p_skey=fake_pskey; Path=/")
			w.WriteHeader(200)

		case strings.Contains(r.URL.Path, "authorize") && r.Method == "POST":
			w.Header().Set("Location", "https://y.qq.com?code=auth_code_123")
			w.WriteHeader(302)
		}
	}))
}

func mockTmeServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"code": 0,
			"music.login.LoginServer.Login": {
				"code": 0,
				"data": {
					"musicid": 123456,
					"musickey": "mock_musickey",
					"openid": "mock_openid",
					"unionid": "mock_unionid",
					"nickname": "MockUser",
					"headurl": "https://avatar.example.com/mock.jpg"
				}
			}
		}`))
	}))
}

func TestGetLoginQRCode_Mocked(t *testing.T) {
	srv := mockLoginServer(t, "test_qrsig", false)
	defer srv.Close()

	c := NewClient()
	c.SetLoginBaseURL(srv.URL)

	qr, err := c.GetLoginQRCode(context.Background())
	if err != nil {
		t.Fatalf("GetLoginQRCode: %v", err)
	}
	if !strings.HasPrefix(qr.QrcodeDataURL, "data:image/png;base64,") {
		t.Error("QrcodeDataURL should be a data URL")
	}
	if qr.Key != "test_qrsig" {
		t.Errorf("key = %q, want test_qrsig", qr.Key)
	}
}

func TestGetLoginQRCode_NoCookie(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte{0x89, 0x50})
	}))
	defer srv.Close()

	c := NewClient()
	c.SetLoginBaseURL(srv.URL)

	_, err := c.GetLoginQRCode(context.Background())
	if err == nil {
		t.Error("expected error when no qrsig cookie")
	}
}

func TestCheckQRCodeStatus_Pending(t *testing.T) {
	srv := mockLoginServer(t, "test_qrsig", false)
	defer srv.Close()

	c := NewClient()
	c.SetLoginBaseURL(srv.URL)

	status, err := c.CheckQRCodeStatus(context.Background(), "test_qrsig")
	if err != nil {
		t.Fatalf("CheckQRCodeStatus: %v", err)
	}
	if status.Status != "pending" {
		t.Errorf("expected pending, got %s", status.Status)
	}
}

func TestCheckQRCodeStatus_Confirmed(t *testing.T) {
	loginSrv := mockLoginServer(t, "confirmed_sig", true)
	defer loginSrv.Close()
	tmeSrv := mockTmeServer(t)
	defer tmeSrv.Close()

	c := NewClient()
	c.SetLoginBaseURL(loginSrv.URL)
	c.SetBaseURL(tmeSrv.URL)

	status, err := c.CheckQRCodeStatus(context.Background(), "confirmed_sig")
	if err != nil {
		t.Fatalf("CheckQRCodeStatus: %v", err)
	}
	if status.Status != "confirmed" {
		t.Errorf("expected confirmed, got %s", status.Status)
	}
	if status.MusicID != "123456" {
		t.Errorf("music_id = %q, want 123456", status.MusicID)
	}
	if status.MusicKey != "mock_musickey" {
		t.Errorf("music_key = %q", status.MusicKey)
	}
	if status.OpenID != "mock_openid" {
		t.Errorf("openid = %q", status.OpenID)
	}
	if status.UserName != "MockUser" {
		t.Errorf("username = %q", status.UserName)
	}
}

func TestCheckQRCodeStatus_Expired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "ptqrlogin") {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`ptuiCB('65','0','','','','','')`))
		}
	}))
	defer srv.Close()

	c := NewClient()
	c.SetLoginBaseURL(srv.URL)

	status, err := c.CheckQRCodeStatus(context.Background(), "expired_sig")
	if err != nil {
		t.Fatalf("CheckQRCodeStatus: %v", err)
	}
	if status.Status != "expired" {
		t.Errorf("expected expired, got %s", status.Status)
	}
}

func TestCheckQRCodeStatus_Scanned(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "ptqrlogin") {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`ptuiCB('67','0','','','','','')`))
		}
	}))
	defer srv.Close()

	c := NewClient()
	c.SetLoginBaseURL(srv.URL)

	status, err := c.CheckQRCodeStatus(context.Background(), "scanned_sig")
	if err != nil {
		t.Fatalf("CheckQRCodeStatus: %v", err)
	}
	if status.Status != "scanned" {
		t.Errorf("expected scanned, got %s", status.Status)
	}
}
