package sqlite

import (
	"testing"
)

func newTestRepo(t *testing.T) *Repository {
	t.Helper()
	repo, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	t.Cleanup(func() { repo.Close() })
	return repo
}

func TestMigrateUp(t *testing.T) {
	repo := newTestRepo(t)

	if err := repo.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp failed: %v", err)
	}

	// Verify all expected tables exist
	tables := []string{
		"committees",
		"users",
		"meetings",
		"agenda_points",
		"attendees",
		"speakers_list",
		"binary_blobs",
		"agenda_attachments",
		"motions",
	}

	for _, table := range tables {
		var name string
		err := repo.DB.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found after MigrateUp: %v", table, err)
		}
	}
}

func TestMigrateUpIdempotent(t *testing.T) {
	repo := newTestRepo(t)

	if err := repo.MigrateUp(); err != nil {
		t.Fatalf("first MigrateUp failed: %v", err)
	}

	if err := repo.MigrateUp(); err != nil {
		t.Fatalf("second MigrateUp should be idempotent: %v", err)
	}
}

func TestMigrateDown(t *testing.T) {
	repo := newTestRepo(t)

	if err := repo.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp failed: %v", err)
	}

	if err := repo.MigrateDown(); err != nil {
		t.Fatalf("MigrateDown failed: %v", err)
	}

	// Verify all application tables are gone
	// Verify only internal tables remain (schema_migrations, sqlite_sequence)
	rows, err := repo.DB.Query(
		"SELECT name FROM sqlite_master WHERE type='table' AND name NOT IN ('schema_migrations', 'sqlite_sequence')",
	)
	if err != nil {
		t.Fatalf("failed to query sqlite_master: %v", err)
	}
	defer rows.Close()

	var remaining []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		remaining = append(remaining, name)
	}

	if len(remaining) != 0 {
		t.Errorf("expected no application tables after MigrateDown, got: %v", remaining)
	}
}

func TestMigrateUpThenDown_CurrentReferences(t *testing.T) {
	repo := newTestRepo(t)

	if err := repo.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp failed: %v", err)
	}

	// Verify the ALTER TABLE columns from migration 006 exist
	columns := map[string]string{
		"committees":    "current_meeting_id",
		"meetings":      "current_agenda_point_id",
		"agenda_points": "current_speaker_id",
	}

	for table, col := range columns {
		var cid int
		err := repo.DB.QueryRow(
			"SELECT cid FROM pragma_table_info(?) WHERE name=?", table, col,
		).Scan(&cid)
		if err != nil {
			t.Errorf("column %s.%s not found: %v", table, col, err)
		}
	}
}

func TestMigrateUp_ForeignKeysEnforced(t *testing.T) {
	repo := newTestRepo(t)

	if err := repo.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp failed: %v", err)
	}

	// Inserting a user with a non-existent committee_id should fail
	_, err := repo.DB.Exec(
		"INSERT INTO users (committee_id, username, password_hash, full_name, gender, role) VALUES (999, 'test', 'hash', 'Test', 'm', 'member')",
	)
	if err == nil {
		t.Error("expected foreign key violation, got nil")
	}
}

func TestMigrateUp_CheckConstraints(t *testing.T) {
	repo := newTestRepo(t)

	if err := repo.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp failed: %v", err)
	}

	// Create a committee and user for FK satisfaction
	repo.DB.Exec("INSERT INTO committees (name) VALUES ('test')")
	repo.DB.Exec("INSERT INTO users (committee_id, username, password_hash, full_name, gender, role) VALUES (1, 'user1', 'hash', 'User One', 'm', 'member')")
	repo.DB.Exec("INSERT INTO meetings (committee_id, name) VALUES (1, 'meeting1')")
	repo.DB.Exec("INSERT INTO agenda_points (meeting_id, position, title) VALUES (1, 1, 'point1')")

	// Invalid gender should fail
	_, err := repo.DB.Exec(
		"INSERT INTO users (committee_id, username, password_hash, full_name, gender, role) VALUES (1, 'bad', 'hash', 'Bad', 'x', 'member')",
	)
	if err == nil {
		t.Error("expected CHECK constraint violation for invalid gender")
	}

	// Invalid speakers_list type should fail
	_, err = repo.DB.Exec(
		"INSERT INTO speakers_list (agenda_point_id, user_id, type) VALUES (1, 1, 'invalid')",
	)
	if err == nil {
		t.Error("expected CHECK constraint violation for invalid speakers_list type")
	}

	// speakers_list start_of_speech set without duration should fail
	_, err = repo.DB.Exec(
		"INSERT INTO speakers_list (agenda_point_id, user_id, type, start_of_speech) VALUES (1, 1, 'regular', '2025-01-01T00:00:00Z')",
	)
	if err == nil {
		t.Error("expected CHECK constraint violation for start_of_speech without duration")
	}

	// Invalid speakers_list status should fail
	_, err = repo.DB.Exec(
		"INSERT INTO speakers_list (agenda_point_id, user_id, type, status) VALUES (1, 1, 'regular', 'INVALID')",
	)
	if err == nil {
		t.Error("expected CHECK constraint violation for invalid status")
	}

	// Motions with partial vote fields should fail
	_, err = repo.DB.Exec(
		"INSERT INTO binary_blobs (filename, content_type, size_bytes, storage_path) VALUES ('f.pdf', 'application/pdf', 100, '/blobs/1')",
	)
	if err != nil {
		t.Fatalf("failed to insert binary_blob: %v", err)
	}

	_, err = repo.DB.Exec(
		"INSERT INTO motions (agenda_point_id, blob_id, title, votes_for) VALUES (1, 1, 'motion1', 5)",
	)
	if err == nil {
		t.Error("expected CHECK constraint violation for partial vote fields on motion")
	}
}
