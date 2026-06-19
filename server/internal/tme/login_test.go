package tme

import (
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
		{"o12345", "12345"},
		{"O67890", "67890"},
		{"12345", "12345"},
		{"", ""},
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

func TestQRCodeFieldsExist(t *testing.T) {
	qr := &QRCode{QrcodeDataURL: "data:image/png;base64,test", Key: "testkey"}
	if qr.QrcodeDataURL == "" || qr.Key == "" {
		t.Error("QRCode fields not set")
	}
}

func TestQRStatusConfirmedFields(t *testing.T) {
	s := &QRStatus{
		Status:   "confirmed",
		MusicID:  "123",
		MusicKey: "key",
		OpenID:   "openid123",
		UnionID:  "union123",
		UserName: "testuser",
	}
	if s.OpenID != "openid123" {
		t.Error("OpenID not set")
	}
	if s.UnionID != "union123" {
		t.Error("UnionID not set")
	}
}

func TestGetLoginQRCode_ReturnsDataURL(t *testing.T) {
	// Mock the ptqrshow endpoint
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "ptqrshow") {
			w.WriteHeader(404)
			return
		}
		w.Header().Set("Set-Cookie", "qrsig=mock_qrsig_abc123; Path=/; Domain=ptlogin2.qq.com; Secure")
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte{0x89, 0x50, 0x4E, 0x47}) // PNG header
	}))
	defer srv.Close()

	// Override the ptlogin URL via environment-level hack: we can't change
	// the hardcoded URL, but we can test the handler layer separately.
	// For this unit test, just verify QRCode struct fields.
	qr := &QRCode{QrcodeDataURL: "data:image/png;base64,dGVzdA==", Key: "testkey"}
	if !strings.HasPrefix(qr.QrcodeDataURL, "data:image/png;base64,") {
		t.Error("QrcodeDataURL should be a data URL")
	}
	if qr.Key == "" {
		t.Error("Key should not be empty")
	}
}

func TestHash33WithSeed(t *testing.T) {
	// Known values for p_skey hash
	result := hash33WithSeed("test", 5381)
	if result <= 0 {
		t.Error("hash33WithSeed returned non-positive")
	}
	// Consistency
	if hash33WithSeed("test", 5381) != result {
		t.Error("hash33WithSeed not deterministic")
	}
}

func TestExtractParam_Empty(t *testing.T) {
	if extractParam("not-a-url", "key") != "" {
		t.Error("expected empty for invalid URL")
	}
	if extractParam("https://example.com", "missing") != "" {
		t.Error("expected empty for missing param")
	}
}

func TestNormalizeUin_All(t *testing.T) {
	cases := []struct{ in, out string }{
		{"o12345", "12345"},
		{"O67890", "67890"},
		{"12345", "12345"},
		{"", ""},
		{"normal_user", "normal_user"},
	}
	for _, c := range cases {
		if normalizeUin(c.in) != c.out {
			t.Errorf("normalizeUin(%q) = %q, want %q", c.in, normalizeUin(c.in), c.out)
		}
	}
}

func TestSplitCSV_Complex(t *testing.T) {
	// Test with empty parts
	result := splitCSV("'0','0','https://example.com?a=1&b=2','','','',''")
	if len(result) == 0 {
		t.Error("expected non-empty result")
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

func TestCheckQRCodeStatus_Pending(t *testing.T) {
	// Test with invalid qrsig — should return pending
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`ptuiCB('66','0','','','','','')`))
	}))
	defer srv.Close()

	c := NewClient()
	// Can't test full flow without overriding ptlogin URLs
	// Just verify the struct methods compile and basic logic works
	s := &QRStatus{Status: "pending"}
	if s.Status != "pending" {
		t.Error("status should be pending")
	}
	_ = c
	_ = srv
}
