package agent

import (
	"context"
	"testing"

	"github.com/music-agent/music-agent/internal/llm"
	"github.com/music-agent/music-agent/internal/tool"
)

type mockLLM struct {
	response string
}

func (m *mockLLM) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{
		Choices: []llm.Choice{
			{Message: llm.Message{Role: "assistant", Content: m.response}},
		},
	}, nil
}

func (m *mockLLM) ChatStream(ctx context.Context, req llm.ChatRequest) (<-chan llm.StreamChunk, <-chan error) {
	ch := make(chan llm.StreamChunk, 2)
	errCh := make(chan error, 1)
	go func() {
		ch <- llm.StreamChunk{Delta: "streamed response"}
		ch <- llm.StreamChunk{Done: true}
		close(ch)
	}()
	return ch, errCh
}

func TestLLMPlanner_Plan(t *testing.T) {
	mock := &mockLLM{
		response: `{"intent":"search","taskType":"search","toolCalls":[{"toolName":"search_songs","args":{"keyword":"周杰伦"}}],"next":"CONTINUE"}`,
	}
	tools := map[string]tool.Tool{
		"search_songs": tool.NewMockSearchSongs(),
	}
	planner := NewLLMPlanner(mock, "test-model", tools)

	state := LoopState{
		Goal: tool.AgentGoal{Intent: "周杰伦的歌", TaskType: "chat"},
	}

	plan, err := planner.Plan(context.Background(), state)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Intent != "search" {
		t.Errorf("got %s", plan.Intent)
	}
	if len(plan.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(plan.ToolCalls))
	}
	if plan.ToolCalls[0].ToolName != "search_songs" {
		t.Errorf("got %s", plan.ToolCalls[0].ToolName)
	}
}

func TestLLMPlanner_Next(t *testing.T) {
	mock := &mockLLM{}
	tools := map[string]tool.Tool{"search_songs": tool.NewMockSearchSongs()}
	planner := NewLLMPlanner(mock, "test", tools)

	n := planner.Next(tool.Observation{Status: "success"})
	if n != FinalAnswer {
		t.Errorf("expected FinalAnswer, got %s", n)
	}

	n = planner.Next(tool.Observation{Status: "error"})
	if n != Continue {
		t.Errorf("expected Continue, got %s", n)
	}
}

func TestLLMPlanner_GenerateAnswer(t *testing.T) {
	mock := &mockLLM{}
	tools := map[string]tool.Tool{"search_songs": tool.NewMockSearchSongs()}
	planner := NewLLMPlanner(mock, "test", tools)

	state := LoopState{
		Goal: tool.AgentGoal{Intent: "周杰伦的歌"},
		Observations: []tool.Observation{
			{ToolName: "search_songs", Status: "success", Result: tool.ToolResult{Data: `[{"title":"晴天"}]`}},
		},
	}

	gen, ok := planner.(AnswerGenerator)
	if !ok {
		t.Fatal("planner does not implement AnswerGenerator")
	}
	ch, err := gen.GenerateAnswer(context.Background(), state)
	if err != nil {
		t.Fatal(err)
	}
	result := ""
	for text := range ch {
		result += text
	}
	if result != "streamed response" {
		t.Errorf("got %s", result)
	}
}
