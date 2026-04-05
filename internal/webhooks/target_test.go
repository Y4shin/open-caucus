package webhooks

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Y4shin/open-caucus/internal/broker"
	"github.com/Y4shin/open-caucus/internal/config"
	"github.com/Y4shin/open-caucus/internal/repository/model"
)

// ── parseTarget unit tests ────────────────────────────────────────────────────

func TestParseTarget_Plain(t *testing.T) {
	got := parseTarget("https://example.com/hook")
	if got.URL != "https://example.com/hook" {
		t.Errorf("URL = %q, want https://example.com/hook", got.URL)
	}
	if got.HeaderName != "" || got.HeaderValue != "" {
		t.Errorf("expected no header auth, got name=%q value=%q", got.HeaderName, got.HeaderValue)
	}
}

func TestParseTarget_WithHeaderAuth(t *testing.T) {
	cases := []struct {
		input       string
		wantURL     string
		wantName    string
		wantValue   string
	}{
		{
			input:     "https://example.com/hook@X-Api-Key:secret123",
			wantURL:   "https://example.com/hook",
			wantName:  "X-Api-Key",
			wantValue: "secret123",
		},
		{
			input:     "https://example.com/hook@Authorization:Bearer my-token",
			wantURL:   "https://example.com/hook",
			wantName:  "Authorization",
			wantValue: "Bearer my-token",
		},
		{
			input:     "http://localhost:5678/webhook@X-Custom_Header:val",
			wantURL:   "http://localhost:5678/webhook",
			wantName:  "X-Custom_Header",
			wantValue: "val",
		},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := parseTarget(tc.input)
			if got.URL != tc.wantURL {
				t.Errorf("URL = %q, want %q", got.URL, tc.wantURL)
			}
			if got.HeaderName != tc.wantName {
				t.Errorf("HeaderName = %q, want %q", got.HeaderName, tc.wantName)
			}
			if got.HeaderValue != tc.wantValue {
				t.Errorf("HeaderValue = %q, want %q", got.HeaderValue, tc.wantValue)
			}
		})
	}
}

func TestParseTarget_URLWithUserinfoAt(t *testing.T) {
	// The @ in userinfo is followed by a hostname which contains dots,
	// so it must NOT be treated as a header auth delimiter.
	cases := []string{
		"https://user@host.example.com/path",
		"https://user:pass@host.example.com/path",
	}
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			got := parseTarget(input)
			if got.URL != input {
				t.Errorf("URL = %q, want full input %q", got.URL, input)
			}
			if got.HeaderName != "" {
				t.Errorf("expected no header auth for userinfo URL, got name=%q", got.HeaderName)
			}
		})
	}
}

func TestParseTarget_NoColon(t *testing.T) {
	// @ present but no colon after it → treat as plain URL.
	got := parseTarget("https://example.com/hook@SomeHeader")
	if got.URL != "https://example.com/hook@SomeHeader" {
		t.Errorf("URL = %q, want full input", got.URL)
	}
	if got.HeaderName != "" {
		t.Errorf("expected no header auth, got name=%q", got.HeaderName)
	}
}

func TestParseTarget_InvalidHeaderName(t *testing.T) {
	// Header name candidate contains a dot → treat as plain URL.
	got := parseTarget("https://example.com/hook@not.a.header:value")
	if got.URL != "https://example.com/hook@not.a.header:value" {
		t.Errorf("URL = %q, want full input", got.URL)
	}
	if got.HeaderName != "" {
		t.Errorf("expected no header auth, got name=%q", got.HeaderName)
	}
}

func TestParseTargets_Mixed(t *testing.T) {
	urls := []string{
		"https://example.com/plain",
		"https://example.com/auth@X-Api-Key:tok",
	}
	targets := parseTargets(urls)
	if len(targets) != 2 {
		t.Fatalf("len = %d, want 2", len(targets))
	}
	if targets[0].HeaderName != "" {
		t.Errorf("target[0] should have no auth header")
	}
	if targets[1].HeaderName != "X-Api-Key" || targets[1].HeaderValue != "tok" {
		t.Errorf("target[1] auth = %q:%q, want X-Api-Key:tok", targets[1].HeaderName, targets[1].HeaderValue)
	}
}

// ── Per-target header integration tests ──────────────────────────────────────

func TestDispatcher_PerTargetHeader(t *testing.T) {
	headers := make(chan http.Header, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers <- r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	b := broker.NewMemoryBroker()
	defer b.Shutdown()

	d := New(&config.WebhookConfig{
		URLs:           []string{srv.URL + "@X-Api-Key:mytoken"},
		TimeoutSeconds: 5,
	})
	d.Start(context.Background(), b)

	b.Publish(broker.SSEEvent{Event: "speakers.updated"})

	select {
	case h := <-headers:
		if got := h.Get("X-Api-Key"); got != "mytoken" {
			t.Errorf("X-Api-Key = %q, want mytoken", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for webhook request")
	}
}

func TestDispatcher_PerTargetHeader_AndSecret(t *testing.T) {
	headers := make(chan http.Header, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers <- r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	b := broker.NewMemoryBroker()
	defer b.Shutdown()

	d := New(&config.WebhookConfig{
		URLs:           []string{srv.URL + "@Authorization:Bearer tok"},
		Secret:         "shared-secret",
		TimeoutSeconds: 5,
	})
	d.Start(context.Background(), b)

	b.Publish(broker.SSEEvent{Event: "votes.updated"})

	select {
	case h := <-headers:
		if got := h.Get("Authorization"); got != "Bearer tok" {
			t.Errorf("Authorization = %q, want Bearer tok", got)
		}
		if got := h.Get("X-Webhook-Secret"); got != "shared-secret" {
			t.Errorf("X-Webhook-Secret = %q, want shared-secret", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for webhook request")
	}
}

func TestCommitteeDispatcher_PerTargetHeader(t *testing.T) {
	headers := make(chan http.Header, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers <- r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := NewCommitteeDispatcher(&config.WebhookConfig{
		URLs:           []string{srv.URL + "@X-Api-Key:committee-token"},
		TimeoutSeconds: 5,
	})
	d.CommitteeCreated(context.Background(), &model.Committee{ID: 1, Name: "Test", Slug: "test"})

	select {
	case h := <-headers:
		if got := h.Get("X-Api-Key"); got != "committee-token" {
			t.Errorf("X-Api-Key = %q, want committee-token", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for committee webhook request")
	}
}
