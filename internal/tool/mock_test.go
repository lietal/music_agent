package tool

import (
	"context"
	"testing"
)

func TestMockSearchSongsName(t *testing.T) {
	m := NewMockSearchSongs()
	if m.Name() != "search_songs" {
		t.Errorf("Name() = %q, want %q", m.Name(), "search_songs")
	}
}

func TestMockSearchSongsDescription(t *testing.T) {
	m := NewMockSearchSongs()
	if m.Description() == "" {
		t.Error("Description() returned empty string")
	}
}

func TestMockSearchSongsExecute(t *testing.T) {
	m := NewMockSearchSongs()
	result, err := m.Execute(context.Background(), map[string]any{
		"query": "rock",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Data == "" {
		t.Error("Execute() returned empty data")
	}
	if result.Error != "" {
		t.Errorf("Execute() returned error = %q", result.Error)
	}
}

func TestMockRecommendSongsName(t *testing.T) {
	m := NewMockRecommendSongs()
	if m.Name() != "recommend_songs" {
		t.Errorf("Name() = %q, want %q", m.Name(), "recommend_songs")
	}
}

func TestMockRecommendSongsExecute(t *testing.T) {
	m := NewMockRecommendSongs()
	result, err := m.Execute(context.Background(), map[string]any{
		"song_id": "song-1",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Data == "" {
		t.Error("Execute() returned empty data")
	}
}

func TestMockToolsSatisfyToolInterface(t *testing.T) {
	var _ Tool = NewMockSearchSongs()
	var _ Tool = NewMockRecommendSongs()
}
