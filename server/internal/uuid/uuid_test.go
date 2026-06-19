package uuid

import "testing"

func TestNew(t *testing.T) {
	id := New()
	if id == "" {
		t.Error("expected non-empty UUID")
	}
	if len(id) != 36 {
		t.Errorf("expected 36 chars UUID, got %d: %s", len(id), id)
	}
}

func TestNew_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := New()
		if seen[id] {
			t.Fatal("duplicate UUID generated:", id)
		}
		seen[id] = true
	}
}
