package apiconnect

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	connect "connectrpc.com/connect"

	attendeesv1 "github.com/Y4shin/conference-tool/gen/go/conference/attendees/v1"
	sessionv1 "github.com/Y4shin/conference-tool/gen/go/conference/session/v1"
	votesv1 "github.com/Y4shin/conference-tool/gen/go/conference/votes/v1"
	"github.com/Y4shin/conference-tool/internal/repository/model"
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
	if !liveResp.Msg.GetView().GetHasActiveAgenda() {
		t.Fatal("expected active agenda flag for attendee panel")
	}
	if !liveResp.Msg.GetView().GetIsEligible() {
		t.Fatal("expected attendee to be eligible")
	}
	if len(liveResp.Msg.GetView().GetVotes()) != 1 {
		t.Fatalf("expected one live vote card, got %d", len(liveResp.Msg.GetView().GetVotes()))
	}

	submitResp, err := guestClient.votes.SubmitBallot(context.Background(), connect.NewRequest(&votesv1.SubmitBallotRequest{
		CommitteeSlug:     "test-committee",
		MeetingId:         fmt.Sprintf("%d", meetingID),
		VoteId:            createResp.Msg.GetVote().GetVoteId(),
		SelectedOptionIds: []string{liveResp.Msg.GetView().GetActiveVote().GetOptions()[0].GetOptionId()},
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
	if len(liveRespAfter.Msg.GetView().GetVotes()) != 1 || !liveRespAfter.Msg.GetView().GetVotes()[0].GetAlreadyVoted() {
		t.Fatal("expected live vote card to reflect attendee submission state")
	}
}

func TestVoteService_GetLiveVotePanel_IncludesClosedTimedResultsAndCountingState(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair One", "chairperson")
	meetingID := ts.seedMeeting(t, "test-committee", "State Meeting", true)
	agendaPointID := ts.seedAgendaPoint(t, "test-committee", "State Meeting", "Budget")
	ts.activateAgendaPoint(t, "test-committee", "State Meeting", agendaPointID)

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

	openVoteResp, err := chairClient.votes.CreateVote(context.Background(), connect.NewRequest(&votesv1.CreateVoteRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		Name:          "Budget Vote",
		Visibility:    "open",
		MinSelections: 1,
		MaxSelections: 1,
		OptionLabels:  []string{"Yes", "No"},
	}))
	if err != nil {
		t.Fatalf("create open vote: %v", err)
	}
	if _, err := chairClient.votes.OpenVote(context.Background(), connect.NewRequest(&votesv1.OpenVoteRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		VoteId:        openVoteResp.Msg.GetVote().GetVoteId(),
	})); err != nil {
		t.Fatalf("open open vote: %v", err)
	}

	liveOpenResp, err := guestClient.votes.GetLiveVotePanel(context.Background(), connect.NewRequest(&votesv1.GetLiveVotePanelRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	}))
	if err != nil {
		t.Fatalf("get live open vote panel: %v", err)
	}
	if _, err := guestClient.votes.SubmitBallot(context.Background(), connect.NewRequest(&votesv1.SubmitBallotRequest{
		CommitteeSlug:     "test-committee",
		MeetingId:         fmt.Sprintf("%d", meetingID),
		VoteId:            openVoteResp.Msg.GetVote().GetVoteId(),
		SelectedOptionIds: []string{liveOpenResp.Msg.GetView().GetActiveVote().GetOptions()[0].GetOptionId()},
	})); err != nil {
		t.Fatalf("submit open ballot: %v", err)
	}
	if _, err := chairClient.votes.CloseVote(context.Background(), connect.NewRequest(&votesv1.CloseVoteRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		VoteId:        openVoteResp.Msg.GetVote().GetVoteId(),
	})); err != nil {
		t.Fatalf("close open vote: %v", err)
	}

	closedResp, err := guestClient.votes.GetLiveVotePanel(context.Background(), connect.NewRequest(&votesv1.GetLiveVotePanelRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	}))
	if err != nil {
		t.Fatalf("get closed live vote panel: %v", err)
	}
	if closedResp.Msg.GetView().GetHasActiveVote() {
		t.Fatal("expected no active vote after close")
	}
	if len(closedResp.Msg.GetView().GetVotes()) != 1 {
		t.Fatalf("expected one recently closed vote card, got %d", len(closedResp.Msg.GetView().GetVotes()))
	}
	if !closedResp.Msg.GetView().GetVotes()[0].GetHasTimedResults() {
		t.Fatal("expected recently closed vote to expose timed results")
	}
	if len(closedResp.Msg.GetView().GetVotes()[0].GetTimedResults()) == 0 {
		t.Fatal("expected recently closed vote to include tally rows")
	}

	secretVoteResp, err := chairClient.votes.CreateVote(context.Background(), connect.NewRequest(&votesv1.CreateVoteRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		Name:          "Secret Vote",
		Visibility:    "secret",
		MinSelections: 1,
		MaxSelections: 1,
		OptionLabels:  []string{"Yes", "No"},
	}))
	if err != nil {
		t.Fatalf("create secret vote: %v", err)
	}
	if _, err := chairClient.votes.OpenVote(context.Background(), connect.NewRequest(&votesv1.OpenVoteRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		VoteId:        secretVoteResp.Msg.GetVote().GetVoteId(),
	})); err != nil {
		t.Fatalf("open secret vote: %v", err)
	}

	liveSecretResp, err := guestClient.votes.GetLiveVotePanel(context.Background(), connect.NewRequest(&votesv1.GetLiveVotePanelRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	}))
	if err != nil {
		t.Fatalf("get live secret vote panel: %v", err)
	}
	if !liveSecretResp.Msg.GetView().GetHasActiveVote() {
		t.Fatal("expected active secret vote before forcing counting state")
	}
	guestAttendeeID, err := strconv.ParseInt(guestJoinResp.Msg.GetAttendee().GetAttendeeId(), 10, 64)
	if err != nil {
		t.Fatalf("parse guest attendee id: %v", err)
	}
	if _, err := ts.repo.RegisterVoteCast(context.Background(), mustInt64(t, secretVoteResp.Msg.GetVote().GetVoteId()), meetingID, guestAttendeeID, model.VoteCastSourceManualSubmission); err != nil {
		t.Fatalf("register manual cast without counted ballot: %v", err)
	}
	if _, err := chairClient.votes.CloseVote(context.Background(), connect.NewRequest(&votesv1.CloseVoteRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		VoteId:        secretVoteResp.Msg.GetVote().GetVoteId(),
	})); err != nil {
		t.Fatalf("close secret vote into counting: %v", err)
	}

	countingResp, err := guestClient.votes.GetLiveVotePanel(context.Background(), connect.NewRequest(&votesv1.GetLiveVotePanelRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	}))
	if err != nil {
		t.Fatalf("get counting live vote panel: %v", err)
	}
	if len(countingResp.Msg.GetView().GetVotes()) != 2 {
		t.Fatalf("expected recently closed + counting vote cards, got %d", len(countingResp.Msg.GetView().GetVotes()))
	}
	var countingCard *votesv1.LiveVoteCardView
	for _, card := range countingResp.Msg.GetView().GetVotes() {
		if card.GetVote().GetState() == "counting" {
			countingCard = card
			break
		}
	}
	if countingCard == nil {
		t.Fatal("expected one counting vote card")
	}
	if !countingCard.GetResultsBlockedCounting() {
		t.Fatal("expected counting vote card to mark results as blocked")
	}
}

func mustInt64(t *testing.T, raw string) int64 {
	t.Helper()

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		t.Fatalf("parse int64 %q: %v", raw, err)
	}
	return value
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

func TestVoteService_VerifyOpenAndSecretReceipts(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair One", "chairperson")
	meetingID := ts.seedMeeting(t, "test-committee", "Receipt Meeting", true)
	agendaPointID := ts.seedAgendaPoint(t, "test-committee", "Receipt Meeting", "Budget")
	ts.activateAgendaPoint(t, "test-committee", "Receipt Meeting", agendaPointID)

	guestClient := newCombinedTestClient(t, ts)
	guestJoinResp, err := guestClient.attendees.GuestJoin(context.Background(), connect.NewRequest(&attendeesv1.GuestJoinRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		FullName:      "Receipt Guest",
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

	openVoteResp, err := chairClient.votes.CreateVote(context.Background(), connect.NewRequest(&votesv1.CreateVoteRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		Name:          "Open Vote",
		Visibility:    "open",
		MinSelections: 1,
		MaxSelections: 1,
		OptionLabels:  []string{"Yes", "No"},
	}))
	if err != nil {
		t.Fatalf("create open vote: %v", err)
	}
	if _, err := chairClient.votes.OpenVote(context.Background(), connect.NewRequest(&votesv1.OpenVoteRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		VoteId:        openVoteResp.Msg.GetVote().GetVoteId(),
	})); err != nil {
		t.Fatalf("open open vote: %v", err)
	}
	liveOpenResp, err := guestClient.votes.GetLiveVotePanel(context.Background(), connect.NewRequest(&votesv1.GetLiveVotePanelRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	}))
	if err != nil {
		t.Fatalf("get live open vote panel: %v", err)
	}
	openSubmitResp, err := guestClient.votes.SubmitBallot(context.Background(), connect.NewRequest(&votesv1.SubmitBallotRequest{
		CommitteeSlug:     "test-committee",
		MeetingId:         fmt.Sprintf("%d", meetingID),
		VoteId:            openVoteResp.Msg.GetVote().GetVoteId(),
		SelectedOptionIds: []string{liveOpenResp.Msg.GetView().GetActiveVote().GetOptions()[0].GetOptionId()},
	}))
	if err != nil {
		t.Fatalf("submit open ballot: %v", err)
	}

	verifyOpenResp, err := guestClient.votes.VerifyOpenReceipt(context.Background(), connect.NewRequest(&votesv1.VerifyOpenReceiptRequest{
		VoteId:       openVoteResp.Msg.GetVote().GetVoteId(),
		ReceiptToken: openSubmitResp.Msg.GetReceiptToken(),
	}))
	if err != nil {
		t.Fatalf("verify open receipt: %v", err)
	}
	if verifyOpenResp.Msg.GetReceiptToken() != openSubmitResp.Msg.GetReceiptToken() {
		t.Fatalf("unexpected open receipt token: %q", verifyOpenResp.Msg.GetReceiptToken())
	}
	if len(verifyOpenResp.Msg.GetChoiceLabels()) != 1 {
		t.Fatalf("expected one verified open choice, got %d", len(verifyOpenResp.Msg.GetChoiceLabels()))
	}
	if _, err := chairClient.votes.CloseVote(context.Background(), connect.NewRequest(&votesv1.CloseVoteRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		VoteId:        openVoteResp.Msg.GetVote().GetVoteId(),
	})); err != nil {
		t.Fatalf("close open vote: %v", err)
	}

	secretVoteResp, err := chairClient.votes.CreateVote(context.Background(), connect.NewRequest(&votesv1.CreateVoteRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		Name:          "Secret Vote",
		Visibility:    "secret",
		MinSelections: 1,
		MaxSelections: 1,
		OptionLabels:  []string{"Yes", "No"},
	}))
	if err != nil {
		t.Fatalf("create secret vote: %v", err)
	}
	if _, err := chairClient.votes.OpenVote(context.Background(), connect.NewRequest(&votesv1.OpenVoteRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		VoteId:        secretVoteResp.Msg.GetVote().GetVoteId(),
	})); err != nil {
		t.Fatalf("open secret vote: %v", err)
	}
	liveSecretResp, err := guestClient.votes.GetLiveVotePanel(context.Background(), connect.NewRequest(&votesv1.GetLiveVotePanelRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	}))
	if err != nil {
		t.Fatalf("get live secret vote panel: %v", err)
	}
	secretSubmitResp, err := guestClient.votes.SubmitBallot(context.Background(), connect.NewRequest(&votesv1.SubmitBallotRequest{
		CommitteeSlug:     "test-committee",
		MeetingId:         fmt.Sprintf("%d", meetingID),
		VoteId:            secretVoteResp.Msg.GetVote().GetVoteId(),
		SelectedOptionIds: []string{liveSecretResp.Msg.GetView().GetActiveVote().GetOptions()[0].GetOptionId()},
	}))
	if err != nil {
		t.Fatalf("submit secret ballot: %v", err)
	}

	verifySecretResp, err := guestClient.votes.VerifySecretReceipt(context.Background(), connect.NewRequest(&votesv1.VerifySecretReceiptRequest{
		VoteId:       secretVoteResp.Msg.GetVote().GetVoteId(),
		ReceiptToken: secretSubmitResp.Msg.GetReceiptToken(),
	}))
	if err != nil {
		t.Fatalf("verify secret receipt: %v", err)
	}
	if verifySecretResp.Msg.GetReceiptToken() != secretSubmitResp.Msg.GetReceiptToken() {
		t.Fatalf("unexpected secret receipt token: %q", verifySecretResp.Msg.GetReceiptToken())
	}
	if verifySecretResp.Msg.GetEncryptedCommitmentB64() == "" {
		t.Fatal("expected encrypted commitment payload")
	}
}
