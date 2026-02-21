//go:build e2e

package e2e_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	playwright "github.com/playwright-community/playwright-go"
)

func protocolURL(baseURL, slug, meetingID string) string {
	return fmt.Sprintf("%s/committee/%s/meeting/%s/protocol", baseURL, slug, meetingID)
}

// seedProtocolWriter sets the protocol writer for a meeting via the repo.
func (ts *testServer) seedProtocolWriter(t *testing.T, slug, meetingName string, attendeeID int64) {
	t.Helper()
	meetingIDStr := ts.getMeetingID(t, slug, meetingName)
	var mid int64
	fmt.Sscanf(meetingIDStr, "%d", &mid)
	if err := ts.repo.SetProtocolWriter(context.Background(), mid, &attendeeID); err != nil {
		t.Fatalf("seed protocol writer: %v", err)
	}
}

// TestManagePage_AssignProtocolWriter verifies that a chairperson can assign
// a protocol writer via the settings section without a full page reload.
func TestManagePage_AssignProtocolWriter(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	ts.seedAttendee(t, "test-committee", "Board Meeting", "Alice Member", "secret-alice")
	aliceID := ts.getAttendeeIDForMeeting(t, "test-committee", "Board Meeting", "Alice Member")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	urlBefore := page.URL()

	if _, err := page.Locator("#protocol_writer_attendee_id").SelectOption(playwright.SelectOptionValues{
		Labels: playwright.StringSlice("Alice Member"),
	}); err != nil {
		t.Fatalf("select attendee: %v", err)
	}
	waitUntil(t, 3*time.Second, func() (bool, error) {
		val, err := page.Locator("#protocol_writer_attendee_id").InputValue()
		return val == aliceID, err
	}, "protocol writer select to persist selected attendee")
	if page.URL() != urlBefore {
		t.Errorf("URL changed on assign: got %s, want %s", page.URL(), urlBefore)
	}
}

// TestProtocolPage_NotWriter verifies that a non-writer attendee gets an error
// when they attempt to access the protocol page.
func TestProtocolPage_NotWriter(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	ts.seedAttendee(t, "test-committee", "Board Meeting", "Alice Member", "secret-alice")
	// No protocol writer assigned — Alice is not the writer

	page := newPage(t)
	if _, err := page.Goto(attendeeLoginURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto attendee-login: %v", err)
	}
	if err := page.Locator("input[name=secret]").Fill("secret-alice"); err != nil {
		t.Fatalf("fill secret: %v", err)
	}
	if err := page.Locator("input[name=secret]").Press("Enter"); err != nil {
		t.Fatalf("submit attendee login: %v", err)
	}
	if err := page.WaitForURL(liveURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("wait for /live: %v", err)
	}

	resp, err := page.Goto(protocolURL(ts.URL, "test-committee", meetingID))
	if err != nil {
		t.Fatalf("goto protocol page: %v", err)
	}
	if resp.Status() != 500 {
		t.Errorf("expected 500 for non-writer, got %d", resp.Status())
	}
}

// TestProtocolPage_WriterCanAccessAndSave verifies that the assigned protocol
// writer can view the page and save text for an agenda point via HTMX.
func TestProtocolPage_WriterCanAccessAndSave(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	attendee := ts.seedAttendee(t, "test-committee", "Board Meeting", "Bob Writer", "secret-bob")
	ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "Item One")
	ts.seedProtocolWriter(t, "test-committee", "Board Meeting", attendee.ID)

	page := newPage(t)
	if _, err := page.Goto(attendeeLoginURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto attendee-login: %v", err)
	}
	if err := page.Locator("input[name=secret]").Fill("secret-bob"); err != nil {
		t.Fatalf("fill secret: %v", err)
	}
	if err := page.Locator("input[name=secret]").Press("Enter"); err != nil {
		t.Fatalf("submit attendee login: %v", err)
	}
	if err := page.WaitForURL(liveURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("wait for /live: %v", err)
	}

	if _, err := page.Goto(protocolURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto protocol page: %v", err)
	}

	if err := page.Locator("h2:has-text('Item One')").WaitFor(); err != nil {
		t.Fatalf("expected agenda point title on protocol page: %v", err)
	}

	urlBefore := page.URL()

	if err := page.Locator("textarea[name=protocol]").First().Fill("Discussion notes here."); err != nil {
		t.Fatalf("fill protocol textarea: %v", err)
	}
	if err := page.Locator("button:has-text('Save')").First().Click(); err != nil {
		t.Fatalf("click save: %v", err)
	}

	if err := page.Locator("p:has-text('Saved.')").First().WaitFor(); err != nil {
		t.Fatalf("expected saved confirmation: %v", err)
	}
	if page.URL() != urlBefore {
		t.Errorf("URL changed on save: got %s, want %s", page.URL(), urlBefore)
	}
}
