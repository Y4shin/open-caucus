package apiconnect

import (
	"context"
	"fmt"
	"testing"

	connect "connectrpc.com/connect"

	attendeesv1 "github.com/Y4shin/conference-tool/gen/go/conference/attendees/v1"
	meetingsv1 "github.com/Y4shin/conference-tool/gen/go/conference/meetings/v1"
	sessionv1 "github.com/Y4shin/conference-tool/gen/go/conference/session/v1"
)

func TestMeetingServiceGetJoinMeeting_AnonymousGuestSeesGuestJoin(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	meetingID := ts.seedMeeting(t, "test-committee", "Open Meeting", true)

	client := newCombinedTestClient(t, ts)

	resp, err := client.meetings.GetJoinMeeting(context.Background(), connect.NewRequest(&meetingsv1.GetJoinMeetingRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	}))
	if err != nil {
		t.Fatalf("get join meeting: %v", err)
	}

	view := resp.Msg.GetMeeting()
	if !view.GetSignupOpen() {
		t.Fatal("expected signup_open=true")
	}
	if !view.GetCapabilities().GetCanGuestJoin() {
		t.Fatal("expected anonymous guest join to be available")
	}
	if view.GetCapabilities().GetAlreadyJoined() {
		t.Fatal("anonymous visitor should not already be joined")
	}
}

func TestMeetingServiceGetJoinMeeting_MemberAlreadyJoined(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Member One", "member")
	meetingID := ts.seedMeeting(t, "test-committee", "Open Meeting", true)

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
		t.Fatalf("self signup: %v", err)
	}

	resp, err := client.meetings.GetJoinMeeting(context.Background(), connect.NewRequest(&meetingsv1.GetJoinMeetingRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	}))
	if err != nil {
		t.Fatalf("get join meeting: %v", err)
	}

	view := resp.Msg.GetMeeting()
	if !view.GetCapabilities().GetAlreadyJoined() {
		t.Fatal("expected already_joined=true for signed-up member")
	}
	if view.GetCurrentAttendee().GetFullName() != "Member One" {
		t.Fatalf("unexpected attendee name: %q", view.GetCurrentAttendee().GetFullName())
	}
	if view.GetCapabilities().GetCanSelfSignup() {
		t.Fatal("already joined member should not see self-signup capability")
	}
}

func TestMeetingServiceGetLiveMeeting_AnonymousUser(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	meetingID := ts.seedMeeting(t, "test-committee", "Spring Meeting", true)

	client := newCombinedTestClient(t, ts)

	resp, err := client.meetings.GetLiveMeeting(context.Background(), connect.NewRequest(&meetingsv1.GetLiveMeetingRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	}))
	if err != nil {
		t.Fatalf("get live meeting: %v", err)
	}

	m := resp.Msg.GetMeeting()
	if m.GetMeetingName() != "Spring Meeting" {
		t.Fatalf("unexpected meeting name: %q", m.GetMeetingName())
	}
	if m.GetCommitteeSlug() != "test-committee" {
		t.Fatalf("unexpected committee slug: %q", m.GetCommitteeSlug())
	}
	if m.GetCapabilities().GetCanSelfSignup() {
		t.Fatal("anonymous user should not be able to self-signup via this RPC (not an attendee)")
	}
}

func TestMeetingServiceGetLiveMeeting_MemberCanSignupWhenOpen(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Member One", "member")
	meetingID := ts.seedMeeting(t, "test-committee", "Open Meeting", true)
	if err := ts.repo.SetActiveMeeting(context.Background(), "test-committee", &meetingID); err != nil {
		t.Fatalf("set active meeting: %v", err)
	}

	client := newCombinedTestClient(t, ts)

	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "member1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	resp, err := client.meetings.GetLiveMeeting(context.Background(), connect.NewRequest(&meetingsv1.GetLiveMeetingRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	}))
	if err != nil {
		t.Fatalf("get live meeting: %v", err)
	}

	caps := resp.Msg.GetMeeting().GetCapabilities()
	if !caps.GetCanSelfSignup() {
		t.Fatal("expected CanSelfSignup when signup is open and user is not yet an attendee")
	}
}

func TestMeetingServiceGetLiveMeeting_NotFound(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")

	client := newCombinedTestClient(t, ts)

	_, err := client.meetings.GetLiveMeeting(context.Background(), connect.NewRequest(&meetingsv1.GetLiveMeetingRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     "99999",
	}))
	if err == nil {
		t.Fatal("expected not found error for non-existent meeting")
	}
}

func TestMeetingServiceGetLiveMeeting_SignupClosedNoSelfSignup(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Member One", "member")
	meetingID := ts.seedMeeting(t, "test-committee", "Closed Meeting", false)
	if err := ts.repo.SetActiveMeeting(context.Background(), "test-committee", &meetingID); err != nil {
		t.Fatalf("set active meeting: %v", err)
	}

	client := newCombinedTestClient(t, ts)

	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "member1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	resp, err := client.meetings.GetLiveMeeting(context.Background(), connect.NewRequest(&meetingsv1.GetLiveMeetingRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	}))
	if err != nil {
		t.Fatalf("get live meeting: %v", err)
	}

	if resp.Msg.GetMeeting().GetCapabilities().GetCanSelfSignup() {
		t.Fatal("expected CanSelfSignup=false when signup is closed")
	}
}
