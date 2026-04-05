package webhooks

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Y4shin/open-caucus/internal/broker"
	"github.com/Y4shin/open-caucus/internal/config"
)

// fakeBroker is a minimal broker.Broker for testing.
type fakeBroker struct {
	ch chan broker.SSEEvent
}

func newFakeBroker() *fakeBroker {
	return &fakeBroker{ch: make(chan broker.SSEEvent, 8)}
}

func (f *fakeBroker) Subscribe(_ context.Context) <-chan broker.SSEEvent { return f.ch }
func (f *fakeBroker) Publish(evt broker.SSEEvent)                        { f.ch <- evt }
func (f *fakeBroker) Shutdown()                                          { close(f.ch) }
func (f *fakeBroker) Increment()                                         {}
func (f *fakeBroker) Counter() int                                       { return 0 }

func TestDispatcher_PayloadAndHeaders(t *testing.T) {
	received := make(chan *http.Request, 1)
	bodies := make(chan []byte, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodies <- body
		received <- r.Clone(context.Background())
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	meetingID := int64(42)
	b := newFakeBroker()

	d := New(&config.WebhookConfig{
		URLs:           []string{srv.URL},
		Secret:         "secret123",
		TimeoutSeconds: 5,
	})
	d.Start(context.Background(), b)

	b.Publish(broker.SSEEvent{
		Event:     "speakers.updated",
		MeetingID: &meetingID,
	})

	select {
	case body := <-bodies:
		var payload webhookPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if payload.Event != "speakers.updated" {
			t.Errorf("event = %q, want %q", payload.Event, "speakers.updated")
		}
		if payload.MeetingID == nil || *payload.MeetingID != 42 {
			t.Errorf("meeting_id = %v, want 42", payload.MeetingID)
		}
		if payload.Timestamp == "" {
			t.Error("timestamp is empty")
		}
		if _, err := time.Parse(time.RFC3339, payload.Timestamp); err != nil {
			t.Errorf("timestamp not RFC3339: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for webhook request")
	}

	select {
	case req := <-received:
		if ct := req.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		if sec := req.Header.Get("X-Webhook-Secret"); sec != "secret123" {
			t.Errorf("X-Webhook-Secret = %q, want secret123", sec)
		}
	default:
		t.Fatal("no request captured")
	}
}

func TestDispatcher_NilMeetingID(t *testing.T) {
	bodies := make(chan []byte, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodies <- body
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	b := newFakeBroker()
	d := New(&config.WebhookConfig{URLs: []string{srv.URL}, TimeoutSeconds: 5})
	d.Start(context.Background(), b)

	b.Publish(broker.SSEEvent{Event: "agenda.updated"})

	select {
	case body := <-bodies:
		var payload webhookPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if payload.MeetingID != nil {
			t.Errorf("meeting_id = %v, want null", *payload.MeetingID)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for webhook request")
	}
}

func TestDispatcher_NoURLs_NoRequests(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	b := newFakeBroker()
	d := New(&config.WebhookConfig{URLs: nil, TimeoutSeconds: 5})
	d.Start(context.Background(), b)

	b.Publish(broker.SSEEvent{Event: "votes.updated"})

	// Give any goroutine a moment to fire (it shouldn't).
	time.Sleep(100 * time.Millisecond)

	if called {
		t.Error("expected no HTTP calls when URLs is empty, but got one")
	}
}

func TestDispatcher_NoSecretHeader(t *testing.T) {
	headers := make(chan http.Header, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers <- r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	b := newFakeBroker()
	d := New(&config.WebhookConfig{URLs: []string{srv.URL}, Secret: "", TimeoutSeconds: 5})
	d.Start(context.Background(), b)

	b.Publish(broker.SSEEvent{Event: "attendees.updated"})

	select {
	case h := <-headers:
		if sec := h.Get("X-Webhook-Secret"); sec != "" {
			t.Errorf("X-Webhook-Secret should be absent, got %q", sec)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for webhook request")
	}
}
