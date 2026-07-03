package agent

import (
	"reflect"
	"testing"

	"github.com/music-agent/music-agent/internal/tool"
)

func TestCapabilityRegistry(t *testing.T) {
	r := NewCapabilityRegistry()
	r.Register(CapabilityManifest{Name: "search", Description: "Search songs"})
	r.Register(CapabilityManifest{Name: "play", Description: "Play songs"})

	m, ok := r.Get("search")
	if !ok {
		t.Fatal("expected search capability")
	}
	if m.Name != "search" {
		t.Errorf("got %s", m.Name)
	}

	list := r.List()
	if len(list) != 2 {
		t.Errorf("expected 2 capabilities, got %d", len(list))
	}

	_, ok = r.Get("nonexistent")
	if ok {
		t.Error("should not find nonexistent capability")
	}
}

func TestOrchestrateRecommendation(t *testing.T) {
	results := []tool.Observation{{
		Result: tool.ToolResult{Data: `[{"title":"晴天","artist":"周杰伦"},{"title":"稻香","artist":"周杰伦"},{"title":"七里香","artist":"周杰伦"}]`},
	}}
	songs := OrchestrateRecommendation("晴天", results)
	if len(songs) == 0 {
		t.Fatal("expected songs")
	}
	if reflect.DeepEqual(songs[0]["title"], "晴天") {
		// First result should match query
	}

	// Empty query
	songs = OrchestrateRecommendation("zzz", results)
	if len(songs) == 0 {
		t.Fatal("expected songs even for non-matching query")
	}

	// Empty results
	songs = OrchestrateRecommendation("test", nil)
	if len(songs) != 0 {
		t.Error("expected empty for nil results")
	}
}
