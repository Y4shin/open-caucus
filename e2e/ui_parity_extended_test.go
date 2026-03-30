//go:build e2e

package e2e_test

import (
	"context"
	"strconv"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

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

	assertEqualStringSlices(t, "committee meeting rows html",
		locatorAllOuterHTML(t, newBrowserPage, "[data-testid='committee-meeting-row']"),
		locatorAllOuterHTML(t, legacyBrowserPage, "[data-testid='committee-meeting-row']"),
	)
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
