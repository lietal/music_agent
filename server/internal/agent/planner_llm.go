package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/music-agent/music-agent/internal/llm"
	"github.com/music-agent/music-agent/internal/tool"
)

type llmPlanner struct {
	client  llm.Client
	model   string
	tools   map[string]tool.Tool
	sysMsg  string
}

func NewLLMPlanner(client llm.Client, model string, tools map[string]tool.Tool) Planner {
	sb := &strings.Builder{}
	sb.WriteString("You are a music search and recommendation assistant. ")
	sb.WriteString("Given a user's music request, you decide what tools to call or answer directly.\n\n")
	sb.WriteString("Available tools:\n")
	for _, t := range tools {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", t.Name(), t.Description()))
	}
	sb.WriteString("\nRespond with JSON only, no markdown:\n")
	sb.WriteString(`{"intent":"<search|recommend|answer>","taskType":"<tool name or answer>","toolCalls":[{"toolName":"<name>","args":{"key":"value"}}],"next":"CONTINUE|FINAL_ANSWER"}`)
	sb.WriteString("\n\nIf you can answer directly without tools, use empty toolCalls and next=FINAL_ANSWER.")

	return &llmPlanner{
		client: client,
		model:  model,
		tools:  tools,
		sysMsg: sb.String(),
	}
}

func (p *llmPlanner) Plan(ctx context.Context, state LoopState) (TurnPlan, error) {
	messages := []llm.Message{
		{Role: "system", Content: p.sysMsg},
		{Role: "user", Content: p.buildUserPrompt(state)},
	}

	resp, err := p.client.Chat(ctx, llm.ChatRequest{
		Model:    p.model,
		Messages: messages,
	})
	if err != nil {
		return TurnPlan{}, fmt.Errorf("llm chat: %w", err)
	}

	if len(resp.Choices) == 0 {
		return TurnPlan{}, fmt.Errorf("empty response from LLM")
	}

	content := resp.Choices[0].Message.Content
	content = extractJSON(content)

	var result struct {
		Intent    string     `json:"intent"`
		TaskType  string     `json:"taskType"`
		ToolCalls []ToolCall `json:"toolCalls"`
		Next      string     `json:"next"`
	}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return TurnPlan{}, fmt.Errorf("parse plan json: %w (content: %s)", err, content)
	}

	plan := TurnPlan{
		Intent:    result.Intent,
		TaskType:  result.TaskType,
		ToolCalls: result.ToolCalls,
	}

	return plan, nil
}

func (p *llmPlanner) Next(obs tool.Observation) string {
	if obs.Status == "success" {
		return FinalAnswer
	}
	return Continue
}

func (p *llmPlanner) buildUserPrompt(state LoopState) string {
	var sb strings.Builder
	sb.WriteString("User request: ")
	sb.WriteString(state.Goal.Intent)

	if len(state.Observations) > 0 {
		sb.WriteString("\n\nPrevious results:\n")
		for i, obs := range state.Observations {
			sb.WriteString(fmt.Sprintf("%d. Tool %s result: %s\n", i+1, obs.ToolName, obs.Result.Data))
		}
		sb.WriteString("\nBased on these results, do you need more tools or is this enough to answer?")
	}

	return sb.String()
}

func (p *llmPlanner) GenerateAnswer(ctx context.Context, state LoopState) (<-chan string, error) {
	var sb strings.Builder
	sb.WriteString("你是一个音乐推荐助手。根据搜索结果给用户一个简短友好的推荐回复。\n\n")
	sb.WriteString(fmt.Sprintf("用户问：%s\n\n", state.Goal.Intent))

	if len(state.Observations) > 0 {
		sb.WriteString("搜索结果：\n")
		for _, obs := range state.Observations {
			sb.WriteString(obs.Result.Data)
			sb.WriteString("\n")
		}
	}

	messages := []llm.Message{
		{Role: "system", Content: sb.String()},
		{Role: "user", Content: "生成推荐回复"},
	}

	chunkCh, errCh := p.client.ChatStream(ctx, llm.ChatRequest{
		Model:    p.model,
		Messages: messages,
	})

	out := make(chan string, 64)
	go func() {
		defer close(out)
		for {
			select {
			case chunk, ok := <-chunkCh:
				if !ok {
					return
				}
				if chunk.Done {
					return
				}
				out <- chunk.Delta
			case err, ok := <-errCh:
				if ok && err != nil {
					out <- fmt.Sprintf("[生成回复出错: %v]", err)
					return
				}
				if !ok {
					return
				}
			case <-ctx.Done():
				out <- "[请求超时]"
				return
			}
		}
	}()

	return out, nil
}

func extractJSON(content string) string {
	content = strings.TrimSpace(content)
	if idx := strings.Index(content, "```json"); idx != -1 {
		content = content[idx+7:]
		if end := strings.LastIndex(content, "```"); end != -1 {
			content = content[:end]
		}
		content = strings.TrimSpace(content)
	}
	if idx := strings.Index(content, "```"); idx != -1 {
		content = strings.TrimSpace(content[:idx])
	}
	return content
}
