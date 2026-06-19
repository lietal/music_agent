package leak

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/music-agent/music-agent/internal/api"
	"github.com/music-agent/music-agent/internal/event"
)

func countGoroutines() int {
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	return runtime.NumGoroutine()
}

func openSSE(bus *event.Bus, runID string) (*httptest.Server, *http.Response, func()) {
	srv := httptest.NewServer(api.SSEHandler(bus))

	ctx, cancel := contextWithTimeout(30 * time.Second)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"?run_id="+runID, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		cancel()
		srv.Close()
		panic(err)
	}

	cleanup := func() {
		resp.Body.Close()
		cancel()
		srv.Close()
	}
	return srv, resp, cleanup
}

func TestGoroutineLeak(t *testing.T) {
	const numConns = 10
	const tolerance = 10

	baseline := countGoroutines()
	t.Logf("baseline goroutines: %d", baseline)

	bus := event.NewBus()
	var cleanups []func()

	for i := 0; i < numConns; i++ {
		runID := "leak-test-" + string(rune('0'+i%10)) + string(rune('a'+i/10))
		_, _, cleanup := openSSE(bus, runID)
		cleanups = append(cleanups, cleanup)
	}

	time.Sleep(500 * time.Millisecond)

	afterOpen := runtime.NumGoroutine()
	t.Logf("goroutines after %d connections: %d", numConns, afterOpen)
	if afterOpen <= baseline {
		t.Errorf("expected goroutine increase, baseline=%d after_open=%d", baseline, afterOpen)
	}

	for _, cleanup := range cleanups {
		cleanup()
	}

	time.Sleep(5 * time.Second)

	current := countGoroutines()
	t.Logf("goroutines after cleanup: %d (baseline=%d, delta=%d)", current, baseline, current-baseline)

	delta := current - baseline
	if delta < 0 {
		delta = -delta
	}
	if delta > tolerance {
		t.Errorf("goroutine leak detected: baseline=%d current=%d delta=%d (tolerance=%d)",
			baseline, current, current-baseline, tolerance)
	}
}

func TestSSEDisconnectSubscriberRemoved(t *testing.T) {
	bus := event.NewBus()
	const runID = "disc-leak-test"

	srv, _, cleanup := openSSE(bus, runID)
	defer srv.Close()

	time.Sleep(100 * time.Millisecond)

	ch := bus.Subscribe(runID)
	if ch == nil {
		bus.Unsubscribe(runID)
		t.Fatal("subscribe returned nil channel")
	}
	bus.Unsubscribe(runID)

	cleanup()

	time.Sleep(200 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		defer close(done)
		ch2 := bus.Subscribe(runID)
		defer bus.Unsubscribe(runID)
		select {
		case evt := <-ch2:
			t.Logf("received event after reconnect: type=%s", evt.Type)
		case <-time.After(2 * time.Second):
			t.Error("timed out waiting for event after reconnect")
		}
	}()

	time.Sleep(100 * time.Millisecond)
	bus.Publish(event.Event{Type: event.TypeDone, RunID: runID})

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for publish verification")
	}

	ch3 := bus.Subscribe(runID)
	bus.Unsubscribe(runID)
	select {
	case _, ok := <-ch3:
		if ok {
			t.Error("expected closed channel from new subscribe; old subscriber should not be present")
		}
	default:
	}
}

type timeoutCtx struct {
	done chan struct{}
}

func (c *timeoutCtx) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *timeoutCtx) Done() <-chan struct{}        { return c.done }
func (c *timeoutCtx) Err() error {
	select {
	case <-c.done:
		return &contextCanceled{}
	default:
		return nil
	}
}
func (c *timeoutCtx) Value(key interface{}) interface{} { return nil }

type contextCanceled struct{}

func (contextCanceled) Error() string { return "context canceled" }

func contextWithTimeout(d time.Duration) (*timeoutCtx, func()) {
	ctx := &timeoutCtx{done: make(chan struct{})}
	cancel := func() {
		select {
		case <-ctx.done:
		default:
			close(ctx.done)
		}
	}
	return ctx, cancel
}
