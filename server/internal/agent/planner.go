package agent

import (
	"context"

	"github.com/music-agent/music-agent/internal/tool"
)

type ToolCall struct {
	ToolName string
	Args     map[string]any
}

type TurnPlan struct {
	Intent    string
	TaskType  string
	ToolCalls []ToolCall
}

type AnswerGenerator interface {
	GenerateAnswer(ctx context.Context, state LoopState) (<-chan string, error)
}

const (
	FinalAnswer = "FINAL_ANSWER"
	Continue    = "CONTINUE"
)

type Planner interface {
	Plan(ctx context.Context, state LoopState) (TurnPlan, error)
	Next(obs tool.Observation) string
}
