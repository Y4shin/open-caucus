package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/Y4shin/open-caucus/internal/config"
	"github.com/Y4shin/open-caucus/internal/repository/model"
)

// CommitteeEventDispatcher is the interface the admin service calls when
// committees or OIDC group rules change. The nil value is safe to use.
type CommitteeEventDispatcher interface {
	CommitteeCreated(ctx context.Context, c *model.Committee)
	CommitteeDeleted(ctx context.Context, c *model.Committee, groups []DeletedGroupInfo)
	OIDCGroupAdded(ctx context.Context, c *model.Committee, rule *model.OAuthCommitteeGroupRule, referencedByOthers bool)
	OIDCGroupRemoved(ctx context.Context, c *model.Committee, rule *model.OAuthCommitteeGroupRule, stillReferenced bool)
}

// DeletedGroupInfo carries details about an OIDC group rule that was attached
// to a committee that is being deleted.
type DeletedGroupInfo struct {
	GroupName       string
	Role            string
	StillReferenced bool
}

// CommitteeDispatcher implements CommitteeEventDispatcher by POSTing JSON
// payloads to every configured URL.
type CommitteeDispatcher struct {
	targets []webhookTarget
	secret  string
	client  *http.Client
}

// NewCommitteeDispatcher creates a CommitteeDispatcher from cfg.
func NewCommitteeDispatcher(cfg *config.WebhookConfig) *CommitteeDispatcher {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &CommitteeDispatcher{
		targets: parseTargets(cfg.URLs),
		secret:  cfg.Secret,
		client:  &http.Client{Timeout: timeout},
	}
}

// ── JSON payload shapes ──────────────────────────────────────────────────────

type committeeInfo struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	CreatedAt string `json:"created_at,omitempty"`
}

type committeeCreatedPayload struct {
	Event     string        `json:"event"`
	Timestamp string        `json:"timestamp"`
	Committee committeeInfo `json:"committee"`
}

type deletedGroupJSON struct {
	GroupName       string `json:"group_name"`
	Role            string `json:"role"`
	StillReferenced bool   `json:"still_referenced_by_other_committees"`
}

type committeeDeletedPayload struct {
	Event      string             `json:"event"`
	Timestamp  string             `json:"timestamp"`
	Committee  committeeInfo      `json:"committee"`
	OIDCGroups []deletedGroupJSON `json:"oidc_groups"`
}

type oidcGroupInfo struct {
	RuleID              int64  `json:"rule_id"`
	GroupName           string `json:"group_name"`
	Role                string `json:"role"`
	ReferencedByOthers  *bool  `json:"referenced_by_other_committees,omitempty"`
	StillReferenced     *bool  `json:"still_referenced_by_other_committees,omitempty"`
}

type oidcGroupEventPayload struct {
	Event     string        `json:"event"`
	Timestamp string        `json:"timestamp"`
	Committee committeeInfo `json:"committee"`
	Group     oidcGroupInfo `json:"group"`
}

// ── Interface implementation ─────────────────────────────────────────────────

func (d *CommitteeDispatcher) CommitteeCreated(_ context.Context, c *model.Committee) {
	if d == nil || len(d.targets) == 0 {
		return
	}
	payload := committeeCreatedPayload{
		Event:     "committee.created",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Committee: committeeInfo{
			ID:        c.ID,
			Name:      c.Name,
			Slug:      c.Slug,
			CreatedAt: c.CreatedAt.UTC().Format(time.RFC3339),
		},
	}
	d.fire("committee.created", payload)
}

func (d *CommitteeDispatcher) CommitteeDeleted(_ context.Context, c *model.Committee, groups []DeletedGroupInfo) {
	if d == nil || len(d.targets) == 0 {
		return
	}
	gs := make([]deletedGroupJSON, len(groups))
	for i, g := range groups {
		gs[i] = deletedGroupJSON{
			GroupName:       g.GroupName,
			Role:            g.Role,
			StillReferenced: g.StillReferenced,
		}
	}
	payload := committeeDeletedPayload{
		Event:     "committee.deleted",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Committee: committeeInfo{ID: c.ID, Name: c.Name, Slug: c.Slug},
		OIDCGroups: gs,
	}
	d.fire("committee.deleted", payload)
}

func (d *CommitteeDispatcher) OIDCGroupAdded(_ context.Context, c *model.Committee, rule *model.OAuthCommitteeGroupRule, referencedByOthers bool) {
	if d == nil || len(d.targets) == 0 {
		return
	}
	payload := oidcGroupEventPayload{
		Event:     "committee.oidc_group_added",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Committee: committeeInfo{ID: c.ID, Name: c.Name, Slug: c.Slug},
		Group: oidcGroupInfo{
			RuleID:             rule.ID,
			GroupName:          rule.GroupName,
			Role:               rule.Role,
			ReferencedByOthers: boolPtr(referencedByOthers),
		},
	}
	d.fire("committee.oidc_group_added", payload)
}

func (d *CommitteeDispatcher) OIDCGroupRemoved(_ context.Context, c *model.Committee, rule *model.OAuthCommitteeGroupRule, stillReferenced bool) {
	if d == nil || len(d.targets) == 0 {
		return
	}
	payload := oidcGroupEventPayload{
		Event:     "committee.oidc_group_removed",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Committee: committeeInfo{ID: c.ID, Name: c.Name, Slug: c.Slug},
		Group: oidcGroupInfo{
			RuleID:          rule.ID,
			GroupName:       rule.GroupName,
			Role:            rule.Role,
			StillReferenced: boolPtr(stillReferenced),
		},
	}
	d.fire("committee.oidc_group_removed", payload)
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func (d *CommitteeDispatcher) fire(event string, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		slog.Warn("committee webhook: failed to marshal payload", "event", event, "err", err)
		return
	}
	for _, t := range d.targets {
		go d.post(t, body, event)
	}
}

func (d *CommitteeDispatcher) post(t webhookTarget, body []byte, event string) {
	req, err := http.NewRequest(http.MethodPost, t.URL, bytes.NewReader(body))
	if err != nil {
		slog.Warn("committee webhook: failed to build request", "url", t.URL, "event", event, "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if d.secret != "" {
		req.Header.Set("X-Webhook-Secret", d.secret)
	}
	if t.HeaderName != "" {
		req.Header.Set(t.HeaderName, t.HeaderValue)
	}

	slog.Debug("committee webhook: dispatching", "url", t.URL, "event", event)

	resp, err := d.client.Do(req)
	if err != nil {
		slog.Warn("committee webhook: request failed", "url", t.URL, "event", event, "err", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slog.Warn("committee webhook: unexpected response status", "url", t.URL, "event", event, "status", resp.StatusCode)
	}
}

func boolPtr(b bool) *bool { return &b }
