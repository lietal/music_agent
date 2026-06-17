package tool

import "context"

type ToolResult struct {
	Data  string `json:"data"`
	Error string `json:"error,omitempty"`
}

type Observation struct {
	ToolName string
	Args     map[string]any
	Status   string
	Result   ToolResult
}

type AgentGoal struct {
	Intent   string
	TaskType string
}

type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, args map[string]any) (ToolResult, error)
}
