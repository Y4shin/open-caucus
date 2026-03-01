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
		"accounts",
		"committees",
		"users",
		"meetings",
		"agenda_points",
		"attendees",
		"speakers_list",
		"binary_blobs",
		"agenda_attachments",
		"vote_definitions",
		"vote_options",
		"eligible_voters",
		"vote_casts",
		"vote_ballots",
		"vote_ballot_selections",
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

	var motionsTable string
	err := repo.DB.QueryRow(
		"SELECT name FROM sqlite_master WHERE type='table' AND name='motions'",
	).Scan(&motionsTable)
	if err == nil {
		t.Errorf("table %q should not exist after latest migration", motionsTable)
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

func TestMigrate027To028_PreservesVoteDefinitionsAndAllowsCounting(t *testing.T) {
	repo := newTestRepo(t)

	if err := repo.MigrateVersion(27); err != nil {
		t.Fatalf("MigrateVersion(27) failed: %v", err)
	}

	if _, err := repo.DB.Exec("INSERT INTO accounts (username) VALUES ('user1')"); err != nil {
		t.Fatalf("insert account: %v", err)
	}
	if _, err := repo.DB.Exec("INSERT INTO committees (name, slug) VALUES ('committee', 'committee')"); err != nil {
		t.Fatalf("insert committee: %v", err)
	}
	if _, err := repo.DB.Exec("INSERT INTO users (account_id, committee_id, role) VALUES (1, 1, 'member')"); err != nil {
		t.Fatalf("insert user membership: %v", err)
	}
	if _, err := repo.DB.Exec("INSERT INTO meetings (committee_id, name, secret, signup_open) VALUES (1, 'meeting1', 'secret1', 0)"); err != nil {
		t.Fatalf("insert meeting: %v", err)
	}
	if _, err := repo.DB.Exec("INSERT INTO agenda_points (meeting_id, position, title) VALUES (1, 1, 'point1')"); err != nil {
		t.Fatalf("insert agenda point: %v", err)
	}
	if _, err := repo.DB.Exec("INSERT INTO binary_blobs (filename, content_type, size_bytes, storage_path) VALUES ('f.pdf', 'application/pdf', 100, '/blobs/1')"); err != nil {
		t.Fatalf("insert binary blob: %v", err)
	}
	if _, err := repo.DB.Exec("INSERT INTO motions (agenda_point_id, blob_id, title) VALUES (1, 1, 'motion1')"); err != nil {
		t.Fatalf("insert motion: %v", err)
	}
	if _, err := repo.DB.Exec("INSERT INTO vote_definitions (meeting_id, agenda_point_id, motion_id, name, visibility, state, min_selections, max_selections) VALUES (1, 1, 1, 'vote-before-028', 'secret', 'open', 0, 1)"); err != nil {
		t.Fatalf("insert vote_definition before migration 028: %v", err)
	}

	if err := repo.MigrateVersion(28); err != nil {
		t.Fatalf("MigrateVersion(28) failed: %v", err)
	}

	var preservedState string
	if err := repo.DB.QueryRow("SELECT state FROM vote_definitions WHERE name = 'vote-before-028'").Scan(&preservedState); err != nil {
		t.Fatalf("load vote_definition after migration 028: %v", err)
	}
	if preservedState != "open" {
		t.Fatalf("expected preserved state 'open', got %q", preservedState)
	}

	if _, err := repo.DB.Exec("INSERT INTO vote_definitions (meeting_id, agenda_point_id, motion_id, name, visibility, state, min_selections, max_selections) VALUES (1, 1, 1, 'vote-counting-028', 'secret', 'counting', 0, 1)"); err != nil {
		t.Fatalf("insert vote_definition with counting state after migration 028: %v", err)
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

	// Create an account first, then try inserting a membership with a non-existent committee_id.
	_, err := repo.DB.Exec(
		"INSERT INTO accounts (username) VALUES ('test')",
	)
	if err != nil {
		t.Fatalf("failed to insert test account: %v", err)
	}
	_, err = repo.DB.Exec(
		"INSERT INTO users (account_id, committee_id, role) VALUES (1, 999, 'member')",
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

	// Create accounts, committee, membership, meeting, and agenda point for FK satisfaction
	if _, err := repo.DB.Exec("INSERT INTO accounts (username) VALUES ('user1')"); err != nil {
		t.Fatalf("insert account user1: %v", err)
	}
	if _, err := repo.DB.Exec("INSERT INTO committees (name, slug) VALUES ('test', 'test')"); err != nil {
		t.Fatalf("insert committee: %v", err)
	}
	if _, err := repo.DB.Exec("INSERT INTO users (account_id, committee_id, role) VALUES (1, 1, 'member')"); err != nil {
		t.Fatalf("insert user membership: %v", err)
	}
	if _, err := repo.DB.Exec("INSERT INTO meetings (committee_id, name, secret, signup_open) VALUES (1, 'meeting1', 'secret1', 0)"); err != nil {
		t.Fatalf("insert meeting: %v", err)
	}
	if _, err := repo.DB.Exec("INSERT INTO agenda_points (meeting_id, position, title) VALUES (1, 1, 'point1')"); err != nil {
		t.Fatalf("insert agenda point: %v", err)
	}

	// Invalid role should fail
	if _, err := repo.DB.Exec("INSERT INTO accounts (username) VALUES ('bad')"); err != nil {
		t.Fatalf("insert bad account: %v", err)
	}
	_, err := repo.DB.Exec(
		"INSERT INTO users (account_id, committee_id, role) VALUES (2, 1, 'admin')",
	)
	if err == nil {
		t.Error("expected CHECK constraint violation for invalid role")
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

	_, err = repo.DB.Exec(
		`INSERT INTO vote_definitions (meeting_id, agenda_point_id, name, visibility, state, min_selections, max_selections)
		 VALUES (
		     (SELECT id FROM meetings ORDER BY id LIMIT 1),
		     (SELECT id FROM agenda_points ORDER BY id LIMIT 1),
		     'vote1',
		     'open',
		     'open',
		     1,
		     2
		 )`,
	)
	if err != nil {
		t.Fatalf("failed to insert vote_definition: %v", err)
	}
	_, err = repo.DB.Exec(
		`INSERT INTO vote_definitions (meeting_id, agenda_point_id, name, visibility, state, min_selections, max_selections)
		 VALUES (
		     (SELECT id FROM meetings ORDER BY id LIMIT 1),
		     (SELECT id FROM agenda_points ORDER BY id LIMIT 1),
		     'vote2',
		     'secret',
		     'counting',
		     0,
		     1
		 )`,
	)
	if err != nil {
		t.Fatalf("failed to insert vote_definition in counting state: %v", err)
	}
	_, err = repo.DB.Exec(
		`INSERT INTO attendees (meeting_id, full_name, secret, quoted, attendee_number)
		 VALUES ((SELECT id FROM meetings ORDER BY id LIMIT 1), 'A', 'secret-a', 0, 1)`,
	)
	if err != nil {
		t.Fatalf("failed to insert attendee: %v", err)
	}
	_, err = repo.DB.Exec(
		`INSERT INTO eligible_voters (vote_definition_id, meeting_id, attendee_id)
		 VALUES (
		     (SELECT id FROM vote_definitions ORDER BY id LIMIT 1),
		     (SELECT id FROM meetings ORDER BY id LIMIT 1),
		     (SELECT id FROM attendees ORDER BY id LIMIT 1)
		 )`,
	)
	if err != nil {
		t.Fatalf("failed to insert eligible_voter: %v", err)
	}
	_, err = repo.DB.Exec(
		`INSERT INTO vote_casts (vote_definition_id, meeting_id, attendee_id, source)
		 VALUES (
		     (SELECT id FROM vote_definitions ORDER BY id LIMIT 1),
		     (SELECT id FROM meetings ORDER BY id LIMIT 1),
		     (SELECT id FROM attendees ORDER BY id LIMIT 1),
		     'self_submission'
		 )`,
	)
	if err != nil {
		t.Fatalf("failed to insert vote_cast: %v", err)
	}

	// Open ballot branch with commitment data should fail.
	_, err = repo.DB.Exec(
		`INSERT INTO vote_ballots (vote_definition_id, cast_id, attendee_id, receipt_token, encrypted_commitment, commitment_cipher, commitment_version)
		 VALUES (
		     (SELECT id FROM vote_definitions ORDER BY id LIMIT 1),
		     (SELECT id FROM vote_casts ORDER BY id LIMIT 1),
		     (SELECT id FROM attendees ORDER BY id LIMIT 1),
		     'r1',
		     x'01',
		     'aes',
		     1
		 )`,
	)
	if err == nil {
		t.Error("expected CHECK constraint violation for open ballot branch with commitment columns set")
	}

	// Secret ballot branch without commitment should fail.
	_, err = repo.DB.Exec(
		`INSERT INTO vote_ballots (vote_definition_id, receipt_token)
		 VALUES ((SELECT id FROM vote_definitions ORDER BY id LIMIT 1), 'r2')`,
	)
	if err == nil {
		t.Error("expected CHECK constraint violation for secret ballot branch without commitment columns")
	}
}
