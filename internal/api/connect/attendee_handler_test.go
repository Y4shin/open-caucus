package apiconnect

import (
	"context"
	"fmt"
	"testing"

	connect "connectrpc.com/connect"

	attendeesv1 "github.com/Y4shin/conference-tool/gen/go/conference/attendees/v1"
	sessionv1 "github.com/Y4shin/conference-tool/gen/go/conference/session/v1"
)

func TestAttendeeService_SelfSignup(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Alice Member", "member")
	meetingID := ts.seedMeeting(t, "test-committee", "Spring Meeting", true)

	client := newCombinedTestClient(t, ts)

	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "member1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	resp, err := client.attendees.SelfSignup(context.Background(), connect.NewRequest(&attendeesv1.SelfSignupRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	}))
	if err != nil {
		t.Fatalf("self signup: %v", err)
	}
	if resp.Msg.GetAlreadyExisted() {
		t.Fatal("expected new attendee, got already_existed=true")
	}
	if resp.Msg.GetAttendee().GetFullName() != "Alice Member" {
		t.Fatalf("unexpected full name: %q", resp.Msg.GetAttendee().GetFullName())
	}

	// Idempotent: second call should return already_existed=true.
	resp2, err := client.attendees.SelfSignup(context.Background(), connect.NewRequest(&attendeesv1.SelfSignupRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	}))
	if err != nil {
		t.Fatalf("second self signup: %v", err)
	}
	if !resp2.Msg.GetAlreadyExisted() {
		t.Fatal("expected already_existed=true on second call")
	}
}

func TestAttendeeService_ListAttendees(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	meetingID := ts.seedMeeting(t, "test-committee", "Spring Meeting", true)

	if _, err := ts.repo.CreateAttendee(context.Background(), meetingID, nil, "Alice Guest", "secret-a", false); err != nil {
		t.Fatalf("create attendee alice: %v", err)
	}
	if _, err := ts.repo.CreateAttendee(context.Background(), meetingID, nil, "Bob Guest", "secret-b", true); err != nil {
		t.Fatalf("create attendee bob: %v", err)
	}

	client := newCombinedTestClient(t, ts)

	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	resp, err := client.attendees.ListAttendees(context.Background(), connect.NewRequest(&attendeesv1.ListAttendeesRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	}))
	if err != nil {
		t.Fatalf("list attendees: %v", err)
	}
	if len(resp.Msg.GetAttendees()) != 2 {
		t.Fatalf("expected 2 attendees, got %d", len(resp.Msg.GetAttendees()))
	}
	if resp.Msg.GetAttendees()[0].GetFullName() != "Alice Guest" {
		t.Fatalf("unexpected first attendee: %q", resp.Msg.GetAttendees()[0].GetFullName())
	}
}

func TestAttendeeService_SelfSignup_SignupClosed_MemberCanAlwaysSignup(t *testing.T) {
	// signupOpen only gates guest joins; committee members may always self-signup.
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Alice Member", "member")
	meetingID := ts.seedMeeting(t, "test-committee", "Spring Meeting", false) // signup closed

	client := newCombinedTestClient(t, ts)

	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "member1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	if _, err := client.attendees.SelfSignup(context.Background(), connect.NewRequest(&attendeesv1.SelfSignupRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	})); err != nil {
		t.Fatalf("expected member to self-signup even when signup is closed: %v", err)
	}
}

func TestAttendeeService_GuestJoin(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	meetingID := ts.seedMeeting(t, "test-committee", "Spring Meeting", true)

	client := newCombinedTestClient(t, ts)

	resp, err := client.attendees.GuestJoin(context.Background(), connect.NewRequest(&attendeesv1.GuestJoinRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		FullName:      "Guest User",
		MeetingSecret: "secret",
		GenderQuoted:  false,
	}))
	if err != nil {
		t.Fatalf("guest join: %v", err)
	}
	if resp.Msg.GetAttendeeSecret() == "" {
		t.Fatal("expected non-empty attendee secret")
	}
	if resp.Msg.GetAttendee().GetFullName() != "Guest User" {
		t.Fatalf("unexpected full name: %q", resp.Msg.GetAttendee().GetFullName())
	}
}

func TestAttendeeService_GuestJoin_WrongSecret(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	meetingID := ts.seedMeeting(t, "test-committee", "Spring Meeting", true)

	client := newCombinedTestClient(t, ts)

	_, err := client.attendees.GuestJoin(context.Background(), connect.NewRequest(&attendeesv1.GuestJoinRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		FullName:      "Guest User",
		MeetingSecret: "wrong-secret",
	}))
	if err == nil {
		t.Fatal("expected error with wrong meeting secret")
	}
}

func TestAttendeeService_AttendeeLogin(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	meetingID := ts.seedMeeting(t, "test-committee", "Spring Meeting", true)

	client := newCombinedTestClient(t, ts)

	// Join as guest first.
	guestResp, err := client.attendees.GuestJoin(context.Background(), connect.NewRequest(&attendeesv1.GuestJoinRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		FullName:      "Guest User",
		MeetingSecret: "secret",
	}))
	if err != nil {
		t.Fatalf("guest join: %v", err)
	}

	loginResp, err := client.attendees.AttendeeLogin(context.Background(), connect.NewRequest(&attendeesv1.AttendeeLoginRequest{
		MeetingId:      fmt.Sprintf("%d", meetingID),
		AttendeeSecret: guestResp.Msg.GetAttendeeSecret(),
	}))
	if err != nil {
		t.Fatalf("attendee login: %v", err)
	}
	if loginResp.Msg.GetActor().GetActorKind() != "guest" {
		t.Fatalf("expected actor kind 'guest', got %q", loginResp.Msg.GetActor().GetActorKind())
	}
	if loginResp.Msg.GetAttendee().GetFullName() != "Guest User" {
		t.Fatalf("unexpected full name: %q", loginResp.Msg.GetAttendee().GetFullName())
	}
}
