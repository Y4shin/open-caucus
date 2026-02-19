//go:build e2e

package e2e_test

import (
	"fmt"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

func manageURL(baseURL, slug, meetingID string) string {
	return fmt.Sprintf("%s/committee/%s/meeting/%s/manage", baseURL, slug, meetingID)
}

// TestManagePage_ShowsAttendeeList verifies that the manage page shows existing
// attendees after they have been seeded.
func TestManagePage_ShowsAttendeeList(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	ts.seedAttendee(t, "test-committee", "Board Meeting", "Alice Member", "secret-alice")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")

	if _, err := page.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}
	if err := page.Locator("td:has-text('Alice Member')").WaitFor(); err != nil {
		t.Fatalf("expected attendee name in table: %v", err)
	}
}

// TestManagePage_AddGuest verifies that the chairperson can add a guest attendee
// via the inline form and the list updates without a full page reload.
func TestManagePage_AddGuest(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	urlBefore := page.URL()

	if err := page.Locator("input[name=full_name]").Fill("Bob Guest"); err != nil {
		t.Fatalf("fill name: %v", err)
	}
	if err := page.Locator("button:has-text('Add Guest')").Click(); err != nil {
		t.Fatalf("click add guest: %v", err)
	}

	if err := page.Locator("#attendee-list-container td:has-text('Bob Guest')").WaitFor(); err != nil {
		t.Fatalf("expected new guest in table: %v", err)
	}
	if page.URL() != urlBefore {
		t.Errorf("URL changed on add guest: got %s, want %s", page.URL(), urlBefore)
	}
}

// TestManagePage_RemoveAttendee verifies that clicking Remove deletes the
// attendee row from the list (after confirmation).
func TestManagePage_RemoveAttendee(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	ts.seedAttendee(t, "test-committee", "Board Meeting", "Carol Guest", "secret-carol")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	if err := page.Locator("td:has-text('Carol Guest')").WaitFor(); err != nil {
		t.Fatalf("attendee not visible before remove: %v", err)
	}

	urlBefore := page.URL()

	// Register dialog handler immediately before the click, following the HTMX hx-confirm pattern
	page.OnDialog(func(d playwright.Dialog) {
		if err := d.Accept(); err != nil {
			t.Logf("accept dialog error: %v", err)
		}
	})

	if err := page.Locator("button:has-text('Remove')").First().Click(); err != nil {
		t.Fatalf("click remove: %v", err)
	}

	if err := page.Locator("td:has-text('Carol Guest')").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected attendee row to disappear: %v", err)
	}
	if page.URL() != urlBefore {
		t.Errorf("URL changed on remove: got %s, want %s", page.URL(), urlBefore)
	}
}

// TestManagePage_ToggleChair verifies that clicking Make Chair promotes an
// attendee and the button changes to Demote, without a full page reload.
func TestManagePage_ToggleChair(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	ts.seedAttendee(t, "test-committee", "Board Meeting", "Dave Member", "secret-dave")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	urlBefore := page.URL()

	if err := page.Locator("button:has-text('Make Chair')").First().Click(); err != nil {
		t.Fatalf("click make chair: %v", err)
	}

	// After promotion the button should now say Demote
	if err := page.Locator("button:has-text('Demote')").First().WaitFor(); err != nil {
		t.Fatalf("expected Demote button after promotion: %v", err)
	}
	if page.URL() != urlBefore {
		t.Errorf("URL changed on toggle chair: got %s, want %s", page.URL(), urlBefore)
	}
}

// TestManagePage_SignupOpen_ShowsCurrentState verifies that the manage page
// reflects the initial signup_open state of the meeting.
func TestManagePage_SignupOpen_ShowsCurrentState(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeetingOpen(t, "test-committee", "Open Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Open Meeting")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	// Should show "Open" and a "Close Signup" button
	if err := page.Locator("strong:has-text('Open')").WaitFor(); err != nil {
		t.Fatalf("expected signup status 'Open': %v", err)
	}
	if err := page.Locator("button:has-text('Close Signup')").WaitFor(); err != nil {
		t.Fatalf("expected 'Close Signup' button: %v", err)
	}
}

// TestManagePage_ToggleSignupOpen verifies that toggling signup_open switches
// the label and button text without a full page reload.
func TestManagePage_ToggleSignupOpen(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "") // signup closed by default
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	// Initial state: closed
	if err := page.Locator("strong:has-text('Closed')").WaitFor(); err != nil {
		t.Fatalf("expected initial state 'Closed': %v", err)
	}

	urlBefore := page.URL()

	// Open signup
	if err := page.Locator("button:has-text('Open Signup')").Click(); err != nil {
		t.Fatalf("click open signup: %v", err)
	}
	if err := page.Locator("strong:has-text('Open')").WaitFor(); err != nil {
		t.Fatalf("expected state 'Open' after toggle: %v", err)
	}
	if err := page.Locator("button:has-text('Close Signup')").WaitFor(); err != nil {
		t.Fatalf("expected 'Close Signup' button after opening: %v", err)
	}

	// Close signup again
	if err := page.Locator("button:has-text('Close Signup')").Click(); err != nil {
		t.Fatalf("click close signup: %v", err)
	}
	if err := page.Locator("strong:has-text('Closed')").WaitFor(); err != nil {
		t.Fatalf("expected state 'Closed' after second toggle: %v", err)
	}

	if page.URL() != urlBefore {
		t.Errorf("URL changed on toggle: got %s, want %s", page.URL(), urlBefore)
	}
}
