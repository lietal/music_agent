package tool

import (
	"context"
	"testing"
)

func TestToolInterfaceSatisfaction(t *testing.T) {
	var _ Tool = (*testTool)(nil)
}

func TestToolResultStruct(t *testing.T) {
	result := ToolResult{
		Data:  "some data",
		Error: "some error",
	}

	if result.Data != "some data" {
		t.Errorf("ToolResult.Data = %q, want %q", result.Data, "some data")
	}
	if result.Error != "some error" {
		t.Errorf("ToolResult.Error = %q, want %q", result.Error, "some error")
	}
}

func TestObservationStruct(t *testing.T) {
	result := ToolResult{Data: "result data"}
	obs := Observation{
		ToolName: "search_songs",
		Args: map[string]any{
			"query": "rock",
			"limit": 5,
		},
		Status: "success",
		Result: result,
	}

	if obs.ToolName != "search_songs" {
		t.Errorf("Observation.ToolName = %q, want %q", obs.ToolName, "search_songs")
	}
	if obs.Args["query"] != "rock" {
		t.Errorf("Observation.Args[query] = %v, want %v", obs.Args["query"], "rock")
	}
	if obs.Args["limit"] != 5 {
		t.Errorf("Observation.Args[limit] = %v, want %v", obs.Args["limit"], 5)
	}
	if obs.Status != "success" {
		t.Errorf("Observation.Status = %q, want %q", obs.Status, "success")
	}
	if obs.Result.Data != "result data" {
		t.Errorf("Observation.Result.Data = %q, want %q", obs.Result.Data, "result data")
	}
}

func TestAgentGoalStruct(t *testing.T) {
	goal := AgentGoal{
		Intent:   "Find rock songs from the 80s",
		TaskType: "search_and_recommend",
	}

	if goal.Intent != "Find rock songs from the 80s" {
		t.Errorf("AgentGoal.Intent = %q, want %q", goal.Intent, "Find rock songs from the 80s")
	}
	if goal.TaskType != "search_and_recommend" {
		t.Errorf("AgentGoal.TaskType = %q, want %q", goal.TaskType, "search_and_recommend")
	}
}

type testTool struct{}

func (t testTool) Name() string                               { return "test" }
func (t testTool) Description() string                        { return "a test tool" }
func (t testTool) Execute(ctx context.Context, args map[string]any) (ToolResult, error) {
	return ToolResult{}, nil
}
