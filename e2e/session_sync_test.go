//go:build e2e

package e2e_test

import (
	"context"
	"fmt"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

func parseID(t *testing.T, idStr string) int64 {
	t.Helper()
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		t.Fatalf("parse id %q: %v", idStr, err)
	}
	return id
}

// TestSync_LiveAndManage_SpeakerLifecycleUpdates verifies speaker add/start/end
// actions in manage propagate to the attendee live session via SSE.
func TestSync_LiveAndManage_SpeakerLifecycleUpdates(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Sync Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Sync Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Sync Meeting", "Main Topic")
	ts.activateAgendaPoint(t, "test-committee", "Sync Meeting", apID)
	ts.seedAttendee(t, "test-committee", "Sync Meeting", "Alice Member", "secret-alice")

	livePage := newPage(t)
	attendeeLoginHelper(t, livePage, ts.URL, "test-committee", meetingID, "secret-alice")
	liveURLBefore := livePage.URL()

	managePage := newPage(t)
	userLogin(t, managePage, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := managePage.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	openSpeakerAddDialog(t, managePage)
	if err := speakerCandidateCard(managePage, "Alice Member").Locator("button[title='Add regular speech']").Click(); err != nil {
		t.Fatalf("add regular speech for Alice: %v", err)
	}

	liveRow := livePage.Locator("#attendee-speakers-list .live-speakers-list-viewport .live-speaker-row").Filter(playwright.LocatorFilterOptions{
		HasText: "Alice Member",
	})
	if err := liveRow.WaitFor(); err != nil {
		t.Fatalf("expected Alice row on live page after add: %v", err)
	}

	manageRow := managePage.Locator("#speakers-list-container .live-speaker-row").Filter(playwright.LocatorFilterOptions{
		HasText: "Alice Member",
	})
	if err := manageRow.Locator("button[title='Start']").Click(); err != nil {
		t.Fatalf("start speech in manage: %v", err)
	}
	if err := livePage.Locator("#attendee-speakers-list .live-speakers-list-viewport .live-speaker-row.speaking:has-text('Alice Member')").WaitFor(); err != nil {
		t.Fatalf("expected speaking state on live after manage start: %v", err)
	}

	if err := manageRow.Locator("button[title='End']").Click(); err != nil {
		t.Fatalf("end speech in manage: %v", err)
	}
	if err := livePage.Locator("#attendee-speakers-list .live-speakers-list-viewport .live-speaker-row.speaking:has-text('Alice Member')").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected speaking state to clear on live after manage end: %v", err)
	}

	if livePage.URL() != liveURLBefore {
		t.Errorf("live page URL changed during SSE updates: before=%s after=%s", liveURLBefore, livePage.URL())
	}
}

// TestSync_LiveAndLive_SelfAddPropagates verifies a self-add action in one live
// attendee session updates another live attendee session for the same meeting.
func TestSync_LiveAndLive_SelfAddPropagates(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeeting(t, "test-committee", "Sync Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Sync Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Sync Meeting", "Main Topic")
	ts.activateAgendaPoint(t, "test-committee", "Sync Meeting", apID)
	ts.seedAttendee(t, "test-committee", "Sync Meeting", "Alice Member", "secret-alice")
	ts.seedAttendee(t, "test-committee", "Sync Meeting", "Bob Member", "secret-bob")

	alicePage := newPage(t)
	attendeeLoginHelper(t, alicePage, ts.URL, "test-committee", meetingID, "secret-alice")

	bobPage := newPage(t)
	attendeeLoginHelper(t, bobPage, ts.URL, "test-committee", meetingID, "secret-bob")
	bobURLBefore := bobPage.URL()

	if err := alicePage.Locator("[data-testid='live-add-self-regular']").Click(); err != nil {
		t.Fatalf("alice self-add regular: %v", err)
	}
	if err := bobPage.Locator("#attendee-speakers-list .live-speakers-list-viewport .live-speaker-row:has-text('Alice Member')").WaitFor(); err != nil {
		t.Fatalf("expected Alice to appear on Bob's live page via SSE: %v", err)
	}
	if bobPage.URL() != bobURLBefore {
		t.Errorf("bob live URL changed during SSE updates: before=%s after=%s", bobURLBefore, bobPage.URL())
	}
}

// TestSync_LiveYield_UpdatesManage verifies yielding an ongoing speech from live
// clears the speaking state on an open manage session.
func TestSync_LiveYield_UpdatesManage(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Sync Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Sync Meeting")
	apIDStr := ts.seedAgendaPoint(t, "test-committee", "Sync Meeting", "Main Topic")
	ts.activateAgendaPoint(t, "test-committee", "Sync Meeting", apIDStr)
	alice := ts.seedAttendee(t, "test-committee", "Sync Meeting", "Alice Member", "secret-alice")

	speakerIDStr := ts.seedSpeaker(t, apIDStr, fmt.Sprintf("%d", alice.ID))
	speakerID := parseID(t, speakerIDStr)
	apID := parseID(t, apIDStr)
	if err := ts.repo.SetSpeakerSpeaking(context.Background(), speakerID, apID); err != nil {
		t.Fatalf("seed speaking state: %v", err)
	}

	livePage := newPage(t)
	attendeeLoginHelper(t, livePage, ts.URL, "test-committee", meetingID, "secret-alice")

	managePage := newPage(t)
	userLogin(t, managePage, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := managePage.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	if err := managePage.Locator("#speakers-list-container .live-speaker-row.speaking:has-text('Alice Member')").WaitFor(); err != nil {
		t.Fatalf("expected speaking row in manage before yield: %v", err)
	}
	if err := livePage.Locator("[data-testid='live-self-yield']").Click(); err != nil {
		t.Fatalf("click yield on live: %v", err)
	}
	if err := managePage.Locator("#speakers-list-container .live-speaker-row.speaking:has-text('Alice Member')").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected speaking state to clear in manage after live yield: %v", err)
	}
}

// TestSync_ManageAndManage_SpeakerActionsPropagate verifies speaker state
// changes in one manage session propagate to another manage session.
func TestSync_ManageAndManage_SpeakerActionsPropagate(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Sync Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Sync Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Sync Meeting", "Main Topic")
	ts.activateAgendaPoint(t, "test-committee", "Sync Meeting", apID)
	alice := ts.seedAttendee(t, "test-committee", "Sync Meeting", "Alice Member", "secret-alice")
	ts.seedSpeaker(t, apID, fmt.Sprintf("%d", alice.ID))

	pageA := newPage(t)
	userLogin(t, pageA, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := pageA.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page A: %v", err)
	}

	pageB := newPage(t)
	userLogin(t, pageB, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := pageB.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page B: %v", err)
	}

	rowA := pageA.Locator("#speakers-list-container .live-speaker-row").Filter(playwright.LocatorFilterOptions{HasText: "Alice Member"})
	if err := rowA.Locator("button[title='Start']").Click(); err != nil {
		t.Fatalf("start speech in manage A: %v", err)
	}
	if err := pageB.Locator("#speakers-list-container .live-speaker-row.speaking:has-text('Alice Member')").WaitFor(); err != nil {
		t.Fatalf("expected speaking state propagated to manage B: %v", err)
	}

	if err := rowA.Locator("button[title='End']").Click(); err != nil {
		t.Fatalf("end speech in manage A: %v", err)
	}
	if err := pageB.Locator("#speakers-list-container .live-speaker-row.speaking:has-text('Alice Member')").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected end state propagated to manage B: %v", err)
	}
}
