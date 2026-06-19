package event

import (
	"encoding/json"
	"testing"
)

func TestSubscribe(t *testing.T) {
	bus := NewBus()
	ch := bus.Subscribe("run-1")
	if ch == nil {
		t.Fatal("expected non-nil channel")
	}
}

func TestPublish(t *testing.T) {
	bus := NewBus()
	ch := bus.Subscribe("run-1")

	data, _ := json.Marshal("hello")
	evt := Event{Type: TypeDelta, RunID: "run-1", Data: data}
	bus.Publish(evt)

	select {
	case received := <-ch:
		if received.Type != TypeDelta {
			t.Errorf("type = %q, want %q", received.Type, TypeDelta)
		}
		if string(received.Data) != `"hello"` {
			t.Errorf("data = %s, want %q", string(received.Data), `"hello"`)
		}
	default:
		t.Fatal("expected event on channel")
	}
}

func TestPublishDropWhenFull(t *testing.T) {
	bus := NewBus()
	ch := bus.Subscribe("run-1")

	fill := Event{Type: TypeDelta, RunID: "run-1"}
	for i := 0; i < 64; i++ {
		fill.Data = json.RawMessage(`"fill"`)
		bus.Publish(fill)
	}

	drop := Event{Type: TypeDelta, RunID: "run-1", Data: json.RawMessage(`"dropped"`)}
	bus.Publish(drop)

	drainCount := 0
	for i := 0; i < 64; i++ {
		select {
		case <-ch:
			drainCount++
		default:
			break
		}
	}
	if drainCount != 64 {
		t.Errorf("drained %d events, want 64", drainCount)
	}

	select {
	case <-ch:
		t.Error("unexpected event in channel after drain")
	default:
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := NewBus()
	ch := bus.Subscribe("run-1")
	bus.Unsubscribe("run-1")

	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed after unsubscribe")
	}

	data, _ := json.Marshal("test")
	bus.Publish(Event{Type: TypeDelta, RunID: "run-1", Data: data})
}

func TestPublishUnknownRunID(t *testing.T) {
	bus := NewBus()
	bus.Publish(Event{Type: TypeDelta, RunID: "nonexistent"})
}

func TestUnsubscribeUnknownRunID(t *testing.T) {
	bus := NewBus()
	bus.Unsubscribe("nonexistent")
}

func TestMultipleSubscribers(t *testing.T) {
	bus := NewBus()
	ch1 := bus.Subscribe("run-1")
	ch2 := bus.Subscribe("run-2")

	data1, _ := json.Marshal("data1")
	bus.Publish(Event{Type: TypeDelta, RunID: "run-1", Data: data1})

	data2, _ := json.Marshal("data2")
	bus.Publish(Event{Type: TypePlan, RunID: "run-2", Data: data2})

	select {
	case evt := <-ch1:
		if evt.RunID != "run-1" {
			t.Errorf("ch1 RunID = %q, want run-1", evt.RunID)
		}
	default:
		t.Fatal("expected event on ch1")
	}

	select {
	case evt := <-ch2:
		if evt.RunID != "run-2" {
			t.Errorf("ch2 RunID = %q, want run-2", evt.RunID)
		}
	default:
		t.Fatal("expected event on ch2")
	}
}

func TestResubscribe(t *testing.T) {
	bus := NewBus()
	ch1 := bus.Subscribe("run-1")

	data, _ := json.Marshal("first")
	bus.Publish(Event{Type: TypeDelta, RunID: "run-1", Data: data})

	select {
	case <-ch1:
	default:
		t.Fatal("expected event on ch1")
	}

	ch2 := bus.Subscribe("run-1")

	_, ok := <-ch1
	if ok {
		t.Error("old channel should be closed after resubscribe")
	}

	data2, _ := json.Marshal("second")
	bus.Publish(Event{Type: TypeDelta, RunID: "run-1", Data: data2})

	select {
	case evt := <-ch2:
		if string(evt.Data) != `"second"` {
			t.Errorf("data = %s, want %q", string(evt.Data), `"second"`)
		}
	default:
		t.Fatal("expected event on new channel")
	}
}
