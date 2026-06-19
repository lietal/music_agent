package agent

import (
	"context"
	"testing"

	"github.com/music-agent/music-agent/internal/event"
	"github.com/music-agent/music-agent/internal/tool"
)

type mockPlanner struct {
	plan TurnPlan
	err  error
}

func (m *mockPlanner) Plan(ctx context.Context, state LoopState) (TurnPlan, error) {
	return m.plan, m.err
}

func (m *mockPlanner) Next(obs tool.Observation) string {
	return FinalAnswer
}

func TestDefaultExecutor_Execute(t *testing.T) {
	exec := NewDefaultExecutor()
	call := ToolCall{ToolName: "search_songs", Args: map[string]any{"keyword": "test"}}
	searchTool := tool.NewMockSearchSongs()

	obs, err := exec.Execute(context.Background(), call, searchTool)
	if err != nil {
		t.Fatal(err)
	}
	if obs.ToolName != "search_songs" {
		t.Errorf("got %s", obs.ToolName)
	}
	if obs.Status != "success" {
		t.Errorf("got %s", obs.Status)
	}
}

func TestAgentLoop_PlanAndExecute(t *testing.T) {
	planner := &mockPlanner{
		plan: TurnPlan{
			Intent:   "search",
			TaskType: "search",
			ToolCalls: []ToolCall{
				{ToolName: "search_songs", Args: map[string]any{"keyword": "test"}},
			},
		},
	}

	tools := map[string]tool.Tool{
		"search_songs": tool.NewMockSearchSongs(),
	}
	loop := NewAgentLoop(planner, NewDefaultExecutor(), tools)

	state := LoopState{
		RunID:         "test-run",
		Goal:          tool.AgentGoal{Intent: "test", TaskType: "search"},
		MaxSteps:      1,
		ExecutedCalls: make(map[string]bool),
	}

	ch := loop.Run(context.Background(), state)

	var types []string
	for evt := range ch {
		types = append(types, evt.Type)
	}

	has := func(typ string) bool {
		for _, t := range types {
			if t == typ {
				return true
			}
		}
		return false
	}
	if !has(event.TypePlan) {
		t.Error("missing plan event")
	}
	if !has(event.TypeToolStart) {
		t.Error("missing tool_start event")
	}
	if !has(event.TypeToolDone) {
		t.Error("missing tool_done event")
	}
	if !has(event.TypeDone) {
		t.Error("missing done event")
	}
}

func TestAgentLoop_Deduplication(t *testing.T) {
	planner := &mockPlanner{
		plan: TurnPlan{
			ToolCalls: []ToolCall{
				{ToolName: "search_songs", Args: map[string]any{"keyword": "test"}},
			},
		},
	}

	tools := map[string]tool.Tool{"search_songs": tool.NewMockSearchSongs()}
	loop := NewAgentLoop(planner, NewDefaultExecutor(), tools)

	executed := make(map[string]bool)
	callKey := callKey(ToolCall{ToolName: "search_songs", Args: map[string]any{"keyword": "test"}})
	executed[callKey] = true

	state := LoopState{
		RunID:         "test-dedup",
		Goal:          tool.AgentGoal{Intent: "test"},
		MaxSteps:      1,
		ExecutedCalls: executed,
	}

	ch := loop.Run(context.Background(), state)
	for evt := range ch {
		if evt.Type == event.TypeToolStart {
			t.Error("should not have tool_start for deduplicated call")
		}
	}
}

func TestAgentLoop_PlannerError(t *testing.T) {
	planner := &mockPlanner{err: fmtError("plan failed")}
	loop := NewAgentLoop(planner, NewDefaultExecutor(), map[string]tool.Tool{})
	state := LoopState{
		RunID:         "test-plan-err",
		Goal:          tool.AgentGoal{Intent: "test"},
		MaxSteps:      1,
		ExecutedCalls: make(map[string]bool),
	}

	ch := loop.Run(context.Background(), state)
	for evt := range ch {
		if evt.Type == event.TypeError {
			return
		}
	}
	t.Error("missing error event")
}

type fmtError string

func (e fmtError) Error() string { return string(e) }
