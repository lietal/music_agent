package api

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/music-agent/music-agent/internal/agent"
	"github.com/music-agent/music-agent/internal/event"
)

func TestWriteSSEEvents_PlanToDone(t *testing.T) {
	ch := make(chan event.Event, 4)
	ch <- event.Event{Type: event.TypePlan, RunID: "test", Data: []byte(`{"intent":"search"}`)}
	ch <- event.Event{Type: event.TypeToolStart, RunID: "test", Data: []byte(`{"name":"search"}`)}
	ch <- event.Event{Type: event.TypeToolDone, RunID: "test", Data: []byte(`{"songs":[]}`)}
	ch <- event.Event{Type: event.TypeDone, RunID: "test"}
	close(ch)

	rec := httptest.NewRecorder()
	rc := http.NewResponseController(rec)
	if rc == nil {
		t.Fatal("ResponseController not supported")
	}

	rec.Header().Set("Content-Type", "text/event-stream")
	rec.WriteHeader(http.StatusOK)

	ctx := context.Background()
	writeSSEEvents(rec, rc, ctx, ch)

	body := rec.Body.String()
	if !strings.Contains(body, "event: plan") {
		t.Error("missing plan event:\n" + body)
	}
	if !strings.Contains(body, "event: tool_start") {
		t.Error("missing tool_start event:\n" + body)
	}
	if !strings.Contains(body, "event: tool_done") {
		t.Error("missing tool_done event:\n" + body)
	}
	if !strings.Contains(body, "event: done") {
		t.Error("missing done event:\n" + body)
	}

	scanner := bufio.NewScanner(strings.NewReader(body))
	eventCount := 0
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "event: ") {
			eventCount++
		}
	}
	if eventCount != 4 {
		t.Errorf("expected 4 events, got %d", eventCount)
	}
}

func TestWriteSSEEvents_ErrorStops(t *testing.T) {
	ch := make(chan event.Event, 2)
	ch <- event.Event{Type: event.TypeError, RunID: "test", Data: []byte(`{"message":"fail"}`)}
	close(ch)

	rec := httptest.NewRecorder()
	rc := http.NewResponseController(rec)
	if rc == nil {
		t.Fatal("ResponseController not supported")
	}
	rec.Header().Set("Content-Type", "text/event-stream")
	rec.WriteHeader(http.StatusOK)

	writeSSEEvents(rec, rc, context.Background(), ch)

	body := rec.Body.String()
	if !strings.Contains(body, "event: error") {
		t.Error("missing error event:\n" + body)
	}
}

func TestWriteSSEEvents_ContextCancel(t *testing.T) {
	ch := make(chan event.Event)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	rec := httptest.NewRecorder()
	rc := http.NewResponseController(rec)
	if rc == nil {
		t.Fatal("ResponseController not supported")
	}
	rec.Header().Set("Content-Type", "text/event-stream")
	rec.WriteHeader(http.StatusOK)

	writeSSEEvents(rec, rc, ctx, ch)
	close(ch)

	if rec.Body.Len() == 0 {
		t.Log("empty body as expected after context cancel")
	}
}

func TestWriteSSEEvents_ClosedChannel(t *testing.T) {
	ch := make(chan event.Event)
	close(ch)

	rec := httptest.NewRecorder()
	rc := http.NewResponseController(rec)
	if rc == nil {
		t.Fatal("ResponseController not supported")
	}
	rec.Header().Set("Content-Type", "text/event-stream")
	rec.WriteHeader(http.StatusOK)

	writeSSEEvents(rec, rc, context.Background(), ch)

	if rec.Body.Len() > 0 {
		t.Error("expected empty body for closed channel")
	}
}

func TestSetupSSE_Success(t *testing.T) {
	rec := httptest.NewRecorder()
	rc, err := setupSSE(rec)
	if err != nil {
		t.Fatal(err)
	}
	if rc == nil {
		t.Fatal("expected non-nil ResponseController")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected text/event-stream, got %s", ct)
	}
}

func TestChatEventsHandler_WithMockAgent(t *testing.T) {
	bus := event.NewBus()
	h := NewHandler(bus, []byte("test-secret"), nil)

	ch := make(chan event.Event, 3)
	ch <- event.Event{Type: event.TypePlan, RunID: "m", Data: []byte(`{}`)}
	ch <- event.Event{Type: event.TypeDelta, RunID: "m", Data: []byte(`{}`)}
	ch <- event.Event{Type: event.TypeDone, RunID: "m"}
	close(ch)

	mockLoop := &testAgentLoop{events: ch}
	h.agent = mockLoop

	token := generateTestToken(h.jwtSecret)
	h.StoreRunMessage("mock-run", "hello")

	router := NewRouter(h)
	req := httptest.NewRequest("GET", "/api/chat/mock-run/events?token="+token, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "event: plan") {
		t.Error("missing plan:\n" + body)
	}
	if !strings.Contains(body, "event: done") {
		t.Error("missing done:\n" + body)
	}
}

type testAgentLoop struct {
	events chan event.Event
}

func (a *testAgentLoop) Run(ctx context.Context, state agent.LoopState) <-chan event.Event {
	return a.events
}
