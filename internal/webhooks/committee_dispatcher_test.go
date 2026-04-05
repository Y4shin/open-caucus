package webhooks

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Y4shin/open-caucus/internal/config"
	"github.com/Y4shin/open-caucus/internal/repository/model"
)

// committeeCapture records inbound committee webhook requests.
type committeeCapture struct {
	bodies  chan []byte
	headers chan http.Header
}

func newCommitteeCapture() *committeeCapture {
	return &committeeCapture{
		bodies:  make(chan []byte, 8),
		headers: make(chan http.Header, 8),
	}
}

func (c *committeeCapture) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		c.bodies <- body
		c.headers <- r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	})
}

func (c *committeeCapture) recv(t *testing.T) (map[string]any, http.Header) {
	t.Helper()
	select {
	case body := <-c.bodies:
		h := <-c.headers
		var m map[string]any
		if err := json.Unmarshal(body, &m); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		return m, h
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for committee webhook request")
		return nil, nil
	}
}

func newTestCommitteeDispatcher(url, secret string) *CommitteeDispatcher {
	return NewCommitteeDispatcher(&config.WebhookConfig{
		URLs:           []string{url},
		Secret:         secret,
		TimeoutSeconds: 5,
	})
}

// ── CommitteeCreated ─────────────────────────────────────────────────────────

func TestCommitteeDispatcher_CommitteeCreated(t *testing.T) {
	c := newCommitteeCapture()
	srv := httptest.NewServer(c.handler())
	defer srv.Close()

	d := newTestCommitteeDispatcher(srv.URL, "sec")
	now := time.Now()
	d.CommitteeCreated(context.Background(), &model.Committee{
		ID: 3, Name: "Finance", Slug: "finance", CreatedAt: now,
	})

	body, h := c.recv(t)

	if body["event"] != "committee.created" {
		t.Errorf("event = %v, want committee.created", body["event"])
	}
	if h.Get("X-Webhook-Secret") != "sec" {
		t.Errorf("X-Webhook-Secret = %q, want sec", h.Get("X-Webhook-Secret"))
	}
	committee, ok := body["committee"].(map[string]any)
	if !ok {
		t.Fatal("committee field missing or wrong type")
	}
	if committee["slug"] != "finance" {
		t.Errorf("committee.slug = %v, want finance", committee["slug"])
	}
	if committee["name"] != "Finance" {
		t.Errorf("committee.name = %v, want Finance", committee["name"])
	}
	if committee["created_at"] == "" || committee["created_at"] == nil {
		t.Error("committee.created_at is empty")
	}
}

// ── CommitteeDeleted ─────────────────────────────────────────────────────────

func TestCommitteeDispatcher_CommitteeDeleted_WithGroups(t *testing.T) {
	c := newCommitteeCapture()
	srv := httptest.NewServer(c.handler())
	defer srv.Close()

	d := newTestCommitteeDispatcher(srv.URL, "")
	d.CommitteeDeleted(context.Background(), &model.Committee{ID: 1, Name: "Budget", Slug: "budget"}, []DeletedGroupInfo{
		{GroupName: "budget-chairs", Role: "chairperson", StillReferenced: false},
		{GroupName: "staff", Role: "member", StillReferenced: true},
	})

	body, _ := c.recv(t)

	if body["event"] != "committee.deleted" {
		t.Errorf("event = %v, want committee.deleted", body["event"])
	}
	groups, ok := body["oidc_groups"].([]any)
	if !ok {
		t.Fatal("oidc_groups field missing or wrong type")
	}
	if len(groups) != 2 {
		t.Fatalf("oidc_groups len = %d, want 2", len(groups))
	}
	g0 := groups[0].(map[string]any)
	if g0["group_name"] != "budget-chairs" {
		t.Errorf("group[0].group_name = %v, want budget-chairs", g0["group_name"])
	}
	if g0["still_referenced_by_other_committees"] != false {
		t.Errorf("group[0].still_referenced = %v, want false", g0["still_referenced_by_other_committees"])
	}
	g1 := groups[1].(map[string]any)
	if g1["still_referenced_by_other_committees"] != true {
		t.Errorf("group[1].still_referenced = %v, want true", g1["still_referenced_by_other_committees"])
	}
}

func TestCommitteeDispatcher_CommitteeDeleted_EmptyGroups(t *testing.T) {
	c := newCommitteeCapture()
	srv := httptest.NewServer(c.handler())
	defer srv.Close()

	d := newTestCommitteeDispatcher(srv.URL, "")
	d.CommitteeDeleted(context.Background(), &model.Committee{ID: 2, Name: "Empty", Slug: "empty"}, nil)

	body, _ := c.recv(t)
	if body["event"] != "committee.deleted" {
		t.Errorf("event = %v, want committee.deleted", body["event"])
	}
	// oidc_groups must be present and be an empty array (not null)
	raw, _ := json.Marshal(body)
	var typed struct {
		OIDCGroups []any `json:"oidc_groups"`
	}
	if err := json.Unmarshal(raw, &typed); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if typed.OIDCGroups == nil {
		t.Error("oidc_groups should be [] not null")
	}
}

// ── OIDCGroupAdded ────────────────────────────────────────────────────────────

func TestCommitteeDispatcher_OIDCGroupAdded(t *testing.T) {
	c := newCommitteeCapture()
	srv := httptest.NewServer(c.handler())
	defer srv.Close()

	d := newTestCommitteeDispatcher(srv.URL, "")
	d.OIDCGroupAdded(
		context.Background(),
		&model.Committee{ID: 1, Name: "Finance", Slug: "finance"},
		&model.OAuthCommitteeGroupRule{ID: 5, GroupName: "finance-chairs", Role: "chairperson"},
		true,
	)

	body, _ := c.recv(t)
	if body["event"] != "committee.oidc_group_added" {
		t.Errorf("event = %v, want committee.oidc_group_added", body["event"])
	}
	group, ok := body["group"].(map[string]any)
	if !ok {
		t.Fatal("group field missing or wrong type")
	}
	if group["group_name"] != "finance-chairs" {
		t.Errorf("group.group_name = %v, want finance-chairs", group["group_name"])
	}
	if group["role"] != "chairperson" {
		t.Errorf("group.role = %v, want chairperson", group["role"])
	}
	if group["referenced_by_other_committees"] != true {
		t.Errorf("group.referenced_by_other_committees = %v, want true", group["referenced_by_other_committees"])
	}
	if _, hasStill := group["still_referenced_by_other_committees"]; hasStill {
		t.Error("still_referenced_by_other_committees should not appear in oidc_group_added payload")
	}
}

// ── OIDCGroupRemoved ─────────────────────────────────────────────────────────

func TestCommitteeDispatcher_OIDCGroupRemoved(t *testing.T) {
	c := newCommitteeCapture()
	srv := httptest.NewServer(c.handler())
	defer srv.Close()

	d := newTestCommitteeDispatcher(srv.URL, "")
	d.OIDCGroupRemoved(
		context.Background(),
		&model.Committee{ID: 1, Name: "Finance", Slug: "finance"},
		&model.OAuthCommitteeGroupRule{ID: 5, GroupName: "finance-chairs", Role: "chairperson"},
		false,
	)

	body, _ := c.recv(t)
	if body["event"] != "committee.oidc_group_removed" {
		t.Errorf("event = %v, want committee.oidc_group_removed", body["event"])
	}
	group := body["group"].(map[string]any)
	if group["still_referenced_by_other_committees"] != false {
		t.Errorf("still_referenced = %v, want false", group["still_referenced_by_other_committees"])
	}
	if _, hasRBO := group["referenced_by_other_committees"]; hasRBO {
		t.Error("referenced_by_other_committees should not appear in oidc_group_removed payload")
	}
}

// ── No-op when URLs empty ─────────────────────────────────────────────────────

func TestCommitteeDispatcher_NoURLs_NoRequests(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := NewCommitteeDispatcher(&config.WebhookConfig{URLs: nil, TimeoutSeconds: 5})
	d.CommitteeCreated(context.Background(), &model.Committee{ID: 1, Name: "X", Slug: "x"})
	time.Sleep(50 * time.Millisecond)
	if called {
		t.Error("expected no HTTP call when URLs is empty")
	}
}

// ── Timestamp format ──────────────────────────────────────────────────────────

func TestCommitteeDispatcher_TimestampRFC3339(t *testing.T) {
	c := newCommitteeCapture()
	srv := httptest.NewServer(c.handler())
	defer srv.Close()

	d := newTestCommitteeDispatcher(srv.URL, "")
	d.CommitteeCreated(context.Background(), &model.Committee{ID: 1, Name: "X", Slug: "x", CreatedAt: time.Now()})

	body, _ := c.recv(t)
	ts, _ := body["timestamp"].(string)
	if _, err := time.Parse(time.RFC3339, ts); err != nil {
		t.Errorf("timestamp %q is not RFC3339: %v", ts, err)
	}
}
