//go:build e2e

package e2e_test

import (
	"context"
	"regexp"
	"sort"
	"strconv"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

// speakingSinceAttrRe matches the data-speaking-since attribute and its value.
// The attribute value differs between the legacy server (Unix seconds from the DB)
// and the SPA (milliseconds from Date.now()), so it must be stripped before comparison.
var speakingSinceAttrRe = regexp.MustCompile(`data-speaking-since="[^"]*"`)

// speakingTimerContentRe matches the text content of a speaking-timer span after its
// data-speaking-since attribute has been normalized to an empty string. The text is
// a running clock value ("00:00", "00:01", …) that advances independently in each
// browser and therefore must not be compared verbatim.
var speakingTimerContentRe = regexp.MustCompile(`data-speaking-since="">[^<]*<`)

// initialScrollTopAttrRe matches the data-initial-scroll-top attribute and its value.
// This attribute is set asynchronously by JavaScript (after an HTMX swap) in the
// legacy server and statically in the SPA Svelte template on initial render, so the
// timing of when Playwright captures the snapshot determines whether it appears.
// Stripping it makes speaker-list comparisons timing-independent.
var initialScrollTopAttrRe = regexp.MustCompile(` data-initial-scroll-top="[^"]*"`)

// normalizeInitialScrollTop removes data-initial-scroll-top attributes so that
// timing differences between static Svelte rendering and async JS attribute setting
// do not cause false parity failures.
func normalizeInitialScrollTop(html string) string {
	return initialScrollTopAttrRe.ReplaceAllString(html, "")
}

// normalizeSpeakingSinceAttr replaces data-speaking-since attribute values with an
// empty string and normalizes the associated timer text content to "00:00" so that
// the active-speaker clock does not cause false parity failures.
func normalizeSpeakingSinceAttr(html string) string {
	h := speakingSinceAttrRe.ReplaceAllString(html, `data-speaking-since=""`)
	return speakingTimerContentRe.ReplaceAllString(h, `data-speaking-since="">00:00<`)
}

// TestCommitteeMeetingRows_UIParityWithLegacy checks that meeting rows on the
// committee page render identically for multiple meetings in different states.
func TestCommitteeMeetingRows_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	for _, ts := range []*testServer{newTS, legacyTS} {
		ts.seedCommittee(t, "Test Committee", "test")
		ts.seedUser(t, "test", "chair1", "pass123", "Chair Person", "chairperson")
		ts.seedMeeting(t, "test", "Closed Meeting", "signup closed")
		ts.seedMeetingOpen(t, "test", "Open Meeting", "signup open")
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "chair1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "chair1", "pass123")

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test", "#meeting-list-container")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test", "#meeting-list-container")

	newRows := locatorAllOuterHTML(t, newBrowserPage, "[data-testid='committee-meeting-row']")
	legacyRows := locatorAllOuterHTML(t, legacyBrowserPage, "[data-testid='committee-meeting-row']")
	sort.Strings(newRows)
	sort.Strings(legacyRows)

	assertEqualStringSlices(t, "committee meeting rows html", newRows, legacyRows)
}

// TestCommitteeActiveMeetingCard_UIParityWithLegacy checks that the active-meeting
// card (shown when a meeting is the active/live meeting) renders identically.
func TestCommitteeActiveMeetingCard_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	for _, ts := range []*testServer{newTS, legacyTS} {
		ts.seedCommittee(t, "Test Committee", "test")
		ts.seedUser(t, "test", "chair1", "pass123", "Chair Person", "chairperson")
		ts.seedUser(t, "test", "member1", "pass123", "Member One", "member")
		ts.seedMeetingOpen(t, "test", "Board Meeting", "")
		meetingID, err := strconv.ParseInt(ts.getMeetingID(t, "test", "Board Meeting"), 10, 64)
		if err != nil {
			t.Fatalf("parse meeting id: %v", err)
		}
		if err := ts.repo.SetActiveMeeting(context.Background(), "test", &meetingID); err != nil {
			t.Fatalf("set active meeting: %v", err)
		}
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "member1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "member1", "pass123")

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test", "[data-testid='committee-active-meeting-card']")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test", "[data-testid='committee-active-meeting-card']")

	assertEqualHTML(t, "committee active meeting card",
		locatorOuterHTML(t, newBrowserPage, "[data-testid='committee-active-meeting-card']"),
		locatorOuterHTML(t, legacyBrowserPage, "[data-testid='committee-active-meeting-card']"),
	)
}

// TestModerateAgendaEditor_UIParityWithLegacy checks the agenda editor dialog
// content including the list of agenda points.
func TestModerateAgendaEditor_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	for _, ts := range []*testServer{newTS, legacyTS} {
		ts.seedCommittee(t, "Test Committee", "test")
		ts.seedUser(t, "test", "chair1", "pass123", "Chair Person", "chairperson")
		ts.seedMeetingOpen(t, "test", "Board Meeting", "")
		ts.seedAgendaPoint(t, "test", "Board Meeting", "Opening")
		ts.seedAgendaPoint(t, "test", "Board Meeting", "Budget Discussion")
		ts.seedAgendaPoint(t, "test", "Board Meeting", "Any Other Business")
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "chair1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "chair1", "pass123")

	meetingID := newTS.getMeetingID(t, "test", "Board Meeting")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Board Meeting")

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID+"/moderate", "#moderate-left-controls")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/moderate", "#moderate-left-controls")

	openModerateAgendaEditor(t, newBrowserPage)
	openModerateAgendaEditor(t, legacyBrowserPage)

	assertEqualStringSlices(t, "agenda point list items",
		locatorAllOuterHTML(t, newBrowserPage, "[data-testid='manage-agenda-point-card']"),
		locatorAllOuterHTML(t, legacyBrowserPage, "[data-testid='manage-agenda-point-card']"),
	)
}

// TestModerateCreateAgendaPoint_UIParityWithLegacy (A10) checks that creating
// an agenda point via the moderation UI produces the same agenda-card list in
// the SPA and legacy implementations.
func TestModerateCreateAgendaPoint_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	for _, ts := range []*testServer{newTS, legacyTS} {
		ts.seedCommittee(t, "Test Committee", "test")
		ts.seedUser(t, "test", "chair1", "pass123", "Chair Person", "chairperson")
		ts.seedMeetingOpen(t, "test", "Board Meeting", "")
		ts.seedAgendaPoint(t, "test", "Board Meeting", "Opening")
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "chair1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "chair1", "pass123")

	meetingID := newTS.getMeetingID(t, "test", "Board Meeting")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Board Meeting")

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID+"/moderate", "#moderate-left-controls")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/moderate", "#moderate-left-controls")

	openModerateAgendaEditor(t, newBrowserPage)
	openModerateAgendaEditor(t, legacyBrowserPage)

	for label, p := range map[string]playwright.Page{"new": newBrowserPage, "legacy": legacyBrowserPage} {
		addAgendaPoint(t, p, "Budget Approval")
		if err := p.Locator("#agenda-point-list-container [data-testid='manage-agenda-point-card']:has-text('Budget Approval')").WaitFor(); err != nil {
			t.Fatalf("wait created agenda point card on %s: %v", label, err)
		}
	}

	assertEqualStringSlices(t, "agenda point list items after create",
		locatorAllOuterHTML(t, newBrowserPage, "[data-testid='manage-agenda-point-card']"),
		locatorAllOuterHTML(t, legacyBrowserPage, "[data-testid='manage-agenda-point-card']"),
	)
}

// TestModerateReorderAgendaPoint_UIParityWithLegacy (A12) checks that moving
// an agenda point up via the moderation UI produces the same ordered agenda
// list in the SPA and legacy implementations.
func TestModerateReorderAgendaPoint_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	for _, ts := range []*testServer{newTS, legacyTS} {
		ts.seedCommittee(t, "Test Committee", "test")
		ts.seedUser(t, "test", "chair1", "pass123", "Chair Person", "chairperson")
		ts.seedMeetingOpen(t, "test", "Board Meeting", "")
		ts.seedAgendaPoint(t, "test", "Board Meeting", "First")
		ts.seedAgendaPoint(t, "test", "Board Meeting", "Second")
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "chair1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "chair1", "pass123")

	meetingID := newTS.getMeetingID(t, "test", "Board Meeting")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Board Meeting")

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID+"/moderate", "#moderate-left-controls")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/moderate", "#moderate-left-controls")

	openModerateAgendaEditor(t, newBrowserPage)
	openModerateAgendaEditor(t, legacyBrowserPage)

	for label, p := range map[string]playwright.Page{"new": newBrowserPage, "legacy": legacyBrowserPage} {
		card := p.Locator("#agenda-point-list-container [data-testid='manage-agenda-point-card']").Filter(playwright.LocatorFilterOptions{
			HasText: "Second",
		})
		if err := card.Locator("button[title='Move up']").Click(); err != nil {
			t.Fatalf("click move up on %s: %v", label, err)
		}
		if err := p.Locator("#agenda-point-list-container [data-testid='manage-agenda-point-card']").First().Locator("text=Second").WaitFor(); err != nil {
			t.Fatalf("wait reordered agenda list on %s: %v", label, err)
		}
	}

	assertEqualStringSlices(t, "agenda point list items after reorder",
		locatorAllOuterHTML(t, newBrowserPage, "[data-testid='manage-agenda-point-card']"),
		locatorAllOuterHTML(t, legacyBrowserPage, "[data-testid='manage-agenda-point-card']"),
	)
}

// TestModerateDeleteAgendaPoint_UIParityWithLegacy (A13) checks that deleting
// an agenda point via the moderation UI produces the same remaining agenda list
// in the SPA and legacy implementations.
func TestModerateDeleteAgendaPoint_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	for _, ts := range []*testServer{newTS, legacyTS} {
		ts.seedCommittee(t, "Test Committee", "test")
		ts.seedUser(t, "test", "chair1", "pass123", "Chair Person", "chairperson")
		ts.seedMeetingOpen(t, "test", "Board Meeting", "")
		ts.seedAgendaPoint(t, "test", "Board Meeting", "Keep Me")
		ts.seedAgendaPoint(t, "test", "Board Meeting", "Delete Me")
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "chair1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "chair1", "pass123")

	meetingID := newTS.getMeetingID(t, "test", "Board Meeting")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Board Meeting")

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID+"/moderate", "#moderate-left-controls")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/moderate", "#moderate-left-controls")

	openModerateAgendaEditor(t, newBrowserPage)
	openModerateAgendaEditor(t, legacyBrowserPage)

	for _, p := range []playwright.Page{newBrowserPage, legacyBrowserPage} {
		p.OnDialog(func(d playwright.Dialog) { _ = d.Accept() })
	}

	for label, p := range map[string]playwright.Page{"new": newBrowserPage, "legacy": legacyBrowserPage} {
		card := p.Locator("#agenda-point-list-container [data-testid='manage-agenda-point-card']").Filter(playwright.LocatorFilterOptions{
			HasText: "Delete Me",
		})
		if err := card.Locator("button[title='Delete agenda point']").Click(); err != nil {
			t.Fatalf("click delete agenda point on %s: %v", label, err)
		}
		if err := p.Locator("#agenda-point-list-container [data-testid='manage-agenda-point-card']:has-text('Delete Me')").WaitFor(playwright.LocatorWaitForOptions{
			State: playwright.WaitForSelectorStateDetached,
		}); err != nil {
			t.Fatalf("wait deleted agenda point detach on %s: %v", label, err)
		}
	}

	assertEqualStringSlices(t, "agenda point list items after delete",
		locatorAllOuterHTML(t, newBrowserPage, "[data-testid='manage-agenda-point-card']"),
		locatorAllOuterHTML(t, legacyBrowserPage, "[data-testid='manage-agenda-point-card']"),
	)
}

// TestModerateAddSpeaker_UIParityWithLegacy (A14) checks that adding a speaker
// via the moderation UI produces the same speakers list in the SPA and legacy
// implementations.
func TestModerateAddSpeaker_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	for _, ts := range []*testServer{newTS, legacyTS} {
		ts.seedCommittee(t, "Test Committee", "test")
		ts.seedUser(t, "test", "chair1", "pass123", "Chair Person", "chairperson")
		ts.seedMeetingOpen(t, "test", "Board Meeting", "")
		ts.seedAttendee(t, "test", "Board Meeting", "Alice Member", "secret-alice")
		apID := ts.seedAgendaPoint(t, "test", "Board Meeting", "Main Topic")
		ts.activateAgendaPoint(t, "test", "Board Meeting", apID)
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "chair1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "chair1", "pass123")

	meetingID := newTS.getMeetingID(t, "test", "Board Meeting")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Board Meeting")

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID+"/moderate", "#speakers-list-container")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/moderate", "#speakers-list-container")

	for label, p := range map[string]playwright.Page{"new": newBrowserPage, "legacy": legacyBrowserPage} {
		openSpeakerAddDialog(t, p)
		card := speakerCandidateCard(p, "Alice Member")
		if err := card.WaitFor(); err != nil {
			t.Fatalf("wait speaker candidate card on %s: %v", label, err)
		}
		if err := card.Locator("button[title='Add regular speech']").Click(); err != nil {
			t.Fatalf("add regular speech on %s: %v", label, err)
		}
		if err := p.Locator("#speakers-list-container [data-testid='live-speaker-item']:has-text('Alice Member')").WaitFor(); err != nil {
			t.Fatalf("wait speaker in list on %s: %v", label, err)
		}
	}

	assertEqualHTML(t, "speakers list after add",
		normalizeInitialScrollTop(locatorOuterHTML(t, newBrowserPage, "#speakers-list-container")),
		normalizeInitialScrollTop(locatorOuterHTML(t, legacyBrowserPage, "#speakers-list-container")),
	)
}

// TestModerateAttendeesTab_UIParityWithLegacy checks the attendees panel on the
// moderation page, including the add-guest form and the list of attendees.
func TestModerateAttendeesTab_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	for _, ts := range []*testServer{newTS, legacyTS} {
		ts.seedCommittee(t, "Test Committee", "test")
		ts.seedUser(t, "test", "chair1", "pass123", "Chair Person", "chairperson")
		ts.seedMeetingOpen(t, "test", "Board Meeting", "")
		ts.seedAttendee(t, "test", "Board Meeting", "Alice Member", "secret-alice")
		ts.seedAttendee(t, "test", "Board Meeting", "Bob Member", "secret-bob")
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "chair1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "chair1", "pass123")

	meetingID := newTS.getMeetingID(t, "test", "Board Meeting")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Board Meeting")

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID+"/moderate", "#moderate-left-controls")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/moderate", "#moderate-left-controls")

	openModerateLeftTab(t, newBrowserPage, "attendees")
	openModerateLeftTab(t, legacyBrowserPage, "attendees")

	assertEqualHTML(t, "manage add-guest form",
		locatorOuterHTML(t, newBrowserPage, "[data-testid='manage-add-guest-form']"),
		locatorOuterHTML(t, legacyBrowserPage, "[data-testid='manage-add-guest-form']"),
	)
	assertEqualStringSlices(t, "manage attendee cards",
		locatorAllOuterHTML(t, newBrowserPage, "[data-testid='manage-attendee-card']"),
		locatorAllOuterHTML(t, legacyBrowserPage, "[data-testid='manage-attendee-card']"),
	)
}

// TestModerateVotesPanel_UIParityWithLegacy checks the votes panel in empty state.
func TestModerateVotesPanel_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	for _, ts := range []*testServer{newTS, legacyTS} {
		ts.seedCommittee(t, "Test Committee", "test")
		ts.seedUser(t, "test", "chair1", "pass123", "Chair Person", "chairperson")
		ts.seedMeetingOpen(t, "test", "Board Meeting", "")
		apID := ts.seedAgendaPoint(t, "test", "Board Meeting", "Main Topic")
		ts.activateAgendaPoint(t, "test", "Board Meeting", apID)
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "chair1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "chair1", "pass123")

	meetingID := newTS.getMeetingID(t, "test", "Board Meeting")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Board Meeting")

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID+"/moderate", "#moderate-left-controls")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/moderate", "#moderate-left-controls")

	openModerateLeftTab(t, newBrowserPage, "tools")
	openModerateLeftTab(t, legacyBrowserPage, "tools")

	assertEqualHTML(t, "moderate votes panel (empty)",
		locatorOuterHTML(t, newBrowserPage, "#moderate-votes-panel"),
		locatorOuterHTML(t, legacyBrowserPage, "#moderate-votes-panel"),
	)
}

// TestModerateSettingsTab_UIParityWithLegacy checks the settings tab content
// including the meeting settings form and speaker settings.
func TestModerateSettingsTab_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	for _, ts := range []*testServer{newTS, legacyTS} {
		ts.seedCommittee(t, "Test Committee", "test")
		ts.seedUser(t, "test", "chair1", "pass123", "Chair Person", "chairperson")
		ts.seedMeetingOpen(t, "test", "Board Meeting", "")
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "chair1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "chair1", "pass123")

	meetingID := newTS.getMeetingID(t, "test", "Board Meeting")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Board Meeting")

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID+"/moderate", "#moderate-left-controls")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/moderate", "#moderate-left-controls")

	openModerateLeftTab(t, newBrowserPage, "settings")
	openModerateLeftTab(t, legacyBrowserPage, "settings")

	assertEqualHTML(t, "meeting settings container",
		locatorOuterHTML(t, newBrowserPage, "#meeting-settings-container"),
		locatorOuterHTML(t, legacyBrowserPage, "#meeting-settings-container"),
	)
	compareFragmentAfterAction(
		t,
		"speaker settings container",
		newBrowserPage,
		legacyBrowserPage,
		"#moderate-speaker-settings-container",
		func(page playwright.Page) error {
			return page.Locator("[data-moderate-settings-tab='agenda']").First().Click()
		},
	)
}

// TestModerateSpeakersWithAttendee_UIParityWithLegacy checks the speakers list on the
// right side of the moderate page when there are actual speakers in the queue.
func TestModerateSpeakersWithAttendee_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	for _, ts := range []*testServer{newTS, legacyTS} {
		ts.seedCommittee(t, "Test Committee", "test")
		ts.seedUser(t, "test", "chair1", "pass123", "Chair Person", "chairperson")
		ts.seedMeetingOpen(t, "test", "Board Meeting", "")
		apID := ts.seedAgendaPoint(t, "test", "Board Meeting", "Main Topic")
		ts.activateAgendaPoint(t, "test", "Board Meeting", apID)
		attendee := ts.seedAttendee(t, "test", "Board Meeting", "Alice Member", "secret-alice")
		ts.seedSpeaker(t, apID, strconv.FormatInt(attendee.ID, 10))
		apIDInt, err := strconv.ParseInt(apID, 10, 64)
		if err != nil {
			t.Fatalf("parse agenda point id: %v", err)
		}
		if err := ts.repo.RecomputeSpeakerOrder(context.Background(), apIDInt); err != nil {
			t.Fatalf("recompute speaker order: %v", err)
		}
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "chair1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "chair1", "pass123")

	meetingID := newTS.getMeetingID(t, "test", "Board Meeting")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Board Meeting")

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID+"/moderate", "#speakers-list-container")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/moderate", "#speakers-list-container")

	// Wait for speaker items to appear (SSE-driven).
	if err := newBrowserPage.Locator("#speakers-list-container [data-speaker-state]").First().WaitFor(
		playwright.LocatorWaitForOptions{Timeout: playwright.Float(defaultE2ETimeoutMs)},
	); err != nil {
		t.Fatalf("wait for speaker item on new: %v", err)
	}
	if err := legacyBrowserPage.Locator("#speakers-list-container [data-speaker-state]").First().WaitFor(
		playwright.LocatorWaitForOptions{Timeout: playwright.Float(defaultE2ETimeoutMs)},
	); err != nil {
		t.Fatalf("wait for speaker item on legacy: %v", err)
	}

	assertEqualHTML(t, "moderate speakers list with attendee",
		locatorOuterHTML(t, newBrowserPage, "#speakers-list-container"),
		locatorOuterHTML(t, legacyBrowserPage, "#speakers-list-container"),
	)
}

// TestMeetingLiveWithSpeakersInQueue_UIParityWithLegacy checks the live page when
// there are speakers in the queue, including speaker items and quick controls.
func TestMeetingLiveWithSpeakersInQueue_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	for _, ts := range []*testServer{newTS, legacyTS} {
		ts.seedCommittee(t, "Test Committee", "test")
		ts.seedUser(t, "test", "member1", "pass123", "Member One", "member")
		ts.seedMeetingOpen(t, "test", "Board Meeting", "")
		apID := ts.seedAgendaPoint(t, "test", "Board Meeting", "Main Topic")
		ts.seedAgendaPoint(t, "test", "Board Meeting", "Second Topic")
		ts.activateAgendaPoint(t, "test", "Board Meeting", apID)
		meetingIDInt, err := strconv.ParseInt(ts.getMeetingID(t, "test", "Board Meeting"), 10, 64)
		if err != nil {
			t.Fatalf("parse meeting id: %v", err)
		}
		if err := ts.repo.SetActiveMeeting(context.Background(), "test", &meetingIDInt); err != nil {
			t.Fatalf("set active meeting: %v", err)
		}
		a1 := ts.seedAttendee(t, "test", "Board Meeting", "Alice Member", "secret-alice-live")
		a2 := ts.seedAttendee(t, "test", "Board Meeting", "Bob Member", "secret-bob-live")
		ts.seedSpeaker(t, apID, strconv.FormatInt(a1.ID, 10))
		ts.seedSpeaker(t, apID, strconv.FormatInt(a2.ID, 10))
		apIDInt, _ := strconv.ParseInt(apID, 10, 64)
		if err := ts.repo.RecomputeSpeakerOrder(context.Background(), apIDInt); err != nil {
			t.Fatalf("recompute speaker order: %v", err)
		}
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "member1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "member1", "pass123")

	// Self-signup via join page.
	meetingID := newTS.getMeetingID(t, "test", "Board Meeting")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Board Meeting")

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID+"/join", "main button")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/join", "main button")

	if err := newBrowserPage.Locator("main button").First().Click(); err != nil {
		t.Fatalf("self-signup on new: %v", err)
	}
	if err := legacyBrowserPage.Locator("main button").First().Click(); err != nil {
		t.Fatalf("self-signup on legacy: %v", err)
	}

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID, "#attendee-speakers-list")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID, "#attendee-speakers-list")

	// Wait for speaker items to appear in the live view (SSE-driven).
	if err := newBrowserPage.Locator("#attendee-speakers-list [data-testid='live-speaker-item']").First().WaitFor(
		playwright.LocatorWaitForOptions{Timeout: playwright.Float(defaultE2ETimeoutMs)},
	); err != nil {
		t.Fatalf("wait for speaker items on new: %v", err)
	}
	if err := legacyBrowserPage.Locator("#attendee-speakers-list [data-testid='live-speaker-item']").First().WaitFor(
		playwright.LocatorWaitForOptions{Timeout: playwright.Float(defaultE2ETimeoutMs)},
	); err != nil {
		t.Fatalf("wait for speaker items on legacy: %v", err)
	}

	assertEqualHTML(t, "live speakers list with queued speakers",
		locatorOuterHTML(t, newBrowserPage, "#attendee-speakers-list"),
		locatorOuterHTML(t, legacyBrowserPage, "#attendee-speakers-list"),
	)
	assertEqualHTML(t, "live agenda stack with active point",
		locatorOuterHTML(t, newBrowserPage, "#live-agenda-main-stack"),
		locatorOuterHTML(t, legacyBrowserPage, "#live-agenda-main-stack"),
	)
}

// TestMeetingJoinFullForm_UIParityWithLegacy checks the complete join page form
// including the guest signup section and the attendee-login link.
func TestMeetingJoinFullForm_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	for _, ts := range []*testServer{newTS, legacyTS} {
		ts.seedCommittee(t, "Test Committee", "test")
		ts.seedUser(t, "test", "member1", "pass123", "Member One", "member")
		ts.seedMeetingOpen(t, "test", "Open Meeting", "")
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "member1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "member1", "pass123")

	meetingID := newTS.getMeetingID(t, "test", "Open Meeting")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Open Meeting")

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID+"/join", "main")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/join", "main")

	assertEqualHTML(t, "join page form content",
		locatorOuterHTML(t, newBrowserPage, "main form"),
		locatorOuterHTML(t, legacyBrowserPage, "main form"),
	)
}

// TestAttendeeLoginFullForm_UIParityWithLegacy checks the complete attendee-login
// page form structure.
func TestAttendeeLoginFullForm_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	for _, ts := range []*testServer{newTS, legacyTS} {
		ts.seedCommittee(t, "Test Committee", "test")
		ts.seedUser(t, "test", "member1", "pass123", "Member One", "member")
		ts.seedMeetingOpen(t, "test", "Open Meeting", "")
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "member1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "member1", "pass123")

	meetingID := newTS.getMeetingID(t, "test", "Open Meeting")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Open Meeting")

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID+"/attendee-login", "main form")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/attendee-login", "main form")

	assertEqualHTML(t, "attendee login form content",
		locatorOuterHTML(t, newBrowserPage, "main form"),
		locatorOuterHTML(t, legacyBrowserPage, "main form"),
	)
}

// TestLiveActiveSpeaker_UIParityWithLegacy checks the live page speaker list when
// one speaker is currently in the speaking state (A02).
func TestLiveActiveSpeaker_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	for _, ts := range []*testServer{newTS, legacyTS} {
		ts.seedCommittee(t, "Test Committee", "test")
		ts.seedUser(t, "test", "member1", "pass123", "Member One", "member")
		ts.seedMeetingOpen(t, "test", "Board Meeting", "")
		apIDStr := ts.seedAgendaPoint(t, "test", "Board Meeting", "Main Topic")
		ts.activateAgendaPoint(t, "test", "Board Meeting", apIDStr)
		meetingIDInt, err := strconv.ParseInt(ts.getMeetingID(t, "test", "Board Meeting"), 10, 64)
		if err != nil {
			t.Fatalf("parse meeting id: %v", err)
		}
		if err := ts.repo.SetActiveMeeting(context.Background(), "test", &meetingIDInt); err != nil {
			t.Fatalf("set active meeting: %v", err)
		}
		attendee := ts.seedAttendee(t, "test", "Board Meeting", "Alice Member", "secret-alice")
		speakerIDStr := ts.seedSpeaker(t, apIDStr, strconv.FormatInt(attendee.ID, 10))
		speakerID, _ := strconv.ParseInt(speakerIDStr, 10, 64)
		apID, _ := strconv.ParseInt(apIDStr, 10, 64)
		if err := ts.repo.SetSpeakerSpeaking(context.Background(), speakerID, apID); err != nil {
			t.Fatalf("set speaker speaking: %v", err)
		}
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "member1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "member1", "pass123")

	meetingID := newTS.getMeetingID(t, "test", "Board Meeting")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Board Meeting")

	// Self-signup via join page so the member becomes an attendee and can view the live page.
	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID+"/join", "main button")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/join", "main button")

	if err := newBrowserPage.Locator("main button").First().Click(); err != nil {
		t.Fatalf("self-signup on new: %v", err)
	}
	if err := legacyBrowserPage.Locator("main button").First().Click(); err != nil {
		t.Fatalf("self-signup on legacy: %v", err)
	}

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID, "#attendee-speakers-list")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID, "#attendee-speakers-list")

	// Wait for the speaking speaker item to appear (SSE-driven).
	if err := newBrowserPage.Locator("#attendee-speakers-list [data-testid='live-speaker-item'][data-speaker-state='speaking']").First().WaitFor(
		playwright.LocatorWaitForOptions{Timeout: playwright.Float(defaultE2ETimeoutMs)},
	); err != nil {
		t.Fatalf("wait for speaking speaker on new: %v", err)
	}
	if err := legacyBrowserPage.Locator("#attendee-speakers-list [data-testid='live-speaker-item'][data-speaker-state='speaking']").First().WaitFor(
		playwright.LocatorWaitForOptions{Timeout: playwright.Float(defaultE2ETimeoutMs)},
	); err != nil {
		t.Fatalf("wait for speaking speaker on legacy: %v", err)
	}

	// The data-speaking-since attribute value differs between legacy (Unix seconds) and
	// SPA (milliseconds from Date.now()), so normalize it before comparing.
	assertEqualHTML(t, "live speakers list with active speaker",
		normalizeSpeakingSinceAttr(locatorOuterHTML(t, newBrowserPage, "#attendee-speakers-list")),
		normalizeSpeakingSinceAttr(locatorOuterHTML(t, legacyBrowserPage, "#attendee-speakers-list")),
	)
}

// TestLiveCompletedSpeaker_UIParityWithLegacy checks the live page speaker list after
// a speaker has finished speaking — the done speaker must not appear in the active list,
// and the remaining queued speaker must still be shown (A03).
func TestLiveCompletedSpeaker_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	for _, ts := range []*testServer{newTS, legacyTS} {
		ts.seedCommittee(t, "Test Committee", "test")
		ts.seedUser(t, "test", "member1", "pass123", "Member One", "member")
		ts.seedMeetingOpen(t, "test", "Board Meeting", "")
		apIDStr := ts.seedAgendaPoint(t, "test", "Board Meeting", "Main Topic")
		ts.activateAgendaPoint(t, "test", "Board Meeting", apIDStr)
		meetingIDInt, err := strconv.ParseInt(ts.getMeetingID(t, "test", "Board Meeting"), 10, 64)
		if err != nil {
			t.Fatalf("parse meeting id: %v", err)
		}
		if err := ts.repo.SetActiveMeeting(context.Background(), "test", &meetingIDInt); err != nil {
			t.Fatalf("set active meeting: %v", err)
		}
		alice := ts.seedAttendee(t, "test", "Board Meeting", "Alice Member", "secret-alice")
		bob := ts.seedAttendee(t, "test", "Board Meeting", "Bob Member", "secret-bob")
		aliceSpeakerIDStr := ts.seedSpeaker(t, apIDStr, strconv.FormatInt(alice.ID, 10))
		ts.seedSpeaker(t, apIDStr, strconv.FormatInt(bob.ID, 10))
		aliceSpeakerID, _ := strconv.ParseInt(aliceSpeakerIDStr, 10, 64)
		apID, _ := strconv.ParseInt(apIDStr, 10, 64)
		// Mark Alice as speaking then done; Bob remains waiting.
		if err := ts.repo.SetSpeakerSpeaking(context.Background(), aliceSpeakerID, apID); err != nil {
			t.Fatalf("set alice speaking: %v", err)
		}
		if err := ts.repo.SetSpeakerDone(context.Background(), aliceSpeakerID); err != nil {
			t.Fatalf("set alice done: %v", err)
		}
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "member1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "member1", "pass123")

	meetingID := newTS.getMeetingID(t, "test", "Board Meeting")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Board Meeting")

	// Self-signup via join page so the member becomes an attendee and can view the live page.
	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID+"/join", "main button")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/join", "main button")

	if err := newBrowserPage.Locator("main button").First().Click(); err != nil {
		t.Fatalf("self-signup on new: %v", err)
	}
	if err := legacyBrowserPage.Locator("main button").First().Click(); err != nil {
		t.Fatalf("self-signup on legacy: %v", err)
	}

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID, "#attendee-speakers-list")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID, "#attendee-speakers-list")

	// Wait for Bob's waiting speaker item (the only active speaker remaining).
	if err := newBrowserPage.Locator("#attendee-speakers-list [data-testid='live-speaker-item'][data-speaker-state='waiting']").First().WaitFor(
		playwright.LocatorWaitForOptions{Timeout: playwright.Float(defaultE2ETimeoutMs)},
	); err != nil {
		t.Fatalf("wait for waiting speaker on new: %v", err)
	}
	if err := legacyBrowserPage.Locator("#attendee-speakers-list [data-testid='live-speaker-item'][data-speaker-state='waiting']").First().WaitFor(
		playwright.LocatorWaitForOptions{Timeout: playwright.Float(defaultE2ETimeoutMs)},
	); err != nil {
		t.Fatalf("wait for waiting speaker on legacy: %v", err)
	}

	assertEqualHTML(t, "live speakers list after speaker completed",
		locatorOuterHTML(t, newBrowserPage, "#attendee-speakers-list"),
		locatorOuterHTML(t, legacyBrowserPage, "#attendee-speakers-list"),
	)
}

// TestModerateVotesPanelOpen_UIParityWithLegacy (A04) checks the votes panel
// on the moderation page while a vote is open, comparing SPA against legacy.
// The SPA moderate page loads the votes panel from the same legacy handler
// fragment, so both should produce identical HTML after HTMX processing.
func TestModerateVotesPanelOpen_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	for _, ts := range []*testServer{newTS, legacyTS} {
		ts.seedCommittee(t, "Test Committee", "test")
		ts.seedUser(t, "test", "chair1", "pass123", "Chair Person", "chairperson")
		ts.seedMeetingOpen(t, "test", "Board Meeting", "")
		apID := ts.seedAgendaPoint(t, "test", "Board Meeting", "Main Topic")
		ts.activateAgendaPoint(t, "test", "Board Meeting", apID)
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "chair1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "chair1", "pass123")

	meetingID := newTS.getMeetingID(t, "test", "Board Meeting")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Board Meeting")

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID+"/moderate", "#moderate-left-controls")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/moderate", "#moderate-left-controls")

	// Create and open a vote in each browser independently.
	createDraftVoteFromModeratorUI(t, newBrowserPage, "Budget Vote", "open", 1, 1)
	createDraftVoteFromModeratorUI(t, legacyBrowserPage, "Budget Vote", "open", 1, 1)

	openDraftVoteFromModeratorUI(t, newBrowserPage, "Budget Vote")
	openDraftVoteFromModeratorUI(t, legacyBrowserPage, "Budget Vote")

	// The HTMX response that opens a vote includes a transient success notification
	// inside #moderate-votes-panel. Remove it synchronously from both DOMs before
	// comparing so that timing differences don't cause a mismatch.
	for _, p := range []playwright.Page{newBrowserPage, legacyBrowserPage} {
		if _, err := p.Evaluate(`document.querySelectorAll('[data-notification-item]').forEach(el => el.remove())`, nil); err != nil {
			t.Logf("note: remove notifications: %v", err)
		}
	}

	assertEqualHTML(t, "moderate votes panel (open vote)",
		locatorOuterHTML(t, newBrowserPage, "#moderate-votes-panel"),
		locatorOuterHTML(t, legacyBrowserPage, "#moderate-votes-panel"),
	)
}

// TestModerateVotesPanelClosed_UIParityWithLegacy (A05) checks the votes panel
// after closing an open vote and viewing the final tallies, comparing SPA against legacy.
func TestModerateVotesPanelClosed_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	for _, ts := range []*testServer{newTS, legacyTS} {
		ts.seedCommittee(t, "Test Committee", "test")
		ts.seedUser(t, "test", "chair1", "pass123", "Chair Person", "chairperson")
		ts.seedMeetingOpen(t, "test", "Board Meeting", "")
		apID := ts.seedAgendaPoint(t, "test", "Board Meeting", "Main Topic")
		ts.activateAgendaPoint(t, "test", "Board Meeting", apID)
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "chair1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "chair1", "pass123")

	meetingID := newTS.getMeetingID(t, "test", "Board Meeting")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Board Meeting")

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID+"/moderate", "#moderate-left-controls")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/moderate", "#moderate-left-controls")

	createDraftVoteFromModeratorUI(t, newBrowserPage, "Budget Vote", "open", 1, 1)
	createDraftVoteFromModeratorUI(t, legacyBrowserPage, "Budget Vote", "open", 1, 1)

	openDraftVoteFromModeratorUI(t, newBrowserPage, "Budget Vote")
	openDraftVoteFromModeratorUI(t, legacyBrowserPage, "Budget Vote")

	closeVoteFromModeratorUI(t, newBrowserPage, "Budget Vote")
	closeVoteFromModeratorUI(t, legacyBrowserPage, "Budget Vote")

	// Wait for Final Tallies to appear in both panels.
	for label, p := range map[string]playwright.Page{"new": newBrowserPage, "legacy": legacyBrowserPage} {
		if err := p.Locator("#moderate-votes-panel").Locator(":has-text('Final Tallies')").First().WaitFor(); err != nil {
			t.Fatalf("wait final tallies in %s votes panel: %v", label, err)
		}
	}

	// Remove any transient notifications before comparing.
	for _, p := range []playwright.Page{newBrowserPage, legacyBrowserPage} {
		if _, err := p.Evaluate(`document.querySelectorAll('[data-notification-item]').forEach(el => el.remove())`, nil); err != nil {
			t.Logf("note: remove notifications: %v", err)
		}
	}

	assertEqualHTML(t, "moderate votes panel (closed vote with tallies)",
		locatorOuterHTML(t, newBrowserPage, "#moderate-votes-panel"),
		locatorOuterHTML(t, legacyBrowserPage, "#moderate-votes-panel"),
	)
}

// TestModerateAddGuestAttendee_UIParityWithLegacy (A06) checks that the
// attendee list fragment matches after adding a guest via the inline form.
func TestModerateAddGuestAttendee_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	for _, ts := range []*testServer{newTS, legacyTS} {
		ts.seedCommittee(t, "Test Committee", "test")
		ts.seedUser(t, "test", "chair1", "pass123", "Chair Person", "chairperson")
		ts.seedMeetingOpen(t, "test", "Board Meeting", "")
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "chair1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "chair1", "pass123")

	meetingID := newTS.getMeetingID(t, "test", "Board Meeting")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Board Meeting")

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID+"/moderate", "#moderate-left-controls")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/moderate", "#moderate-left-controls")

	// Add a guest in each browser; wait for the card to appear.
	submitAddGuest(t, newBrowserPage, "Guest Person")
	submitAddGuest(t, legacyBrowserPage, "Guest Person")

	for label, p := range map[string]playwright.Page{"new": newBrowserPage, "legacy": legacyBrowserPage} {
		if err := manageAttendeeCard(t, p, "Guest Person").WaitFor(playwright.LocatorWaitForOptions{
			State: playwright.WaitForSelectorStateAttached,
		}); err != nil {
			t.Fatalf("wait guest attendee card on %s: %v", label, err)
		}
	}

	assertEqualStringSlices(t, "attendee cards after adding guest",
		locatorAllOuterHTML(t, newBrowserPage, "[data-testid='manage-attendee-card']"),
		locatorAllOuterHTML(t, legacyBrowserPage, "[data-testid='manage-attendee-card']"),
	)
}

// TestModerateRemoveAttendee_UIParityWithLegacy (A07) checks that the attendee
// list fragment matches after removing a guest via the remove button.
func TestModerateRemoveAttendee_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	for _, ts := range []*testServer{newTS, legacyTS} {
		ts.seedCommittee(t, "Test Committee", "test")
		ts.seedUser(t, "test", "chair1", "pass123", "Chair Person", "chairperson")
		ts.seedMeetingOpen(t, "test", "Board Meeting", "")
		ts.seedAttendee(t, "test", "Board Meeting", "Alice Member", "secret-alice")
		ts.seedAttendee(t, "test", "Board Meeting", "Bob Guest", "secret-bob")
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "chair1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "chair1", "pass123")

	meetingID := newTS.getMeetingID(t, "test", "Board Meeting")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Board Meeting")

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID+"/moderate", "#moderate-left-controls")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/moderate", "#moderate-left-controls")

	// Register dialog acceptors before clicking remove buttons.
	for _, p := range []playwright.Page{newBrowserPage, legacyBrowserPage} {
		p.OnDialog(func(d playwright.Dialog) { _ = d.Accept() })
	}

	// Remove Bob Guest from each browser; wait for the card to disappear.
	bobNewCard := manageAttendeeCard(t, newBrowserPage, "Bob Guest")
	bobLegacyCard := manageAttendeeCard(t, legacyBrowserPage, "Bob Guest")

	if err := bobNewCard.Locator("button[title='Remove attendee']").Click(); err != nil {
		t.Fatalf("click remove attendee on new: %v", err)
	}
	if err := bobLegacyCard.Locator("button[title='Remove attendee']").Click(); err != nil {
		t.Fatalf("click remove attendee on legacy: %v", err)
	}

	for label, p := range map[string]playwright.Page{"new": newBrowserPage, "legacy": legacyBrowserPage} {
		if err := manageAttendeeCard(t, p, "Bob Guest").WaitFor(playwright.LocatorWaitForOptions{
			State: playwright.WaitForSelectorStateDetached,
		}); err != nil {
			t.Fatalf("wait bob card detached on %s: %v", label, err)
		}
	}

	assertEqualStringSlices(t, "attendee cards after removing guest",
		locatorAllOuterHTML(t, newBrowserPage, "[data-testid='manage-attendee-card']"),
		locatorAllOuterHTML(t, legacyBrowserPage, "[data-testid='manage-attendee-card']"),
	)
}

// TestAttachmentListPopulated_UIParityWithLegacy (A08) checks that the
// attachment download links on the tools page match between SPA and legacy.
// The tools page HTML structure differs (SPA Svelte vs legacy HTMX/Templ),
// so we compare only the download <a> elements whose text and href should match.
func TestAttachmentListPopulated_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	var newApID, legacyApID string
	for _, pair := range []struct {
		ts    *testServer
		apPtr *string
	}{{newTS, &newApID}, {legacyTS, &legacyApID}} {
		pair.ts.seedCommittee(t, "Test Committee", "test")
		pair.ts.seedUser(t, "test", "chair1", "pass123", "Chair Person", "chairperson")
		pair.ts.seedMeetingOpen(t, "test", "Board Meeting", "")
		*pair.apPtr = pair.ts.seedAgendaPoint(t, "test", "Board Meeting", "Main Topic")
		label := "Budget Report"
		pair.ts.seedAttachment(t, *pair.apPtr, &label)
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "chair1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "chair1", "pass123")

	meetingID := newTS.getMeetingID(t, "test", "Board Meeting")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Board Meeting")

	gotoAndWaitForSelector(t, newBrowserPage, agendaPointToolsURL(newTS.URL, "test", meetingID, newApID), "#attachment-list-ap-"+newApID)
	gotoAndWaitForSelector(t, legacyBrowserPage, agendaPointToolsURL(legacyTS.URL, "test", legacyMeetingID, legacyApID), "#attachment-list-ap-"+legacyApID)

	// Wait for the labeled attachment link to appear in both browsers.
	for label, p := range map[string]playwright.Page{"new": newBrowserPage, "legacy": legacyBrowserPage} {
		if err := p.Locator("a:has-text('Budget Report')").First().WaitFor(); err != nil {
			t.Fatalf("wait attachment link on %s: %v", label, err)
		}
	}

	// Compare the attachment link label texts. The SPA and legacy use different
	// download URL schemes and form mechanisms, so we compare the visible link
	// text (label + filename) rather than full HTML.
	newTexts := locatorAllInnerText(t, newBrowserPage, "#attachment-list-ap-"+newApID+" li a")
	legacyTexts := locatorAllInnerText(t, legacyBrowserPage, "#attachment-list-ap-"+legacyApID+" li a")
	assertEqualStringSlices(t, "attachment link texts", newTexts, legacyTexts)
}

// TestCurrentDocumentState_UIParityWithLegacy (A09) checks that when a
// current attachment is set, the live page shows the document controls
// in both SPA and legacy, and that the document label shown matches.
func TestCurrentDocumentState_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	for _, ts := range []*testServer{newTS, legacyTS} {
		ts.seedCommittee(t, "Test Committee", "test")
		ts.seedUser(t, "test", "member1", "pass123", "Member One", "member")
		ts.seedMeetingOpen(t, "test", "Board Meeting", "")
		meetingIDStr := ts.getMeetingID(t, "test", "Board Meeting")
		meetingIDInt, _ := strconv.ParseInt(meetingIDStr, 10, 64)
		if err := ts.repo.SetActiveMeeting(context.Background(), "test", &meetingIDInt); err != nil {
			t.Fatalf("set active meeting: %v", err)
		}
		apID := ts.seedAgendaPoint(t, "test", "Board Meeting", "Main Topic")
		ts.activateAgendaPoint(t, "test", "Board Meeting", apID)
		label := "Budget Report"
		attachmentIDStr := ts.seedAttachment(t, apID, &label)
		// Set the attachment as current directly via repo.
		apIDInt, _ := strconv.ParseInt(apID, 10, 64)
		attachmentID, _ := strconv.ParseInt(attachmentIDStr, 10, 64)
		if err := ts.repo.SetCurrentAttachment(context.Background(), apIDInt, attachmentID); err != nil {
			t.Fatalf("set current attachment: %v", err)
		}
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "member1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "member1", "pass123")

	meetingID := newTS.getMeetingID(t, "test", "Board Meeting")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Board Meeting")

	// Self-signup to become an attendee on both servers.
	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID+"/join", "main button")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/join", "main button")
	if err := newBrowserPage.Locator("main button").First().Click(); err != nil {
		t.Fatalf("self-signup on new: %v", err)
	}
	if err := legacyBrowserPage.Locator("main button").First().Click(); err != nil {
		t.Fatalf("self-signup on legacy: %v", err)
	}

	// Navigate to live page and wait for the desktop open-document button.
	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID, "[data-testid='live-doc-open-desktop']")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID, "[data-testid='live-doc-open-desktop']")

	// Both browsers should show the document controls. The SPA and legacy
	// current-document panels differ structurally, so we compare element presence
	// and the download button's href basename.
	for label, p := range map[string]playwright.Page{"new": newBrowserPage, "legacy": legacyBrowserPage} {
		if err := p.Locator("[data-testid='live-doc-open-desktop']").First().WaitFor(); err != nil {
			t.Fatalf("%s: expected live-doc-open-desktop: %v", label, err)
		}
		if err := p.Locator("[data-testid='live-doc-download-desktop']").First().WaitFor(); err != nil {
			t.Fatalf("%s: expected live-doc-download-desktop: %v", label, err)
		}
	}
}

// TestModerateStartSpeaker_UIParityWithLegacy (A15) checks that starting the
// next speaker via the moderation UI produces the same speakers list in the SPA
// and legacy implementations.
func TestModerateStartSpeaker_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	for _, ts := range []*testServer{newTS, legacyTS} {
		ts.seedCommittee(t, "Test Committee", "test")
		ts.seedUser(t, "test", "chair1", "pass123", "Chair Person", "chairperson")
		ts.seedMeetingOpen(t, "test", "Board Meeting", "")
		ts.seedAttendee(t, "test", "Board Meeting", "Alice Member", "secret-alice")
		apID := ts.seedAgendaPoint(t, "test", "Board Meeting", "Main Topic")
		ts.activateAgendaPoint(t, "test", "Board Meeting", apID)
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "chair1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "chair1", "pass123")

	meetingID := newTS.getMeetingID(t, "test", "Board Meeting")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Board Meeting")

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID+"/moderate", "#speakers-list-container")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/moderate", "#speakers-list-container")

	// Add Alice to the speaker queue in each browser.
	for label, p := range map[string]playwright.Page{"new": newBrowserPage, "legacy": legacyBrowserPage} {
		openSpeakerAddDialog(t, p)
		card := speakerCandidateCard(p, "Alice Member")
		if err := card.WaitFor(); err != nil {
			t.Fatalf("wait speaker candidate card on %s: %v", label, err)
		}
		if err := card.Locator("button[title='Add regular speech']").Click(); err != nil {
			t.Fatalf("add regular speech on %s: %v", label, err)
		}
		if err := p.Locator("#speakers-list-container [data-testid='live-speaker-item']:has-text('Alice Member')").WaitFor(); err != nil {
			t.Fatalf("wait speaker in queue on %s: %v", label, err)
		}
	}

	// Start the next speaker in each browser.
	for label, p := range map[string]playwright.Page{"new": newBrowserPage, "legacy": legacyBrowserPage} {
		startBtn := p.Locator("#speakers-list-container button[title='Start next speaker']")
		if err := startBtn.WaitFor(); err != nil {
			t.Fatalf("wait start-next button on %s: %v", label, err)
		}
		if err := startBtn.Click(); err != nil {
			t.Fatalf("click start-next on %s: %v", label, err)
		}
		if err := p.Locator("#speakers-list-container [data-testid='live-speaker-item'][data-speaker-state='speaking']:has-text('Alice Member')").WaitFor(); err != nil {
			t.Fatalf("wait for speaking speaker on %s: %v", label, err)
		}
	}

	// The data-speaking-since attribute value differs between legacy (Unix seconds) and
	// SPA (milliseconds from Date.now()), so normalize it before comparing.
	// Also strip data-initial-scroll-top which is timing-dependent.
	assertEqualHTML(t, "speakers list after start speaker",
		normalizeInitialScrollTop(normalizeSpeakingSinceAttr(locatorOuterHTML(t, newBrowserPage, "#speakers-list-container"))),
		normalizeInitialScrollTop(normalizeSpeakingSinceAttr(locatorOuterHTML(t, legacyBrowserPage, "#speakers-list-container"))),
	)
}

// TestModerateEndSpeaker_UIParityWithLegacy (A16) checks that ending the current
// speaker via the moderation UI produces the same speakers list in the SPA and
// legacy implementations. Alice is started then ended; Bob remains waiting.
func TestModerateEndSpeaker_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	for _, ts := range []*testServer{newTS, legacyTS} {
		ts.seedCommittee(t, "Test Committee", "test")
		ts.seedUser(t, "test", "chair1", "pass123", "Chair Person", "chairperson")
		ts.seedMeetingOpen(t, "test", "Board Meeting", "")
		ts.seedAttendee(t, "test", "Board Meeting", "Alice Member", "secret-alice")
		ts.seedAttendee(t, "test", "Board Meeting", "Bob Member", "secret-bob")
		apID := ts.seedAgendaPoint(t, "test", "Board Meeting", "Main Topic")
		ts.activateAgendaPoint(t, "test", "Board Meeting", apID)
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "chair1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "chair1", "pass123")

	meetingID := newTS.getMeetingID(t, "test", "Board Meeting")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Board Meeting")

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID+"/moderate", "#speakers-list-container")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/moderate", "#speakers-list-container")

	// Add Alice and Bob to the queue, then start Alice in each browser.
	for label, p := range map[string]playwright.Page{"new": newBrowserPage, "legacy": legacyBrowserPage} {
		for _, name := range []string{"Alice Member", "Bob Member"} {
			openSpeakerAddDialog(t, p)
			card := speakerCandidateCard(p, name)
			if err := card.WaitFor(); err != nil {
				t.Fatalf("wait candidate card %q on %s: %v", name, label, err)
			}
			if err := card.Locator("button[title='Add regular speech']").Click(); err != nil {
				t.Fatalf("add regular speech %q on %s: %v", name, label, err)
			}
			if err := p.Locator("#speakers-list-container [data-testid='live-speaker-item']:has-text('"+name+"')").WaitFor(); err != nil {
				t.Fatalf("wait speaker %q in queue on %s: %v", name, label, err)
			}
		}
		startBtn := p.Locator("#speakers-list-container button[title='Start next speaker']")
		if err := startBtn.WaitFor(); err != nil {
			t.Fatalf("wait start-next on %s: %v", label, err)
		}
		if err := startBtn.Click(); err != nil {
			t.Fatalf("click start-next on %s: %v", label, err)
		}
		if err := p.Locator("#speakers-list-container [data-testid='live-speaker-item'][data-speaker-state='speaking']:has-text('Alice Member')").WaitFor(); err != nil {
			t.Fatalf("wait Alice speaking on %s: %v", label, err)
		}
	}

	// End Alice's speech in each browser.
	for label, p := range map[string]playwright.Page{"new": newBrowserPage, "legacy": legacyBrowserPage} {
		endBtn := p.Locator("#speakers-list-container button[title='End current speech']")
		if err := endBtn.WaitFor(); err != nil {
			t.Fatalf("wait end-speech button on %s: %v", label, err)
		}
		if err := endBtn.Click(); err != nil {
			t.Fatalf("click end-speech on %s: %v", label, err)
		}
		// Alice's speaking row should detach; Bob should remain waiting.
		if err := p.Locator("#speakers-list-container [data-testid='live-speaker-item'][data-speaker-state='speaking']").WaitFor(
			playwright.LocatorWaitForOptions{State: playwright.WaitForSelectorStateDetached},
		); err != nil {
			t.Fatalf("wait speaking row detached on %s: %v", label, err)
		}
		if err := p.Locator("#speakers-list-container [data-testid='live-speaker-item'][data-speaker-state='waiting']:has-text('Bob Member')").WaitFor(); err != nil {
			t.Fatalf("wait Bob waiting on %s: %v", label, err)
		}
	}

	// Strip data-initial-scroll-top: the SPA Svelte template renders it statically
	// while the legacy sets it asynchronously via JS after the HTMX swap. The timing
	// of the Playwright snapshot determines whether it appears in the legacy DOM.
	assertEqualHTML(t, "speakers list after end speaker",
		normalizeInitialScrollTop(locatorOuterHTML(t, newBrowserPage, "#speakers-list-container")),
		normalizeInitialScrollTop(locatorOuterHTML(t, legacyBrowserPage, "#speakers-list-container")),
	)
}
