package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const maxRetries = 2

type OpenAIClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewOpenAI(baseURL, apiKey string, httpClient *http.Client) *OpenAIClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &OpenAIClient{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: httpClient,
	}
}

func (c *OpenAIClient) chatURL() string {
	if strings.HasSuffix(c.baseURL, "/v1") {
		return c.baseURL + "/chat/completions"
	}
	return c.baseURL + "/v1/chat/completions"
}

func (c *OpenAIClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url_ := c.chatURL()

	var lastStatus int
	for attempt := 0; attempt <= maxRetries; attempt++ {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url_, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
		httpReq.Header.Set("Content-Type", "application/json")

		httpResp, err := c.httpClient.Do(httpReq)
		if err != nil {
			return nil, err
		}

		if httpResp.StatusCode >= 500 {
			httpResp.Body.Close()
			lastStatus = httpResp.StatusCode
			continue
		}

		defer httpResp.Body.Close()

		if httpResp.StatusCode >= 400 {
			return nil, fmt.Errorf("client error: %d", httpResp.StatusCode)
		}

		var resp ChatResponse
		if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}
		return &resp, nil
	}

	return nil, fmt.Errorf("server error %d after %d retries", lastStatus, maxRetries+1)
}

func (c *OpenAIClient) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, <-chan error) {
	chunkCh := make(chan StreamChunk)
	errCh := make(chan error, 1)

	go func() {
		defer close(chunkCh)
		defer close(errCh)

		req.Stream = true

		body, err := json.Marshal(req)
		if err != nil {
			errCh <- fmt.Errorf("marshal request: %w", err)
			return
		}

	url_ := c.chatURL()
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url_, bytes.NewReader(body))
		if err != nil {
			errCh <- fmt.Errorf("create request: %w", err)
			return
		}
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "text/event-stream")

		httpResp, err := c.httpClient.Do(httpReq)
		if err != nil {
			errCh <- err
			return
		}
		defer httpResp.Body.Close()

		if httpResp.StatusCode >= 400 {
			errCh <- fmt.Errorf("server error: %d", httpResp.StatusCode)
			return
		}

		scanner := bufio.NewScanner(httpResp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				chunkCh <- StreamChunk{Done: true}
				return
			}

			var sseResp struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}
			if err := json.Unmarshal([]byte(data), &sseResp); err != nil {
				errCh <- fmt.Errorf("parse SSE data: %w", err)
				return
			}

			delta := ""
			if len(sseResp.Choices) > 0 {
				delta = sseResp.Choices[0].Delta.Content
			}

			chunkCh <- StreamChunk{Delta: delta}
		}

		if err := scanner.Err(); err != nil {
			errCh <- err
		}
	}()

	return chunkCh, errCh
}
