package apiconnect

import (
	"context"
	"testing"

	connect "connectrpc.com/connect"

	committeesv1 "github.com/Y4shin/conference-tool/gen/go/conference/committees/v1"
	sessionv1 "github.com/Y4shin/conference-tool/gen/go/conference/session/v1"
)

func TestCommitteeServiceListMyCommittees_Chairperson(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Alpha Committee", "alpha")
	ts.seedCommittee(t, "Beta Committee", "beta")
	ts.seedUser(t, "alpha", "chair1", "pass123", "Chair One", "chairperson")
	ts.seedUser(t, "beta", "chair1", "pass123", "Chair One", "member")

	client := newCombinedTestClient(t, ts)

	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	resp, err := client.committees.ListMyCommittees(context.Background(), connect.NewRequest(&committeesv1.ListMyCommitteesRequest{}))
	if err != nil {
		t.Fatalf("list committees: %v", err)
	}

	if got := len(resp.Msg.GetCommittees()); got != 2 {
		t.Fatalf("expected 2 committees, got %d", got)
	}
}

func TestCommitteeServiceListMyCommittees_Unauthenticated(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	client := newCombinedTestClient(t, ts)

	_, err := client.committees.ListMyCommittees(context.Background(), connect.NewRequest(&committeesv1.ListMyCommitteesRequest{}))
	if err == nil {
		t.Fatal("expected error for unauthenticated request")
	}
}

func TestCommitteeServiceGetCommitteeOverview_Member(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Regular Member", "member")

	client := newCombinedTestClient(t, ts)

	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "member1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	resp, err := client.committees.GetCommitteeOverview(context.Background(), connect.NewRequest(&committeesv1.GetCommitteeOverviewRequest{
		CommitteeSlug: "test-committee",
	}))
	if err != nil {
		t.Fatalf("get overview: %v", err)
	}

	overview := resp.Msg.GetOverview()
	if got := overview.GetCommittee().GetSlug(); got != "test-committee" {
		t.Fatalf("unexpected slug: %q", got)
	}
	if overview.GetCommittee().GetIsChairperson() {
		t.Fatal("expected member, not chairperson")
	}
	if !overview.GetCommittee().GetIsMember() {
		t.Fatal("expected IsMember flag")
	}
}

func TestCommitteeServiceGetCommitteeOverview_NonMemberForbidden(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Private Committee", "private")
	ts.seedCommittee(t, "Other Committee", "other")
	ts.seedUser(t, "other", "user1", "pass123", "User One", "member")

	client := newCombinedTestClient(t, ts)

	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "user1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	_, err := client.committees.GetCommitteeOverview(context.Background(), connect.NewRequest(&committeesv1.GetCommitteeOverviewRequest{
		CommitteeSlug: "private",
	}))
	if err == nil {
		t.Fatal("expected permission denied for non-member")
	}
}

func TestCommitteeServiceGetCommitteeOverview_HasActiveMeeting(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair One", "chairperson")
	meetingID := ts.seedMeeting(t, "test-committee", "Spring Meeting", true)

	if err := ts.repo.SetActiveMeeting(context.Background(), "test-committee", &meetingID); err != nil {
		t.Fatalf("set active meeting: %v", err)
	}

	client := newCombinedTestClient(t, ts)

	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	resp, err := client.committees.GetCommitteeOverview(context.Background(), connect.NewRequest(&committeesv1.GetCommitteeOverviewRequest{
		CommitteeSlug: "test-committee",
	}))
	if err != nil {
		t.Fatalf("get overview: %v", err)
	}

	meetings := resp.Msg.GetOverview().GetMeetings()
	if len(meetings) != 1 {
		t.Fatalf("expected 1 meeting, got %d", len(meetings))
	}
	if !meetings[0].GetCanViewLive() {
		t.Fatal("expected CanViewLive for active meeting")
	}
}

func TestCommitteeServiceCreateDeleteAndToggleMeetingActive(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair One", "chairperson")

	client := newCombinedTestClient(t, ts)
	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	createResp, err := client.committees.CreateMeeting(context.Background(), connect.NewRequest(&committeesv1.CreateMeetingRequest{
		CommitteeSlug: "test-committee",
		Name:          "Budget Meeting",
		Description:   "Quarterly budget review",
	}))
	if err != nil {
		t.Fatalf("create meeting: %v", err)
	}
	meetingID := createResp.Msg.GetMeeting().GetMeetingId()
	if meetingID == "" {
		t.Fatal("expected created meeting id")
	}

	toggleResp, err := client.committees.ToggleMeetingActive(context.Background(), connect.NewRequest(&committeesv1.ToggleMeetingActiveRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     meetingID,
	}))
	if err != nil {
		t.Fatalf("toggle active meeting: %v", err)
	}
	if !toggleResp.Msg.GetActive() {
		t.Fatal("expected meeting to become active")
	}

	toggleResp, err = client.committees.ToggleMeetingActive(context.Background(), connect.NewRequest(&committeesv1.ToggleMeetingActiveRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     meetingID,
	}))
	if err != nil {
		t.Fatalf("toggle active meeting off: %v", err)
	}
	if toggleResp.Msg.GetActive() {
		t.Fatal("expected meeting to become inactive")
	}

	deleteResp, err := client.committees.DeleteMeeting(context.Background(), connect.NewRequest(&committeesv1.DeleteMeetingRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     meetingID,
	}))
	if err != nil {
		t.Fatalf("delete meeting: %v", err)
	}
	if !deleteResp.Msg.GetDeleted() {
		t.Fatal("expected deleted=true")
	}
}
