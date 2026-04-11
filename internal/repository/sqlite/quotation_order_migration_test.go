package sqlite

import (
	"testing"
)

func TestMigrate034_QuotationOrderMigration(t *testing.T) {
	repo := newTestRepo(t)

	if err := repo.MigrateVersion(33); err != nil {
		t.Fatalf("MigrateVersion(33) failed: %v", err)
	}

	// Insert test data with old boolean columns.
	if _, err := repo.DB.Exec("INSERT INTO committees (name, slug) VALUES ('Test', 'test')"); err != nil {
		t.Fatalf("insert committee: %v", err)
	}

	// Meeting with both quotation types enabled (default).
	if _, err := repo.DB.Exec("INSERT INTO meetings (committee_id, name, secret, signup_open, gender_quotation_enabled, first_speaker_quotation_enabled) VALUES (1, 'Both', 'secret', 0, 1, 1)"); err != nil {
		t.Fatalf("insert meeting both: %v", err)
	}
	// Meeting with only gender enabled.
	if _, err := repo.DB.Exec("INSERT INTO meetings (committee_id, name, secret, signup_open, gender_quotation_enabled, first_speaker_quotation_enabled) VALUES (1, 'GenderOnly', 'secret', 0, 1, 0)"); err != nil {
		t.Fatalf("insert meeting gender-only: %v", err)
	}
	// Meeting with both disabled.
	if _, err := repo.DB.Exec("INSERT INTO meetings (committee_id, name, secret, signup_open, gender_quotation_enabled, first_speaker_quotation_enabled) VALUES (1, 'None', 'secret', 0, 0, 0)"); err != nil {
		t.Fatalf("insert meeting none: %v", err)
	}

	// Run migration 034.
	if err := repo.MigrateVersion(34); err != nil {
		t.Fatalf("MigrateVersion(34) failed: %v", err)
	}

	// Verify meeting "Both" has '["gender","first_speaker"]'.
	var bothOrder string
	if err := repo.DB.QueryRow("SELECT quotation_order FROM meetings WHERE name = 'Both'").Scan(&bothOrder); err != nil {
		t.Fatalf("query both: %v", err)
	}
	if bothOrder != `["gender","first_speaker"]` {
		t.Fatalf("expected both order '[\"gender\",\"first_speaker\"]', got %q", bothOrder)
	}

	// Verify meeting "GenderOnly" has '["gender"]'.
	var genderOrder string
	if err := repo.DB.QueryRow("SELECT quotation_order FROM meetings WHERE name = 'GenderOnly'").Scan(&genderOrder); err != nil {
		t.Fatalf("query gender: %v", err)
	}
	if genderOrder != `["gender"]` {
		t.Fatalf("expected gender order '[\"gender\"]', got %q", genderOrder)
	}

	// Verify meeting "None" has '[]'.
	var noneOrder string
	if err := repo.DB.QueryRow("SELECT quotation_order FROM meetings WHERE name = 'None'").Scan(&noneOrder); err != nil {
		t.Fatalf("query none: %v", err)
	}
	if noneOrder != `[]` {
		t.Fatalf("expected none order '[]', got %q", noneOrder)
	}

	// Verify old columns are gone.
	var dummy string
	err := repo.DB.QueryRow("SELECT gender_quotation_enabled FROM meetings LIMIT 1").Scan(&dummy)
	if err == nil {
		t.Fatal("expected gender_quotation_enabled column to be dropped")
	}
}
