package apiconnect

import (
	"context"
	"testing"

	connect "connectrpc.com/connect"

	committeesv1 "github.com/Y4shin/open-caucus/gen/go/conference/committees/v1"
	sessionv1 "github.com/Y4shin/open-caucus/gen/go/conference/session/v1"
)

func TestCreateMeeting_WithDatetime(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "DateTime Committee", "dt-comm")
	ts.seedUser(t, "dt-comm", "chair1", "pass123", "Chair One", "chairperson")

	client := newCombinedTestClient(t, ts)
	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	startAt := "2026-05-15T14:00:00Z"
	endAt := "2026-05-15T16:00:00Z"

	createResp, err := client.committees.CreateMeeting(context.Background(), connect.NewRequest(&committeesv1.CreateMeetingRequest{
		CommitteeSlug: "dt-comm",
		Name:          "Timed Meeting",
		StartAt:       &startAt,
		EndAt:         &endAt,
	}))
	if err != nil {
		t.Fatalf("create meeting: %v", err)
	}

	ref := createResp.Msg.GetMeeting()
	if ref.StartAt == nil || *ref.StartAt != startAt {
		t.Fatalf("expected start_at %q, got %v", startAt, ref.StartAt)
	}
	if ref.EndAt == nil || *ref.EndAt != endAt {
		t.Fatalf("expected end_at %q, got %v", endAt, ref.EndAt)
	}

	// Verify via repo as well.
	meetings, err := ts.repo.ListMeetingsForCommittee(context.Background(), "dt-comm", 1, 0)
	if err != nil || len(meetings) == 0 {
		t.Fatalf("list meetings: %v", err)
	}
	m := meetings[0]
	if m.StartAt == nil {
		t.Fatal("expected non-nil StartAt in model")
	}
	if m.EndAt == nil {
		t.Fatal("expected non-nil EndAt in model")
	}
	if m.StartAt.UTC().Format("2006-01-02T15:04:05Z") != startAt {
		t.Fatalf("model start_at mismatch: got %v", m.StartAt)
	}
}

func TestCreateMeeting_WithoutDatetime(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "NoDT Committee", "nodt-comm")
	ts.seedUser(t, "nodt-comm", "chair1", "pass123", "Chair One", "chairperson")

	client := newCombinedTestClient(t, ts)
	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	createResp, err := client.committees.CreateMeeting(context.Background(), connect.NewRequest(&committeesv1.CreateMeetingRequest{
		CommitteeSlug: "nodt-comm",
		Name:          "No Time Meeting",
	}))
	if err != nil {
		t.Fatalf("create meeting: %v", err)
	}

	ref := createResp.Msg.GetMeeting()
	if ref.StartAt != nil {
		t.Fatalf("expected nil start_at, got %q", *ref.StartAt)
	}
	if ref.EndAt != nil {
		t.Fatalf("expected nil end_at, got %q", *ref.EndAt)
	}
}

func TestCreateMeeting_EndBeforeStartFails(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Invalid Committee", "inv-comm")
	ts.seedUser(t, "inv-comm", "chair1", "pass123", "Chair One", "chairperson")

	client := newCombinedTestClient(t, ts)
	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	startAt := "2026-05-15T16:00:00Z"
	endAt := "2026-05-15T14:00:00Z" // before start

	_, err := client.committees.CreateMeeting(context.Background(), connect.NewRequest(&committeesv1.CreateMeetingRequest{
		CommitteeSlug: "inv-comm",
		Name:          "Bad Meeting",
		StartAt:       &startAt,
		EndAt:         &endAt,
	}))
	if err == nil {
		t.Fatal("expected error when end is before start")
	}
}

func TestGetCommitteeOverview_IncludesDatetime(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Overview Committee", "ov-comm")
	ts.seedUser(t, "ov-comm", "chair1", "pass123", "Chair One", "chairperson")

	client := newCombinedTestClient(t, ts)
	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	startAt := "2026-06-01T10:00:00Z"
	if _, err := client.committees.CreateMeeting(context.Background(), connect.NewRequest(&committeesv1.CreateMeetingRequest{
		CommitteeSlug: "ov-comm",
		Name:          "Overview Meeting",
		StartAt:       &startAt,
	})); err != nil {
		t.Fatalf("create meeting: %v", err)
	}

	overviewResp, err := client.committees.GetCommitteeOverview(context.Background(), connect.NewRequest(&committeesv1.GetCommitteeOverviewRequest{
		CommitteeSlug: "ov-comm",
	}))
	if err != nil {
		t.Fatalf("get overview: %v", err)
	}

	meetings := overviewResp.Msg.GetOverview().GetMeetings()
	if len(meetings) == 0 {
		t.Fatal("expected at least one meeting")
	}
	ref := meetings[0].GetMeeting()
	if ref.StartAt == nil || *ref.StartAt != startAt {
		t.Fatalf("expected start_at %q in overview, got %v", startAt, ref.StartAt)
	}
}
