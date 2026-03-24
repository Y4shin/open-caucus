package apiconnect

import (
	"context"
	"fmt"
	"testing"

	connect "connectrpc.com/connect"

	meetingsv1 "github.com/Y4shin/conference-tool/gen/go/conference/meetings/v1"
	sessionv1 "github.com/Y4shin/conference-tool/gen/go/conference/session/v1"
)

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
