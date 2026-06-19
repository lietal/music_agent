package tme

import (
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
