package integration

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/music-agent/music-agent/internal/api"
	"github.com/music-agent/music-agent/internal/auth"
	"github.com/music-agent/music-agent/internal/event"
)

func newTestServer() (*httptest.Server, []byte) {
	bus := event.NewBus()
	jwtSecret := []byte("integration-test-secret")
	handler := api.NewHandler(bus, jwtSecret, nil)
	router := api.NewRouter(handler)
	return httptest.NewServer(router), jwtSecret
}

func generateToken(secret []byte) string {
	token, err := auth.GenerateToken("test-user", "test", secret)
	if err != nil {
		panic(fmt.Sprintf("failed to generate token: %v", err))
	}
	return token
}

func TestFullChatFlow(t *testing.T) {
	ts, jwtSecret := newTestServer()
	defer ts.Close()

	token := generateToken(jwtSecret)

	runID := createChat(t, ts.URL, token)

	events := connectSSE(t, ts.URL, runID, token)

	verifyEventSequence(t, events)
}

func TestUnauthorizedNoToken(t *testing.T) {
	ts, _ := newTestServer()
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/api/chat", "application/json", strings.NewReader(`{"message":"test"}`))
	if err != nil {
		t.Fatalf("failed to POST /api/chat: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestUnauthorizedInvalidToken(t *testing.T) {
	ts, _ := newTestServer()
	defer ts.Close()

	req, _ := http.NewRequest("POST", ts.URL+"/api/chat", strings.NewReader(`{"message":"test"}`))
	req.Header.Set("Authorization", "Bearer invalid-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to POST /api/chat: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestSSEUnauthorizedNoToken(t *testing.T) {
	ts, _ := newTestServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/chat/test-run/events")
	if err != nil {
		t.Fatalf("failed to GET SSE: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func createChat(t *testing.T, baseURL, token string) string {
	t.Helper()

	reqBody := strings.NewReader(`{"message":"test message"}`)
	req, err := http.NewRequest("POST", baseURL+"/api/chat", reqBody)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to POST /api/chat: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	runID := body["runId"]
	if runID == "" {
		t.Fatal("expected non-empty run_id in response")
	}

	return runID
}

func connectSSE(t *testing.T, baseURL, runID, token string) []event.Event {
	t.Helper()

	url := fmt.Sprintf("%s/api/chat/%s/events?token=%s", baseURL, runID, token)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("failed to connect SSE: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for SSE, got %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Errorf("expected text/event-stream Content-Type, got %q", ct)
	}

	var events []event.Event
	scanner := bufio.NewScanner(resp.Body)
	var currentType string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "event: ") {
			currentType = strings.TrimPrefix(line, "event: ")
		}
		if strings.HasPrefix(line, "data: ") {
			payload := strings.TrimPrefix(line, "data: ")

			evt := event.Event{
				Type: currentType,
				Data: json.RawMessage(payload),
			}

			events = append(events, evt)

			if evt.Type == event.TypeDone || evt.Type == event.TypeError {
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		t.Logf("SSE scanner error: %v", err)
	}

	return events
}

func verifyEventSequence(t *testing.T, events []event.Event) {
	t.Helper()

	if len(events) == 0 {
		t.Fatal("expected at least one SSE event")
	}

	expected := []string{
		event.TypePlan,
		event.TypeToolStart,
		event.TypeToolDone,
		event.TypeDelta,
		event.TypeDone,
	}

	if len(events) != len(expected) {
		t.Errorf("expected %d events, got %d", len(expected), len(events))
	}

	minLen := len(events)
	if len(expected) < minLen {
		minLen = len(expected)
	}

	for i := 0; i < minLen; i++ {
		if events[i].Type != expected[i] {
			t.Errorf("event[%d]: expected %q, got %q", i, expected[i], events[i].Type)
		}
	}

	for i, evt := range events {
		switch evt.Type {
		case event.TypePlan:
			if evt.Data == nil {
				t.Errorf("event[%d] plan: expected non-nil data", i)
			}
		case event.TypeToolStart:
			if evt.Data == nil {
				t.Errorf("event[%d] tool_start: expected non-nil data", i)
			}
		case event.TypeToolDone:
			if evt.Data == nil {
				t.Errorf("event[%d] tool_done: expected non-nil data", i)
			}
		case event.TypeDelta:
			if evt.Data == nil {
				t.Errorf("event[%d] delta: expected non-nil data", i)
			}
		case event.TypeDone:
			if evt.Data != nil {
				t.Logf("event[%d] done has data (may be ok): %s", i, string(evt.Data))
			}
		default:
			t.Errorf("event[%d]: unexpected event type %q", i, evt.Type)
		}
	}
}
