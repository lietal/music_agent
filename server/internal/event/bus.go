package event

import (
	"sync"
)

type Bus struct {
	mu   sync.RWMutex
	subs map[string]chan Event
}

func NewBus() *Bus {
	return &Bus{
		subs: make(map[string]chan Event),
	}
}

func (b *Bus) Subscribe(runID string) <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ch, ok := b.subs[runID]; ok {
		close(ch)
	}

	ch := make(chan Event, 64)
	b.subs[runID] = ch
	return ch
}

func (b *Bus) Publish(evt Event) {
	b.mu.RLock()
	ch, ok := b.subs[evt.RunID]
	b.mu.RUnlock()

	if !ok {
		return
	}

	select {
	case ch <- evt:
	default:
	}
}

func (b *Bus) Unsubscribe(runID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ch, ok := b.subs[runID]; ok {
		close(ch)
		delete(b.subs, runID)
	}
}
