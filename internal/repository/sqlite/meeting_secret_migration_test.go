package sqlite

import (
	"testing"
)

func TestMigrate032_BackfillsMeetingSecrets(t *testing.T) {
	repo := newTestRepo(t)

	if err := repo.MigrateVersion(31); err != nil {
		t.Fatalf("MigrateVersion(31) failed: %v", err)
	}

	// Insert a committee and meetings with empty/NULL secrets.
	if _, err := repo.DB.Exec("INSERT INTO committees (name, slug) VALUES ('Test Committee', 'test')"); err != nil {
		t.Fatalf("insert committee: %v", err)
	}
	if _, err := repo.DB.Exec("INSERT INTO meetings (committee_id, name, secret, signup_open) VALUES (1, 'Empty Secret', '', 0)"); err != nil {
		t.Fatalf("insert meeting with empty secret: %v", err)
	}
	if _, err := repo.DB.Exec("INSERT INTO meetings (committee_id, name, secret, signup_open) VALUES (1, 'Has Secret', 'existing-secret', 0)"); err != nil {
		t.Fatalf("insert meeting with existing secret: %v", err)
	}

	// Run migration 032.
	if err := repo.MigrateVersion(32); err != nil {
		t.Fatalf("MigrateVersion(32) failed: %v", err)
	}

	// Verify previously-empty secret is now populated.
	var emptySecretAfter string
	if err := repo.DB.QueryRow("SELECT secret FROM meetings WHERE name = 'Empty Secret'").Scan(&emptySecretAfter); err != nil {
		t.Fatalf("query empty secret meeting: %v", err)
	}
	if emptySecretAfter == "" {
		t.Fatal("expected backfilled secret for 'Empty Secret' meeting, still empty")
	}
	if len(emptySecretAfter) != 32 {
		t.Fatalf("expected 32-char hex secret, got %d chars: %q", len(emptySecretAfter), emptySecretAfter)
	}

	// Verify existing secret is preserved.
	var existingSecretAfter string
	if err := repo.DB.QueryRow("SELECT secret FROM meetings WHERE name = 'Has Secret'").Scan(&existingSecretAfter); err != nil {
		t.Fatalf("query existing secret meeting: %v", err)
	}
	if existingSecretAfter != "existing-secret" {
		t.Fatalf("expected preserved secret 'existing-secret', got %q", existingSecretAfter)
	}
}

func TestMigrate032_BackfilledSecretsAreUnique(t *testing.T) {
	repo := newTestRepo(t)

	if err := repo.MigrateVersion(31); err != nil {
		t.Fatalf("MigrateVersion(31) failed: %v", err)
	}

	if _, err := repo.DB.Exec("INSERT INTO committees (name, slug) VALUES ('Test Committee', 'test')"); err != nil {
		t.Fatalf("insert committee: %v", err)
	}

	// Insert multiple meetings with empty secrets.
	for i := 0; i < 5; i++ {
		if _, err := repo.DB.Exec("INSERT INTO meetings (committee_id, name, secret, signup_open) VALUES (1, ?, '', 0)", i); err != nil {
			t.Fatalf("insert meeting %d: %v", i, err)
		}
	}

	if err := repo.MigrateVersion(32); err != nil {
		t.Fatalf("MigrateVersion(32) failed: %v", err)
	}

	rows, err := repo.DB.Query("SELECT secret FROM meetings")
	if err != nil {
		t.Fatalf("query secrets: %v", err)
	}
	defer rows.Close()

	seen := make(map[string]bool)
	for rows.Next() {
		var secret string
		if err := rows.Scan(&secret); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if secret == "" {
			t.Fatal("found empty secret after migration")
		}
		if seen[secret] {
			t.Fatalf("duplicate secret found: %q", secret)
		}
		seen[secret] = true
	}
}
