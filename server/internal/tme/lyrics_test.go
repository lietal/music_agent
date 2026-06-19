package tme

import (
	"testing"
)

func TestDecryptLyrics(t *testing.T) {
	_, err := decryptLyrics("")
	if err == nil {
		t.Error("expected error for empty string")
	}
}

func TestDecryptLyrics_InvalidHex(t *testing.T) {
	_, err := decryptLyrics("not-hex!")
	if err == nil {
		t.Error("expected error")
	}
}

func TestStripLRC(t *testing.T) {
	input := "[00:15.50]雨下整夜\n[00:20.00]我的爱溢出就像雨水"
	got := stripLRCTimestamps(input)
	if got != "雨下整夜\n我的爱溢出就像雨水" {
		t.Errorf("got %q", got)
	}
}

func TestStripLRC_Empty(t *testing.T) {
	if s := stripLRCTimestamps(""); s != "" {
		t.Errorf("expected empty, got %q", s)
	}
}

func TestTrimLRC(t *testing.T) {
	if s := trimLRC("[00:15.50]hello"); s != "hello" {
		t.Errorf("got %q", s)
	}
	if s := trimLRC("no bracket"); s != "no bracket" {
		t.Errorf("got %q", s)
	}
}

func TestPKCS5Unpad(t *testing.T) {
	data := []byte("hello\x03\x03\x03")
	if got := pkcs5Unpad(data); string(got) != "hello" {
		t.Errorf("got %q", string(got))
	}
}
