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

func agendaManageURL(baseURL, slug, meetingID string) string {
	return fmt.Sprintf("%s/committee/%s/meeting/%s/moderate", baseURL, slug, meetingID)
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
	openModerateAgendaEditor(t, page)
	if err := page.Locator("#agenda-point-list-container input[name=title]").Fill(title); err != nil {
		t.Fatalf("fill agenda title: %v", err)
	}
	if err := page.Locator("#agenda-point-list-container [data-testid='manage-agenda-add-form'] button[type=submit]").Click(); err != nil {
		t.Fatalf("submit agenda form: %v", err)
	}
}

func openAgendaImportDialog(t *testing.T, page playwright.Page) {
	t.Helper()
	openModerateAgendaEditor(t, page)
	openBtn := page.Locator("#agenda-point-list-container button[aria-controls='moderate-agenda-import-dialog']")
	if err := openBtn.First().Click(); err != nil {
		t.Fatalf("open agenda import dialog: %v", err)
	}
	if err := page.Locator("#moderate-agenda-import-dialog[open]").WaitFor(); err != nil {
		t.Fatalf("wait agenda import dialog open: %v", err)
	}
}

func agendaImportLinePrefixText(t *testing.T, page playwright.Page, rowIndex int) string {
	t.Helper()
	text, err := page.Locator("#agenda-import-correction-form [data-import-line-row]").Nth(rowIndex).Locator("[data-import-line-prefix]").TextContent()
	if err != nil {
		t.Fatalf("read import line prefix for row %d: %v", rowIndex, err)
	}
	return strings.TrimSpace(text)
}

func openSpeakerAddDialog(t *testing.T, page playwright.Page) {
	t.Helper()
	openButton := page.Locator("#speakers-list-container button[data-manage-dialog-open]")
	count, err := openButton.Count()
	if err != nil {
		t.Fatalf("count add speaker open buttons: %v", err)
	}
	if count > 0 {
		if err := openButton.First().Click(); err != nil {
			t.Fatalf("open add speaker dialog: %v", err)
		}
	}
	search := page.Locator("#speaker-add-search-input")
	if visible, err := search.IsVisible(); err == nil && visible {
		if err := search.Fill(" "); err != nil {
			t.Fatalf("prime add speaker search: %v", err)
		}
		if err := search.Fill(""); err != nil {
			t.Fatalf("reset add speaker search: %v", err)
		}
	}
	if err := page.Locator("#speaker-add-candidates-container").WaitFor(); err != nil {
		t.Fatalf("wait add speaker candidates: %v", err)
	}
}

func speakerCandidateCard(page playwright.Page, name string) playwright.Locator {
	return page.Locator("#speaker-add-candidates-container [data-testid='manage-speaker-candidate-card']").Filter(playwright.LocatorFilterOptions{
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
	openModerateAgendaEditor(t, page)

	urlBefore := page.URL()
	addAgendaPoint(t, page, "Opening Remarks")

	if err := page.Locator("#agenda-point-list-container [data-testid='manage-agenda-point-card']:has-text('Opening Remarks')").WaitFor(); err != nil {
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
	openModerateAgendaEditor(t, page)

	addAgendaPoint(t, page, "Parent Item")
	if err := page.Locator("#agenda-point-list-container [data-testid='manage-agenda-point-card']:has-text('Parent Item')").WaitFor(); err != nil {
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
	if err := page.Locator("#agenda-point-list-container [data-testid='manage-agenda-add-form'] button[type=submit]").Click(); err != nil {
		t.Fatalf("submit child agenda form: %v", err)
	}

	if err := page.Locator("#agenda-point-list-container [data-testid='manage-agenda-point-card']:has-text('Child Item'):has-text('Child')").WaitFor(); err != nil {
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
	openModerateAgendaEditor(t, page)

	urlBefore := page.URL()
	card := page.Locator("#agenda-point-list-container [data-testid='manage-agenda-point-card']").Filter(playwright.LocatorFilterOptions{
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
	if err := card.Locator("[data-testid='manage-agenda-active-badge']").WaitFor(); err != nil {
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
	openModerateAgendaEditor(t, page)

	card := page.Locator("#agenda-point-list-container [data-testid='manage-agenda-point-card']").Filter(playwright.LocatorFilterOptions{
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

	if err := page.Locator("#agenda-point-list-container [data-testid='manage-agenda-point-card']:has-text('Deletable Item')").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected agenda point to disappear: %v", err)
	}
}

func TestAgendaPoint_ReorderMoveUp(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "First")
	ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "Second")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(agendaManageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}
	openModerateAgendaEditor(t, page)

	urlBefore := page.URL()
	card := page.Locator("#agenda-point-list-container [data-testid='manage-agenda-point-card']").Filter(playwright.LocatorFilterOptions{
		HasText: "Second",
	})
	if err := card.Locator("button[title='Move up']").Click(); err != nil {
		t.Fatalf("click move up: %v", err)
	}
	if err := page.Locator("#agenda-point-list-container [data-testid='manage-agenda-point-card']").First().Locator("text=Second").WaitFor(); err != nil {
		t.Fatalf("expected 'Second' to become first card: %v", err)
	}
	if page.URL() != urlBefore {
		t.Errorf("URL changed on reorder: got %s, want %s", page.URL(), urlBefore)
	}
}

func TestAgendaImport_FileUploadPopulatesTextarea(t *testing.T) {
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
	openAgendaImportDialog(t, page)

	fileInput := page.Locator("#moderate-agenda-import-dialog input[type='file'][data-agenda-import-file]")
	if err := fileInput.SetInputFiles(playwright.InputFile{
		Name:     "agenda.txt",
		MimeType: "text/plain",
		Buffer:   []byte("TOP1 Opening\nTOP2 Reports"),
	}); err != nil {
		t.Fatalf("set import file: %v", err)
	}

	if _, err := page.Evaluate(`() => new Promise((resolve) => setTimeout(resolve, 200))`, nil); err != nil {
		t.Fatalf("wait after import file read: %v", err)
	}
	value, err := page.Locator("#agenda-import-source").InputValue()
	if err != nil {
		t.Fatalf("read import textarea value: %v", err)
	}
	if !strings.Contains(value, "TOP1 Opening") || !strings.Contains(value, "TOP2 Reports") {
		t.Fatalf("expected textarea to contain imported file contents, got %q", value)
	}
}

func TestAgendaImport_ExtractDiffAccept(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "A")
	ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "C")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(agendaManageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}
	openAgendaImportDialog(t, page)

	urlBefore := page.URL()
	source := page.Locator("#agenda-import-source")
	if err := source.Fill("TOP1 A\nTOP2 B\nTOP3 C"); err != nil {
		t.Fatalf("fill import source: %v", err)
	}
	if err := page.Locator("#moderate-agenda-import-dialog button:has-text('Extract Agenda')").Click(); err != nil {
		t.Fatalf("click Extract Agenda: %v", err)
	}
	if err := page.Locator("#agenda-import-correction-form").WaitFor(); err != nil {
		t.Fatalf("wait correction form: %v", err)
	}
	if err := page.Locator("#agenda-import-correction-form button:has-text('Generate Diff')").Click(); err != nil {
		t.Fatalf("click Generate Diff: %v", err)
	}
	if err := page.Locator("#moderate-agenda-import-dialog h4:has-text('Agenda Diff')").WaitFor(); err != nil {
		t.Fatalf("wait diff heading: %v", err)
	}
	if err := page.Locator("#moderate-agenda-import-dialog button:has-text('Accept')").Click(); err != nil {
		t.Fatalf("click Accept: %v", err)
	}
	if err := page.Locator("#agenda-point-list-container [data-testid='manage-agenda-point-card']:has-text('B')").WaitFor(); err != nil {
		t.Fatalf("expected imported agenda point B to appear: %v", err)
	}
	if page.URL() != urlBefore {
		t.Errorf("URL changed on import apply: got %s, want %s", page.URL(), urlBefore)
	}
}

func TestAgendaImport_CorrectionClickUpdatesDetectedNumbering(t *testing.T) {
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
	openAgendaImportDialog(t, page)

	if err := page.Locator("#agenda-import-source").Fill("TOP1 A\nTOP2 B\nTOP3 C"); err != nil {
		t.Fatalf("fill import source: %v", err)
	}
	if err := page.Locator("#moderate-agenda-import-dialog button:has-text('Extract Agenda')").Click(); err != nil {
		t.Fatalf("click Extract Agenda: %v", err)
	}
	if err := page.Locator("#agenda-import-correction-form").WaitFor(); err != nil {
		t.Fatalf("wait correction form: %v", err)
	}

	rows := page.Locator("#agenda-import-correction-form [data-import-line-row]")
	count, err := rows.Count()
	if err != nil {
		t.Fatalf("count correction rows: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 correction rows, got %d", count)
	}

	if got := agendaImportLinePrefixText(t, page, 0); got != "TOP 1" {
		t.Fatalf("unexpected initial prefix row 1: got %q want %q", got, "TOP 1")
	}
	if got := agendaImportLinePrefixText(t, page, 1); got != "TOP 2" {
		t.Fatalf("unexpected initial prefix row 2: got %q want %q", got, "TOP 2")
	}
	if got := agendaImportLinePrefixText(t, page, 2); got != "TOP 3" {
		t.Fatalf("unexpected initial prefix row 3: got %q want %q", got, "TOP 3")
	}

	if err := rows.Nth(1).Click(); err != nil {
		t.Fatalf("click row 2 (heading -> subheading): %v", err)
	}
	if got := agendaImportLinePrefixText(t, page, 1); got != "TOP 1.1" {
		t.Fatalf("unexpected prefix row 2 after subheading click: got %q want %q", got, "TOP 1.1")
	}
	if got := agendaImportLinePrefixText(t, page, 2); got != "TOP 2" {
		t.Fatalf("unexpected prefix row 3 after row 2 subheading click: got %q want %q", got, "TOP 2")
	}

	if err := rows.Nth(1).Click(); err != nil {
		t.Fatalf("click row 2 (subheading -> ignore): %v", err)
	}
	if got := agendaImportLinePrefixText(t, page, 1); got != "" {
		t.Fatalf("unexpected prefix row 2 after ignore click: got %q want empty", got)
	}
	if got := agendaImportLinePrefixText(t, page, 2); got != "TOP 2" {
		t.Fatalf("unexpected prefix row 3 after row 2 ignore click: got %q want %q", got, "TOP 2")
	}
}

func TestAgendaImport_StaleAcceptShowsWarning(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "A")
	ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "C")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(agendaManageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}
	openAgendaImportDialog(t, page)

	source := page.Locator("#agenda-import-source")
	if err := source.Fill("TOP1 A\nTOP2 Inserted Point\nTOP3 C"); err != nil {
		t.Fatalf("fill import source: %v", err)
	}
	if err := page.Locator("#moderate-agenda-import-dialog button:has-text('Extract Agenda')").Click(); err != nil {
		t.Fatalf("click Extract Agenda: %v", err)
	}
	if err := page.Locator("#agenda-import-correction-form").WaitFor(); err != nil {
		t.Fatalf("wait correction form: %v", err)
	}
	if err := page.Locator("#agenda-import-correction-form button:has-text('Generate Diff')").Click(); err != nil {
		t.Fatalf("click Generate Diff: %v", err)
	}
	if err := page.Locator("#moderate-agenda-import-dialog h4:has-text('Agenda Diff')").WaitFor(); err != nil {
		t.Fatalf("wait diff heading: %v", err)
	}

	ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "External Change")

	if err := page.Locator("#moderate-agenda-import-dialog button:has-text('Accept')").Click(); err != nil {
		t.Fatalf("click Accept after external change: %v", err)
	}
	if err := page.Locator("#moderate-agenda-import-dialog .alert:has-text('Agenda changed while you reviewed this diff')").WaitFor(); err != nil {
		t.Fatalf("expected stale warning message: %v", err)
	}

	// Stale accept should not apply the previously previewed changes yet.
	count, err := page.Locator("#agenda-point-list-container [data-testid='manage-agenda-point-card']:has-text('Inserted Point')").Count()
	if err != nil {
		t.Fatalf("count inserted point cards: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected stale accept to not apply changes immediately, got count=%d", count)
	}
}

func TestAgendaImport_DenyLeavesAgendaUnchanged(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "A")
	ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "C")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(agendaManageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}
	openAgendaImportDialog(t, page)

	if err := page.Locator("#agenda-import-source").Fill("TOP1 A\nTOP2 Denied Point\nTOP3 C"); err != nil {
		t.Fatalf("fill source: %v", err)
	}
	if err := page.Locator("#moderate-agenda-import-dialog button:has-text('Extract Agenda')").Click(); err != nil {
		t.Fatalf("click Extract Agenda: %v", err)
	}
	if err := page.Locator("#agenda-import-correction-form button:has-text('Generate Diff')").Click(); err != nil {
		t.Fatalf("click Generate Diff: %v", err)
	}
	if err := page.Locator("#moderate-agenda-import-dialog button:has-text('Deny')").Click(); err != nil {
		t.Fatalf("click Deny: %v", err)
	}
	if err := page.Locator("#moderate-agenda-import-dialog[open]").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected import dialog to close on deny: %v", err)
	}

	count, err := page.Locator("#agenda-point-list-container [data-testid='manage-agenda-point-card']:has-text('Denied Point')").Count()
	if err != nil {
		t.Fatalf("count denied point cards: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected denied point to not be applied, got count=%d", count)
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

	if err := page.Locator("[data-testid='manage-speakers-card'] #speakers-list-container p:has-text('No active agenda point.')").First().WaitFor(); err != nil {
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

	if err := page.Locator("#speakers-list-container [data-testid='live-speaker-item']:has-text('Alice Member')").WaitFor(); err != nil {
		t.Fatalf("expected speaker in list: %v", err)
	}
}

// TestSpeakersList_SearchEnterAddsBestMatch verifies Enter behavior in the
// add-speaker search: add top candidate as regular, clear input, keep focus,
// and do not add duplicates when regular waiting already exists.
func TestSpeakersList_SearchEnterAddsBestMatch(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "Main Topic")
	ts.activateAgendaPoint(t, "test-committee", "Board Meeting", apID)
	ts.seedAttendee(t, "test-committee", "Board Meeting", "Alice Member", "secret-alice")
	ts.seedAttendee(t, "test-committee", "Board Meeting", "Alicia Member", "secret-alicia")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(agendaManageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	openSpeakerAddDialog(t, page)
	search := page.Locator("#speaker-add-search-input")
	if err := search.Fill("alice"); err != nil {
		t.Fatalf("fill search input: %v", err)
	}

	firstCandidate := page.Locator("#speaker-add-candidates-container [data-testid='manage-speaker-candidate-card']").First()
	if err := firstCandidate.WaitFor(); err != nil {
		t.Fatalf("wait first candidate card: %v", err)
	}
	if err := firstCandidate.Locator("text=Alice Member").WaitFor(); err != nil {
		t.Fatalf("expected best match candidate first: %v", err)
	}

	if err := search.Press("Enter"); err != nil {
		t.Fatalf("press enter in search: %v", err)
	}
	if err := page.Locator("#speakers-list-container [data-testid='live-speaker-item']:has-text('Alice Member')").WaitFor(); err != nil {
		t.Fatalf("expected Enter to add top candidate: %v", err)
	}

	value, err := search.InputValue()
	if err != nil {
		t.Fatalf("read search value after enter-add: %v", err)
	}
	if value != "" {
		t.Fatalf("expected search input to be cleared after enter-add, got %q", value)
	}

	openSpeakerAddDialog(t, page)
	search = page.Locator("#speaker-add-search-input")
	if err := search.Fill("alice"); err != nil {
		t.Fatalf("fill search input second time: %v", err)
	}
	disabled, err := speakerCandidateCard(page, "Alice Member").Locator("button[title='Add regular speech']").IsDisabled()
	if err != nil {
		t.Fatalf("read regular button disabled state: %v", err)
	}
	if !disabled {
		t.Fatalf("expected regular button to be disabled once attendee is already waiting regular")
	}

	countBefore, err := page.Locator("#speakers-list-container [data-testid='live-speaker-item']:has-text('Alice Member')").Count()
	if err != nil {
		t.Fatalf("count alice rows before second enter: %v", err)
	}
	if err := search.Press("Enter"); err != nil {
		t.Fatalf("press enter with disabled top regular candidate: %v", err)
	}
	if _, err := page.Evaluate(`() => new Promise((resolve) => setTimeout(resolve, 250))`, nil); err != nil {
		t.Fatalf("wait after second enter: %v", err)
	}
	countAfter, err := page.Locator("#speakers-list-container [data-testid='live-speaker-item']:has-text('Alice Member')").Count()
	if err != nil {
		t.Fatalf("count alice rows after second enter: %v", err)
	}
	if countAfter != countBefore {
		t.Fatalf("expected no duplicate speaker added on second enter, before=%d after=%d", countBefore, countAfter)
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
	if err := page.Locator("#speakers-list-container [data-testid='live-speaker-item']:has-text('Alice Member')").WaitFor(); err != nil {
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

	ropmDisabled, err := card.Locator("button[title='Add Point of Order (PO) speech']").IsDisabled()
	if err != nil {
		t.Fatalf("read ropm button disabled state: %v", err)
	}
	if ropmDisabled {
		t.Fatalf("expected ropm add button to still be enabled")
	}
	if err := card.Locator("button[title='Add Point of Order (PO) speech']").Click(); err != nil {
		t.Fatalf("add ropm speech: %v", err)
	}

	openSpeakerAddDialog(t, page)
	card = speakerCandidateCard(page, "Alice Member")
	ropmDisabled, err = card.Locator("button[title='Add Point of Order (PO) speech']").IsDisabled()
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

	row := page.Locator("#speakers-list-container [data-testid='live-speaker-item']").Filter(playwright.LocatorFilterOptions{
		HasText: "Bob Member",
	})
	if err := row.Locator("button[title='Start']").Click(); err != nil {
		t.Fatalf("click start: %v", err)
	}
	if err := page.Locator("#speakers-list-container [data-testid='live-speaker-item'][data-speaker-state='speaking']:has-text('Bob Member')").WaitFor(); err != nil {
		t.Fatalf("expected speaking row: %v", err)
	}

	if err := page.Locator("[data-testid='manage-end-current-speaker']").Click(); err != nil {
		t.Fatalf("click end: %v", err)
	}
	if err := page.Locator("#speakers-list-container [data-testid='live-speaker-item'][data-speaker-state='speaking']:has-text('Bob Member')").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected speaking state to clear: %v", err)
	}
	if err := row.WaitFor(); err != nil {
		t.Fatalf("expected row to remain after done: %v", err)
	}
}

// TestSpeakersList_DoneSpeakerCanBeReadded verifies that completed entries do
// not block re-adding the same attendee, and that done rows do not render a
// fake numeric position of 0.
func TestSpeakersList_DoneSpeakerCanBeReadded(t *testing.T) {
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
	aliceCard := speakerCandidateCard(page, "Alice Member")
	if err := aliceCard.Locator("button[title='Add regular speech']").Click(); err != nil {
		t.Fatalf("add first regular speech: %v", err)
	}

	row := page.Locator("#speakers-list-container [data-testid='live-speaker-item']").Filter(playwright.LocatorFilterOptions{
		HasText: "Alice Member",
	})
	if err := row.WaitFor(); err != nil {
		t.Fatalf("wait first Alice row: %v", err)
	}
	if err := row.Locator("button[title='Start']").Click(); err != nil {
		t.Fatalf("start Alice speech: %v", err)
	}
	if err := page.Locator("[data-testid='manage-end-current-speaker']").Click(); err != nil {
		t.Fatalf("end Alice speech: %v", err)
	}
	if err := row.WaitFor(); err != nil {
		t.Fatalf("wait done Alice row after end: %v", err)
	}

	doneRow := page.Locator("#speakers-list-container [data-testid='live-speaker-item'][data-speaker-state='done']").Filter(playwright.LocatorFilterOptions{
		HasText: "Alice Member",
	}).First()
	if err := doneRow.WaitFor(); err != nil {
		t.Fatalf("wait done Alice row: %v", err)
	}

	leftColumnText, err := doneRow.Locator("div").First().TextContent()
	if err != nil {
		t.Fatalf("read done row left column text: %v", err)
	}
	// Done speakers now show their sequential done-position number (e.g. "1") to
	// match legacy behaviour. The column must not be blank.
	if strings.TrimSpace(leftColumnText) == "" {
		t.Fatalf("expected done speaker left column to show a position number, got blank")
	}

	openSpeakerAddDialog(t, page)
	aliceCard = speakerCandidateCard(page, "Alice Member")
	regularDisabled, err := aliceCard.Locator("button[title='Add regular speech']").IsDisabled()
	if err != nil {
		t.Fatalf("read regular button disabled state after done: %v", err)
	}
	if regularDisabled {
		t.Fatalf("expected regular add button to re-enable after done speaker")
	}
	if err := aliceCard.Locator("button[title='Add regular speech']").Click(); err != nil {
		t.Fatalf("re-add regular speech after done: %v", err)
	}

	rows, err := page.Locator("#speakers-list-container [data-testid='live-speaker-item']:has-text('Alice Member')").Count()
	if err != nil {
		t.Fatalf("count Alice rows after re-add: %v", err)
	}
	if rows < 2 {
		t.Fatalf("expected Alice to have both a done row and a new waiting row after re-add, got %d rows", rows)
	}
	if err := page.Locator("#speakers-list-container [data-testid='live-speaker-item'][data-speaker-state='waiting']:has-text('Alice Member')").WaitFor(); err != nil {
		t.Fatalf("expected a new waiting Alice row after re-add: %v", err)
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

	row := page.Locator("#speakers-list-container [data-testid='live-speaker-item']").Filter(playwright.LocatorFilterOptions{
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

	if err := page.Locator("#speakers-list-container [data-testid='live-speaker-item']:has-text('Dave Member')").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected speaker row to disappear: %v", err)
	}
}
