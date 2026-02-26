package sqlite

import (
	"context"
	"testing"

	"github.com/Y4shin/conference-tool/internal/repository"
)

func seedMeetingForAgendaTests(t *testing.T, repo *Repository) int64 {
	t.Helper()
	if _, err := repo.DB.Exec("INSERT INTO committees (name, slug) VALUES ('Test Committee', 'test-committee')"); err != nil {
		t.Fatalf("insert committee: %v", err)
	}
	if _, err := repo.DB.Exec("INSERT INTO meetings (committee_id, name, secret, signup_open) VALUES (1, 'Test Meeting', 'secret-1', 0)"); err != nil {
		t.Fatalf("insert meeting: %v", err)
	}
	return 1
}

func TestMoveAgendaPointUpAndDown(t *testing.T) {
	repo := newTestRepo(t)
	if err := repo.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp failed: %v", err)
	}
	meetingID := seedMeetingForAgendaTests(t, repo)

	a, err := repo.CreateAgendaPoint(context.Background(), meetingID, "A")
	if err != nil {
		t.Fatalf("create A: %v", err)
	}
	b, err := repo.CreateAgendaPoint(context.Background(), meetingID, "B")
	if err != nil {
		t.Fatalf("create B: %v", err)
	}
	_, err = repo.CreateAgendaPoint(context.Background(), meetingID, "C")
	if err != nil {
		t.Fatalf("create C: %v", err)
	}

	if err := repo.MoveAgendaPointUp(context.Background(), meetingID, b.ID); err != nil {
		t.Fatalf("move up B: %v", err)
	}

	top, err := repo.ListAgendaPointsForMeeting(context.Background(), meetingID)
	if err != nil {
		t.Fatalf("list top-level: %v", err)
	}
	if len(top) != 3 || top[0].Title != "B" || top[1].Title != "A" || top[2].Title != "C" {
		t.Fatalf("unexpected top-level order after move up: %+v", []string{top[0].Title, top[1].Title, top[2].Title})
	}

	if err := repo.MoveAgendaPointUp(context.Background(), meetingID, top[0].ID); err != nil {
		t.Fatalf("move up first should no-op: %v", err)
	}

	if err := repo.MoveAgendaPointDown(context.Background(), meetingID, a.ID); err != nil {
		t.Fatalf("move down A: %v", err)
	}
	top, err = repo.ListAgendaPointsForMeeting(context.Background(), meetingID)
	if err != nil {
		t.Fatalf("list top-level after move down: %v", err)
	}
	if len(top) != 3 || top[0].Title != "B" || top[1].Title != "C" || top[2].Title != "A" {
		t.Fatalf("unexpected top-level order after move down: %+v", []string{top[0].Title, top[1].Title, top[2].Title})
	}
}

func TestApplyAgendaPoints_ReplacesAgendaStructure(t *testing.T) {
	repo := newTestRepo(t)
	if err := repo.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp failed: %v", err)
	}
	meetingID := seedMeetingForAgendaTests(t, repo)

	apA, err := repo.CreateAgendaPoint(context.Background(), meetingID, "A")
	if err != nil {
		t.Fatalf("create A: %v", err)
	}
	apC, err := repo.CreateAgendaPoint(context.Background(), meetingID, "C")
	if err != nil {
		t.Fatalf("create C: %v", err)
	}

	points := []repository.AgendaApplyPoint{
		{Key: "k-a", ExistingID: &apA.ID, Title: "A", Position: 1},
		{Key: "k-b", Title: "B", Position: 2},
		{Key: "k-c", ExistingID: &apC.ID, Title: "C", Position: 3},
	}
	if err := repo.ApplyAgendaPoints(context.Background(), meetingID, points, nil); err != nil {
		t.Fatalf("apply points: %v", err)
	}

	top, err := repo.ListAgendaPointsForMeeting(context.Background(), meetingID)
	if err != nil {
		t.Fatalf("list top-level: %v", err)
	}
	if len(top) != 3 {
		t.Fatalf("expected 3 points after apply, got=%d", len(top))
	}
	if top[0].Title != "A" || top[1].Title != "B" || top[2].Title != "C" {
		t.Fatalf("unexpected order/titles: %v, %v, %v", top[0].Title, top[1].Title, top[2].Title)
	}
}
