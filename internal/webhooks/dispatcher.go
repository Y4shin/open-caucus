package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/Y4shin/open-caucus/internal/broker"
	"github.com/Y4shin/open-caucus/internal/config"
)

// Dispatcher subscribes to the broker and fires outbound HTTP webhooks for
// every SSE event received.
type Dispatcher struct {
	urls    []string
	secret  string
	client  *http.Client
}

// New creates a Dispatcher from cfg. Returns nil when cfg is nil or has no URLs
// (callers should guard with len(cfg.URLs) > 0 before calling Start).
func New(cfg *config.WebhookConfig) *Dispatcher {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Dispatcher{
		urls:   cfg.URLs,
		secret: cfg.Secret,
		client: &http.Client{Timeout: timeout},
	}
}

// webhookPayload is the JSON body sent on every outbound request.
type webhookPayload struct {
	Event     string  `json:"event"`
	MeetingID *int64  `json:"meeting_id"`
	Timestamp string  `json:"timestamp"`
}

// Start subscribes to b and dispatches webhooks in the background.
// It returns immediately; cancel ctx to stop.
func (d *Dispatcher) Start(ctx context.Context, b broker.Broker) {
	if len(d.urls) == 0 {
		return
	}
	ch := b.Subscribe(ctx)
	go func() {
		for evt := range ch {
			d.dispatch(evt)
		}
	}()
}

func (d *Dispatcher) dispatch(evt broker.SSEEvent) {
	payload := webhookPayload{
		Event:     evt.Event,
		MeetingID: evt.MeetingID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		slog.Warn("webhook: failed to marshal payload", "event", evt.Event, "err", err)
		return
	}

	for _, url := range d.urls {
		go d.post(url, body, evt.Event)
	}
}

func (d *Dispatcher) post(url string, body []byte, event string) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		slog.Warn("webhook: failed to build request", "url", url, "event", event, "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if d.secret != "" {
		req.Header.Set("X-Webhook-Secret", d.secret)
	}

	slog.Debug("webhook: dispatching", "url", url, "event", event)

	resp, err := d.client.Do(req)
	if err != nil {
		slog.Warn("webhook: request failed", "url", url, "event", event, "err", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slog.Warn("webhook: unexpected response status", "url", url, "event", event, "status", resp.StatusCode)
	}
}
