//go:build e2e

package e2e_test

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

// attendeeLoginHelper navigates to the attendee-login page and authenticates with the given secret.
func attendeeLoginHelper(t *testing.T, page playwright.Page, baseURL, slug, meetingID, secret string) {
	t.Helper()
	if _, err := page.Goto(attendeeLoginURL(baseURL, slug, meetingID)); err != nil {
		t.Fatalf("goto attendee-login: %v", err)
	}
	if err := page.Locator("input[name=secret]").Fill(secret); err != nil {
		t.Fatalf("fill secret: %v", err)
	}
	if err := page.Locator("button[type=submit]").Click(); err != nil {
		t.Fatalf("click submit: %v", err)
	}
	if err := page.WaitForURL(liveURL(baseURL, slug, meetingID)); err != nil {
		t.Fatalf("wait for /live: %v", err)
	}
}

// TestAttendee_SpeakersListUpdates_ViaSSE verifies that when a chairperson adds
// a speaker via the manage page, the attendee's live page updates automatically
// via SSE without a full page reload.
func TestAttendee_SpeakersListUpdates_ViaSSE(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Live Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Live Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Live Meeting", "Main Topic")
	ts.activateAgendaPoint(t, "test-committee", "Live Meeting", apID)
	ts.seedAttendee(t, "test-committee", "Live Meeting", "Alice Speaker", "secret-alice")

	// Attendee page: Alice opens /live and waits for the initial state.
	attendeePage := newPage(t)
	attendeeLoginHelper(t, attendeePage, ts.URL, "test-committee", meetingID, "secret-alice")

	// Verify no speakers are queued yet.
	if err := attendeePage.Locator("#attendee-speakers-list p:has-text('No speakers')").WaitFor(); err != nil {
		t.Fatalf("expected 'no speakers' message on attendee page: %v", err)
	}

	urlBefore := attendeePage.URL()

	// Chair page: log in and add Alice as a speaker.
	chairPage := newPage(t)
	userLogin(t, chairPage, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := chairPage.Goto(agendaManageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}
	if _, err := chairPage.Locator("#speaker_attendee_id").SelectOption(playwright.SelectOptionValues{
		Labels: playwright.StringSlice("Alice Speaker"),
	}); err != nil {
		t.Fatalf("select attendee on chair page: %v", err)
	}
	if err := chairPage.Locator("button:has-text('Add Speaker')").Click(); err != nil {
		t.Fatalf("click add speaker: %v", err)
	}

	// Confirm speaker was added on the chair's page first.
	if err := chairPage.Locator("#speakers-list-container td:has-text('Alice Speaker')").WaitFor(); err != nil {
		t.Fatalf("chair page: expected Alice Speaker in speakers table: %v", err)
	}

	// Attendee page: SSE should deliver the update; Alice's row must appear.
	if err := attendeePage.Locator("#attendee-speakers-list td:has-text('Alice Speaker')").WaitFor(); err != nil {
		t.Fatalf("attendee page: expected Alice Speaker via SSE update: %v", err)
	}

	// Confirm no full-page navigation occurred on the attendee's page.
	if attendeePage.URL() != urlBefore {
		t.Errorf("attendee page URL changed: got %s, want %s", attendeePage.URL(), urlBefore)
	}
}

// TestAttendee_SeesOwnPositionHighlighted verifies that an attendee who is in
// the speakers queue sees their own row marked with the "my-turn" CSS class.
func TestAttendee_SeesOwnPositionHighlighted(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeeting(t, "test-committee", "Highlight Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Highlight Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Highlight Meeting", "Agenda Item")
	ts.activateAgendaPoint(t, "test-committee", "Highlight Meeting", apID)
	alice := ts.seedAttendee(t, "test-committee", "Highlight Meeting", "Alice Member", "secret-alice")
	aliceIDStr := strconv.FormatInt(alice.ID, 10)
	ts.seedSpeaker(t, apID, aliceIDStr)

	page := newPage(t)
	attendeeLoginHelper(t, page, ts.URL, "test-committee", meetingID, "secret-alice")

	// Alice's row should be present and carry the "my-turn" highlight class.
	if err := page.Locator("#attendee-speakers-list tr.my-turn:has-text('Alice Member')").WaitFor(); err != nil {
		t.Fatalf("expected Alice's row to have 'my-turn' class: %v", err)
	}
}

// TestAttendee_QuotedBadgeVisible verifies that a speaker whose gender-quoted
// flag is set sees the "Q" indicator in their row on the live page.
func TestAttendee_QuotedBadgeVisible(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeeting(t, "test-committee", "Quoted Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Quoted Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Quoted Meeting", "Agenda Item")
	ts.activateAgendaPoint(t, "test-committee", "Quoted Meeting", apID)
	bob := ts.seedAttendee(t, "test-committee", "Quoted Meeting", "Bob Quoted", "secret-bob")

	// Add Bob as a speaker with gender_quoted=true.
	var apid int64
	fmt.Sscanf(apID, "%d", &apid)
	if _, err := ts.repo.AddSpeaker(context.Background(), apid, bob.ID, "regular", true, false); err != nil {
		t.Fatalf("add quoted speaker: %v", err)
	}

	page := newPage(t)
	attendeeLoginHelper(t, page, ts.URL, "test-committee", meetingID, "secret-bob")

	// Bob's row must be visible.
	if err := page.Locator("#attendee-speakers-list td:has-text('Bob Quoted')").WaitFor(); err != nil {
		t.Fatalf("expected Bob's row in speakers list: %v", err)
	}

	// The row must contain the "Q" quoted badge.
	bobRow := page.Locator("#attendee-speakers-list tr:has-text('Bob Quoted')")
	rowText, err := bobRow.TextContent()
	if err != nil {
		t.Fatalf("get Bob's row text: %v", err)
	}
	if !strings.Contains(rowText, "Q") {
		t.Errorf("expected 'Q' badge in Bob's row, got: %q", rowText)
	}
}
