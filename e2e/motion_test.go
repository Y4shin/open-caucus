//go:build e2e

package e2e_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

func agendaPointToolsURL(baseURL, slug, meetingID, agendaPointID string) string {
	return fmt.Sprintf("%s/committee/%s/meeting/%s/agenda-point/%s/tools", baseURL, slug, meetingID, agendaPointID)
}

// writeTempPDF writes a minimal dummy PDF to a temp file and returns its path.
func writeTempPDF(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "motion-*.pdf")
	if err != nil {
		t.Fatalf("create temp pdf: %v", err)
	}
	if _, err := f.WriteString("%PDF-1.0 dummy content"); err != nil {
		t.Fatalf("write temp pdf: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close temp pdf: %v", err)
	}
	abs, err := filepath.Abs(f.Name())
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	return abs
}

// TestMotions_NoAgendaPoints_NotShownOnManage verifies motions are no longer
// rendered on the main manage page.
func TestMotions_NoAgendaPoints_ShowsPlaceholder(t *testing.T) {
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

	if err := page.Locator("h2:has-text('Motions')").WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(1500),
	}); err == nil {
		t.Fatalf("did not expect Motions section on main manage page")
	}
}

// TestMotions_ShowsMotionFormPerAgendaPoint verifies that the Motions section
// renders one upload form per agenda point.
func TestMotions_ShowsMotionFormPerAgendaPoint(t *testing.T) {
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

	if err := page.Locator("h4:has-text('Budget — Motions')").WaitFor(); err != nil {
		t.Fatalf("expected motion list heading for agenda point: %v", err)
	}

	if err := page.Locator("button:has-text('Upload Motion')").WaitFor(); err != nil {
		t.Fatalf("expected Upload Motion button: %v", err)
	}
}

// TestMotions_UploadMotion_AppearsInList verifies that uploading a motion via
// the form adds it to the list via HTMX without a full page reload.
func TestMotions_UploadMotion_AppearsInList(t *testing.T) {
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

	if err := page.Locator("h4:has-text('Budget — Motions')").WaitFor(); err != nil {
		t.Fatalf("motion section not loaded: %v", err)
	}

	urlBefore := page.URL()

	// Use the AP-ID-scoped input IDs to avoid ambiguity with the agenda point form's input[name=title].
	titleInput := page.Locator("#motion-title-" + apID)
	if err := titleInput.Fill("Budget Approval"); err != nil {
		t.Fatalf("fill motion title: %v", err)
	}

	fileInput := page.Locator("#motion-file-" + apID)
	if err := fileInput.SetInputFiles(playwright.InputFile{Name: "budget.pdf", MimeType: "application/pdf", Buffer: mustReadFile(t, pdfPath)}); err != nil {
		t.Fatalf("set input file: %v", err)
	}

	if err := page.Locator("button:has-text('Upload Motion')").Click(); err != nil {
		t.Fatalf("click Upload Motion: %v", err)
	}

	if err := page.Locator("strong:has-text('Budget Approval')").WaitFor(); err != nil {
		t.Fatalf("expected motion title to appear in list: %v", err)
	}

	if page.URL() != urlBefore {
		t.Errorf("HTMX swap caused unexpected navigation: before=%s after=%s", urlBefore, page.URL())
	}
}

// TestMotions_DeleteMotion_RemovesFromList verifies that deleting a motion
// removes it from the list via HTMX without a full page reload.
func TestMotions_DeleteMotion_RemovesFromList(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Spring Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Spring Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Spring Meeting", "Budget")
	motionID := ts.seedMotion(t, apID, "Budget Approval")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")

	if _, err := page.Goto(agendaPointToolsURL(ts.URL, "test-committee", meetingID, apID)); err != nil {
		t.Fatalf("goto agenda-point tools page: %v", err)
	}

	if err := page.Locator("strong:has-text('Budget Approval')").WaitFor(); err != nil {
		t.Fatalf("motion not visible before delete: %v", err)
	}

	urlBefore := page.URL()

	page.OnDialog(func(d playwright.Dialog) { _ = d.Accept() })

	// Scope the Delete click to the specific motion item to avoid hitting the agenda point's delete button.
	motionContainer := page.Locator("#motion-item-" + motionID)
	if err := motionContainer.Locator("button:has-text('Delete')").Click(); err != nil {
		t.Fatalf("click Delete: %v", err)
	}

	if err := page.Locator("strong:has-text('Budget Approval')").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected motion to be removed after delete: %v", err)
	}

	if page.URL() != urlBefore {
		t.Errorf("HTMX swap caused unexpected navigation: before=%s after=%s", urlBefore, page.URL())
	}
}

// mustReadFile reads a file and returns its contents, failing the test on error.
func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}
	return data
}

