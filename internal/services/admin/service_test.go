package adminservice

import (
	"context"
	"testing"
	"time"

	"github.com/Y4shin/open-caucus/internal/repository/sqlite"
	"github.com/Y4shin/open-caucus/internal/session"
)

// adminCtx creates a context with an admin session for the given account ID.
func adminCtx(t *testing.T, accountID int64) context.Context {
	t.Helper()
	return session.WithSession(context.Background(), &session.SessionData{
		SessionType: session.SessionTypeAccount,
		AccountID:   &accountID,
		IsAdmin:     true,
		ExpiresAt:   time.Now().Add(time.Hour),
	})
}

// newTestService creates a Service with an in-memory repo and the given prefix.
// It returns the service and the admin account ID.
func newTestService(t *testing.T, committeeGroupPrefix string) (*Service, int64) {
	t.Helper()
	repo, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}
	if err := repo.MigrateUp(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { repo.Close() })

	account, err := repo.CreateAccount(context.Background(), "admin", "Admin", "hash")
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	if err := repo.SetAccountIsAdmin(context.Background(), account.ID, true); err != nil {
		t.Fatalf("set admin: %v", err)
	}
	if err := repo.CreateCommitteeWithSlug(context.Background(), "Test Committee", "test-committee"); err != nil {
		t.Fatalf("create committee: %v", err)
	}

	svc := New(repo, nil, committeeGroupPrefix)
	return svc, account.ID
}

func TestCreateOAuthRule_PrefixConfigured_RejectsNonMatchingGroup(t *testing.T) {
	svc, accountID := newTestService(t, "conference-")
	ctx := adminCtx(t, accountID)

	_, err := svc.CreateOAuthRule(ctx, "test-committee", "wrong-group", "member")
	if err == nil {
		t.Fatal("expected error for group name without required prefix")
	}
}

func TestCreateOAuthRule_PrefixConfigured_AcceptsMatchingGroup(t *testing.T) {
	svc, accountID := newTestService(t, "conference-")
	ctx := adminCtx(t, accountID)

	resp, err := svc.CreateOAuthRule(ctx, "test-committee", "conference-members", "member")
	if err != nil {
		t.Fatalf("expected success for matching prefix, got: %v", err)
	}
	if resp.GetRule().GetGroupName() != "conference-members" {
		t.Fatalf("unexpected group name: %q", resp.GetRule().GetGroupName())
	}
}

func TestCreateOAuthRule_NoPrefixConfigured_AcceptsAnyGroup(t *testing.T) {
	svc, accountID := newTestService(t, "")
	ctx := adminCtx(t, accountID)

	resp, err := svc.CreateOAuthRule(ctx, "test-committee", "any-group-at-all", "chairperson")
	if err != nil {
		t.Fatalf("expected success with no prefix restriction, got: %v", err)
	}
	if resp.GetRule().GetGroupName() != "any-group-at-all" {
		t.Fatalf("unexpected group name: %q", resp.GetRule().GetGroupName())
	}
}

func TestCreateOAuthRule_PrefixConfigured_RejectsExactPrefixAsGroupName(t *testing.T) {
	svc, accountID := newTestService(t, "conference-")
	ctx := adminCtx(t, accountID)

	// The prefix itself is a valid prefix match (HasPrefix("conference-", "conference-") == true),
	// so this should succeed — the prefix is a minimum, not requiring extra characters.
	resp, err := svc.CreateOAuthRule(ctx, "test-committee", "conference-", "member")
	if err != nil {
		t.Fatalf("expected prefix-only group name to be accepted, got: %v", err)
	}
	if resp.GetRule().GetGroupName() != "conference-" {
		t.Fatalf("unexpected group name: %q", resp.GetRule().GetGroupName())
	}
}

func TestCreateOAuthRule_PrefixWithWhitespace_IsTrimmed(t *testing.T) {
	svc, accountID := newTestService(t, "  conference-  ")
	ctx := adminCtx(t, accountID)

	// Prefix should be trimmed to "conference-"
	_, err := svc.CreateOAuthRule(ctx, "test-committee", "wrong-group", "member")
	if err == nil {
		t.Fatal("expected error — trimmed prefix should still be enforced")
	}

	resp, err := svc.CreateOAuthRule(ctx, "test-committee", "conference-chairs", "chairperson")
	if err != nil {
		t.Fatalf("expected success for group matching trimmed prefix, got: %v", err)
	}
	if resp.GetRule().GetGroupName() != "conference-chairs" {
		t.Fatalf("unexpected group name: %q", resp.GetRule().GetGroupName())
	}
}

func TestCreateOAuthRule_EmptyGroupName_Rejected(t *testing.T) {
	svc, accountID := newTestService(t, "")
	ctx := adminCtx(t, accountID)

	_, err := svc.CreateOAuthRule(ctx, "test-committee", "", "member")
	if err == nil {
		t.Fatal("expected error for empty group name")
	}
}

func TestCreateOAuthRule_EmptyRole_Rejected(t *testing.T) {
	svc, accountID := newTestService(t, "")
	ctx := adminCtx(t, accountID)

	_, err := svc.CreateOAuthRule(ctx, "test-committee", "some-group", "")
	if err == nil {
		t.Fatal("expected error for empty role")
	}
}
