package tool

import (
	"context"
	"testing"
)

func TestTMESearchSongs_Name(t *testing.T) {
	tool := &TMESearchSongs{}
	if tool.Name() != "search_songs" {
		t.Errorf("got %s", tool.Name())
	}
}

func TestTMESearchSongs_Description(t *testing.T) {
	tool := &TMESearchSongs{}
	if tool.Description() == "" {
		t.Error("empty description")
	}
}

func TestTMESearchSongs_NewTMESearchSongs(t *testing.T) {
	tool := NewTMESearchSongs()
	if tool == nil {
		t.Fatal("expected non-nil")
	}
	if tool.client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestTMESearchSongs_Execute_MissingKeyword(t *testing.T) {
	tool := NewTMESearchSongs()
	_, err := tool.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTMESearchSongs_Execute_InvalidLimit(t *testing.T) {
	tool := NewTMESearchSongs()
	_, _ = tool.Execute(context.Background(), map[string]any{"keyword": "test", "limit": "not-a-number"})
}

func TestTMESearchSongs_IsAvailable(t *testing.T) {
	tool := NewTMESearchSongs()
	ctx, cancel := context.WithTimeout(context.Background(), 1)
	defer cancel()
	available := tool.IsAvailable(ctx)
	_ = available
}

func TestTMESearchSongs_QueryFallback(t *testing.T) {
	tool := NewTMESearchSongs()
	_, _ = tool.Execute(context.Background(), map[string]any{"query": "test"})
}
