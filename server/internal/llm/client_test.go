package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestChat_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Authorization: Bearer test-key, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}

		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if req.Model != "test-model" {
			t.Errorf("expected model test-model, got %s", req.Model)
		}
		if len(req.Messages) != 1 || req.Messages[0].Role != "user" || req.Messages[0].Content != "Hi" {
			t.Errorf("unexpected messages: %+v", req.Messages)
		}

		resp := ChatResponse{
			Choices: []Choice{
				{Message: Message{Role: "assistant", Content: "Hello!"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAI(server.URL, "test-key", nil)

	req := ChatRequest{
		Model:    "test-model",
		Messages: []Message{{Role: "user", Content: "Hi"}},
	}
	resp, err := client.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(resp.Choices))
	}
	if resp.Choices[0].Message.Content != "Hello!" {
		t.Fatalf("expected 'Hello!', got '%s'", resp.Choices[0].Message.Content)
	}
}

func TestChat_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewOpenAI(server.URL, "test-key", nil)

	ctx, cancel := context.WithCancel(context.Background())
	req := ChatRequest{
		Model:    "test-model",
		Messages: []Message{{Role: "user", Content: "Hi"}},
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := client.Chat(ctx, req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !isContextCanceledError(err) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func TestChat_RetriesOn5xx(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		resp := ChatResponse{
			Choices: []Choice{
				{Message: Message{Role: "assistant", Content: "OK!"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAI(server.URL, "test-key", nil)

	req := ChatRequest{
		Model:    "test-model",
		Messages: []Message{{Role: "user", Content: "Hi"}},
	}
	resp, err := client.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
	if resp.Choices[0].Message.Content != "OK!" {
		t.Fatalf("expected 'OK!', got '%s'", resp.Choices[0].Message.Content)
	}
}

func TestChat_DoesNotRetryOn4xx(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewOpenAI(server.URL, "test-key", nil)

	req := ChatRequest{
		Model:    "test-model",
		Messages: []Message{{Role: "user", Content: "Hi"}},
	}
	_, err := client.Chat(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt (no retry on 4xx), got %d", attempts)
	}
}

func TestChatStream_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Errorf("expected Accept: text/event-stream, got %s", r.Header.Get("Accept"))
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected http.Flusher")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		chunks := []struct {
			delta string
			done  bool
		}{
			{delta: "Hello", done: false},
			{delta: " world", done: false},
			{delta: "", done: true},
		}

		for _, ch := range chunks {
			data := map[string]interface{}{
				"choices": []map[string]interface{}{
					{"delta": map[string]interface{}{"content": ch.delta}},
				},
			}
			jsonData, _ := json.Marshal(data)
			fmt.Fprintf(w, "data: %s\n\n", jsonData)
			flusher.Flush()
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	client := NewOpenAI(server.URL, "test-key", nil)

	req := ChatRequest{
		Model:    "test-model",
		Messages: []Message{{Role: "user", Content: "Hi"}},
	}

	chunks, errs := client.ChatStream(context.Background(), req)

	var deltas []string
	var streamDone bool

loop:
	for {
		select {
		case chunk, ok := <-chunks:
			if !ok {
				break loop
			}
			deltas = append(deltas, chunk.Delta)
			if chunk.Done {
				streamDone = true
			}
		case err := <-errs:
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}
	}

	combined := strings.Join(deltas, "")
	if combined != "Hello world" {
		t.Fatalf("expected 'Hello world', got '%s'", combined)
	}
	if !streamDone {
		t.Fatal("expected final chunk with Done=true")
	}
}

func TestChatStream_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected http.Flusher")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		for i := 0; i < 10; i++ {
			data := map[string]interface{}{
				"choices": []map[string]interface{}{
					{"delta": map[string]interface{}{"content": "."}},
				},
			}
			jsonData, _ := json.Marshal(data)
			fmt.Fprintf(w, "data: %s\n\n", jsonData)
			flusher.Flush()
			time.Sleep(100 * time.Millisecond)
		}
	}))
	defer server.Close()

	client := NewOpenAI(server.URL, "test-key", nil)

	ctx, cancel := context.WithCancel(context.Background())
	req := ChatRequest{
		Model:    "test-model",
		Messages: []Message{{Role: "user", Content: "Hi"}},
	}

	chunks, errs := client.ChatStream(ctx, req)

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	var gotErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case _, ok := <-chunks:
				if !ok {
					return
				}
			case err := <-errs:
				if err != nil {
					gotErr = err
				}
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for stream to close")
	}

	if gotErr == nil {
		t.Fatal("expected error from context cancellation, got nil")
	}
	if !isContextCanceledError(gotErr) {
		t.Fatalf("expected context.Canceled, got: %v", gotErr)
	}
}

func isContextCanceledError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, context.Canceled.Error()) || strings.Contains(msg, "canceled")
}
