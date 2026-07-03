package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/music-agent/music-agent/internal/event"
	"github.com/music-agent/music-agent/internal/llm"
	"github.com/music-agent/music-agent/internal/tool"
)

type ReActLoop struct {
	client  llm.Client
	model   string
	prompt  string
	tools   map[string]tool.Tool
	exec    Executor
	maxStep int
}

func NewReActLoop(client llm.Client, model string, prompt string, tools map[string]tool.Tool, exec Executor, maxSteps int) *ReActLoop {
	if maxSteps <= 0 {
		maxSteps = 5
	}
	return &ReActLoop{
		client:  client,
		model:   model,
		prompt:  prompt,
		tools:   tools,
		exec:    exec,
		maxStep: maxSteps,
	}
}

type reactAction struct {
	StepType string         `json:"stepType"`
	ToolName string         `json:"toolName,omitempty"`
	Args     map[string]any `json:"args,omitempty"`
	Message  string         `json:"message,omitempty"`
}

func (r *ReActLoop) Run(ctx context.Context, state LoopState, ch chan<- event.Event) ([]tool.Observation, error) {
	var observations []tool.Observation

	for step := 0; step < r.maxStep; step++ {
		if ctx.Err() != nil {
			return observations, ctx.Err()
		}

		toolsDesc := buildToolsDesc(r.tools)
		obsDesc := buildObsDesc(observations)
		reactPrompt := strings.Replace(r.prompt, "{tools}", toolsDesc, 1)
		reactPrompt = strings.Replace(reactPrompt, "{observations}", obsDesc, 1)

		resp, err := r.client.Chat(ctx, llm.ChatRequest{
			Model: r.model,
			Messages: []llm.Message{
				{Role: "system", Content: reactPrompt},
				{Role: "user", Content: state.Goal.Intent},
			},
		})
		if err != nil {
			return observations, fmt.Errorf("react think: %w", err)
		}

		if len(resp.Choices) == 0 {
			return observations, fmt.Errorf("empty react response")
		}

		var action reactAction
		content := trimJSON(resp.Choices[0].Message.Content)
		if err := json.Unmarshal([]byte(content), &action); err != nil {
			return observations, nil
		}

		if action.StepType == "FINAL_ANSWER" || action.StepType == "confirmation" {
			return observations, nil
		}

		t, ok := r.tools[action.ToolName]
		if !ok {
			continue
		}

		call := ToolCall{
			ToolName: action.ToolName,
			Args:     action.Args,
		}

		startData, _ := json.Marshal(call)
		emit(ch, event.Event{Type: event.TypeToolStart, RunID: state.RunID, Data: startData})

		obs, err := r.exec.Execute(ctx, call, t)
		if err != nil {
			return observations, fmt.Errorf("exec: %w", err)
		}

		observations = append(observations, obs)

		doneData, _ := json.Marshal(obs)
		emit(ch, event.Event{Type: event.TypeToolDone, RunID: state.RunID, Data: doneData})

		state = state.WithObservation(obs)

		stepCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		_ = stepCtx
		cancel()
	}

	return observations, nil
}

func buildToolsDesc(tools map[string]tool.Tool) string {
	var b strings.Builder
	for name, t := range tools {
		b.WriteString(fmt.Sprintf("- %s: %s\n", name, t.Description()))
	}
	return b.String()
}

func buildObsDesc(observations []tool.Observation) string {
	if len(observations) == 0 {
		return "No observations yet."
	}
	var b strings.Builder
	for i, o := range observations {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, o.Result.Data))
	}
	return b.String()
}

func trimJSON(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == '{' || s[i] == '[' {
			s = s[i:]
			break
		}
	}
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '}' || s[i] == ']' {
			s = s[:i+1]
			break
		}
	}
	return s
}

func emit(ch chan<- event.Event, evt event.Event) {
	select {
	case ch <- evt:
	default:
	}
}
