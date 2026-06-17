package tool

import (
	"context"
	"testing"
)

func TestRegistryRegisterAndGet(t *testing.T) {
	reg := NewRegistry()

	t1 := &testNamedTool{name: "search_songs"}
	t2 := &testNamedTool{name: "recommend_songs"}

	reg.Register(t1)
	reg.Register(t2)

	got, ok := reg.Get("search_songs")
	if !ok {
		t.Fatal("expected search_songs to be registered")
	}
	if got.Name() != "search_songs" {
		t.Errorf("got.Name() = %q, want %q", got.Name(), "search_songs")
	}

	got, ok = reg.Get("recommend_songs")
	if !ok {
		t.Fatal("expected recommend_songs to be registered")
	}
	if got.Name() != "recommend_songs" {
		t.Errorf("got.Name() = %q, want %q", got.Name(), "recommend_songs")
	}
}

func TestRegistryGetMissing(t *testing.T) {
	reg := NewRegistry()

	_, ok := reg.Get("nonexistent")
	if ok {
		t.Error("expected false for missing tool, got true")
	}
}

func TestRegistryList(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&testNamedTool{name: "a"})
	reg.Register(&testNamedTool{name: "b"})

	names := reg.List()
	if len(names) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(names))
	}
}

type testNamedTool struct {
	name string
}

func (t *testNamedTool) Name() string                                    { return t.name }
func (t *testNamedTool) Description() string                             { return "test" }
func (t *testNamedTool) Execute(ctx context.Context, args map[string]any) (ToolResult, error) {
	return ToolResult{}, nil
}
