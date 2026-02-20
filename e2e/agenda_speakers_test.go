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

// TestAgendaPoint_CreateAndShow verifies that the chairperson can create an
// agenda point via the inline form and see it in the list.
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

	if err := page.Locator("#agenda-point-list-container input[name=title]").Fill("Opening Remarks"); err != nil {
		t.Fatalf("fill title: %v", err)
	}
	if err := page.Locator("button:has-text('Add Agenda Point')").Click(); err != nil {
		t.Fatalf("click add agenda point: %v", err)
	}

	if err := page.Locator("#agenda-point-list-container td:has-text('Opening Remarks')").WaitFor(); err != nil {
		t.Fatalf("expected agenda point in table: %v", err)
	}
	if page.URL() != urlBefore {
		t.Errorf("URL changed on add: got %s, want %s", page.URL(), urlBefore)
	}
}

// TestAgendaPoint_CreateSubAgendaPoint verifies that selecting a parent creates
// a sub-agenda point shown as a child row.
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

	// Create top-level parent.
	if err := page.Locator("#agenda-point-list-container input[name=title]").Fill("Parent Item"); err != nil {
		t.Fatalf("fill parent title: %v", err)
	}
	if err := page.Locator("button:has-text('Add Agenda Point')").Click(); err != nil {
		t.Fatalf("click add parent agenda point: %v", err)
	}
	if err := page.Locator("#agenda-point-list-container td:has-text('Parent Item')").WaitFor(); err != nil {
		t.Fatalf("expected parent agenda point in table: %v", err)
	}

	// Create child under selected parent.
	if err := page.Locator("#agenda-point-list-container input[name=title]").Fill("Child Item"); err != nil {
		t.Fatalf("fill child title: %v", err)
	}
	if _, err := page.Locator("#ap_parent_id").SelectOption(playwright.SelectOptionValues{
		Labels: playwright.StringSlice("Parent Item"),
	}); err != nil {
		t.Fatalf("select parent agenda point: %v", err)
	}
	if err := page.Locator("button:has-text('Add Agenda Point')").Click(); err != nil {
		t.Fatalf("click add child agenda point: %v", err)
	}

	if err := page.Locator("#agenda-point-list-container td:has-text('-> Child Item')").WaitFor(); err != nil {
		t.Fatalf("expected child agenda point row: %v", err)
	}
}

// TestAgendaPoint_CreateSubAgendaPoint_AfterReload verifies that a top-level
// agenda point can be created, then after reloading the page, selected as the
// parent to create a sub-agenda point.
func TestAgendaPoint_CreateSubAgendaPoint_AfterReload(t *testing.T) {
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

	if err := page.Locator("#agenda-point-list-container input[name=title]").Fill("Parent Reloaded"); err != nil {
		t.Fatalf("fill parent title: %v", err)
	}
	if err := page.Locator("button:has-text('Add Agenda Point')").Click(); err != nil {
		t.Fatalf("click add parent agenda point: %v", err)
	}
	if err := page.Locator("#agenda-point-list-container td:has-text('Parent Reloaded')").WaitFor(); err != nil {
		t.Fatalf("expected parent agenda point in table: %v", err)
	}

	var meetingIDInt int64
	fmt.Sscanf(meetingID, "%d", &meetingIDInt)
	topLevel, err := ts.repo.ListAgendaPointsForMeeting(context.Background(), meetingIDInt)
	if err != nil {
		t.Fatalf("list agenda points to resolve parent ID: %v", err)
	}
	parentIDValue := ""
	for _, ap := range topLevel {
		if ap.Title == "Parent Reloaded" {
			parentIDValue = strconv.FormatInt(ap.ID, 10)
			break
		}
	}
	if parentIDValue == "" {
		t.Fatalf("failed to resolve parent agenda ID for 'Parent Reloaded'")
	}

	if _, err := page.Reload(); err != nil {
		t.Fatalf("reload manage page: %v", err)
	}
	if err := page.Locator("#agenda-point-list-container td:has-text('Parent Reloaded')").WaitFor(); err != nil {
		t.Fatalf("expected parent agenda point after reload: %v", err)
	}

	if err := page.Locator("#agenda-point-list-container input[name=title]").Fill("Child After Reload"); err != nil {
		t.Fatalf("fill child title: %v", err)
	}
	if _, err := page.Locator("#ap_parent_id").SelectOption(playwright.SelectOptionValues{
		Values: playwright.StringSlice(parentIDValue),
	}); err != nil {
		t.Fatalf("select parent agenda point after reload: %v", err)
	}
	if err := page.Locator("button:has-text('Add Agenda Point')").Click(); err != nil {
		t.Fatalf("click add child agenda point: %v", err)
	}

	if err := page.Locator("#agenda-point-list-container td:has-text('-> Child After Reload')").WaitFor(); err != nil {
		t.Fatalf("expected child agenda point row after reload flow: %v", err)
	}
}

// TestAgendaPoint_Activate verifies that clicking Activate marks the agenda
// point as active in the table without a full page reload.
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

	if err := page.Locator("button:has-text('Activate')").First().Click(); err != nil {
		t.Fatalf("click activate: %v", err)
	}

	// After activation the Active column should show "Yes"
	if err := page.Locator("td:has-text('Yes')").WaitFor(); err != nil {
		t.Fatalf("expected active indicator: %v", err)
	}
	if page.URL() != urlBefore {
		t.Errorf("URL changed on activate: got %s, want %s", page.URL(), urlBefore)
	}
}

// TestAgendaPoint_Delete verifies that deleting an agenda point removes it
// from the list without a full page reload.
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

	if err := page.Locator("td:has-text('Deletable Item')").WaitFor(); err != nil {
		t.Fatalf("agenda point not visible before delete: %v", err)
	}

	urlBefore := page.URL()

	page.OnDialog(func(d playwright.Dialog) {
		if err := d.Accept(); err != nil {
			t.Logf("accept dialog error: %v", err)
		}
	})

	if err := page.Locator("button:has-text('Delete')").First().Click(); err != nil {
		t.Fatalf("click delete: %v", err)
	}

	if err := page.Locator("td:has-text('Deletable Item')").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected agenda point to disappear: %v", err)
	}
	if page.URL() != urlBefore {
		t.Errorf("URL changed on delete: got %s, want %s", page.URL(), urlBefore)
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

// TestSpeakersList_AddSpeaker verifies that the chairperson can add a speaker
// to the active agenda point and see them in the table.
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

	urlBefore := page.URL()

	if _, err := page.Locator("#speaker_attendee_id").SelectOption(playwright.SelectOptionValues{
		Labels: playwright.StringSlice("Alice Member"),
	}); err != nil {
		t.Fatalf("select attendee: %v", err)
	}
	if err := page.Locator("button:has-text('Add Speaker')").Click(); err != nil {
		t.Fatalf("click add speaker: %v", err)
	}

	if err := page.Locator("#speakers-list-container td:has-text('Alice Member')").WaitFor(); err != nil {
		t.Fatalf("expected speaker in table: %v", err)
	}
	if err := page.Locator("#speakers-list-container td:has-text('WAITING')").WaitFor(); err != nil {
		t.Fatalf("expected WAITING status: %v", err)
	}
	if page.URL() != urlBefore {
		t.Errorf("URL changed on add speaker: got %s, want %s", page.URL(), urlBefore)
	}
}

// TestSpeakersList_OneNonDoneEntryPerType verifies that an attendee can have
// at most one non-DONE entry for each speaker type (regular, ropm).
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

	// Add first regular entry.
	if _, err := page.Locator("#speaker_attendee_id").SelectOption(playwright.SelectOptionValues{
		Labels: playwright.StringSlice("Alice Member"),
	}); err != nil {
		t.Fatalf("select attendee: %v", err)
	}
	if _, err := page.Locator("#speaker_type").SelectOption(playwright.SelectOptionValues{
		Values: playwright.StringSlice("regular"),
	}); err != nil {
		t.Fatalf("select regular type: %v", err)
	}
	if err := page.Locator("#speakers-list-container button:has-text('Add Speaker')").Click(); err != nil {
		t.Fatalf("click add regular speaker: %v", err)
	}
	if err := page.Locator("#speakers-list-container tr:has(td:has-text('Alice Member')):has(td:has-text('regular'))").WaitFor(); err != nil {
		t.Fatalf("expected regular speaker row: %v", err)
	}

	// Try to add second non-done regular entry -> must be rejected.
	if _, err := page.Locator("#speaker_attendee_id").SelectOption(playwright.SelectOptionValues{
		Labels: playwright.StringSlice("Alice Member"),
	}); err != nil {
		t.Fatalf("re-select attendee: %v", err)
	}
	if _, err := page.Locator("#speaker_type").SelectOption(playwright.SelectOptionValues{
		Values: playwright.StringSlice("regular"),
	}); err != nil {
		t.Fatalf("re-select regular type: %v", err)
	}
	if err := page.Locator("#speakers-list-container button:has-text('Add Speaker')").Click(); err != nil {
		t.Fatalf("click add duplicate regular speaker: %v", err)
	}
	if err := page.Locator("#speakers-list-container p:has-text('already has a non-done regular')").WaitFor(); err != nil {
		t.Fatalf("expected duplicate-regular error: %v", err)
	}
	regularCount, err := page.Locator("#speakers-list-container tr:has(td:has-text('Alice Member')):has(td:has-text('regular'))").Count()
	if err != nil {
		t.Fatalf("count regular rows: %v", err)
	}
	if regularCount != 1 {
		t.Fatalf("expected exactly one non-done regular row, got %d", regularCount)
	}

	// Add one ropm entry -> allowed (different type).
	if _, err := page.Locator("#speaker_attendee_id").SelectOption(playwright.SelectOptionValues{
		Labels: playwright.StringSlice("Alice Member"),
	}); err != nil {
		t.Fatalf("select attendee for ropm: %v", err)
	}
	if _, err := page.Locator("#speaker_type").SelectOption(playwright.SelectOptionValues{
		Values: playwright.StringSlice("ropm"),
	}); err != nil {
		t.Fatalf("select ropm type: %v", err)
	}
	if err := page.Locator("#speakers-list-container button:has-text('Add Speaker')").Click(); err != nil {
		t.Fatalf("click add ropm speaker: %v", err)
	}
	if err := page.Locator("#speakers-list-container tr:has(td:has-text('Alice Member')):has(td:has-text('ropm'))").WaitFor(); err != nil {
		t.Fatalf("expected ropm speaker row: %v", err)
	}

	// Try to add second non-done ropm entry -> must be rejected.
	if _, err := page.Locator("#speaker_attendee_id").SelectOption(playwright.SelectOptionValues{
		Labels: playwright.StringSlice("Alice Member"),
	}); err != nil {
		t.Fatalf("re-select attendee for duplicate ropm: %v", err)
	}
	if _, err := page.Locator("#speaker_type").SelectOption(playwright.SelectOptionValues{
		Values: playwright.StringSlice("ropm"),
	}); err != nil {
		t.Fatalf("re-select ropm type: %v", err)
	}
	if err := page.Locator("#speakers-list-container button:has-text('Add Speaker')").Click(); err != nil {
		t.Fatalf("click add duplicate ropm speaker: %v", err)
	}
	if err := page.Locator("#speakers-list-container p:has-text('already has a non-done ropm')").WaitFor(); err != nil {
		t.Fatalf("expected duplicate-ropm error: %v", err)
	}
	ropmCount, err := page.Locator("#speakers-list-container tr:has(td:has-text('Alice Member')):has(td:has-text('ropm'))").Count()
	if err != nil {
		t.Fatalf("count ropm rows: %v", err)
	}
	if ropmCount != 1 {
		t.Fatalf("expected exactly one non-done ropm row, got %d", ropmCount)
	}
}

// TestSpeakersList_StartEnd verifies the full Start → End flow for a speaker.
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

	urlBefore := page.URL()

	if err := page.Locator("button:has-text('Start')").First().Click(); err != nil {
		t.Fatalf("click start: %v", err)
	}
	if err := page.Locator("#speakers-list-container td:has-text('SPEAKING')").WaitFor(); err != nil {
		t.Fatalf("expected SPEAKING status: %v", err)
	}

	if err := page.Locator("#speakers-list-container button:has-text('End')").First().Click(); err != nil {
		t.Fatalf("click end: %v", err)
	}
	if err := page.Locator("#speakers-list-container td:has-text('DONE')").WaitFor(); err != nil {
		t.Fatalf("expected DONE status: %v", err)
	}

	if page.URL() != urlBefore {
		t.Errorf("URL changed: got %s, want %s", page.URL(), urlBefore)
	}
}

// TestSpeakersList_Withdraw verifies that withdrawing a speaker changes their
// status to WITHDRAWN and removes the action buttons.
func TestSpeakersList_Withdraw(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	ts.seedAttendee(t, "test-committee", "Board Meeting", "Carol Member", "secret-carol")
	apID := ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "Main Topic")
	ts.activateAgendaPoint(t, "test-committee", "Board Meeting", apID)
	attendeeID := ts.getAttendeeIDForMeeting(t, "test-committee", "Board Meeting", "Carol Member")
	ts.seedSpeaker(t, apID, attendeeID)

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(agendaManageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	urlBefore := page.URL()

	if err := page.Locator("#speakers-list-container button:has-text('Withdraw')").First().Click(); err != nil {
		t.Fatalf("click withdraw: %v", err)
	}
	if err := page.Locator("#speakers-list-container td:has-text('WITHDRAWN')").WaitFor(); err != nil {
		t.Fatalf("expected WITHDRAWN status: %v", err)
	}

	if page.URL() != urlBefore {
		t.Errorf("URL changed: got %s, want %s", page.URL(), urlBefore)
	}
}

// TestSpeakersList_Remove verifies that removing a speaker entry deletes the row.
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

	if err := page.Locator("#speakers-list-container td:has-text('Dave Member')").WaitFor(); err != nil {
		t.Fatalf("speaker not visible before remove: %v", err)
	}

	urlBefore := page.URL()

	page.OnDialog(func(d playwright.Dialog) {
		if err := d.Accept(); err != nil {
			t.Logf("accept dialog error: %v", err)
		}
	})

	if err := page.Locator("#speakers-list-container button:has-text('Remove')").First().Click(); err != nil {
		t.Fatalf("click remove speaker: %v", err)
	}

	if err := page.Locator("#speakers-list-container td:has-text('Dave Member')").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected speaker row to disappear: %v", err)
	}
	if page.URL() != urlBefore {
		t.Errorf("URL changed: got %s, want %s", page.URL(), urlBefore)
	}
}
