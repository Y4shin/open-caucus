package sqlite

import (
	"context"
	"testing"

	"github.com/Y4shin/open-caucus/internal/repository/model"
)

func TestSyncOAuthCommitteeMemberships_RemovesOnlyOAuthManaged(t *testing.T) {
	repo := newTestRepo(t)
	if err := repo.MigrateUp(); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	ctx := context.Background()
	account, err := repo.CreateOAuthAccount(ctx, "oauth-user", "OAuth User")
	if err != nil {
		t.Fatalf("create oauth account: %v", err)
	}

	manualCommitteeID := seedCommitteeID(t, repo, "Manual Committee", "manual-committee")
	managedCommitteeID := seedCommitteeID(t, repo, "Managed Committee", "managed-committee")

	manualUserID := seedMembership(t, repo, account.ID, manualCommitteeID, "member")
	managedUserID := seedMembership(t, repo, account.ID, managedCommitteeID, "member")
	seedOAuthManagedMembership(t, repo, managedUserID)

	if err := repo.SyncOAuthCommitteeMemberships(ctx, account.ID, nil); err != nil {
		t.Fatalf("sync oauth memberships: %v", err)
	}

	if !membershipExists(t, repo, manualUserID) {
		t.Fatalf("manual membership should remain after sync")
	}
	if membershipExists(t, repo, managedUserID) {
		t.Fatalf("oauth-managed membership should be deleted when no longer desired")
	}
}

func TestSyncOAuthCommitteeMemberships_CreatesManagedAndPromotesManualWithoutManagingIt(t *testing.T) {
	repo := newTestRepo(t)
	if err := repo.MigrateUp(); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	ctx := context.Background()
	account, err := repo.CreateOAuthAccount(ctx, "oauth-user2", "OAuth User Two")
	if err != nil {
		t.Fatalf("create oauth account: %v", err)
	}

	managedCommitteeID := seedCommitteeID(t, repo, "Managed Committee", "managed-committee-2")
	manualCommitteeID := seedCommitteeID(t, repo, "Manual Committee", "manual-committee-2")
	manualUserID := seedMembership(t, repo, account.ID, manualCommitteeID, "member")

	if err := repo.SyncOAuthCommitteeMemberships(ctx, account.ID, []model.OAuthDesiredMembership{
		{CommitteeID: managedCommitteeID, Role: "member"},
		{CommitteeID: manualCommitteeID, Role: "chairperson"},
	}); err != nil {
		t.Fatalf("first sync oauth memberships: %v", err)
	}

	managedUserID, managedRole := findMembershipByCommittee(t, repo, account.ID, managedCommitteeID)
	if managedRole != "member" {
		t.Fatalf("expected managed membership role member, got=%q", managedRole)
	}
	if !oauthManagedMarkerExists(t, repo, managedUserID) {
		t.Fatalf("expected managed membership marker to exist")
	}

	manualRole := membershipRole(t, repo, manualUserID)
	if manualRole != "chairperson" {
		t.Fatalf("manual membership role should be promoted to chairperson, got=%q", manualRole)
	}
	if oauthManagedMarkerExists(t, repo, manualUserID) {
		t.Fatalf("manual membership should not become oauth-managed")
	}

	if err := repo.SyncOAuthCommitteeMemberships(ctx, account.ID, []model.OAuthDesiredMembership{
		{CommitteeID: managedCommitteeID, Role: "chairperson"},
		{CommitteeID: manualCommitteeID, Role: "chairperson"},
	}); err != nil {
		t.Fatalf("second sync oauth memberships: %v", err)
	}

	_, managedRole = findMembershipByCommittee(t, repo, account.ID, managedCommitteeID)
	if managedRole != "chairperson" {
		t.Fatalf("expected managed membership role to update to chairperson, got=%q", managedRole)
	}

	manualRole = membershipRole(t, repo, manualUserID)
	if manualRole != "chairperson" {
		t.Fatalf("manual membership should not be downgraded by oauth sync, got=%q", manualRole)
	}
}

func TestUpsertOAuthIdentity_ReconcilesIssuerChangeForSameAccount(t *testing.T) {
	repo := newTestRepo(t)
	if err := repo.MigrateUp(); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	ctx := context.Background()
	account, err := repo.CreateOAuthAccount(ctx, "oauth-user3", "OAuth User Three")
	if err != nil {
		t.Fatalf("create oauth account: %v", err)
	}

	username := "oauth-user3"
	fullName := "OAuth User Three"
	email := "oauth-user3@example.test"
	groupsJSON := `["committee-a-chair"]`

	firstIdentity, err := repo.UpsertOAuthIdentity(
		ctx,
		"http://127.0.0.1:9096",
		"id1",
		account.ID,
		&username,
		&fullName,
		&email,
		&groupsJSON,
	)
	if err != nil {
		t.Fatalf("first upsert oauth identity: %v", err)
	}

	secondIdentity, err := repo.UpsertOAuthIdentity(
		ctx,
		"http://localhost:9096",
		"id1",
		account.ID,
		&username,
		&fullName,
		&email,
		&groupsJSON,
	)
	if err != nil {
		t.Fatalf("second upsert oauth identity after issuer change: %v", err)
	}

	if secondIdentity.ID != firstIdentity.ID {
		t.Fatalf("expected identity row to be updated in place, got first=%d second=%d", firstIdentity.ID, secondIdentity.ID)
	}
	if secondIdentity.Issuer != "http://localhost:9096" {
		t.Fatalf("expected issuer to be updated, got %q", secondIdentity.Issuer)
	}

	refreshed, err := repo.GetOAuthIdentityByIssuerSubject(ctx, "http://localhost:9096", "id1")
	if err != nil {
		t.Fatalf("get refreshed oauth identity: %v", err)
	}
	if refreshed.AccountID != account.ID {
		t.Fatalf("expected refreshed identity account id %d, got %d", account.ID, refreshed.AccountID)
	}

	var count int
	if err := repo.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM oauth_identities WHERE account_id = ?`, account.ID).Scan(&count); err != nil {
		t.Fatalf("count oauth identities by account id: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one oauth identity row for account, got %d", count)
	}
}

func seedCommitteeID(t *testing.T, repo *Repository, name, slug string) int64 {
	t.Helper()
	if err := repo.CreateCommitteeWithSlug(context.Background(), name, slug); err != nil {
		t.Fatalf("create committee %q: %v", slug, err)
	}
	id, err := repo.GetCommitteeIDBySlug(context.Background(), slug)
	if err != nil {
		t.Fatalf("get committee id %q: %v", slug, err)
	}
	return id
}

func seedMembership(t *testing.T, repo *Repository, accountID, committeeID int64, role string) int64 {
	t.Helper()
	var userID int64
	if err := repo.DB.QueryRow(
		`INSERT INTO users (account_id, committee_id, role, quoted, created_at, updated_at)
		 VALUES (?, ?, ?, 0, datetime('now'), datetime('now'))
		 RETURNING id`,
		accountID, committeeID, role,
	).Scan(&userID); err != nil {
		t.Fatalf("insert membership: %v", err)
	}
	return userID
}

func seedOAuthManagedMembership(t *testing.T, repo *Repository, userID int64) {
	t.Helper()
	if _, err := repo.DB.Exec(
		`INSERT INTO oauth_managed_memberships (user_id, last_synced_at)
		 VALUES (?, datetime('now'))`,
		userID,
	); err != nil {
		t.Fatalf("insert oauth managed membership marker: %v", err)
	}
}

func membershipExists(t *testing.T, repo *Repository, userID int64) bool {
	t.Helper()
	var count int64
	if err := repo.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE id = ?`, userID).Scan(&count); err != nil {
		t.Fatalf("query membership exists: %v", err)
	}
	return count == 1
}

func oauthManagedMarkerExists(t *testing.T, repo *Repository, userID int64) bool {
	t.Helper()
	var count int64
	if err := repo.DB.QueryRow(`SELECT COUNT(*) FROM oauth_managed_memberships WHERE user_id = ?`, userID).Scan(&count); err != nil {
		t.Fatalf("query oauth managed marker: %v", err)
	}
	return count == 1
}

func membershipRole(t *testing.T, repo *Repository, userID int64) string {
	t.Helper()
	var role string
	if err := repo.DB.QueryRow(`SELECT role FROM users WHERE id = ?`, userID).Scan(&role); err != nil {
		t.Fatalf("query membership role for user_id=%d: %v", userID, err)
	}
	return role
}

func findMembershipByCommittee(t *testing.T, repo *Repository, accountID, committeeID int64) (int64, string) {
	t.Helper()
	var (
		userID int64
		role   string
	)
	if err := repo.DB.QueryRow(
		`SELECT id, role
		   FROM users
		  WHERE account_id = ? AND committee_id = ?`,
		accountID,
		committeeID,
	).Scan(&userID, &role); err != nil {
		t.Fatalf("query membership by committee: %v", err)
	}
	return userID, role
}
