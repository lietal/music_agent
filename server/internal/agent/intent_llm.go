package agent

import (
	"context"
	"encoding/json"

	"github.com/music-agent/music-agent/internal/llm"
)

type llmTurnPlanner struct {
	client llm.Client
	model  string
	prompt string
}

func NewLLMTurnPlanner(client llm.Client, model string, prompt string) AgentTurnPlanner {
	return &llmTurnPlanner{
		client: client,
		model:  model,
		prompt: prompt,
	}
}

func (p *llmTurnPlanner) Plan(ctx context.Context, message string, history []llm.Message) ([]Intent, error) {
	messages := append([]llm.Message{{Role: "system", Content: p.prompt}}, history...)
	messages = append(messages, llm.Message{Role: "user", Content: message})

	resp, err := p.client.Chat(ctx, llm.ChatRequest{
		Model:    p.model,
		Messages: messages,
	})
	if err != nil {
		return []Intent{{Type: "chat", Query: message}}, nil
	}

	if len(resp.Choices) == 0 {
		return []Intent{{Type: "chat", Query: message}}, nil
	}

	content := resp.Choices[0].Message.Content
	var intents []Intent
	if err := json.Unmarshal([]byte(content), &intents); err != nil {
		return []Intent{{Type: "chat", Query: message}}, nil
	}

	if len(intents) == 0 {
		return []Intent{{Type: "chat", Query: message}}, nil
	}

	for i := range intents {
		if intents[i].Params == nil {
			intents[i].Params = map[string]any{}
		}
	}

	return intents, nil
}
