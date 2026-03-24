package apiconnect

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	connect "connectrpc.com/connect"

	moderationv1 "github.com/Y4shin/conference-tool/gen/go/conference/moderation/v1"
	sessionv1 "github.com/Y4shin/conference-tool/gen/go/conference/session/v1"
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

	// Open the SSE connection before triggering the mutation.
	sseURL := fmt.Sprintf("%s/api/realtime/meetings/%d/events", ts.server.URL, meetingID)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, sseURL, nil)
	if err != nil {
		t.Fatalf("build SSE request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("open SSE connection: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected SSE status: %d", resp.StatusCode)
	}

	// Channel to receive parsed SSE event lines.
	eventCh := make(chan string, 8)
	go func() {
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data:") {
				eventCh <- strings.TrimPrefix(line, "data:")
			}
		}
	}()

	// Consume the initial "connected" ping.
	select {
	case <-eventCh:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for connected ping")
	}

	// Trigger the mutation via a logged-in chairperson.
	client := newCombinedTestClient(t, ts)

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

	// The SSE stream should deliver the invalidation event.
	select {
	case data := <-eventCh:
		if !strings.Contains(data, "attendees.updated") {
			t.Fatalf("unexpected event data: %q", data)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for invalidation event")
	}
}

