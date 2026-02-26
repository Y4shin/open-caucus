//go:build e2e

package e2e_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

func moderateURL(baseURL, slug, meetingID string) string {
	return fmt.Sprintf("%s/committee/%s/meeting/%s/moderate", baseURL, slug, meetingID)
}

// TestModeratePage_ChairpersonCanAccess verifies a chairperson user session can
// access the moderate page, including left workspace tabs and right-column cards.
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

	for _, tabName := range []string{"Agenda", "Tools", "Attendees", "Settings"} {
		if err := page.Locator("#moderate-left-controls [data-moderate-left-tab]:has-text('" + tabName + "')").WaitFor(); err != nil {
			t.Fatalf("expected %q left tab on moderate page: %v", tabName, err)
		}
	}
	for _, heading := range []string{"Speakers", "Add Speaker"} {
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
	if err := page.Locator("#speakers-list-container [data-testid='live-speaker-item'][data-speaker-state='speaking']:has-text('First Speaker')").WaitFor(); err != nil {
		t.Fatalf("expected First Speaker to be speaking after start-next: %v", err)
	}

	endBtn := page.Locator("#speakers-list-container button[title='End current speech']")
	if err := endBtn.WaitFor(); err != nil {
		t.Fatalf("expected end-speech control: %v", err)
	}
	if err := endBtn.Click(); err != nil {
		t.Fatalf("click end-speech: %v", err)
	}
	if err := page.Locator("#speakers-list-container [data-testid='live-speaker-item'][data-speaker-state='speaking']").WaitFor(playwright.LocatorWaitForOptions{
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

	if err := page.Locator("#speaker-add-search-input").Fill("Alice"); err != nil {
		t.Fatalf("fill speaker search: %v", err)
	}
	candidateCard := page.Locator("#speaker-add-candidates-container [data-testid='manage-speaker-candidate-card']").Filter(playwright.LocatorFilterOptions{
		HasText: "Alice Speaker",
	})
	if err := candidateCard.WaitFor(); err != nil {
		t.Fatalf("expected Alice Speaker candidate card: %v", err)
	}
	if err := candidateCard.Locator("button[title='Add regular speech']").Click(); err != nil {
		t.Fatalf("click add regular speech: %v", err)
	}
	if err := page.Locator("#speakers-list-container [data-testid='live-speaker-item']:has-text('Alice Speaker')").WaitFor(); err != nil {
		t.Fatalf("expected Alice Speaker in speakers list after add: %v", err)
	}
	if page.URL() != urlBefore {
		t.Errorf("URL changed during add-speaker from attendees card: got %s, want %s", page.URL(), urlBefore)
	}
}

// TestModeratePage_SearchEnterAddsBestMatch verifies Enter behavior in the
// inline add-speaker search: add top candidate as regular, then clear and
// retain focus for rapid re-entry.
func TestModeratePage_SearchEnterAddsBestMatch(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "Main Topic")
	ts.activateAgendaPoint(t, "test-committee", "Board Meeting", apID)
	ts.seedAttendee(t, "test-committee", "Board Meeting", "Alice Speaker", "secret-alice-s")
	ts.seedAttendee(t, "test-committee", "Board Meeting", "Alicia Speaker", "secret-alicia-s")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(moderateURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto moderate page: %v", err)
	}

	search := page.Locator("#speaker-add-search-input")
	if err := search.Fill("alice"); err != nil {
		t.Fatalf("fill moderate speaker search: %v", err)
	}

	firstCandidate := page.Locator("#speaker-add-candidates-container [data-testid='manage-speaker-candidate-card']").First()
	if err := firstCandidate.WaitFor(); err != nil {
		t.Fatalf("wait first candidate card: %v", err)
	}
	if err := firstCandidate.Locator("text=Alice Speaker").WaitFor(); err != nil {
		t.Fatalf("expected best match candidate first in moderate pane: %v", err)
	}

	if err := search.Press("Enter"); err != nil {
		t.Fatalf("press enter in moderate search: %v", err)
	}
	if err := page.Locator("#speakers-list-container [data-testid='live-speaker-item']:has-text('Alice Speaker')").WaitFor(); err != nil {
		t.Fatalf("expected Enter to add top candidate on moderate pane: %v", err)
	}

	value, err := search.InputValue()
	if err != nil {
		t.Fatalf("read moderate search value after enter-add: %v", err)
	}
	if value != "" {
		t.Fatalf("expected moderate search input to be cleared after enter-add, got %q", value)
	}

	activeIDRaw, err := page.Evaluate(`() => (document.activeElement && document.activeElement.id) || ""`, nil)
	if err != nil {
		t.Fatalf("read active element id: %v", err)
	}
	activeID, _ := activeIDRaw.(string)
	if activeID != "speaker-add-search-input" {
		t.Fatalf("expected moderate search input to stay focused after enter-add, got active id %q", activeID)
	}
}

// TestModeratePage_SearchEnterAddsMultipleConsecutive verifies repeated add
// operations via textbox + Enter, including numeric exact-priority matching and
// duplicate-protection when top candidate is already waiting regular.
func TestModeratePage_SearchEnterAddsMultipleConsecutive(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "Main Topic")
	ts.activateAgendaPoint(t, "test-committee", "Board Meeting", apID)

	alpha := ts.seedAttendee(t, "test-committee", "Board Meeting", "Alpha Zero", "secret-alpha")
	ts.seedAttendee(t, "test-committee", "Board Meeting", "Bravo One", "secret-bravo")
	ts.seedAttendee(t, "test-committee", "Board Meeting", "Charlie Two", "secret-charlie")
	_ = ts.seedAttendee(t, "test-committee", "Board Meeting", "Delta Four", "secret-delta")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(moderateURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto moderate page: %v", err)
	}

	search := page.Locator("#speaker-add-search-input")
	addViaEnter := func(query, expectedTopName string) {
		t.Helper()
		if err := search.Fill(query); err != nil {
			t.Fatalf("fill search input with %q: %v", query, err)
		}

		if _, err := page.Evaluate(`() => new Promise((resolve) => setTimeout(resolve, 260))`, nil); err != nil {
			t.Fatalf("wait for search update for query %q: %v", query, err)
		}
		if err := page.Locator("#speaker-add-candidates-container [data-testid='manage-speaker-candidate-card']").First().WaitFor(); err != nil {
			t.Fatalf("wait candidates after query %q: %v", query, err)
		}
		expectedCard := page.Locator("#speaker-add-candidates-container [data-testid='manage-speaker-candidate-card']").Filter(playwright.LocatorFilterOptions{
			HasText: expectedTopName,
		})
		if err := expectedCard.First().WaitFor(); err != nil {
			t.Fatalf("wait expected candidate %q after query %q: %v", expectedTopName, query, err)
		}
		if err := page.Locator("#speaker-add-candidates-container [data-testid='manage-speaker-candidate-card']").First().Locator("text=" + expectedTopName).WaitFor(); err != nil {
			t.Fatalf("expected %q to be first candidate after query %q: %v", expectedTopName, query, err)
		}

		if err := search.Press("Enter"); err != nil {
			t.Fatalf("press Enter for query %q: %v", query, err)
		}
		if err := page.Locator("#speakers-list-container [data-testid='live-speaker-item']:has-text('" + expectedTopName + "')").WaitFor(); err != nil {
			t.Fatalf("expected %q to be added after Enter: %v", expectedTopName, err)
		}

		value, err := search.InputValue()
		if err != nil {
			t.Fatalf("read search value after Enter for %q: %v", query, err)
		}
		if value != "" {
			t.Fatalf("expected search input cleared after add for query %q, got %q", query, value)
		}

		activeIDRaw, err := page.Evaluate(`() => (document.activeElement && document.activeElement.id) || ""`, nil)
		if err != nil {
			t.Fatalf("read active element id after Enter for %q: %v", query, err)
		}
		activeID, _ := activeIDRaw.(string)
		if activeID != "speaker-add-search-input" {
			t.Fatalf("expected moderate search focus after Enter for %q, got %q", query, activeID)
		}

		// Let the clear-triggered candidate refresh settle before the next query.
		if _, err := page.Evaluate(`() => new Promise((resolve) => setTimeout(resolve, 260))`, nil); err != nil {
			t.Fatalf("wait for post-add candidate refresh for query %q: %v", query, err)
		}
	}

	// Text ranking.
	addViaEnter("alpha", "Alpha Zero")
	// Additional consecutive entries using distinct text queries.
	addViaEnter("charlie", "Charlie Two")
	addViaEnter("bravo", "Bravo One")

	for _, name := range []string{"Alpha Zero", "Charlie Two", "Bravo One"} {
		if err := page.Locator("#speakers-list-container [data-testid='live-speaker-item']:has-text('" + name + "')").WaitFor(); err != nil {
			t.Fatalf("expected %q in speakers list after consecutive adds: %v", name, err)
		}
	}

	// Duplicate-protection: top candidate already waiting regular -> Enter no-op.
	alphaRowsBefore, err := page.Locator("#speakers-list-container [data-testid='live-speaker-item']:has-text('Alpha Zero')").Count()
	if err != nil {
		t.Fatalf("count Alpha rows before duplicate Enter: %v", err)
	}
	if err := search.Fill(strconv.FormatInt(alpha.AttendeeNumber, 10)); err != nil {
		t.Fatalf("fill duplicate-protection query: %v", err)
	}
	if _, err := page.Evaluate(`() => new Promise((resolve) => setTimeout(resolve, 260))`, nil); err != nil {
		t.Fatalf("wait for duplicate-protection search update: %v", err)
	}
	if err := search.Press("Enter"); err != nil {
		t.Fatalf("press Enter for duplicate-protection query: %v", err)
	}
	if _, err := page.Evaluate(`() => new Promise((resolve) => setTimeout(resolve, 250))`, nil); err != nil {
		t.Fatalf("wait after duplicate Enter: %v", err)
	}
	alphaRowsAfter, err := page.Locator("#speakers-list-container [data-testid='live-speaker-item']:has-text('Alpha Zero')").Count()
	if err != nil {
		t.Fatalf("count Alpha rows after duplicate Enter: %v", err)
	}
	if alphaRowsAfter != alphaRowsBefore {
		t.Fatalf("expected no duplicate Alpha row on Enter, before=%d after=%d", alphaRowsBefore, alphaRowsAfter)
	}

	valueAfterNoop, err := search.InputValue()
	if err != nil {
		t.Fatalf("read search value after duplicate no-op: %v", err)
	}
	if strings.TrimSpace(valueAfterNoop) == "" {
		t.Fatalf("expected search value to remain when Enter does not add a speaker")
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

	if err := moderatePage.Locator("#speakers-list-container [data-testid='live-speaker-item']:has-text('Alice Member')").WaitFor(); err != nil {
		t.Fatalf("expected Alice Member to appear on moderate page via SSE: %v", err)
	}
	if moderatePage.URL() != modURLBefore {
		t.Errorf("moderate page URL changed during SSE update: before=%s after=%s", modURLBefore, moderatePage.URL())
	}
}
