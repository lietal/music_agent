package agent

import (
	"context"

	"github.com/music-agent/music-agent/internal/tool"
)

type Executor interface {
	Execute(ctx context.Context, call ToolCall, t tool.Tool) (tool.Observation, error)
}
