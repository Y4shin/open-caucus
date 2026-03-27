package apiconnect

import (
	"context"
	"fmt"
	"testing"

	connect "connectrpc.com/connect"

	attendeesv1 "github.com/Y4shin/conference-tool/gen/go/conference/attendees/v1"
	sessionv1 "github.com/Y4shin/conference-tool/gen/go/conference/session/v1"
	votesv1 "github.com/Y4shin/conference-tool/gen/go/conference/votes/v1"
)

func TestVoteService_CreateOpenCloseVote(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair One", "chairperson")
	meetingID := ts.seedMeeting(t, "test-committee", "Spring Meeting", true)
	agendaPointID := ts.seedAgendaPoint(t, "test-committee", "Spring Meeting", "Budget")
	ts.activateAgendaPoint(t, "test-committee", "Spring Meeting", agendaPointID)

	client := newCombinedTestClient(t, ts)

	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	createResp, err := client.votes.CreateVote(context.Background(), connect.NewRequest(&votesv1.CreateVoteRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		Name:          "Budget Vote",
		Visibility:    "open",
		MinSelections: 1,
		MaxSelections: 1,
		OptionLabels:  []string{"Yes", "No"},
	}))
	if err != nil {
		t.Fatalf("create vote: %v", err)
	}
	if createResp.Msg.GetVote().GetName() != "Budget Vote" {
		t.Fatalf("unexpected vote name: %q", createResp.Msg.GetVote().GetName())
	}

	openResp, err := client.votes.OpenVote(context.Background(), connect.NewRequest(&votesv1.OpenVoteRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		VoteId:        createResp.Msg.GetVote().GetVoteId(),
	}))
	if err != nil {
		t.Fatalf("open vote: %v", err)
	}
	if openResp.Msg.GetVote().GetState() != "open" {
		t.Fatalf("expected open state, got %q", openResp.Msg.GetVote().GetState())
	}

	closeResp, err := client.votes.CloseVote(context.Background(), connect.NewRequest(&votesv1.CloseVoteRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		VoteId:        createResp.Msg.GetVote().GetVoteId(),
	}))
	if err != nil {
		t.Fatalf("close vote: %v", err)
	}
	if closeResp.Msg.GetVote().GetState() != "closed" {
		t.Fatalf("expected closed state, got %q", closeResp.Msg.GetVote().GetState())
	}

	panelResp, err := client.votes.GetVotesPanel(context.Background(), connect.NewRequest(&votesv1.GetVotesPanelRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	}))
	if err != nil {
		t.Fatalf("get votes panel: %v", err)
	}
	if len(panelResp.Msg.GetView().GetVotes()) != 1 {
		t.Fatalf("expected 1 vote definition, got %d", len(panelResp.Msg.GetView().GetVotes()))
	}
}

func TestVoteService_GetLiveVotePanel_AndSubmitBallot(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair One", "chairperson")
	meetingID := ts.seedMeeting(t, "test-committee", "Spring Meeting", true)
	agendaPointID := ts.seedAgendaPoint(t, "test-committee", "Spring Meeting", "Budget")
	ts.activateAgendaPoint(t, "test-committee", "Spring Meeting", agendaPointID)

	guestClient := newCombinedTestClient(t, ts)
	guestJoinResp, err := guestClient.attendees.GuestJoin(context.Background(), connect.NewRequest(&attendeesv1.GuestJoinRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		FullName:      "Guest Voter",
		MeetingSecret: "secret",
	}))
	if err != nil {
		t.Fatalf("guest join: %v", err)
	}
	if _, err := guestClient.attendees.AttendeeLogin(context.Background(), connect.NewRequest(&attendeesv1.AttendeeLoginRequest{
		MeetingId:      fmt.Sprintf("%d", meetingID),
		AttendeeSecret: guestJoinResp.Msg.GetAttendeeSecret(),
	})); err != nil {
		t.Fatalf("attendee login: %v", err)
	}

	chairClient := newCombinedTestClient(t, ts)
	if _, err := chairClient.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("chair login: %v", err)
	}

	createResp, err := chairClient.votes.CreateVote(context.Background(), connect.NewRequest(&votesv1.CreateVoteRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		Name:          "Budget Vote",
		Visibility:    "open",
		MinSelections: 1,
		MaxSelections: 1,
		OptionLabels:  []string{"Yes", "No"},
	}))
	if err != nil {
		t.Fatalf("create vote: %v", err)
	}

	if _, err := chairClient.votes.OpenVote(context.Background(), connect.NewRequest(&votesv1.OpenVoteRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		VoteId:        createResp.Msg.GetVote().GetVoteId(),
	})); err != nil {
		t.Fatalf("open vote: %v", err)
	}

	liveResp, err := guestClient.votes.GetLiveVotePanel(context.Background(), connect.NewRequest(&votesv1.GetLiveVotePanelRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	}))
	if err != nil {
		t.Fatalf("get live vote panel: %v", err)
	}
	if !liveResp.Msg.GetView().GetHasActiveVote() {
		t.Fatal("expected an active vote for attendee")
	}
	if !liveResp.Msg.GetView().GetIsEligible() {
		t.Fatal("expected attendee to be eligible")
	}

	submitResp, err := guestClient.votes.SubmitBallot(context.Background(), connect.NewRequest(&votesv1.SubmitBallotRequest{
		CommitteeSlug:      "test-committee",
		MeetingId:          fmt.Sprintf("%d", meetingID),
		VoteId:             createResp.Msg.GetVote().GetVoteId(),
		SelectedOptionIds:  []string{liveResp.Msg.GetView().GetActiveVote().GetOptions()[0].GetOptionId()},
	}))
	if err != nil {
		t.Fatalf("submit ballot: %v", err)
	}
	if submitResp.Msg.GetReceiptToken() == "" {
		t.Fatal("expected non-empty receipt token")
	}

	liveRespAfter, err := guestClient.votes.GetLiveVotePanel(context.Background(), connect.NewRequest(&votesv1.GetLiveVotePanelRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	}))
	if err != nil {
		t.Fatalf("get live vote panel after submit: %v", err)
	}
	if !liveRespAfter.Msg.GetView().GetAlreadyVoted() {
		t.Fatal("expected attendee to be marked as already voted")
	}
}

func TestVoteService_CreateVote_MemberForbidden(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Alice Member", "member")
	meetingID := ts.seedMeeting(t, "test-committee", "Spring Meeting", true)
	agendaPointID := ts.seedAgendaPoint(t, "test-committee", "Spring Meeting", "Budget")
	ts.activateAgendaPoint(t, "test-committee", "Spring Meeting", agendaPointID)

	client := newCombinedTestClient(t, ts)
	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "member1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	_, err := client.votes.CreateVote(context.Background(), connect.NewRequest(&votesv1.CreateVoteRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		Name:          "Forbidden Vote",
		Visibility:    "open",
		MinSelections: 1,
		MaxSelections: 1,
		OptionLabels:  []string{"Yes", "No"},
	}))
	if err == nil {
		t.Fatal("expected permission error for member creating vote")
	}
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("expected permission denied, got %v", connect.CodeOf(err))
	}
}

func TestVoteService_SubmitSecretBallot(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair One", "chairperson")
	meetingID := ts.seedMeeting(t, "test-committee", "Secret Meeting", true)
	agendaPointID := ts.seedAgendaPoint(t, "test-committee", "Secret Meeting", "Budget")
	ts.activateAgendaPoint(t, "test-committee", "Secret Meeting", agendaPointID)

	guestClient := newCombinedTestClient(t, ts)
	guestJoinResp, err := guestClient.attendees.GuestJoin(context.Background(), connect.NewRequest(&attendeesv1.GuestJoinRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		FullName:      "Secret Voter",
		MeetingSecret: "secret",
	}))
	if err != nil {
		t.Fatalf("guest join: %v", err)
	}
	if _, err := guestClient.attendees.AttendeeLogin(context.Background(), connect.NewRequest(&attendeesv1.AttendeeLoginRequest{
		MeetingId:      fmt.Sprintf("%d", meetingID),
		AttendeeSecret: guestJoinResp.Msg.GetAttendeeSecret(),
	})); err != nil {
		t.Fatalf("attendee login: %v", err)
	}

	chairClient := newCombinedTestClient(t, ts)
	if _, err := chairClient.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("chair login: %v", err)
	}

	createResp, err := chairClient.votes.CreateVote(context.Background(), connect.NewRequest(&votesv1.CreateVoteRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		Name:          "Secret Vote",
		Visibility:    "secret",
		MinSelections: 1,
		MaxSelections: 1,
		OptionLabels:  []string{"Yes", "No"},
	}))
	if err != nil {
		t.Fatalf("create vote: %v", err)
	}

	if _, err := chairClient.votes.OpenVote(context.Background(), connect.NewRequest(&votesv1.OpenVoteRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		VoteId:        createResp.Msg.GetVote().GetVoteId(),
	})); err != nil {
		t.Fatalf("open vote: %v", err)
	}

	liveResp, err := guestClient.votes.GetLiveVotePanel(context.Background(), connect.NewRequest(&votesv1.GetLiveVotePanelRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	}))
	if err != nil {
		t.Fatalf("get live vote panel: %v", err)
	}

	submitResp, err := guestClient.votes.SubmitBallot(context.Background(), connect.NewRequest(&votesv1.SubmitBallotRequest{
		CommitteeSlug:     "test-committee",
		MeetingId:         fmt.Sprintf("%d", meetingID),
		VoteId:            createResp.Msg.GetVote().GetVoteId(),
		SelectedOptionIds: []string{liveResp.Msg.GetView().GetActiveVote().GetOptions()[0].GetOptionId()},
	}))
	if err != nil {
		t.Fatalf("submit secret ballot: %v", err)
	}
	if submitResp.Msg.GetReceiptToken() == "" {
		t.Fatal("expected non-empty receipt token")
	}

	liveRespAfter, err := guestClient.votes.GetLiveVotePanel(context.Background(), connect.NewRequest(&votesv1.GetLiveVotePanelRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	}))
	if err != nil {
		t.Fatalf("get live vote panel after submit: %v", err)
	}
	if !liveRespAfter.Msg.GetView().GetAlreadyVoted() {
		t.Fatal("expected attendee to be marked as already voted after secret ballot")
	}
}
