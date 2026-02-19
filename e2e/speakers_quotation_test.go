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

// TestSpeakers_PriorityToggle_MovesToFront seeds two speakers, toggles priority
// on the second, and verifies it appears first in the WAITING list.
func TestSpeakers_PriorityToggle_MovesToFront(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "Item One")

	a1 := ts.seedAttendee(t, "test-committee", "Board Meeting", "Alice Speaker", "secret-alice")
	a2 := ts.seedAttendee(t, "test-committee", "Board Meeting", "Bob Speaker", "secret-bob")
	aid1Str := strconv.FormatInt(a1.ID, 10)
	aid2Str := strconv.FormatInt(a2.ID, 10)
	ts.seedSpeaker(t, apID, aid1Str)
	ts.seedSpeaker(t, apID, aid2Str)
	ts.activateAgendaPoint(t, "test-committee", "Board Meeting", apID)

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(agendaManageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	// Both speakers should be visible.
	if err := page.Locator("#speakers-list-container td:has-text('Alice Speaker')").WaitFor(); err != nil {
		t.Fatalf("expected Alice in speakers list: %v", err)
	}

	urlBefore := page.URL()

	// Toggle priority on Bob (second speaker).
	priorityBtn := page.Locator("#speakers-list-container tr:has-text('Bob Speaker') button[title='Give priority']")
	if err := priorityBtn.Click(); err != nil {
		t.Fatalf("click priority toggle for Bob: %v", err)
	}

	// After the HTMX swap, Bob's row should show the filled star.
	if err := page.Locator("#speakers-list-container tr:has-text('Bob Speaker') button[title='Remove priority']").WaitFor(); err != nil {
		t.Fatalf("expected filled star on Bob after priority toggle: %v", err)
	}

	// Bob should now appear before Alice (priority=true sorts first).
	rows, err := page.Locator("#speakers-list-container tbody tr").All()
	if err != nil {
		t.Fatalf("get rows: %v", err)
	}
	if len(rows) < 2 {
		t.Fatalf("expected at least 2 speaker rows, got %d", len(rows))
	}
	firstRowText, err := rows[0].TextContent()
	if err != nil {
		t.Fatalf("get first row text: %v", err)
	}
	if !strings.Contains(firstRowText, "Bob Speaker") {
		t.Errorf("expected Bob (high priority) to appear first, got: %q", firstRowText)
	}

	if page.URL() != urlBefore {
		t.Errorf("URL changed on priority toggle: got %s, want %s", page.URL(), urlBefore)
	}
}

// TestSpeakers_FirstSpeaker_Badge verifies that the "1st" badge is shown only
// for speakers whose firstSpeaker flag is true.
func TestSpeakers_FirstSpeaker_Badge(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "Item One")

	a1 := ts.seedAttendee(t, "test-committee", "Board Meeting", "First Timer", "secret-first")
	a2 := ts.seedAttendee(t, "test-committee", "Board Meeting", "Repeat Speaker", "secret-repeat")

	var apid int64
	fmt.Sscanf(apID, "%d", &apid)
	if _, err := ts.repo.AddSpeaker(context.Background(), apid, a1.ID, "regular", false, true); err != nil {
		t.Fatalf("add first-timer speaker: %v", err)
	}
	if _, err := ts.repo.AddSpeaker(context.Background(), apid, a2.ID, "regular", false, false); err != nil {
		t.Fatalf("add repeat speaker: %v", err)
	}
	ts.activateAgendaPoint(t, "test-committee", "Board Meeting", apID)

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(agendaManageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	if err := page.Locator("#speakers-list-container td:has-text('First Timer')").WaitFor(); err != nil {
		t.Fatalf("wait for First Timer row: %v", err)
	}

	// First Timer row should contain the "1st" badge.
	firstTimerRow := page.Locator("#speakers-list-container tr:has-text('First Timer')")
	firstTimerText, err := firstTimerRow.TextContent()
	if err != nil {
		t.Fatalf("get first-timer row text: %v", err)
	}
	if !strings.Contains(firstTimerText, "1st") {
		t.Errorf("expected '1st' badge in First Timer row, got: %q", firstTimerText)
	}

	// Repeat Speaker row should NOT contain the "1st" badge.
	repeatRow := page.Locator("#speakers-list-container tr:has-text('Repeat Speaker')")
	repeatText, err := repeatRow.TextContent()
	if err != nil {
		t.Fatalf("get repeat row text: %v", err)
	}
	if strings.Contains(repeatText, "1st") {
		t.Errorf("did not expect '1st' badge in Repeat Speaker row, got: %q", repeatText)
	}
}

// TestSpeakers_MeetingModerator_SetAndClear sets a meeting-level moderator via
// the settings form and then clears it, verifying HTMX partial updates each time.
func TestSpeakers_MeetingModerator_SetAndClear(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	ts.seedAttendee(t, "test-committee", "Board Meeting", "Alice Moderator", "secret-am")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(agendaManageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	urlBefore := page.URL()

	// Set Alice as meeting moderator.
	if _, err := page.Locator("#meeting_moderator_attendee_id").SelectOption(playwright.SelectOptionValues{
		Labels: playwright.StringSlice("Alice Moderator"),
	}); err != nil {
		t.Fatalf("select moderator: %v", err)
	}
	if err := page.Locator("button:has-text('Set Moderator')").Click(); err != nil {
		t.Fatalf("click set moderator: %v", err)
	}

	// Moderator name should now be shown in bold.
	if err := page.Locator("#meeting-settings-container strong:has-text('Alice Moderator')").WaitFor(); err != nil {
		t.Fatalf("expected moderator name in settings: %v", err)
	}
	if page.URL() != urlBefore {
		t.Errorf("URL changed on set moderator: got %s, want %s", page.URL(), urlBefore)
	}

	// Clear the moderator (select "-- none --").
	if _, err := page.Locator("#meeting_moderator_attendee_id").SelectOption(playwright.SelectOptionValues{
		Values: playwright.StringSlice(""),
	}); err != nil {
		t.Fatalf("select none for moderator: %v", err)
	}
	if err := page.Locator("button:has-text('Set Moderator')").Click(); err != nil {
		t.Fatalf("click set moderator (clear): %v", err)
	}

	// Moderator section should show "None" again.
	if err := page.Locator("#meeting-settings-container p:has-text('Moderator:') strong:has-text('None')").WaitFor(); err != nil {
		t.Fatalf("expected 'None' after clearing moderator: %v", err)
	}
}

// TestSpeakers_MeetingQuotation_ToggleDisablesGender verifies that setting the
// meeting-level gender quotation to "Disabled" persists and updates the UI.
func TestSpeakers_MeetingQuotation_ToggleDisablesGender(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(agendaManageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	urlBefore := page.URL()

	// Default: gender quotation select should have "true" selected.
	genderSelect := page.Locator("#gender_quotation_enabled")
	if err := genderSelect.WaitFor(); err != nil {
		t.Fatalf("wait for gender quotation select: %v", err)
	}
	initialVal, err := genderSelect.InputValue()
	if err != nil {
		t.Fatalf("get initial gender quotation value: %v", err)
	}
	if initialVal != "true" {
		t.Errorf("expected initial gender quotation to be 'true', got %q", initialVal)
	}

	// Set gender quotation to Disabled.
	if _, err := genderSelect.SelectOption(playwright.SelectOptionValues{
		Values: playwright.StringSlice("false"),
	}); err != nil {
		t.Fatalf("select disabled: %v", err)
	}
	if err := page.Locator("#meeting-settings-container button:has-text('Apply')").Click(); err != nil {
		t.Fatalf("click apply: %v", err)
	}

	// After the partial swap, the select should reflect the new "false" value.
	if err := page.Locator("#gender_quotation_enabled").WaitFor(); err != nil {
		t.Fatalf("wait for gender quotation select after update: %v", err)
	}
	updatedVal, err := page.Locator("#gender_quotation_enabled").InputValue()
	if err != nil {
		t.Fatalf("get updated gender quotation value: %v", err)
	}
	if updatedVal != "false" {
		t.Errorf("expected gender quotation to be 'false' after applying, got %q", updatedVal)
	}

	if page.URL() != urlBefore {
		t.Errorf("URL changed on quotation toggle: got %s, want %s", page.URL(), urlBefore)
	}
}
