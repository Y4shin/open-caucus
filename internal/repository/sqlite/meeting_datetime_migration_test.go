package sqlite

import (
	"testing"
)

func TestMigrate033_AddsMeetingDatetimeColumns(t *testing.T) {
	repo := newTestRepo(t)

	if err := repo.MigrateVersion(32); err != nil {
		t.Fatalf("MigrateVersion(32) failed: %v", err)
	}

	// Insert a committee and meeting before migration 033.
	if _, err := repo.DB.Exec("INSERT INTO committees (name, slug) VALUES ('Test', 'test')"); err != nil {
		t.Fatalf("insert committee: %v", err)
	}
	if _, err := repo.DB.Exec("INSERT INTO meetings (committee_id, name, secret, signup_open) VALUES (1, 'Pre-033', 'secret', 0)"); err != nil {
		t.Fatalf("insert meeting: %v", err)
	}

	if err := repo.MigrateVersion(33); err != nil {
		t.Fatalf("MigrateVersion(33) failed: %v", err)
	}

	// Verify columns exist and are NULL for existing meetings.
	var startAt, endAt *string
	err := repo.DB.QueryRow("SELECT start_at, end_at FROM meetings WHERE name = 'Pre-033'").Scan(&startAt, &endAt)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if startAt != nil {
		t.Fatalf("expected NULL start_at for existing meeting, got %q", *startAt)
	}
	if endAt != nil {
		t.Fatalf("expected NULL end_at for existing meeting, got %q", *endAt)
	}

	// Verify new meetings can use the columns.
	if _, err := repo.DB.Exec(
		"INSERT INTO meetings (committee_id, name, secret, signup_open, start_at, end_at) VALUES (1, 'With-Times', 'secret2', 0, '2026-05-15T14:00:00Z', '2026-05-15T16:00:00Z')",
	); err != nil {
		t.Fatalf("insert meeting with datetimes: %v", err)
	}

	var gotStart, gotEnd string
	err = repo.DB.QueryRow("SELECT start_at, end_at FROM meetings WHERE name = 'With-Times'").Scan(&gotStart, &gotEnd)
	if err != nil {
		t.Fatalf("query with-times: %v", err)
	}
	if gotStart != "2026-05-15T14:00:00Z" {
		t.Fatalf("expected start_at '2026-05-15T14:00:00Z', got %q", gotStart)
	}
	if gotEnd != "2026-05-15T16:00:00Z" {
		t.Fatalf("expected end_at '2026-05-15T16:00:00Z', got %q", gotEnd)
	}
}
