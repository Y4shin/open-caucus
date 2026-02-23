//go:build e2e

package e2e_test

import (
	"fmt"
	"strconv"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

func moderateURL(baseURL, slug, meetingID string) string {
	return fmt.Sprintf("%s/committee/%s/meeting/%s/moderate", baseURL, slug, meetingID)
}

// TestModeratePage_ChairpersonCanAccess verifies a chairperson user session can
// access the moderate page and all four card headings are rendered.
func TestModeratePage_ChairpersonCanAccess(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(moderateURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto moderate page: %v", err)
	}

	for _, heading := range []string{"Agenda", "Tools", "Speakers", "Add Speaker"} {
		if err := page.Locator("h2:has-text('" + heading + "')").WaitFor(); err != nil {
			t.Fatalf("expected %q card heading on moderate page: %v", heading, err)
		}
	}
}

// TestModeratePage_AttendeeNonChair_Forbidden verifies that a non-chair attendee
// session receives a 403 when attempting to access the moderate page.
func TestModeratePage_AttendeeNonChair_Forbidden(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeetingOpen(t, "test-committee", "Open Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Open Meeting")
	ts.seedAttendee(t, "test-committee", "Open Meeting", "Nonchair Guest", "secret-nonchair")

	page := newPage(t)
	attendeeLoginHelper(t, page, ts.URL, "test-committee", meetingID, "secret-nonchair")

	resp, err := page.Goto(moderateURL(ts.URL, "test-committee", meetingID))
	if err != nil {
		t.Fatalf("goto moderate page: %v", err)
	}
	if resp.Status() != 403 {
		t.Fatalf("expected 403 for /moderate as non-chair attendee, got %d", resp.Status())
	}
}

// TestModeratePage_AttendeeChair_CanAccess verifies a chair attendee session can
// access the moderate page.
func TestModeratePage_AttendeeChair_CanAccess(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeetingOpen(t, "test-committee", "Open Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Open Meeting")
	attendee := ts.seedAttendee(t, "test-committee", "Open Meeting", "Chair Guest", "secret-chair")
	ts.setAttendeeChair(t, attendee.ID, true)

	page := newPage(t)
	attendeeLoginHelper(t, page, ts.URL, "test-committee", meetingID, "secret-chair")
	if _, err := page.Goto(moderateURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto moderate page: %v", err)
	}
	if err := page.Locator("h2:has-text('Speakers')").WaitFor(); err != nil {
		t.Fatalf("expected moderate page to load for chair attendee: %v", err)
	}
}

// TestModeratePage_AttendeeModerator_CanAccess verifies that a non-chair attendee
// who is the meeting's designated moderator can access the moderate page.
func TestModeratePage_AttendeeModerator_CanAccess(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeetingOpen(t, "test-committee", "Open Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Open Meeting")
	attendee := ts.seedAttendee(t, "test-committee", "Open Meeting", "Mod Guest", "secret-mod")

	// Not a chair, but assigned as meeting moderator.
	ts.setMeetingModerator(t, "test-committee", "Open Meeting", &attendee.ID)

	page := newPage(t)
	attendeeLoginHelper(t, page, ts.URL, "test-committee", meetingID, "secret-mod")
	if _, err := page.Goto(moderateURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto moderate page: %v", err)
	}
	if err := page.Locator("h2:has-text('Speakers')").WaitFor(); err != nil {
		t.Fatalf("expected moderate page to load for designated meeting moderator: %v", err)
	}
}

// TestModeratePage_SpeakerQuickControls verifies the start-next and end-speech
// quick controls work on the moderate page without a full page reload.
func TestModeratePage_SpeakerQuickControls(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "Main Topic")
	ts.activateAgendaPoint(t, "test-committee", "Board Meeting", apID)
	first := ts.seedAttendee(t, "test-committee", "Board Meeting", "First Speaker", "secret-first-m")
	ts.seedSpeaker(t, apID, strconv.FormatInt(first.ID, 10))

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(moderateURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto moderate page: %v", err)
	}
	urlBefore := page.URL()

	startBtn := page.Locator("#speakers-list-container button[title='Start next speaker']")
	if err := startBtn.WaitFor(); err != nil {
		t.Fatalf("expected start-next control on moderate page: %v", err)
	}
	if err := startBtn.Click(); err != nil {
		t.Fatalf("click start-next: %v", err)
	}
	if err := page.Locator("#speakers-list-container .live-speaker-row.speaking:has-text('First Speaker')").WaitFor(); err != nil {
		t.Fatalf("expected First Speaker to be speaking after start-next: %v", err)
	}

	endBtn := page.Locator("#speakers-list-container button[title='End current speech']")
	if err := endBtn.WaitFor(); err != nil {
		t.Fatalf("expected end-speech control: %v", err)
	}
	if err := endBtn.Click(); err != nil {
		t.Fatalf("click end-speech: %v", err)
	}
	if err := page.Locator("#speakers-list-container .live-speaker-row.speaking").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected no speaking row after end-speech: %v", err)
	}
	if page.URL() != urlBefore {
		t.Errorf("URL changed during moderate quick controls: got %s, want %s", page.URL(), urlBefore)
	}
}

// TestModeratePage_AddSpeakerFromAttendeesCard verifies that the inline search
// in the Add Speaker card can add an attendee to the speakers queue.
func TestModeratePage_AddSpeakerFromAttendeesCard(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "Main Topic")
	ts.activateAgendaPoint(t, "test-committee", "Board Meeting", apID)
	ts.seedAttendee(t, "test-committee", "Board Meeting", "Alice Speaker", "secret-alice-s")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(moderateURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto moderate page: %v", err)
	}
	urlBefore := page.URL()

	if err := page.Locator("#moderate-speaker-search").Fill("Alice"); err != nil {
		t.Fatalf("fill speaker search: %v", err)
	}
	candidateCard := page.Locator("#speaker-add-candidates-container .manage-speaker-candidate-card").Filter(playwright.LocatorFilterOptions{
		HasText: "Alice Speaker",
	})
	if err := candidateCard.WaitFor(); err != nil {
		t.Fatalf("expected Alice Speaker candidate card: %v", err)
	}
	if err := candidateCard.Locator("button[title='Add regular speech']").Click(); err != nil {
		t.Fatalf("click add regular speech: %v", err)
	}
	if err := page.Locator("#speakers-list-container .live-speaker-row:has-text('Alice Speaker')").WaitFor(); err != nil {
		t.Fatalf("expected Alice Speaker in speakers list after add: %v", err)
	}
	if page.URL() != urlBefore {
		t.Errorf("URL changed during add-speaker from attendees card: got %s, want %s", page.URL(), urlBefore)
	}
}

// TestModeratePage_SSE_SpeakerUpdatePropagates verifies that a speaker added via
// the manage page appears on an open moderate page via SSE without a reload.
func TestModeratePage_SSE_SpeakerUpdatePropagates(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "Main Topic")
	ts.activateAgendaPoint(t, "test-committee", "Board Meeting", apID)
	ts.seedAttendee(t, "test-committee", "Board Meeting", "Alice Member", "secret-alice-m")

	moderatePage := newPage(t)
	userLogin(t, moderatePage, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := moderatePage.Goto(moderateURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto moderate page: %v", err)
	}
	if err := moderatePage.Locator("h2:has-text('Speakers')").WaitFor(); err != nil {
		t.Fatalf("moderate page did not load: %v", err)
	}
	modURLBefore := moderatePage.URL()

	managePage := newPage(t)
	userLogin(t, managePage, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := managePage.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}
	openSpeakerAddDialog(t, managePage)
	if err := speakerCandidateCard(managePage, "Alice Member").Locator("button[title='Add regular speech']").Click(); err != nil {
		t.Fatalf("add speaker from manage page: %v", err)
	}

	if err := moderatePage.Locator("#speakers-list-container .live-speaker-row:has-text('Alice Member')").WaitFor(); err != nil {
		t.Fatalf("expected Alice Member to appear on moderate page via SSE: %v", err)
	}
	if moderatePage.URL() != modURLBefore {
		t.Errorf("moderate page URL changed during SSE update: before=%s after=%s", modURLBefore, moderatePage.URL())
	}
}
