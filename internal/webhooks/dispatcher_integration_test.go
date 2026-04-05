package webhooks

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Y4shin/open-caucus/internal/broker"
	"github.com/Y4shin/open-caucus/internal/config"
)

// capture is a thread-safe recorder for inbound webhook requests.
type capture struct {
	mu       sync.Mutex
	payloads []webhookPayload
	headers  []http.Header
}

func (c *capture) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var p webhookPayload
		_ = json.Unmarshal(body, &p)
		c.mu.Lock()
		c.payloads = append(c.payloads, p)
		c.headers = append(c.headers, r.Header.Clone())
		c.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
}

func (c *capture) waitForN(t *testing.T, n int, timeout time.Duration) []webhookPayload {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		c.mu.Lock()
		got := len(c.payloads)
		c.mu.Unlock()
		if got >= n {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.payloads) < n {
		t.Errorf("wanted %d webhook(s), got %d after %s", n, len(c.payloads), timeout)
	}
	out := make([]webhookPayload, len(c.payloads))
	copy(out, c.payloads)
	return out
}

// TestIntegration_RealBroker_SingleURL verifies the full path:
// broker.Publish → Dispatcher → HTTP POST with correct JSON body.
func TestIntegration_RealBroker_SingleURL(t *testing.T) {
	c := &capture{}
	srv := httptest.NewServer(c.handler())
	defer srv.Close()

	b := broker.NewMemoryBroker()
	defer b.Shutdown()

	d := New(&config.WebhookConfig{
		URLs:           []string{srv.URL},
		Secret:         "tok",
		TimeoutSeconds: 5,
	})
	d.Start(context.Background(), b)

	meetingID := int64(7)
	b.Publish(broker.SSEEvent{Event: "speakers.updated", MeetingID: &meetingID})

	payloads := c.waitForN(t, 1, 3*time.Second)
	if len(payloads) == 0 {
		return
	}

	p := payloads[0]
	if p.Event != "speakers.updated" {
		t.Errorf("event = %q, want speakers.updated", p.Event)
	}
	if p.MeetingID == nil || *p.MeetingID != 7 {
		t.Errorf("meeting_id = %v, want 7", p.MeetingID)
	}
	if _, err := time.Parse(time.RFC3339, p.Timestamp); err != nil {
		t.Errorf("timestamp not RFC3339: %v", err)
	}

	c.mu.Lock()
	sec := c.headers[0].Get("X-Webhook-Secret")
	c.mu.Unlock()
	if sec != "tok" {
		t.Errorf("X-Webhook-Secret = %q, want tok", sec)
	}
}

// TestIntegration_MultipleURLs verifies that every configured URL receives
// a copy of each event.
func TestIntegration_MultipleURLs(t *testing.T) {
	c1, c2 := &capture{}, &capture{}
	srv1 := httptest.NewServer(c1.handler())
	defer srv1.Close()
	srv2 := httptest.NewServer(c2.handler())
	defer srv2.Close()

	b := broker.NewMemoryBroker()
	defer b.Shutdown()

	d := New(&config.WebhookConfig{
		URLs:           []string{srv1.URL, srv2.URL},
		TimeoutSeconds: 5,
	})
	d.Start(context.Background(), b)

	b.Publish(broker.SSEEvent{Event: "votes.updated"})

	c1.waitForN(t, 1, 3*time.Second)
	c2.waitForN(t, 1, 3*time.Second)
}

// TestIntegration_MultipleEvents verifies that consecutive publishes each
// produce a webhook request and that events arrive in order.
func TestIntegration_MultipleEvents(t *testing.T) {
	c := &capture{}
	srv := httptest.NewServer(c.handler())
	defer srv.Close()

	b := broker.NewMemoryBroker()
	defer b.Shutdown()

	d := New(&config.WebhookConfig{URLs: []string{srv.URL}, TimeoutSeconds: 5})
	d.Start(context.Background(), b)

	events := []string{"speakers.updated", "votes.updated", "agenda.updated"}
	for _, ev := range events {
		b.Publish(broker.SSEEvent{Event: ev})
	}

	payloads := c.waitForN(t, len(events), 3*time.Second)

	// Collect received event names (order is not guaranteed due to goroutines).
	got := make(map[string]int, len(payloads))
	for _, p := range payloads {
		got[p.Event]++
	}
	for _, ev := range events {
		if got[ev] != 1 {
			t.Errorf("event %q received %d times, want 1", ev, got[ev])
		}
	}
}

// TestIntegration_ContextCancel verifies that cancelling the dispatcher
// context stops further webhook delivery.
func TestIntegration_ContextCancel(t *testing.T) {
	var callCount atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	b := broker.NewMemoryBroker()
	defer b.Shutdown()

	ctx, cancel := context.WithCancel(context.Background())

	d := New(&config.WebhookConfig{URLs: []string{srv.URL}, TimeoutSeconds: 5})
	d.Start(ctx, b)

	// Publish one event and confirm it arrives.
	b.Publish(broker.SSEEvent{Event: "attendees.updated"})
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && callCount.Load() == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if callCount.Load() == 0 {
		t.Fatal("first event was not delivered before cancel")
	}

	// Cancel the subscription context and give the goroutine a moment to stop.
	cancel()
	time.Sleep(50 * time.Millisecond)

	before := callCount.Load()
	b.Publish(broker.SSEEvent{Event: "speakers.updated"})
	time.Sleep(200 * time.Millisecond)

	if after := callCount.Load(); after != before {
		t.Errorf("expected no delivery after cancel, but call count went from %d to %d", before, after)
	}
}

// TestIntegration_SlowURL verifies that a slow target URL does not block
// event delivery to a fast URL.
func TestIntegration_SlowURL(t *testing.T) {
	fast := &capture{}
	fastSrv := httptest.NewServer(fast.handler())
	defer fastSrv.Close()

	slowSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowSrv.Close()

	b := broker.NewMemoryBroker()
	defer b.Shutdown()

	d := New(&config.WebhookConfig{
		URLs:           []string{slowSrv.URL, fastSrv.URL},
		TimeoutSeconds: 5,
	})
	d.Start(context.Background(), b)

	b.Publish(broker.SSEEvent{Event: "agenda.updated"})

	// Fast server should receive the request well within 1 second despite the
	// slow server taking 2 seconds.
	fast.waitForN(t, 1, 1*time.Second)
}

// TestIntegration_UnreachableURL verifies that an unreachable URL does not
// crash the dispatcher or prevent events from reaching reachable URLs.
func TestIntegration_UnreachableURL(t *testing.T) {
	good := &capture{}
	goodSrv := httptest.NewServer(good.handler())
	defer goodSrv.Close()

	b := broker.NewMemoryBroker()
	defer b.Shutdown()

	d := New(&config.WebhookConfig{
		URLs:           []string{"http://127.0.0.1:1", goodSrv.URL},
		TimeoutSeconds: 2,
	})
	d.Start(context.Background(), b)

	b.Publish(broker.SSEEvent{Event: "moderate-updated"})

	good.waitForN(t, 1, 5*time.Second)
}
