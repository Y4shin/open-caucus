//go:build e2e

package e2e_test

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	playwright "github.com/playwright-community/playwright-go"
)

func manageSpeakerRow(page playwright.Page, attendeeName string) playwright.Locator {
	return page.Locator("#speakers-list-container [data-testid='manage-speakers-viewport'] [data-testid='live-speaker-item']").Filter(playwright.LocatorFilterOptions{
		HasText: attendeeName,
	})
}

func manageSpeakerNamesInDisplayedOrder(t *testing.T, page playwright.Page) []string {
	t.Helper()
	raw, err := page.Evaluate(`() => {
		const container = document.querySelector("[data-testid='manage-speakers-card'] #speakers-list-container")
			|| document.querySelector("#speakers-list-container");
		if (!container) return [];
		const rows = Array.from(container.querySelectorAll("[data-testid='manage-speakers-viewport'] [data-testid='live-speaker-item']"));
		return rows.map((row) => {
			const nameEl = row.querySelector("[data-testid='live-speaker-name']");
			return (nameEl ? nameEl.textContent : row.textContent || "").trim();
		});
	}`, nil)
	if err != nil {
		t.Fatalf("read speakers names order: %v", err)
	}
	namesRaw, ok := raw.([]interface{})
	if !ok {
		t.Fatalf("unexpected speakers names payload: %#v", raw)
	}
	names := make([]string, 0, len(namesRaw))
	for _, v := range namesRaw {
		if s, ok := v.(string); ok {
			names = append(names, s)
		}
	}
	return names
}

func seedSpeakerOrderingScenario(t *testing.T, withActiveSpeaker bool) (*testServer, string, []string) {
	t.Helper()

	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "Main Topic")
	ts.activateAgendaPoint(t, "test-committee", "Board Meeting", apID)
	apIDInt := parseID(t, apID)

	addSpeaker := func(name, speakerType string, quoted, firstSpeaker, priority bool) int64 {
		secret := "secret-" + strings.ToLower(strings.ReplaceAll(name, " ", "-"))
		attendee := ts.seedAttendee(t, "test-committee", "Board Meeting", name, secret)
		entry, err := ts.repo.AddSpeaker(context.Background(), apIDInt, attendee.ID, speakerType, quoted, firstSpeaker)
		if err != nil {
			t.Fatalf("add speaker %q: %v", name, err)
		}
		if priority {
			if err := ts.repo.SetSpeakerPriority(context.Background(), entry.ID, true); err != nil {
				t.Fatalf("set priority for %q: %v", name, err)
			}
		}
		return entry.ID
	}

	doneNames := []string{"Done A", "Done B", "Done C"}
	for _, name := range doneNames {
		id := addSpeaker(name, "regular", false, false, false)
		if err := ts.repo.SetSpeakerSpeaking(context.Background(), id, apIDInt); err != nil {
			t.Fatalf("set speaking for %q: %v", name, err)
		}
		time.Sleep(5 * time.Millisecond)
		if err := ts.repo.SetSpeakerDone(context.Background(), id); err != nil {
			t.Fatalf("set done for %q: %v", name, err)
		}
		time.Sleep(5 * time.Millisecond)
	}

	activeName := "Active Speaker"
	if withActiveSpeaker {
		activeID := addSpeaker(activeName, "regular", false, false, false)
		if err := ts.repo.SetSpeakerSpeaking(context.Background(), activeID, apIDInt); err != nil {
			t.Fatalf("set active speaker: %v", err)
		}
		time.Sleep(5 * time.Millisecond)
	}

	waitingNames := []string{
		"Waiting RoPM Priority",
		"Waiting RoPM Plain",
		"Waiting Regular Priority",
		"Waiting Regular Quoted",
		"Waiting Regular First",
		"Waiting Regular Plain",
	}
	addSpeaker(waitingNames[0], "ropm", true, false, true)
	time.Sleep(5 * time.Millisecond)
	addSpeaker(waitingNames[1], "ropm", false, false, false)
	time.Sleep(5 * time.Millisecond)
	addSpeaker(waitingNames[2], "regular", false, false, true)
	time.Sleep(5 * time.Millisecond)
	addSpeaker(waitingNames[3], "regular", true, false, false)
	time.Sleep(5 * time.Millisecond)
	addSpeaker(waitingNames[4], "regular", false, true, false)
	time.Sleep(5 * time.Millisecond)
	addSpeaker(waitingNames[5], "regular", false, false, false)

	if err := ts.repo.RecomputeSpeakerOrder(context.Background(), apIDInt); err != nil {
		t.Fatalf("recompute speaker order: %v", err)
	}

	expected := append([]string{}, doneNames...)
	if withActiveSpeaker {
		expected = append(expected, activeName)
	}
	expected = append(expected, waitingNames...)
	return ts, meetingID, expected
}

// TestSpeakers_PriorityToggle_MovesToFront seeds two speakers, toggles priority
// on the second, and verifies it appears first in the waiting list.
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

	if err := manageSpeakerRow(page, "Alice Speaker").WaitFor(); err != nil {
		t.Fatalf("expected Alice in speakers list: %v", err)
	}

	urlBefore := page.URL()

	bobRow := manageSpeakerRow(page, "Bob Speaker")
	if err := bobRow.Locator("button[title='Give Priority']").Click(); err != nil {
		t.Fatalf("click priority toggle for Bob: %v", err)
	}
	if err := bobRow.Locator("button[title='Remove Priority']").WaitFor(); err != nil {
		t.Fatalf("expected remove-priority button after toggle: %v", err)
	}

	rows, err := page.Locator("#speakers-list-container [data-testid='manage-speakers-viewport'] [data-testid='live-speaker-item']").All()
	if err != nil {
		t.Fatalf("get speaker rows: %v", err)
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

// TestSpeakers_FirstSpeaker_Badge verifies that first-speaker entries render an
// extra badge compared to regular entries.
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

	firstTimerRow := manageSpeakerRow(page, "First Timer")
	if err := firstTimerRow.WaitFor(); err != nil {
		t.Fatalf("wait for First Timer row: %v", err)
	}
	repeatRow := manageSpeakerRow(page, "Repeat Speaker")
	if err := repeatRow.WaitFor(); err != nil {
		t.Fatalf("wait for Repeat Speaker row: %v", err)
	}

	firstHasBadge, err := firstTimerRow.Locator("[data-testid='live-speaker-first-badge']").Count()
	if err != nil {
		t.Fatalf("count first-speaker badges: %v", err)
	}
	repeatHasBadge, err := repeatRow.Locator("[data-testid='live-speaker-first-badge']").Count()
	if err != nil {
		t.Fatalf("count repeat-speaker first badges: %v", err)
	}
	if firstHasBadge == 0 {
		t.Errorf("expected first-speaker badge in First Timer row")
	}
	if repeatHasBadge != 0 {
		t.Errorf("expected no first-speaker badge in Repeat Speaker row, got %d", repeatHasBadge)
	}
}

// TestSpeakers_MeetingModerator_SetAndClear sets a meeting-level moderator via
// the settings form and then clears it, verifying auto-submit updates.
func TestSpeakers_MeetingModerator_SetAndClear(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	ts.seedAttendee(t, "test-committee", "Board Meeting", "Alice Moderator", "secret-am")
	aliceID := ts.getAttendeeIDForMeeting(t, "test-committee", "Board Meeting", "Alice Moderator")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(agendaManageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	urlBefore := page.URL()

	if _, err := page.Locator("#meeting_moderator_attendee_id").SelectOption(playwright.SelectOptionValues{
		Labels: playwright.StringSlice("Alice Moderator"),
	}); err != nil {
		t.Fatalf("select moderator: %v", err)
	}

	waitUntil(t, 3*time.Second, func() (bool, error) {
		val, err := page.Locator("#meeting_moderator_attendee_id").InputValue()
		return val == aliceID, err
	}, "moderator select to persist selected attendee")

	if _, err := page.Locator("#meeting_moderator_attendee_id").SelectOption(playwright.SelectOptionValues{
		Values: playwright.StringSlice(""),
	}); err != nil {
		t.Fatalf("clear moderator: %v", err)
	}
	waitUntil(t, 3*time.Second, func() (bool, error) {
		val, err := page.Locator("#meeting_moderator_attendee_id").InputValue()
		return val == "", err
	}, "moderator select to clear")

	if page.URL() != urlBefore {
		t.Errorf("URL changed during moderator updates: got %s, want %s", page.URL(), urlBefore)
	}
}

// TestSpeakers_MeetingQuotation_ToggleDisablesGender verifies that setting the
// meeting-level gender quotation to disabled persists in the UI.
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
	genderSelect := page.Locator("#gender_quotation_enabled")
	if err := genderSelect.WaitFor(); err != nil {
		t.Fatalf("wait for gender quotation select: %v", err)
	}
	initialVal, err := genderSelect.InputValue()
	if err != nil {
		t.Fatalf("read initial gender quotation value: %v", err)
	}
	if initialVal != "true" {
		t.Fatalf("expected initial gender quotation value 'true', got %q", initialVal)
	}

	if _, err := genderSelect.SelectOption(playwright.SelectOptionValues{
		Values: playwright.StringSlice("false"),
	}); err != nil {
		t.Fatalf("set gender quotation to false: %v", err)
	}
	waitUntil(t, 3*time.Second, func() (bool, error) {
		val, err := page.Locator("#gender_quotation_enabled").InputValue()
		return val == "false", err
	}, "gender quotation select to persist false")

	if page.URL() != urlBefore {
		t.Errorf("URL changed on quotation toggle: got %s, want %s", page.URL(), urlBefore)
	}
}

// TestSpeakers_SortingOrder_WithActiveSpeaker verifies displayed row order:
// DONE entries first, then SPEAKING, then WAITING according queue ordering.
func TestSpeakers_SortingOrder_WithActiveSpeaker(t *testing.T) {
	ts, meetingID, expectedOrder := seedSpeakerOrderingScenario(t, true)

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(agendaManageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	if err := manageSpeakerRow(page, expectedOrder[len(expectedOrder)-1]).WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateAttached,
	}); err != nil {
		t.Fatalf("expected all speaker rows attached: %v", err)
	}

	gotOrder := manageSpeakerNamesInDisplayedOrder(t, page)
	if !reflect.DeepEqual(gotOrder, expectedOrder) {
		t.Errorf("unexpected speaker order with active speaker:\n got: %v\nwant: %v", gotOrder, expectedOrder)
	}
}

// TestSpeakers_SortingOrder_WithoutActiveSpeaker verifies displayed row order:
// DONE entries first, then WAITING according queue ordering.
func TestSpeakers_SortingOrder_WithoutActiveSpeaker(t *testing.T) {
	ts, meetingID, expectedOrder := seedSpeakerOrderingScenario(t, false)

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(agendaManageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	if err := manageSpeakerRow(page, expectedOrder[len(expectedOrder)-1]).WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateAttached,
	}); err != nil {
		t.Fatalf("expected all speaker rows attached: %v", err)
	}

	gotOrder := manageSpeakerNamesInDisplayedOrder(t, page)
	if !reflect.DeepEqual(gotOrder, expectedOrder) {
		t.Errorf("unexpected speaker order without active speaker:\n got: %v\nwant: %v", gotOrder, expectedOrder)
	}
}

