package tme

import "testing"

func TestCredentialStoreInitialState(t *testing.T) {
	cs := NewCredentialStore()
	if cs.IsLoggedIn() {
		t.Error("expected IsLoggedIn to be false initially")
	}
}

func TestCredentialStoreSetGet(t *testing.T) {
	cs := NewCredentialStore()
	cs.Set("mid-123", "key-abc")

	if !cs.IsLoggedIn() {
		t.Error("expected IsLoggedIn to be true after Set")
	}

	musicid, musickey := cs.Get()
	if musicid != "mid-123" {
		t.Errorf("expected musicid 'mid-123', got '%s'", musicid)
	}
	if musickey != "key-abc" {
		t.Errorf("expected musickey 'key-abc', got '%s'", musickey)
	}
}

func TestCredentialStoreClear(t *testing.T) {
	cs := NewCredentialStore()
	cs.Set("mid-123", "key-abc")
	cs.Clear()

	if cs.IsLoggedIn() {
		t.Error("expected IsLoggedIn to be false after Clear")
	}

	musicid, musickey := cs.Get()
	if musicid != "" {
		t.Errorf("expected empty musicid after Clear, got '%s'", musicid)
	}
	if musickey != "" {
		t.Errorf("expected empty musickey after Clear, got '%s'", musickey)
	}
}
