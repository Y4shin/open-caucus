//go:build e2e

package e2e_test

import (
	"testing"
	"time"

	playwright "github.com/playwright-community/playwright-go"
)

// TestCurrentDoc_SetAndClearAttachment_UpdatesLivePreview verifies that setting
// and clearing the current attachment on the tools page updates attendee live
// preview via SSE, without full-page navigation.
func TestCurrentDoc_SetAndClearAttachment_UpdatesLivePreview(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Live Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Live Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Live Meeting", "Budget")
	ts.activateAgendaPoint(t, "test-committee", "Live Meeting", apID)
	label := "Budget Attachment"
	attachmentID := ts.seedAttachment(t, apID, &label)
	ts.seedAttendee(t, "test-committee", "Live Meeting", "Alice Guest", "secret-alice-current-doc")

	livePage := newPage(t)
	attendeeLoginHelper(t, livePage, ts.URL, "test-committee", meetingID, "secret-alice-current-doc")
	waitUntil(t, 3*time.Second, func() (bool, error) {
		count, err := livePage.Locator("#live-current-doc").Count()
		return count == 0, err
	}, "no current-doc pane before set-current")
	time.Sleep(800 * time.Millisecond)

	toolsPage := newPage(t)
	userLogin(t, toolsPage, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := toolsPage.Goto(agendaPointToolsURL(ts.URL, "test-committee", meetingID, apID)); err != nil {
		t.Fatalf("goto agenda-point tools page: %v", err)
	}
	toolsURLBefore := toolsPage.URL()

	attachmentRow := toolsPage.Locator("#attachment-item-" + attachmentID)
	if err := attachmentRow.WaitFor(); err != nil {
		t.Fatalf("expected attachment row before set-current: %v", err)
	}
	if err := attachmentRow.Locator("button:has-text('Set Current')").Click(); err != nil {
		t.Fatalf("set current attachment: %v", err)
	}
	if err := attachmentRow.Locator("button:has-text('Clear')").WaitFor(); err != nil {
		t.Fatalf("expected clear button after setting current attachment: %v", err)
	}
	if err := livePage.Locator("[data-testid='live-doc-open-desktop']").First().WaitFor(); err != nil {
		t.Fatalf("expected desktop open-document button after set-current: %v", err)
	}
	if err := livePage.Locator("[data-testid='live-doc-download-desktop']").First().WaitFor(); err != nil {
		t.Fatalf("expected desktop download-document button after set-current: %v", err)
	}
	if toolsPage.URL() != toolsURLBefore {
		t.Errorf("tools page URL changed after set-current: before=%s after=%s", toolsURLBefore, toolsPage.URL())
	}

	if err := attachmentRow.Locator("button:has-text('Clear')").Click(); err != nil {
		t.Fatalf("clear current attachment: %v", err)
	}
	if err := attachmentRow.Locator("button:has-text('Set Current')").WaitFor(); err != nil {
		t.Fatalf("expected set-current button after clear: %v", err)
	}
	if err := livePage.Locator("[data-testid='live-doc-open-desktop']").First().WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected desktop document action buttons to be removed after clearing current doc: %v", err)
	}
	if toolsPage.URL() != toolsURLBefore {
		t.Errorf("tools page URL changed after clear-current: before=%s after=%s", toolsURLBefore, toolsPage.URL())
	}
}
