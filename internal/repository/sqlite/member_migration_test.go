package sqlite

import (
	"testing"
)

func TestMigrate035_AllowNonAccountMembers(t *testing.T) {
	repo := newTestRepo(t)

	if err := repo.MigrateVersion(34); err != nil {
		t.Fatalf("MigrateVersion(34) failed: %v", err)
	}

	// Insert test data with old schema (account_id NOT NULL).
	if _, err := repo.DB.Exec("INSERT INTO committees (name, slug) VALUES ('Test', 'test')"); err != nil {
		t.Fatalf("insert committee: %v", err)
	}
	if _, err := repo.DB.Exec("INSERT INTO accounts (username, full_name, auth_method) VALUES ('alice', 'Alice', 'password')"); err != nil {
		t.Fatalf("insert account: %v", err)
	}
	if _, err := repo.DB.Exec("INSERT INTO users (account_id, committee_id, role, quoted) VALUES (1, 1, 'chairperson', 0)"); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	// Run migration 035.
	if err := repo.MigrateVersion(35); err != nil {
		t.Fatalf("MigrateVersion(35) failed: %v", err)
	}

	// Existing member preserved with account_id.
	var accountID *int64
	if err := repo.DB.QueryRow("SELECT account_id FROM users WHERE id = 1").Scan(&accountID); err != nil {
		t.Fatalf("query existing member: %v", err)
	}
	if accountID == nil || *accountID != 1 {
		t.Fatalf("expected account_id=1, got %v", accountID)
	}

	// Can now insert email-only member (no account_id).
	if _, err := repo.DB.Exec(
		"INSERT INTO users (committee_id, email, full_name, role, quoted, invite_secret) VALUES (1, 'bob@example.com', 'Bob', 'member', 0, 'secret123')",
	); err != nil {
		t.Fatalf("insert email member: %v", err)
	}

	// Verify email member has NULL account_id.
	var emailAccountID *int64
	if err := repo.DB.QueryRow("SELECT account_id FROM users WHERE email = 'bob@example.com'").Scan(&emailAccountID); err != nil {
		t.Fatalf("query email member: %v", err)
	}
	if emailAccountID != nil {
		t.Fatalf("expected NULL account_id for email member, got %v", *emailAccountID)
	}

	// Verify unique constraint on (committee_id, email).
	_, err := repo.DB.Exec(
		"INSERT INTO users (committee_id, email, full_name, role, quoted) VALUES (1, 'bob@example.com', 'Bob2', 'member', 0)",
	)
	if err == nil {
		t.Fatal("expected unique constraint violation for duplicate email in same committee")
	}
}
