package tool

import (
	"context"
	"testing"
)

func TestMockRecommendSongs_Execute(t *testing.T) {
	tool := NewMockRecommendSongs()
	result, err := tool.Execute(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Data == "" {
		t.Error("expected data")
	}
}

func TestMockRecommendSongs_NameDesc(t *testing.T) {
	tool := NewMockRecommendSongs()
	if tool.Name() != "recommend_songs" {
		t.Errorf("got %s", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("empty description")
	}
}

func TestMockSearchSongs_NonJayChou(t *testing.T) {
	tool := NewMockSearchSongs()
	result, err := tool.Execute(context.Background(), map[string]any{"query": "queen"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Data == "" {
		t.Error("expected data")
	}
}

func TestRegistry_FullFlow(t *testing.T) {
	r := NewRegistry()
	r.Register(NewMockSearchSongs())
	r.Register(NewMockRecommendSongs())

	if len(r.List()) != 2 {
		t.Errorf("expected 2, got %d", len(r.List()))
	}

	tool, ok := r.Get("search_songs")
	if !ok {
		t.Fatal("search_songs not found")
	}
	if tool.Name() != "search_songs" {
		t.Errorf("got %s", tool.Name())
	}

	_, ok = r.Get("nonexistent")
	if ok {
		t.Error("expected false")
	}
}

func TestToolResult_ErrorField(t *testing.T) {
	tr := ToolResult{Error: "test error"}
	if tr.Error != "test error" {
		t.Error("bad error")
	}
}
