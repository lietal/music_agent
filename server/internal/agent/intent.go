package agent

import (
	"context"

	"github.com/music-agent/music-agent/internal/llm"
	"github.com/music-agent/music-agent/internal/tool"
)

type Intent struct {
	Type   string         `json:"type"`
	Query  string         `json:"query"`
	Params map[string]any `json:"params"`
}

type IntentResult struct {
	Intent Intent `json:"intent"`
	Output any    `json:"output"`
	Error  string `json:"error,omitempty"`
}

type PipelineContext struct {
	UserMessage  string             `json:"user_message"`
	UserID       string             `json:"user_id"`
	Intents      []Intent           `json:"intents"`
	Results      []IntentResult     `json:"results"`
	Observations []tool.Observation `json:"observations"`
}

type AgentTurnPlanner interface {
	Plan(ctx context.Context, message string, history []llm.Message) ([]Intent, error)
}

var intentTools = map[string][]string{
	"search_music":    {"search_songs"},
	"recommend_music": {"recommend_songs"},
	"playlist_write":  {"create_playlist", "add_to_playlist", "remove_song", "rename_playlist", "delete_playlist"},
	"playlist_read":   {"list_playlists", "get_playlist", "play_playlist"},
	"chat":            {},
}

func ToolsForIntent(intentType string) []string {
	if tools, ok := intentTools[intentType]; ok {
		return tools
	}
	return nil
}
