package api

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/music-agent/music-agent/internal/event"
)

func TestSSEHandlerMissingRunID(t *testing.T) {
	bus := event.NewBus()
	handler := SSEHandler(bus)

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestSSEHandlerHeaders(t *testing.T) {
	bus := event.NewBus()

	srv := httptest.NewServer(SSEHandler(bus))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"?run_id=test-headers", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", resp.Header.Get("Content-Type"))
	}
	if resp.Header.Get("Cache-Control") != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", resp.Header.Get("Cache-Control"))
	}
	if resp.Header.Get("X-Accel-Buffering") != "no" {
		t.Errorf("X-Accel-Buffering = %q, want no", resp.Header.Get("X-Accel-Buffering"))
	}
	if resp.Header.Get("Connection") != "keep-alive" {
		t.Errorf("Connection = %q, want keep-alive", resp.Header.Get("Connection"))
	}
}

func TestSSEHandlerEvents(t *testing.T) {
	bus := event.NewBus()

	srv := httptest.NewServer(SSEHandler(bus))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"?run_id=test-events", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	data, _ := json.Marshal(map[string]string{"msg": "hello"})
	bus.Publish(event.Event{Type: event.TypeDelta, RunID: "test-events", Data: data})

	reader := bufio.NewReader(resp.Body)

	var mu sync.Mutex
	var lines []string

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 4; i++ {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			mu.Lock()
			lines = append(lines, strings.TrimRight(line, "\n"))
			mu.Unlock()
		}
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
	}

	mu.Lock()
	defer mu.Unlock()

	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d: %v", len(lines), lines)
	}

	hasEvent := false
	hasData := false
	for _, line := range lines {
		if strings.HasPrefix(line, "event: delta") {
			hasEvent = true
		}
		if strings.HasPrefix(line, "data: ") {
			hasData = true
		}
	}
	if !hasEvent {
		t.Error("expected event line")
	}
	if !hasData {
		t.Error("expected data line")
	}
}

func TestSSEHandlerHeartbeat(t *testing.T) {
	bus := event.NewBus()

	srv := httptest.NewServer(SSEHandler(bus))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"?run_id=test-hb-fmt", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	found := make(chan struct{})
	go func() {
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			if strings.Contains(line, "heartbeat") {
				close(found)
				return
			}
		}
	}()

	select {
	case <-found:
		t.Log("heartbeat comment received")
	case <-time.After(35 * time.Second):
		t.Fatal("timed out waiting for heartbeat")
	}

	cancel()
}

func TestSSEDisconnect(t *testing.T) {
	bus := event.NewBus()

	ctx, cancel := context.WithCancel(context.Background())

	srv := httptest.NewServer(SSEHandler(bus))
	defer srv.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"?run_id=test-disc", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	resp.Body.Close()
	cancel()

	time.Sleep(200 * time.Millisecond)

	ch := bus.Subscribe("test-disc")
	defer bus.Unsubscribe("test-disc")

	select {
	case evt := <-ch:
		t.Logf("new subscriber received event (old handler disconnected): %s", evt.Type)
	default:
	}
}
