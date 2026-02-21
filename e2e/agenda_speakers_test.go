//go:build e2e

package e2e_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

func agendaManageURL(baseURL, slug, meetingID string) string {
	return fmt.Sprintf("%s/committee/%s/meeting/%s/manage", baseURL, slug, meetingID)
}

// activateAgendaPoint sets the meeting's current agenda point directly via the repo.
func (ts *testServer) activateAgendaPoint(t *testing.T, slug, meetingName, apIDStr string) {
	t.Helper()
	meetingIDStr := ts.getMeetingID(t, slug, meetingName)
	var mid, apid int64
	fmt.Sscanf(meetingIDStr, "%d", &mid)
	fmt.Sscanf(apIDStr, "%d", &apid)
	if err := ts.repo.SetCurrentAgendaPoint(context.Background(), mid, &apid); err != nil {
		t.Fatalf("activate agenda point: %v", err)
	}
}

// seedSpeaker adds a speaker entry to an agenda point directly via the repo.
func (ts *testServer) seedSpeaker(t *testing.T, apIDStr, attendeeIDStr string) string {
	t.Helper()
	var apid, aid int64
	fmt.Sscanf(apIDStr, "%d", &apid)
	fmt.Sscanf(attendeeIDStr, "%d", &aid)
	entry, err := ts.repo.AddSpeaker(context.Background(), apid, aid, "regular", false, false)
	if err != nil {
		t.Fatalf("seed speaker: %v", err)
	}
	return strconv.FormatInt(entry.ID, 10)
}

// getAttendeeIDForMeeting returns the string attendee ID for the first attendee with the given name.
func (ts *testServer) getAttendeeIDForMeeting(t *testing.T, slug, meetingName, fullName string) string {
	t.Helper()
	meetingIDStr := ts.getMeetingID(t, slug, meetingName)
	var mid int64
	fmt.Sscanf(meetingIDStr, "%d", &mid)
	attendees, err := ts.repo.ListAttendeesForMeeting(context.Background(), mid)
	if err != nil {
		t.Fatalf("list attendees: %v", err)
	}
	for _, a := range attendees {
		if a.FullName == fullName {
			return strconv.FormatInt(a.ID, 10)
		}
	}
	t.Fatalf("attendee %q not found in meeting %q", fullName, meetingName)
	return ""
}

func addAgendaPoint(t *testing.T, page playwright.Page, title string) {
	t.Helper()
	if err := page.Locator("#agenda-point-list-container input[name=title]").Fill(title); err != nil {
		t.Fatalf("fill agenda title: %v", err)
	}
	if err := page.Locator("#agenda-point-list-container .manage-agenda-add-form button[type=submit]").Click(); err != nil {
		t.Fatalf("submit agenda form: %v", err)
	}
}

func openSpeakerAddDialog(t *testing.T, page playwright.Page) {
	t.Helper()
	if err := page.Locator("#speakers-list-container button[data-manage-dialog-open]").Click(); err != nil {
		t.Fatalf("open add speaker dialog: %v", err)
	}
	if err := page.Locator("#speaker-add-candidates-container").WaitFor(); err != nil {
		t.Fatalf("wait add speaker candidates: %v", err)
	}
}

func speakerCandidateCard(page playwright.Page, name string) playwright.Locator {
	return page.Locator("#speaker-add-candidates-container .manage-speaker-candidate-card").Filter(playwright.LocatorFilterOptions{
		HasText: name,
	})
}

// TestAgendaPoint_CreateAndShow verifies that the chairperson can create an
// agenda point via the inline form and see it in the card list.
func TestAgendaPoint_CreateAndShow(t *testing.T) {
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
	addAgendaPoint(t, page, "Opening Remarks")

	if err := page.Locator("#agenda-point-list-container .manage-agenda-point-card:has-text('Opening Remarks')").WaitFor(); err != nil {
		t.Fatalf("expected agenda point card: %v", err)
	}
	if page.URL() != urlBefore {
		t.Errorf("URL changed on add: got %s, want %s", page.URL(), urlBefore)
	}
}

// TestAgendaPoint_CreateSubAgendaPoint verifies that selecting a parent creates
// a sub-agenda point rendered as an indented child card.
func TestAgendaPoint_CreateSubAgendaPoint(t *testing.T) {
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

	addAgendaPoint(t, page, "Parent Item")
	if err := page.Locator("#agenda-point-list-container .manage-agenda-point-card:has-text('Parent Item')").WaitFor(); err != nil {
		t.Fatalf("expected parent agenda card: %v", err)
	}

	if err := page.Locator("#agenda-point-list-container input[name=title]").Fill("Child Item"); err != nil {
		t.Fatalf("fill child title: %v", err)
	}
	if _, err := page.Locator("#ap_parent_id").SelectOption(playwright.SelectOptionValues{
		Labels: playwright.StringSlice("Parent Item"),
	}); err != nil {
		t.Fatalf("select parent agenda point: %v", err)
	}
	if err := page.Locator("#agenda-point-list-container .manage-agenda-add-form button[type=submit]").Click(); err != nil {
		t.Fatalf("submit child agenda form: %v", err)
	}

	if err := page.Locator("#agenda-point-list-container .manage-agenda-point-card:has-text('Child Item'):has-text('Child')").WaitFor(); err != nil {
		t.Fatalf("expected child agenda card: %v", err)
	}
}

// TestAgendaPoint_Activate verifies that activating an agenda point marks its
// card as active without a full page reload.
func TestAgendaPoint_Activate(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "Item One")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(agendaManageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	urlBefore := page.URL()
	card := page.Locator("#agenda-point-list-container .manage-agenda-point-card").Filter(playwright.LocatorFilterOptions{
		HasText: "Item One",
	})
	if err := card.Locator("button[title='Activate agenda point']").Click(); err != nil {
		t.Fatalf("click activate: %v", err)
	}

	if err := card.Locator("button[title='Activate agenda point']").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected activate button to disappear after activation: %v", err)
	}
	if err := card.Locator(".manage-agenda-point-badge-active, [class*='ManageAgendaPointBadgeActive']").WaitFor(); err != nil {
		t.Fatalf("expected active badge on activated agenda point: %v", err)
	}
	if page.URL() != urlBefore {
		t.Errorf("URL changed on activate: got %s, want %s", page.URL(), urlBefore)
	}
}

// TestAgendaPoint_Delete verifies deleting an agenda point removes its card.
func TestAgendaPoint_Delete(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "Deletable Item")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(agendaManageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	card := page.Locator("#agenda-point-list-container .manage-agenda-point-card").Filter(playwright.LocatorFilterOptions{
		HasText: "Deletable Item",
	})
	if err := card.WaitFor(); err != nil {
		t.Fatalf("agenda point not visible before delete: %v", err)
	}

	page.OnDialog(func(d playwright.Dialog) {
		if err := d.Accept(); err != nil {
			t.Logf("accept dialog error: %v", err)
		}
	})
	if err := card.Locator("button[title='Delete agenda point']").Click(); err != nil {
		t.Fatalf("click delete: %v", err)
	}

	if err := page.Locator("#agenda-point-list-container .manage-agenda-point-card:has-text('Deletable Item')").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected agenda point to disappear: %v", err)
	}
}

// TestSpeakersList_NoActivePoint verifies that the speakers section shows a
// message when no agenda point is active.
func TestSpeakersList_NoActivePoint(t *testing.T) {
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

	if err := page.Locator("text=No active agenda point").WaitFor(); err != nil {
		t.Fatalf("expected no-active-point message: %v", err)
	}
}

// TestSpeakersList_AddSpeaker verifies add-speaker modal flow for an active point.
func TestSpeakersList_AddSpeaker(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	ts.seedAttendee(t, "test-committee", "Board Meeting", "Alice Member", "secret-alice")
	apID := ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "Main Topic")
	ts.activateAgendaPoint(t, "test-committee", "Board Meeting", apID)

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(agendaManageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	openSpeakerAddDialog(t, page)
	card := speakerCandidateCard(page, "Alice Member")
	if err := card.WaitFor(); err != nil {
		t.Fatalf("expected speaker candidate card: %v", err)
	}
	if err := card.Locator("button[title='Add regular speech']").Click(); err != nil {
		t.Fatalf("add regular speech: %v", err)
	}

	if err := page.Locator("#speakers-list-container .live-speaker-row:has-text('Alice Member')").WaitFor(); err != nil {
		t.Fatalf("expected speaker in list: %v", err)
	}
}

// TestSpeakersList_OneNonDoneEntryPerType verifies one waiting entry per type.
func TestSpeakersList_OneNonDoneEntryPerType(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	ts.seedAttendee(t, "test-committee", "Board Meeting", "Alice Member", "secret-alice")
	apID := ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "Main Topic")
	ts.activateAgendaPoint(t, "test-committee", "Board Meeting", apID)

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(agendaManageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	openSpeakerAddDialog(t, page)
	card := speakerCandidateCard(page, "Alice Member")
	if err := card.Locator("button[title='Add regular speech']").Click(); err != nil {
		t.Fatalf("add first regular speech: %v", err)
	}
	if err := page.Locator("#speakers-list-container .live-speaker-row:has-text('Alice Member')").WaitFor(); err != nil {
		t.Fatalf("expected speaker row after first add: %v", err)
	}

	openSpeakerAddDialog(t, page)
	card = speakerCandidateCard(page, "Alice Member")
	regularDisabled, err := card.Locator("button[title='Add regular speech']").IsDisabled()
	if err != nil {
		t.Fatalf("read regular button disabled state: %v", err)
	}
	if !regularDisabled {
		t.Fatalf("expected regular add button to be disabled after waiting regular entry")
	}

	ropmDisabled, err := card.Locator("button[title='Add RoPM speech']").IsDisabled()
	if err != nil {
		t.Fatalf("read ropm button disabled state: %v", err)
	}
	if ropmDisabled {
		t.Fatalf("expected ropm add button to still be enabled")
	}
	if err := card.Locator("button[title='Add RoPM speech']").Click(); err != nil {
		t.Fatalf("add ropm speech: %v", err)
	}

	openSpeakerAddDialog(t, page)
	card = speakerCandidateCard(page, "Alice Member")
	ropmDisabled, err = card.Locator("button[title='Add RoPM speech']").IsDisabled()
	if err != nil {
		t.Fatalf("read ropm button disabled state after add: %v", err)
	}
	if !ropmDisabled {
		t.Fatalf("expected ropm add button to be disabled after waiting ropm entry")
	}
}

// TestSpeakersList_StartEnd verifies waiting -> speaking -> done transitions.
func TestSpeakersList_StartEnd(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	ts.seedAttendee(t, "test-committee", "Board Meeting", "Bob Member", "secret-bob")
	apID := ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "Main Topic")
	ts.activateAgendaPoint(t, "test-committee", "Board Meeting", apID)
	attendeeID := ts.getAttendeeIDForMeeting(t, "test-committee", "Board Meeting", "Bob Member")
	ts.seedSpeaker(t, apID, attendeeID)

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(agendaManageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	row := page.Locator("#speakers-list-container .live-speaker-row").Filter(playwright.LocatorFilterOptions{
		HasText: "Bob Member",
	})
	if err := row.Locator("button[title='Start']").Click(); err != nil {
		t.Fatalf("click start: %v", err)
	}
	if err := page.Locator("#speakers-list-container .live-speaker-row.speaking:has-text('Bob Member')").WaitFor(); err != nil {
		t.Fatalf("expected speaking row: %v", err)
	}

	if err := row.Locator("button[title='End']").Click(); err != nil {
		t.Fatalf("click end: %v", err)
	}
	if err := page.Locator("#speakers-list-container .live-speaker-row.speaking:has-text('Bob Member')").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected speaking state to clear: %v", err)
	}
	if err := row.WaitFor(); err != nil {
		t.Fatalf("expected row to remain after done: %v", err)
	}
}

// TestSpeakersList_Remove verifies that removing a waiting speaker deletes the row.
func TestSpeakersList_Remove(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	ts.seedAttendee(t, "test-committee", "Board Meeting", "Dave Member", "secret-dave")
	apID := ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "Main Topic")
	ts.activateAgendaPoint(t, "test-committee", "Board Meeting", apID)
	attendeeID := ts.getAttendeeIDForMeeting(t, "test-committee", "Board Meeting", "Dave Member")
	ts.seedSpeaker(t, apID, attendeeID)

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(agendaManageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	row := page.Locator("#speakers-list-container .live-speaker-row").Filter(playwright.LocatorFilterOptions{
		HasText: "Dave Member",
	})
	if err := row.WaitFor(); err != nil {
		t.Fatalf("speaker not visible before remove: %v", err)
	}

	page.OnDialog(func(d playwright.Dialog) {
		if err := d.Accept(); err != nil {
			t.Logf("accept dialog error: %v", err)
		}
	})
	if err := row.Locator("button[title='Remove']").Click(); err != nil {
		t.Fatalf("click remove speaker: %v", err)
	}

	if err := page.Locator("#speakers-list-container .live-speaker-row:has-text('Dave Member')").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected speaker row to disappear: %v", err)
	}
}
