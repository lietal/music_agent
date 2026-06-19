package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/music-agent/music-agent/internal/event"
	"github.com/music-agent/music-agent/internal/tool"
)

type countingPlanner struct {
	plan         TurnPlan
	nextDecision string
	planCalls    int
}

func (m *countingPlanner) Plan(ctx context.Context, state LoopState) (TurnPlan, error) {
	m.planCalls++
	return m.plan, nil
}

func (m *countingPlanner) Next(obs tool.Observation) string {
	return m.nextDecision
}

type fixedExecutor struct {
	result tool.ToolResult
}

func (e *fixedExecutor) Execute(ctx context.Context, call ToolCall, t tool.Tool) (tool.Observation, error) {
	result, execErr := t.Execute(ctx, call.Args)
	status := "success"
	if execErr != nil || result.Error != "" {
		status = "error"
	}
	return tool.Observation{
		ToolName: call.ToolName,
		Args:     call.Args,
		Status:   status,
		Result:   result,
	}, nil
}

type panicPlanner struct{}

func (p *panicPlanner) Plan(ctx context.Context, state LoopState) (TurnPlan, error) {
	panic("planner panic")
}

func (p *panicPlanner) Next(obs tool.Observation) string {
	return FinalAnswer
}

func TestMaxStepsLimit(t *testing.T) {
	tools := map[string]tool.Tool{
		"search_songs": tool.NewMockSearchSongs(),
	}

	executor := &fixedExecutor{result: tool.ToolResult{Data: `[{"id":"song-1"}]`}}
	planner := &countingPlanner{
		plan: TurnPlan{
			Intent:   "Find songs",
			TaskType: "search",
			ToolCalls: []ToolCall{
				{ToolName: "search_songs", Args: map[string]any{"query": "rock"}},
			},
		},
		nextDecision: Continue,
	}

	loop := NewAgentLoop(planner, executor, tools)

	state := LoopState{
		RunID:    "run-1",
		UserID:   "user-1",
		MaxSteps: 3,
		Goal:     tool.AgentGoal{Intent: "Find songs", TaskType: "search"},
	}

	ctx := context.Background()
	events := loop.Run(ctx, state)

	var lastEvent event.Event
	stepCount := 0
	for evt := range events {
		lastEvent = evt
		if evt.Type == event.TypePlan {
			stepCount++
		}
	}

	if stepCount > state.MaxSteps {
		t.Errorf("stepCount = %d, want <= %d", stepCount, state.MaxSteps)
	}

	if lastEvent.Type != event.TypeDone {
		t.Errorf("last event type = %q, want %q", lastEvent.Type, event.TypeDone)
	}
}

func TestMaxStepsHardLimitForcesCompose(t *testing.T) {
	tools := map[string]tool.Tool{
		"search_songs": tool.NewMockSearchSongs(),
	}

	executor := &fixedExecutor{result: tool.ToolResult{Data: `[]`}}
	planner := &countingPlanner{
		plan: TurnPlan{
			Intent:   "Find songs",
			TaskType: "search",
			ToolCalls: []ToolCall{
				{ToolName: "search_songs", Args: map[string]any{"query": "rock"}},
			},
		},
		nextDecision: Continue,
	}

	loop := NewAgentLoop(planner, executor, tools)

	state := LoopState{
		RunID:    "run-max",
		MaxSteps: 5,
		Goal:     tool.AgentGoal{Intent: "Find songs", TaskType: "search"},
	}

	ctx := context.Background()
	events := loop.Run(ctx, state)

	var doneEvent *event.Event
	for evt := range events {
		e := evt
		if e.Type == event.TypeDone {
			doneEvent = &e
			break
		}
	}

	if doneEvent == nil {
		t.Error("expected DoneEvent when max steps reached")
	}

	if planner.planCalls > state.MaxSteps {
		t.Errorf("planCalls = %d, want <= %d", planner.planCalls, state.MaxSteps)
	}
}

func TestContextCancellation(t *testing.T) {
	tools := map[string]tool.Tool{
		"search_songs": tool.NewMockSearchSongs(),
	}

	executor := &fixedExecutor{result: tool.ToolResult{Data: `[]`}}
	planner := &countingPlanner{
		plan: TurnPlan{
			Intent:   "Find songs",
			TaskType: "search",
			ToolCalls: []ToolCall{
				{ToolName: "search_songs", Args: map[string]any{"query": "rock"}},
			},
		},
		nextDecision: Continue,
	}

	loop := NewAgentLoop(planner, executor, tools)

	state := LoopState{
		RunID:    "run-cancel",
		MaxSteps: 10,
		Goal:     tool.AgentGoal{Intent: "Find songs", TaskType: "search"},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	events := loop.Run(ctx, state)

	eventCount := 0
	for range events {
		eventCount++
	}

	if eventCount > 1 {
		t.Errorf("expected at most 1 event for cancelled context, got %d", eventCount)
	}
}

func TestToolErrorProducesErrorEvent(t *testing.T) {
	tools := map[string]tool.Tool{
		"error_tool": &errorTool{},
	}

	executor := &fixedExecutor{}
	planner := &countingPlanner{
		plan: TurnPlan{
			Intent:   "Test",
			TaskType: "search",
			ToolCalls: []ToolCall{
				{ToolName: "error_tool", Args: map[string]any{}},
			},
		},
		nextDecision: FinalAnswer,
	}

	loop := NewAgentLoop(planner, executor, tools)

	state := LoopState{
		RunID:    "run-error",
		MaxSteps: 3,
		Goal:     tool.AgentGoal{Intent: "Test", TaskType: "search"},
	}

	ctx := context.Background()
	events := loop.Run(ctx, state)

	hasError := false
	for evt := range events {
		if evt.Type == event.TypeError {
			hasError = true
		}
	}

	if !hasError {
		t.Error("expected error event when tool returns error")
	}
}

func TestPanicRecovery(t *testing.T) {
	tools := map[string]tool.Tool{
		"search_songs": tool.NewMockSearchSongs(),
	}

	executor := &fixedExecutor{result: tool.ToolResult{Data: `[]`}}
	planner := &panicPlanner{}

	loop := NewAgentLoop(planner, executor, tools)

	state := LoopState{
		RunID:    "run-panic",
		MaxSteps: 3,
		Goal:     tool.AgentGoal{Intent: "Find songs", TaskType: "search"},
	}

	ctx := context.Background()
	events := loop.Run(ctx, state)

	var errorEvent *event.Event
	for evt := range events {
		e := evt
		if e.Type == event.TypeError {
			errorEvent = &e
		}
	}

	if errorEvent == nil {
		t.Error("expected error event from panic recovery")
	}

	if errorEvent != nil {
		var errData map[string]string
		if err := json.Unmarshal(errorEvent.Data, &errData); err != nil {
			t.Errorf("error event data should be valid JSON: %v", err)
		}
		if errData["message"] == "" {
			t.Error("error event should contain a message")
		}
	}
}

func TestDeduplicationSkipsExecutedCalls(t *testing.T) {
	tools := map[string]tool.Tool{
		"search_songs": tool.NewMockSearchSongs(),
	}

	executor := &fixedExecutor{result: tool.ToolResult{Data: `[]`}}
	planner := &countingPlanner{
		plan: TurnPlan{
			Intent:   "Find songs",
			TaskType: "search",
			ToolCalls: []ToolCall{
				{ToolName: "search_songs", Args: map[string]any{"query": "rock"}},
			},
		},
		nextDecision: FinalAnswer,
	}

	loop := NewAgentLoop(planner, executor, tools)

	state := LoopState{
		RunID:    "run-dedup",
		MaxSteps: 3,
		Goal:     tool.AgentGoal{Intent: "Find songs", TaskType: "search"},
		ExecutedCalls: map[string]bool{
			`search_songs:{"query":"rock"}`: true,
		},
	}

	ctx := context.Background()
	events := loop.Run(ctx, state)

	toolStartCount := 0
	for evt := range events {
		if evt.Type == event.TypeToolStart {
			toolStartCount++
		}
	}

	if toolStartCount > 0 {
		t.Errorf("expected 0 tool starts for deduplicated call, got %d", toolStartCount)
	}
}

func TestImmutabilityOfState(t *testing.T) {
	tools := map[string]tool.Tool{
		"search_songs": tool.NewMockSearchSongs(),
	}

	executor := &fixedExecutor{result: tool.ToolResult{Data: `[{"id":"song-1"}]`}}
	planner := &countingPlanner{
		plan: TurnPlan{
			Intent:   "Find songs",
			TaskType: "search",
			ToolCalls: []ToolCall{
				{ToolName: "search_songs", Args: map[string]any{"query": "rock"}},
			},
		},
		nextDecision: FinalAnswer,
	}

	loop := NewAgentLoop(planner, executor, tools)

	originalState := LoopState{
		RunID:    "run-immutable",
		MaxSteps: 3,
		Goal:     tool.AgentGoal{Intent: "Find songs", TaskType: "search"},
	}

	ctx := context.Background()
	for range loop.Run(ctx, originalState) {
	}

	if len(originalState.Observations) != 0 {
		t.Error("original state should not be mutated after Run")
	}
}

type errorTool struct{}

func (e *errorTool) Name() string        { return "error_tool" }
func (e *errorTool) Description() string { return "always returns an error" }
func (e *errorTool) Execute(ctx context.Context, args map[string]any) (tool.ToolResult, error) {
	return tool.ToolResult{Error: "tool failed"}, nil
}
