package agent

import (
	"testing"

	"github.com/music-agent/music-agent/internal/tool"
)

func TestLoopStateStruct(t *testing.T) {
	state := LoopState{
		RunID:  "run-1",
		UserID: "user-1",
		Goal: tool.AgentGoal{
			Intent:   "Find rock songs",
			TaskType: "search_and_recommend",
		},
		RequiredOutcomes: []string{"songs_found", "recommendations_made"},
		MaxSteps:         10,
		CurrentStep:      0,
	}

	if state.RunID != "run-1" {
		t.Errorf("RunID = %q, want %q", state.RunID, "run-1")
	}
	if state.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", state.UserID, "user-1")
	}
	if state.Goal.Intent != "Find rock songs" {
		t.Errorf("Goal.Intent = %q, want %q", state.Goal.Intent, "Find rock songs")
	}
	if len(state.RequiredOutcomes) != 2 {
		t.Errorf("len(RequiredOutcomes) = %d, want 2", len(state.RequiredOutcomes))
	}
	if state.MaxSteps != 10 {
		t.Errorf("MaxSteps = %d, want 10", state.MaxSteps)
	}
}

func TestWithObservationReturnsNewStruct(t *testing.T) {
	original := LoopState{
		RunID:  "run-1",
		UserID: "user-1",
		Goal: tool.AgentGoal{
			Intent:   "Find songs",
			TaskType: "search",
		},
		Observations:  []tool.Observation{},
		MaxSteps:      10,
		CurrentStep:   0,
	}

	obs := tool.Observation{
		ToolName: "search_songs",
		Args:     map[string]any{"query": "rock"},
		Status:   "success",
		Result:   tool.ToolResult{Data: `[{"title":"Song 1"}]`},
	}

	newState := original.WithObservation(obs)

	if len(original.Observations) != 0 {
		t.Error("original.Observations should still be empty (immutability violation)")
	}

	if len(newState.Observations) != 1 {
		t.Fatalf("len(newState.Observations) = %d, want 1", len(newState.Observations))
	}

	if newState.Observations[0].ToolName != "search_songs" {
		t.Errorf("newState.Observations[0].ToolName = %q, want %q",
			newState.Observations[0].ToolName, "search_songs")
	}
}

func TestWithObservationPreservesOtherFields(t *testing.T) {
	original := LoopState{
		RunID:   "run-1",
		UserID:  "user-1",
		Goal:    tool.AgentGoal{Intent: "Find songs", TaskType: "search"},
		MaxSteps: 10,
		CurrentStep: 3,
		ExecutedCalls: map[string]bool{"search_songs:hash1": true},
		RequiredOutcomes: []string{"found"},
	}

	obs := tool.Observation{
		ToolName: "recommend_songs",
		Args:     map[string]any{"song_id": "song-1"},
		Status:   "success",
		Result:   tool.ToolResult{Data: "recs"},
	}

	newState := original.WithObservation(obs)

	if newState.RunID != "run-1" {
		t.Errorf("RunID = %q, want %q", newState.RunID, "run-1")
	}
	if newState.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", newState.UserID, "user-1")
	}
	if newState.Goal.Intent != "Find songs" {
		t.Errorf("Goal.Intent = %q, want %q", newState.Goal.Intent, "Find songs")
	}
	if newState.MaxSteps != 10 {
		t.Errorf("MaxSteps = %d, want 10", newState.MaxSteps)
	}
	if newState.CurrentStep != 3 {
		t.Errorf("CurrentStep = %d, want 3", newState.CurrentStep)
	}
	if !newState.ExecutedCalls["search_songs:hash1"] {
		t.Error("ExecutedCalls should still contain search_songs:hash1")
	}
	if len(newState.RequiredOutcomes) != 1 {
		t.Errorf("len(RequiredOutcomes) = %d, want 1", len(newState.RequiredOutcomes))
	}
}

func TestMultipleObservations(t *testing.T) {
	state := LoopState{
		RunID:  "run-1",
		UserID: "user-1",
		Goal:   tool.AgentGoal{Intent: "Find songs", TaskType: "search"},
		MaxSteps: 10,
	}

	obs1 := tool.Observation{
		ToolName: "search_songs",
		Status:   "success",
		Result:   tool.ToolResult{Data: "found 3 songs"},
	}
	obs2 := tool.Observation{
		ToolName: "recommend_songs",
		Status:   "success",
		Result:   tool.ToolResult{Data: "3 recommendations"},
	}

	state = state.WithObservation(obs1)
	state = state.WithObservation(obs2)

	if len(state.Observations) != 2 {
		t.Fatalf("len(Observations) = %d, want 2", len(state.Observations))
	}
	if state.Observations[0].ToolName != "search_songs" {
		t.Errorf("Observations[0].ToolName = %q, want %q", state.Observations[0].ToolName, "search_songs")
	}
	if state.Observations[1].ToolName != "recommend_songs" {
		t.Errorf("Observations[1].ToolName = %q, want %q", state.Observations[1].ToolName, "recommend_songs")
	}
}
