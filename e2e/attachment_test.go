//go:build e2e

package e2e_test

import (
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

// TestAttachments_NoAgendaPoints_NotShownOnManage verifies attachments are no
// longer rendered on the main manage page.
func TestAttachments_NoAgendaPoints_ShowsPlaceholder(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Spring Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Spring Meeting")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")

	if _, err := page.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	if err := page.Locator("h2:has-text('Attachments')").WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(1500),
	}); err == nil {
		t.Fatalf("did not expect Attachments section on main manage page")
	}
}

// TestAttachments_ShowsFormPerAgendaPoint verifies that the Attachments section
// renders one upload form per agenda point.
func TestAttachments_ShowsFormPerAgendaPoint(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Spring Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Spring Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Spring Meeting", "Budget")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")

	if _, err := page.Goto(agendaPointToolsURL(ts.URL, "test-committee", meetingID, apID)); err != nil {
		t.Fatalf("goto agenda-point tools page: %v", err)
	}

	if err := page.Locator("h4:has-text('Budget - Attachments')").WaitFor(); err != nil {
		t.Fatalf("expected attachment list heading for agenda point: %v", err)
	}

	if err := page.Locator("button:has-text('Upload')").WaitFor(); err != nil {
		t.Fatalf("expected Upload button: %v", err)
	}
}

// TestAttachments_UploadAttachment_AppearsInList verifies that uploading an attachment
// adds it to the list via HTMX without a full page reload.
func TestAttachments_UploadAttachment_AppearsInList(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Spring Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Spring Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Spring Meeting", "Budget")

	pdfPath := writeTempPDF(t)

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")

	if _, err := page.Goto(agendaPointToolsURL(ts.URL, "test-committee", meetingID, apID)); err != nil {
		t.Fatalf("goto agenda-point tools page: %v", err)
	}

	if err := page.Locator("h4:has-text('Budget - Attachments')").WaitFor(); err != nil {
		t.Fatalf("attachment section not loaded: %v", err)
	}

	urlBefore := page.URL()

	labelInput := page.Locator("#attachment-label-" + apID)
	if err := labelInput.Fill("Budget Proposal"); err != nil {
		t.Fatalf("fill attachment label: %v", err)
	}

	fileInput := page.Locator("#attachment-file-" + apID)
	if err := fileInput.SetInputFiles(playwright.InputFile{Name: "budget.pdf", MimeType: "application/pdf", Buffer: mustReadFile(t, pdfPath)}); err != nil {
		t.Fatalf("set input file: %v", err)
	}

	if err := page.Locator("button:has-text('Upload')").Click(); err != nil {
		t.Fatalf("click Upload: %v", err)
	}

	if err := page.Locator("a:has-text('Budget Proposal')").WaitFor(); err != nil {
		t.Fatalf("expected attachment link with label to appear: %v", err)
	}

	if page.URL() != urlBefore {
		t.Errorf("HTMX swap caused unexpected navigation: before=%s after=%s", urlBefore, page.URL())
	}
}

// TestAttachments_UploadWithoutLabel_ShowsFilename verifies that when no label is given,
// the attachment link shows the filename.
func TestAttachments_UploadWithoutLabel_ShowsFilename(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Spring Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Spring Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Spring Meeting", "Budget")

	pdfPath := writeTempPDF(t)

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")

	if _, err := page.Goto(agendaPointToolsURL(ts.URL, "test-committee", meetingID, apID)); err != nil {
		t.Fatalf("goto agenda-point tools page: %v", err)
	}

	if err := page.Locator("h4:has-text('Budget - Attachments')").WaitFor(); err != nil {
		t.Fatalf("attachment section not loaded: %v", err)
	}

	// Leave label empty — only upload file
	fileInput := page.Locator("#attachment-file-" + apID)
	if err := fileInput.SetInputFiles(playwright.InputFile{Name: "report.pdf", MimeType: "application/pdf", Buffer: mustReadFile(t, pdfPath)}); err != nil {
		t.Fatalf("set input file: %v", err)
	}

	if err := page.Locator("button:has-text('Upload')").Click(); err != nil {
		t.Fatalf("click Upload: %v", err)
	}

	if err := page.Locator("a:has-text('report.pdf')").WaitFor(); err != nil {
		t.Fatalf("expected attachment link showing filename when no label: %v", err)
	}
}

// TestAttachments_DeleteAttachment_RemovesFromList verifies that deleting an attachment
// removes it from the list via HTMX without a full page reload.
func TestAttachments_DeleteAttachment_RemovesFromList(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Spring Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Spring Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Spring Meeting", "Budget")
	label := "Budget Proposal"
	attachmentID := ts.seedAttachment(t, apID, &label)

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")

	if _, err := page.Goto(agendaPointToolsURL(ts.URL, "test-committee", meetingID, apID)); err != nil {
		t.Fatalf("goto agenda-point tools page: %v", err)
	}

	if err := page.Locator("a:has-text('Budget Proposal')").WaitFor(); err != nil {
		t.Fatalf("attachment not visible before delete: %v", err)
	}

	urlBefore := page.URL()

	page.OnDialog(func(d playwright.Dialog) { _ = d.Accept() })

	attachmentContainer := page.Locator("#attachment-item-" + attachmentID)
	if err := attachmentContainer.Locator("button:has-text('Delete')").Click(); err != nil {
		t.Fatalf("click Delete: %v", err)
	}

	if err := page.Locator("a:has-text('Budget Proposal')").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected attachment to be removed after delete: %v", err)
	}

	if page.URL() != urlBefore {
		t.Errorf("HTMX swap caused unexpected navigation: before=%s after=%s", urlBefore, page.URL())
	}
}
