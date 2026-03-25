package apiconnect

import (
	"context"
	"fmt"
	"testing"

	connect "connectrpc.com/connect"

	agendav1 "github.com/Y4shin/conference-tool/gen/go/conference/agenda/v1"
	sessionv1 "github.com/Y4shin/conference-tool/gen/go/conference/session/v1"
)

func TestAgendaService_ListAgendaPoints_Member(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Alice Member", "member")
	meetingID := ts.seedMeeting(t, "test-committee", "Spring Meeting", false)

	client := newCombinedTestClient(t, ts)

	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "member1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	resp, err := client.agenda.ListAgendaPoints(context.Background(), connect.NewRequest(&agendav1.ListAgendaPointsRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	}))
	if err != nil {
		t.Fatalf("list agenda points: %v", err)
	}
	if len(resp.Msg.GetAgendaPoints()) != 0 {
		t.Fatalf("expected empty agenda, got %d points", len(resp.Msg.GetAgendaPoints()))
	}
}

func TestAgendaService_CreateAndDelete(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair One", "chairperson")
	meetingID := ts.seedMeeting(t, "test-committee", "Spring Meeting", false)

	client := newCombinedTestClient(t, ts)

	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	createResp, err := client.agenda.CreateAgendaPoint(context.Background(), connect.NewRequest(&agendav1.CreateAgendaPointRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		Title:         "Opening",
	}))
	if err != nil {
		t.Fatalf("create agenda point: %v", err)
	}
	apID := createResp.Msg.GetAgendaPoint().GetAgendaPointId()
	if apID == "" {
		t.Fatal("expected non-empty agenda point id")
	}
	if createResp.Msg.GetAgendaPoint().GetTitle() != "Opening" {
		t.Fatalf("unexpected title: %q", createResp.Msg.GetAgendaPoint().GetTitle())
	}

	// Verify it appears in the list.
	listResp, err := client.agenda.ListAgendaPoints(context.Background(), connect.NewRequest(&agendav1.ListAgendaPointsRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	}))
	if err != nil {
		t.Fatalf("list after create: %v", err)
	}
	if len(listResp.Msg.GetAgendaPoints()) != 1 {
		t.Fatalf("expected 1 agenda point, got %d", len(listResp.Msg.GetAgendaPoints()))
	}

	// Delete it.
	_, err = client.agenda.DeleteAgendaPoint(context.Background(), connect.NewRequest(&agendav1.DeleteAgendaPointRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		AgendaPointId: apID,
	}))
	if err != nil {
		t.Fatalf("delete agenda point: %v", err)
	}

	// Verify empty again.
	listResp2, err := client.agenda.ListAgendaPoints(context.Background(), connect.NewRequest(&agendav1.ListAgendaPointsRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	}))
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(listResp2.Msg.GetAgendaPoints()) != 0 {
		t.Fatalf("expected empty agenda after delete, got %d points", len(listResp2.Msg.GetAgendaPoints()))
	}
}

func TestAgendaService_CreateAgendaPoint_MemberForbidden(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Alice Member", "member")
	meetingID := ts.seedMeeting(t, "test-committee", "Spring Meeting", false)

	client := newCombinedTestClient(t, ts)

	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "member1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	_, err := client.agenda.CreateAgendaPoint(context.Background(), connect.NewRequest(&agendav1.CreateAgendaPointRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		Title:         "Should Fail",
	}))
	if err == nil {
		t.Fatal("expected permission error for member creating agenda point")
	}
}

func TestAgendaService_ActivateAgendaPoint(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair One", "chairperson")
	meetingID := ts.seedMeeting(t, "test-committee", "Spring Meeting", false)

	client := newCombinedTestClient(t, ts)

	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	createResp, err := client.agenda.CreateAgendaPoint(context.Background(), connect.NewRequest(&agendav1.CreateAgendaPointRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		Title:         "Item 1",
	}))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	apID := createResp.Msg.GetAgendaPoint().GetAgendaPointId()

	activateResp, err := client.agenda.ActivateAgendaPoint(context.Background(), connect.NewRequest(&agendav1.ActivateAgendaPointRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		AgendaPointId: apID,
	}))
	if err != nil {
		t.Fatalf("activate: %v", err)
	}
	if activateResp.Msg.GetActiveAgendaPointId() != apID {
		t.Fatalf("expected active id %q, got %q", apID, activateResp.Msg.GetActiveAgendaPointId())
	}
}

func TestAgendaService_MoveAgendaPoint_ReordersTopLevelAgenda(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair One", "chairperson")
	meetingID := ts.seedMeeting(t, "test-committee", "Spring Meeting", false)

	client := newCombinedTestClient(t, ts)

	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	firstResp, err := client.agenda.CreateAgendaPoint(context.Background(), connect.NewRequest(&agendav1.CreateAgendaPointRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		Title:         "Opening",
	}))
	if err != nil {
		t.Fatalf("create first agenda point: %v", err)
	}

	secondResp, err := client.agenda.CreateAgendaPoint(context.Background(), connect.NewRequest(&agendav1.CreateAgendaPointRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		Title:         "Voting",
	}))
	if err != nil {
		t.Fatalf("create second agenda point: %v", err)
	}

	moveResp, err := client.agenda.MoveAgendaPoint(context.Background(), connect.NewRequest(&agendav1.MoveAgendaPointRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		AgendaPointId: secondResp.Msg.GetAgendaPoint().GetAgendaPointId(),
		Direction:     "up",
	}))
	if err != nil {
		t.Fatalf("move agenda point: %v", err)
	}

	if len(moveResp.Msg.GetAgendaPoints()) != 2 {
		t.Fatalf("expected 2 agenda points after move, got %d", len(moveResp.Msg.GetAgendaPoints()))
	}
	if moveResp.Msg.GetAgendaPoints()[0].GetAgendaPointId() != secondResp.Msg.GetAgendaPoint().GetAgendaPointId() {
		t.Fatalf("expected moved agenda point first, got %q", moveResp.Msg.GetAgendaPoints()[0].GetAgendaPointId())
	}
	if moveResp.Msg.GetAgendaPoints()[1].GetAgendaPointId() != firstResp.Msg.GetAgendaPoint().GetAgendaPointId() {
		t.Fatalf("expected original first agenda point second, got %q", moveResp.Msg.GetAgendaPoints()[1].GetAgendaPointId())
	}
}

func TestAgendaService_ActivateAgendaPoint_MemberForbidden(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair One", "chairperson")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Alice Member", "member")
	meetingID := ts.seedMeeting(t, "test-committee", "Spring Meeting", false)

	chairClient := newCombinedTestClient(t, ts)

	if _, err := chairClient.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("chair login: %v", err)
	}

	createResp, err := chairClient.agenda.CreateAgendaPoint(context.Background(), connect.NewRequest(&agendav1.CreateAgendaPointRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		Title:         "Item 1",
	}))
	if err != nil {
		t.Fatalf("create agenda point: %v", err)
	}

	memberClient := newCombinedTestClient(t, ts)

	if _, err := memberClient.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "member1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("member login: %v", err)
	}

	_, err = memberClient.agenda.ActivateAgendaPoint(context.Background(), connect.NewRequest(&agendav1.ActivateAgendaPointRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		AgendaPointId: createResp.Msg.GetAgendaPoint().GetAgendaPointId(),
	}))
	if err == nil {
		t.Fatal("expected permission error for member activating agenda point")
	}
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("expected permission denied, got %v", connect.CodeOf(err))
	}
}
