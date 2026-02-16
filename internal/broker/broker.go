package broker

import "context"

// SSEEvent is the wire format sent to clients over a Server-Sent Events connection.
type SSEEvent struct {
	// Event is the SSE "event:" field name (e.g. "counter").
	Event string
	// Data is the payload (typically HTML for htmx SSE swaps).
	Data []byte
}

// Broker manages SSE client subscriptions and publishes strictly typed domain events.
type Broker interface {
	// Subscribe returns a channel that receives SSE events for the lifetime of ctx.
	// The channel is closed when ctx is cancelled.
	Subscribe(ctx context.Context) <-chan SSEEvent

	// Shutdown gracefully closes all client channels and stops the broker.
	Shutdown()

	// Increment atomically increments the counter and broadcasts the new value.
	Increment()

	// Counter returns the current counter value.
	Counter() int
}
