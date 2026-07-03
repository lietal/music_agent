package agent

import (
	"github.com/music-agent/music-agent/internal/llm"
	"github.com/music-agent/music-agent/internal/tool"
)

type LoopState struct {
	RunID            string
	UserID           string
	Goal             tool.AgentGoal
	Observations     []tool.Observation
	MessageHistory   []llm.Message
	ExecutedCalls    map[string]bool
	RequiredOutcomes []string
	MaxSteps         int
	CurrentStep      int
}

func (s LoopState) WithObservation(obs tool.Observation) LoopState {
	next := s
	next.Observations = make([]tool.Observation, len(s.Observations), len(s.Observations)+1)
	copy(next.Observations, s.Observations)
	next.Observations = append(next.Observations, obs)
	return next
}
