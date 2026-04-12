package apiconnect

import (
	"context"
	"testing"

	connect "connectrpc.com/connect"

	committeesv1 "github.com/Y4shin/open-caucus/gen/go/conference/committees/v1"
	sessionv1 "github.com/Y4shin/open-caucus/gen/go/conference/session/v1"
)

func TestUpdateMeeting_ChangesNameAndDescription(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-comm")
	ts.seedUser(t, "test-comm", "chair1", "pass", "Chair", "chairperson")

	client := newCombinedTestClient(t, ts)
	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	// Create a meeting.
	createResp, err := client.committees.CreateMeeting(context.Background(), connect.NewRequest(&committeesv1.CreateMeetingRequest{
		CommitteeSlug: "test-comm",
		Name:          "Original Name",
		Description:   "Original Desc",
	}))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	meetingID := createResp.Msg.Meeting.MeetingId

	// Update it.
	startAt := "2026-06-01T10:00:00Z"
	endAt := "2026-06-01T12:00:00Z"
	updateResp, err := client.committees.UpdateMeeting(context.Background(), connect.NewRequest(&committeesv1.UpdateMeetingRequest{
		CommitteeSlug: "test-comm",
		MeetingId:     meetingID,
		Name:          "Updated Name",
		Description:   "Updated Desc",
		StartAt:       &startAt,
		EndAt:         &endAt,
	}))
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	m := updateResp.Msg.Meeting
	if m.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got %q", m.Name)
	}
	if m.Description != "Updated Desc" {
		t.Errorf("expected desc 'Updated Desc', got %q", m.Description)
	}
	if m.StartAt == nil || *m.StartAt != startAt {
		t.Errorf("expected startAt %q, got %v", startAt, m.StartAt)
	}
	if m.EndAt == nil || *m.EndAt != endAt {
		t.Errorf("expected endAt %q, got %v", endAt, m.EndAt)
	}
}

func TestUpdateMeeting_MemberCannotUpdate(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-comm")
	ts.seedUser(t, "test-comm", "chair1", "pass", "Chair", "chairperson")
	ts.seedUser(t, "test-comm", "member1", "pass", "Member", "member")

	client := newCombinedTestClient(t, ts)

	// Create as chair.
	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass",
	})); err != nil {
		t.Fatalf("login chair: %v", err)
	}
	createResp, _ := client.committees.CreateMeeting(context.Background(), connect.NewRequest(&committeesv1.CreateMeetingRequest{
		CommitteeSlug: "test-comm",
		Name:          "Meeting",
	}))
	meetingID := createResp.Msg.Meeting.MeetingId

	// Login as member and try to update.
	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "member1",
		Password: "pass",
	})); err != nil {
		t.Fatalf("login member: %v", err)
	}

	_, err := client.committees.UpdateMeeting(context.Background(), connect.NewRequest(&committeesv1.UpdateMeetingRequest{
		CommitteeSlug: "test-comm",
		MeetingId:     meetingID,
		Name:          "Hacked",
	}))
	if err == nil {
		t.Fatal("expected permission denied for member")
	}
}
