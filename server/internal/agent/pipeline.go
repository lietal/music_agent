package agent

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/music-agent/music-agent/internal/event"
	"github.com/music-agent/music-agent/internal/llm"
	"github.com/music-agent/music-agent/internal/tool"
)

type AgentPipeline struct {
	turnPlanner AgentTurnPlanner
	client      llm.Client
	model       string
	prompts     Prompts
	allTools    map[string]tool.Tool
	executor    Executor
	maxSteps    int
}

func NewAgentPipeline(
	turnPlanner AgentTurnPlanner,
	client llm.Client,
	model string,
	prompts Prompts,
	allTools map[string]tool.Tool,
	executor Executor,
	maxSteps int,
) *AgentPipeline {
	if maxSteps <= 0 {
		maxSteps = 5
	}
	return &AgentPipeline{
		turnPlanner: turnPlanner,
		client:      client,
		model:       model,
		prompts:     prompts,
		allTools:    allTools,
		executor:    executor,
		maxSteps:    maxSteps,
	}
}

func (p *AgentPipeline) Run(ctx context.Context, state LoopState) <-chan event.Event {
	ch := make(chan event.Event, 64)

	go func() {
		defer close(ch)
		defer func() {
			if r := recover(); r != nil {
				errData, _ := json.Marshal(map[string]string{"message": "panic: " + anyToString(r)})
				emit(ch, event.Event{Type: event.TypeError, RunID: state.RunID, Data: errData})
			}
		}()

		intents, _ := p.turnPlanner.Plan(ctx, state.Goal.Intent, state.MessageHistory)
		if len(intents) == 0 {
			intents = []Intent{{Type: "chat", Query: state.Goal.Intent}}
		}

		intentsJSON, _ := json.Marshal(intents)
		turnMsgs := append([]llm.Message{{Role: "system", Content: p.prompts.IntentRouter}}, state.MessageHistory...)
		turnMsgs = append(turnMsgs, llm.Message{Role: "user", Content: state.Goal.Intent})
		planData, _ := json.Marshal(map[string]any{
			"intents":       intents,
			"plan":          string(intentsJSON),
			"system_prompt": p.prompts.IntentRouter,
			"user_message":  state.Goal.Intent,
			"history":       messagesToText(state.MessageHistory),
			"messages":      turnMsgs,
		})
		emit(ch, event.Event{Type: event.TypePlan, RunID: state.RunID, Data: planData})

		prompt := buildPrompt(p.prompts.ReactThink, p.allTools, string(intentsJSON))

		var observations []tool.Observation
		confirmed := false
		for step := 0; step < p.maxSteps; step++ {
			if ctx.Err() != nil {
				return
			}

			obsText := observationsToText(observations)
			contextText := "Context:\n- User intents: " + string(intentsJSON) + "\n- Current observations: " + obsText + "\n\nUser Input:\n" + state.Goal.Intent

			messages := append([]llm.Message{{Role: "system", Content: prompt}}, state.MessageHistory...)
			messages = append(messages, llm.Message{Role: "user", Content: contextText})

			stepCtx, cancel := context.WithTimeout(ctx, 60*time.Second)

			resp, err := p.client.Chat(stepCtx, llm.ChatRequest{
				Model:    p.model,
				Messages: messages,
			})
			cancel()
			if err != nil {
				continue
			}
			if len(resp.Choices) == 0 {
				continue
			}

			var action reactAction
			content := trimJSON(resp.Choices[0].Message.Content)
			if json.Unmarshal([]byte(content), &action) != nil {
				continue
			}

			stepData, _ := json.Marshal(map[string]any{
				"stepType":     action.StepType,
				"toolName":     action.ToolName,
				"message":      action.Message,
				"systemPrompt": prompt,
				"userMessage":  contextText,
				"messages":     messages,
			})
			emit(ch, event.Event{Type: event.TypeStep, RunID: state.RunID, Data: stepData})

			if action.StepType == "FINAL_ANSWER" || action.StepType == "confirmation" {
				if action.StepType == "confirmation" {
					confirmed = true
				}
				if action.Message != "" {
					msgData, _ := json.Marshal(map[string]string{"message": action.Message})
					emit(ch, event.Event{Type: event.TypeDelta, RunID: state.RunID, Data: msgData})
				}
				break
			}

			if action.StepType != "tool_call" || action.ToolName == "" {
				continue
			}

			t, ok := p.allTools[action.ToolName]
			if !ok {
				continue
			}

			startData, _ := json.Marshal(map[string]any{
				"name":  action.ToolName,
				"input": action.Args,
				"args":  action.Args,
			})
			emit(ch, event.Event{Type: event.TypeToolStart, RunID: state.RunID, Data: startData})

			call := ToolCall{ToolName: action.ToolName, Args: action.Args}
			obs, err := p.executor.Execute(ctx, call, t)
			if err != nil {
				continue
			}
			observations = append(observations, obs)

			doneData, _ := json.Marshal(map[string]any{
				"name":   action.ToolName,
				"output": obs.Result.Data,
				"result": obs.Result,
			})
			emit(ch, event.Event{Type: event.TypeToolDone, RunID: state.RunID, Data: doneData})
		}

		if !confirmed {
			p.agentAnswerPlanner(ctx, state, intents, observations, ch)
		}
		emit(ch, event.Event{Type: event.TypeDone, RunID: state.RunID})
	}()

	return ch
}

func (p *AgentPipeline) agentAnswerPlanner(ctx context.Context, state LoopState, intents []Intent, observations []tool.Observation, ch chan<- event.Event) {
	intentsJSON, _ := json.Marshal(intents)
	obsJSON, _ := json.Marshal(observations)
	answerPrompt := replacePlaceholders(p.prompts.AnswerGen, map[string]string{
		"{user_message}":   state.Goal.Intent,
		"{intents}":        string(intentsJSON),
		"{intent_results}": string(obsJSON),
	})

	messages := append([]llm.Message{{Role: "system", Content: answerPrompt}}, state.MessageHistory...)

	resp, err := p.client.Chat(ctx, llm.ChatRequest{
		Model:    p.model,
		Messages: messages,
	})
	if err != nil {
		return
	}
	if len(resp.Choices) > 0 {
		deltaData, _ := json.Marshal(map[string]string{"message": resp.Choices[0].Message.Content})
		emit(ch, event.Event{Type: event.TypeDelta, RunID: state.RunID, Data: deltaData})
	}
}

func buildPrompt(template string, tools map[string]tool.Tool, intentsJSON string) string {
	toolList := ""
	for name, t := range tools {
		toolList += "- " + name + ": " + t.Description() + "\n"
	}
	return replacePlaceholders(template, map[string]string{
		"{tools}":   toolList,
		"{intents}": intentsJSON,
	})
}

func replacePlaceholders(template string, replacements map[string]string) string {
	result := template
	for k, v := range replacements {
		result = strings.ReplaceAll(result, k, v)
	}
	return result
}

func observationsToText(obs []tool.Observation) string {
	if len(obs) == 0 {
		return "(no observations yet)"
	}
	data, _ := json.Marshal(obs)
	return string(data)
}

func anyToString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	data, _ := json.Marshal(v)
	return string(data)
}

func messagesToText(msgs []llm.Message) string {
	if len(msgs) == 0 {
		return "(none)"
	}
	var result string
	for _, m := range msgs {
		result += "[" + m.Role + "]\n" + m.Content + "\n\n"
	}
	return result
}
