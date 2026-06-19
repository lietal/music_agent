package event

import "encoding/json"

type Event struct {
	Type  string          `json:"type"`
	RunID string          `json:"run_id"`
	Data  json.RawMessage `json:"data,omitempty"`
}

const (
	TypePlan      = "plan"
	TypeToolStart = "tool_start"
	TypeToolDone  = "tool_done"
	TypeDelta     = "delta"
	TypeDone      = "done"
	TypeError     = "error"
)
