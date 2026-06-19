package agent

import (
	"context"
	"encoding/json"

	"github.com/music-agent/music-agent/internal/tool"
)

type defaultExecutor struct{}

func NewDefaultExecutor() Executor {
	return &defaultExecutor{}
}

func (e *defaultExecutor) Execute(ctx context.Context, call ToolCall, t tool.Tool) (tool.Observation, error) {
	result, err := t.Execute(ctx, call.Args)
	if err != nil {
		return tool.Observation{}, err
	}

	argsJSON, _ := json.Marshal(call.Args)
	obs := tool.Observation{
		ToolName: call.ToolName,
		Args:     call.Args,
		Status:   "success",
		Result:   result,
	}

	_ = argsJSON
	return obs, nil
}
