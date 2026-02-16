package broker

import (
	"bytes"
	"context"
	"strconv"
	"sync"
)

// MemoryBroker is an in-memory implementation of Broker.
type MemoryBroker struct {
	mu       sync.RWMutex
	counter  int
	clients  map[chan SSEEvent]struct{}
	shutdown chan struct{}
}

func NewMemoryBroker() *MemoryBroker {
	return &MemoryBroker{
		clients:  make(map[chan SSEEvent]struct{}),
		shutdown: make(chan struct{}),
	}
}

func (b *MemoryBroker) Subscribe(ctx context.Context) <-chan SSEEvent {
	ch := make(chan SSEEvent, 16)

	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()

	go func() {
		select {
		case <-ctx.Done():
		case <-b.shutdown:
		}
		b.mu.Lock()
		delete(b.clients, ch)
		close(ch)
		b.mu.Unlock()
	}()

	return ch
}

func (b *MemoryBroker) Shutdown() {
	close(b.shutdown)
}

func (b *MemoryBroker) Increment() {
	b.mu.Lock()
	b.counter++
	val := b.counter
	b.mu.Unlock()

	b.broadcast(renderCounter(val))
}

func (b *MemoryBroker) Counter() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.counter
}

func (b *MemoryBroker) broadcast(evt SSEEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- evt:
		default:
			// drop if client is slow
		}
	}
}

func renderCounter(val int) SSEEvent {
	var buf bytes.Buffer
	buf.WriteString(`<span id="counter-value">`)
	buf.WriteString(strconv.Itoa(val))
	buf.WriteString(`</span>`)
	return SSEEvent{
		Event: "counter",
		Data:  buf.Bytes(),
	}
}
