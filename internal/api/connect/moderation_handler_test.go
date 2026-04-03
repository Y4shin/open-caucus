package apiconnect

import (
	"context"
	"fmt"
	"testing"
	"time"

	connect "connectrpc.com/connect"

	meetingsv1 "github.com/Y4shin/open-caucus/gen/go/conference/meetings/v1"
	moderationv1 "github.com/Y4shin/open-caucus/gen/go/conference/moderation/v1"
	sessionv1 "github.com/Y4shin/open-caucus/gen/go/conference/session/v1"
)

func TestModerationServiceGetModerationView_Chairperson(t *testing.T) {
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

	resp, err := client.moderation.GetModerationView(context.Background(), connect.NewRequest(&moderationv1.GetModerationViewRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	}))
	if err != nil {
		t.Fatalf("get moderation view: %v", err)
	}

	view := resp.Msg.GetView()
	if view.GetMeeting().GetMeetingName() != "Spring Meeting" {
		t.Fatalf("unexpected meeting name: %q", view.GetMeeting().GetMeetingName())
	}
	if view.GetAttendees().GetSignupOpen() {
		t.Fatal("expected signup closed")
	}
}

func TestModerationServiceGetModerationView_MemberForbidden(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Regular Member", "member")
	meetingID := ts.seedMeeting(t, "test-committee", "Spring Meeting", false)

	client := newCombinedTestClient(t, ts)

	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "member1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	_, err := client.moderation.GetModerationView(context.Background(), connect.NewRequest(&moderationv1.GetModerationViewRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	}))
	if err == nil {
		t.Fatal("expected permission denied for non-chairperson")
	}
}

func TestModerationServiceGetModerationView_Unauthenticated(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	meetingID := ts.seedMeeting(t, "test-committee", "Spring Meeting", false)

	client := newCombinedTestClient(t, ts)

	_, err := client.moderation.GetModerationView(context.Background(), connect.NewRequest(&moderationv1.GetModerationViewRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	}))
	if err == nil {
		t.Fatal("expected unauthenticated error")
	}
}

func TestModerationServiceToggleSignupOpen_ChangesState(t *testing.T) {
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

	resp, err := client.moderation.ToggleSignupOpen(context.Background(), connect.NewRequest(&moderationv1.ToggleSignupOpenRequest{
		CommitteeSlug:   "test-committee",
		MeetingId:       fmt.Sprintf("%d", meetingID),
		DesiredOpen:     true,
		ExpectedVersion: 0, // skip version check
	}))
	if err != nil {
		t.Fatalf("toggle signup: %v", err)
	}

	if !resp.Msg.GetSignupOpen() {
		t.Fatal("expected signup_open=true after toggle")
	}
	if resp.Msg.GetVersion() == 0 {
		t.Fatal("expected non-zero version after mutation")
	}
	if len(resp.Msg.GetInvalidatedViews()) == 0 {
		t.Fatal("expected invalidated_views to be populated")
	}
}

func TestModerationServiceToggleSignupOpen_VersionConflict(t *testing.T) {
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

	// Toggle once to bump version to 1
	if _, err := client.moderation.ToggleSignupOpen(context.Background(), connect.NewRequest(&moderationv1.ToggleSignupOpenRequest{
		CommitteeSlug:   "test-committee",
		MeetingId:       fmt.Sprintf("%d", meetingID),
		DesiredOpen:     true,
		ExpectedVersion: 0,
	})); err != nil {
		t.Fatalf("first toggle: %v", err)
	}

	// Now try with stale version 0
	_, err := client.moderation.ToggleSignupOpen(context.Background(), connect.NewRequest(&moderationv1.ToggleSignupOpenRequest{
		CommitteeSlug:   "test-committee",
		MeetingId:       fmt.Sprintf("%d", meetingID),
		DesiredOpen:     false,
		ExpectedVersion: 0, // version 0 means skip check, so use a wrong stale version
	}))
	// Version 0 skips the check, so this should succeed
	if err != nil {
		t.Fatalf("expected success with version=0 sentinel: %v", err)
	}

	// Now try with wrong expected version (2 when actual is 2, so let's use 1 which is stale)
	_, err = client.moderation.ToggleSignupOpen(context.Background(), connect.NewRequest(&moderationv1.ToggleSignupOpenRequest{
		CommitteeSlug:   "test-committee",
		MeetingId:       fmt.Sprintf("%d", meetingID),
		DesiredOpen:     true,
		ExpectedVersion: 1, // stale — actual is 2
	}))
	if err == nil {
		t.Fatal("expected conflict error for stale version")
	}
}

func TestRealtime_MeetingEvents_PublishesSignupInvalidation(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair One", "chairperson")
	meetingID := ts.seedMeeting(t, "test-committee", "Spring Meeting", false)

	client := newCombinedTestClient(t, ts)
	streamCtx, cancelStream := context.WithCancel(context.Background())
	defer cancelStream()
	eventCh := make(chan meetingsv1.MeetingEventKind, 8)
	streamErrCh := make(chan error, 1)
	stream, err := client.meetings.SubscribeMeetingEvents(streamCtx, connect.NewRequest(&meetingsv1.SubscribeMeetingEventsRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
	}))
	if err != nil {
		t.Fatalf("subscribe meeting events: %v", err)
	}
	defer stream.Close()
	streamDone := make(chan struct{})
	go func() {
		defer close(streamDone)
		for stream.Receive() {
			eventCh <- stream.Msg().GetKind()
		}
		if err := stream.Err(); err != nil {
			select {
			case streamErrCh <- err:
			default:
			}
		}
	}()

	// Consume the initial event that confirms the stream is live.
	select {
	case <-eventCh:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for initial stream event")
	case err := <-streamErrCh:
		t.Fatalf("meeting event stream failed before initial event: %v", err)
	}

	// Trigger the mutation via a logged-in chairperson.
	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	if _, err := client.moderation.ToggleSignupOpen(context.Background(), connect.NewRequest(&moderationv1.ToggleSignupOpenRequest{
		CommitteeSlug:   "test-committee",
		MeetingId:       fmt.Sprintf("%d", meetingID),
		DesiredOpen:     true,
		ExpectedVersion: 0,
	})); err != nil {
		t.Fatalf("toggle signup: %v", err)
	}

	// The Connect stream should deliver the typed invalidation event.
	select {
	case kind := <-eventCh:
		if kind != meetingsv1.MeetingEventKind_MEETING_EVENT_KIND_ATTENDEES_UPDATED {
			t.Fatalf("unexpected event kind: %v", kind)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for invalidation event")
	case err := <-streamErrCh:
		t.Fatalf("meeting event stream failed before invalidation: %v", err)
	}

	cancelStream()
	select {
	case <-streamDone:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for meeting event stream shutdown")
	}
}
