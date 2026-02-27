//go:build e2e

package e2e_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	playwright "github.com/playwright-community/playwright-go"
)

func attendeeLiveSpeakerRows(page playwright.Page) playwright.Locator {
	return page.Locator("#attendee-speakers-list [data-testid='live-speakers-active-list'] [data-testid='live-speaker-item']")
}

// attendeeLoginHelper navigates to the attendee-login page and authenticates with the given secret.
func attendeeLoginHelper(t *testing.T, page playwright.Page, baseURL, slug, meetingID, secret string) {
	t.Helper()
	if _, err := page.Goto(attendeeLoginURL(baseURL, slug, meetingID)); err != nil {
		t.Fatalf("goto attendee-login: %v", err)
	}
	if err := page.Locator("input[name=secret]").Fill(secret); err != nil {
		t.Fatalf("fill secret: %v", err)
	}
	if err := page.Locator("input[name=secret]").Press("Enter"); err != nil {
		t.Fatalf("submit attendee login: %v", err)
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

	attendeePage := newPage(t)
	attendeeLoginHelper(t, attendeePage, ts.URL, "test-committee", meetingID, "secret-alice")
	// Give the HTMX SSE extension a moment to establish the live stream before publishing updates.
	time.Sleep(800 * time.Millisecond)
	urlBefore := attendeePage.URL()

	initialRows, err := attendeeLiveSpeakerRows(attendeePage).Count()
	if err != nil {
		t.Fatalf("count initial live speaker rows: %v", err)
	}
	if initialRows != 0 {
		t.Fatalf("expected empty speakers list before manage update, got %d rows", initialRows)
	}

	chairPage := newPage(t)
	userLogin(t, chairPage, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := chairPage.Goto(agendaManageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	openSpeakerAddDialog(t, chairPage)
	aliceCard := speakerCandidateCard(chairPage, "Alice Speaker")
	if err := aliceCard.Locator("button[title='Add regular speech']").Click(); err != nil {
		t.Fatalf("add regular speech for Alice: %v", err)
	}
	if err := chairPage.Locator("#speakers-list-container [data-testid='live-speaker-item']:has-text('Alice Speaker')").WaitFor(); err != nil {
		t.Fatalf("chair page should show Alice in speaker list: %v", err)
	}

	if err := attendeePage.Locator("#attendee-speakers-list [data-testid='live-speakers-active-viewport'] [data-testid='live-speaker-item']:has-text('Alice Speaker')").WaitFor(); err != nil {
		t.Fatalf("attendee page should receive SSE speaker update: %v", err)
	}
	if attendeePage.URL() != urlBefore {
		t.Errorf("attendee page URL changed: got %s, want %s", attendeePage.URL(), urlBefore)
	}
}

// TestAttendee_SeesOwnPositionHighlighted verifies that an attendee who is in
// the speakers queue sees their own row marked with a mine-state attribute.
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

	if err := page.Locator("#attendee-speakers-list [data-testid='live-speakers-active-viewport'] [data-testid='live-speaker-item'][data-speaker-mine='true']:has-text('Alice Member')").WaitFor(); err != nil {
		t.Fatalf("expected Alice's row to be marked as mine: %v", err)
	}
}

// TestAttendee_QuotedBadgeVisible verifies that a speaker whose gender-quoted
// flag is set sees the quoted indicator in their row on the live page.
func TestAttendee_QuotedBadgeVisible(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeeting(t, "test-committee", "Quoted Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Quoted Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Quoted Meeting", "Agenda Item")
	ts.activateAgendaPoint(t, "test-committee", "Quoted Meeting", apID)
	bob := ts.seedAttendee(t, "test-committee", "Quoted Meeting", "Bob Quoted", "secret-bob")

	var apid int64
	fmt.Sscanf(apID, "%d", &apid)
	if _, err := ts.repo.AddSpeaker(context.Background(), apid, bob.ID, "regular", true, false); err != nil {
		t.Fatalf("add quoted speaker: %v", err)
	}

	page := newPage(t)
	attendeeLoginHelper(t, page, ts.URL, "test-committee", meetingID, "secret-bob")

	bobRow := page.Locator("#attendee-speakers-list [data-testid='live-speakers-active-viewport'] [data-testid='live-speaker-item']").Filter(playwright.LocatorFilterOptions{
		HasText: "Bob Quoted",
	})
	if err := bobRow.WaitFor(); err != nil {
		t.Fatalf("expected Bob's row in speakers list: %v", err)
	}
	if err := bobRow.Locator("[data-testid='live-speaker-quoted-badge']").WaitFor(); err != nil {
		t.Fatalf("expected quoted speaker badge in Bob's row: %v", err)
	}
}

// TestAttendeeLive_SelfAddButtons verifies attendees can add themselves to the
// speakers queue via the live-page quick actions for regular and ropm.
func TestAttendeeLive_SelfAddButtons(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeeting(t, "test-committee", "Live Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Live Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Live Meeting", "Main Topic")
	ts.activateAgendaPoint(t, "test-committee", "Live Meeting", apID)
	ts.seedAttendee(t, "test-committee", "Live Meeting", "Alice Speaker", "secret-alice")

	page := newPage(t)
	attendeeLoginHelper(t, page, ts.URL, "test-committee", meetingID, "secret-alice")

	if err := page.Locator("[data-testid='live-add-self-regular']").Click(); err != nil {
		t.Fatalf("click add self regular: %v", err)
	}
	if err := page.Locator("#attendee-speakers-list [data-testid='live-speakers-active-viewport'] [data-testid='live-speaker-item']:has-text('Alice Speaker')").WaitFor(); err != nil {
		t.Fatalf("expected regular self-added speaker row: %v", err)
	}

	regularDisabled, err := page.Locator("[data-testid='live-add-self-regular']").IsDisabled()
	if err != nil {
		t.Fatalf("read regular self-add disabled state: %v", err)
	}
	if !regularDisabled {
		t.Fatalf("expected regular self-add button to disable after regular entry")
	}

	if err := page.Locator("[data-testid='live-add-self-ropm']").Click(); err != nil {
		t.Fatalf("click add self ropm: %v", err)
	}
	waitUntil(t, 3*time.Second, func() (bool, error) {
		count, err := page.Locator("#attendee-speakers-list [data-testid='live-speakers-active-viewport'] [data-testid='live-speaker-item']:has-text('Alice Speaker')").Count()
		return count >= 2, err
	}, "second speaker row for Alice after ropm self-add")
}



