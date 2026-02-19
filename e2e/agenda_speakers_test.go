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
