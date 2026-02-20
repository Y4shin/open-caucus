//go:build e2e

package e2e_test

import (
	"fmt"
	"strings"
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

// TestManagePage_SelfSignup verifies that the chairperson can sign themselves
// up as attendee from the manage page without a full page reload.
func TestManagePage_SelfSignup(t *testing.T) {
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

	if err := page.Locator("button:has-text('Sign yourself up')").Click(); err != nil {
		t.Fatalf("click sign yourself up: %v", err)
	}

	if err := page.Locator("#attendee-list-container td:has-text('Chair Person')").WaitFor(); err != nil {
		t.Fatalf("expected chairperson in attendee table: %v", err)
	}
	if page.URL() != urlBefore {
		t.Errorf("URL changed on self-signup: got %s, want %s", page.URL(), urlBefore)
	}
}

// TestManagePage_SelfSignup_ThenNavigateToLive verifies that after adding
// themselves as attendee on the manage page, a chairperson can navigate
// directly to /live using their user session and gets auto-authenticated.
func TestManagePage_SelfSignup_ThenNavigateToLive(t *testing.T) {
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

	if err := page.Locator("button:has-text('Sign yourself up')").Click(); err != nil {
		t.Fatalf("click sign yourself up: %v", err)
	}
	if err := page.Locator("#attendee-list-container td:has-text('Chair Person')").WaitFor(); err != nil {
		t.Fatalf("expected chairperson in attendee table: %v", err)
	}

	// Directly open /live with the existing user session.
	if _, err := page.Goto(liveURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto live page: %v", err)
	}
	if err := page.WaitForURL(liveURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("expected /live to load with auto-authenticated attendee session: %v", err)
	}
	if err := page.Locator("p:has-text('Chair Person')").WaitFor(); err != nil {
		t.Fatalf("expected attendee name on live page: %v", err)
	}
	if err := page.Locator("a:has-text('Manage')").WaitFor(); err != nil {
		t.Fatalf("expected manage button for chairperson user on live page: %v", err)
	}
}

// TestManagePage_SelfSignup_ThenAssignSelfModerator verifies that after the
// chairperson signs themselves up as attendee, they can assign themselves as
// meeting moderator via the settings partial.
func TestManagePage_SelfSignup_ThenAssignSelfModerator(t *testing.T) {
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

	if err := page.Locator("button:has-text('Sign yourself up')").Click(); err != nil {
		t.Fatalf("click sign yourself up: %v", err)
	}
	if err := page.Locator("#attendee-list-container td:has-text('Chair Person')").WaitFor(); err != nil {
		t.Fatalf("expected chairperson in attendee table: %v", err)
	}
	if err := page.Locator("#meeting-settings-container").WaitFor(); err != nil {
		t.Fatalf("meeting settings not visible after signup: %v", err)
	}

	if _, err := page.Locator("#meeting_moderator_attendee_id").SelectOption(playwright.SelectOptionValues{
		Labels: playwright.StringSlice("Chair Person"),
	}); err != nil {
		t.Fatalf("select self as moderator: %v", err)
	}
	if err := page.Locator("#meeting-settings-container button:has-text('Set Moderator')").Click(); err != nil {
		t.Fatalf("click set moderator: %v", err)
	}

	if err := page.Locator("#meeting-settings-container p:has-text('Moderator:') strong:has-text('Chair Person')").WaitFor(); err != nil {
		t.Fatalf("expected self as moderator: %v", err)
	}
	if page.URL() != urlBefore {
		t.Errorf("URL changed on self-signup/moderator assignment: got %s, want %s", page.URL(), urlBefore)
	}
}

// TestManagePage_SelfSignup_AssignSelfModerator_Reassign verifies that self
// moderator assignment can be cleared and set again after self-signup.
func TestManagePage_SelfSignup_AssignSelfModerator_Reassign(t *testing.T) {
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

	if err := page.Locator("button:has-text('Sign yourself up')").Click(); err != nil {
		t.Fatalf("click sign yourself up: %v", err)
	}
	if err := page.Locator("#attendee-list-container td:has-text('Chair Person')").WaitFor(); err != nil {
		t.Fatalf("expected chairperson in attendee table: %v", err)
	}
	if err := page.Locator("#meeting-settings-container").WaitFor(); err != nil {
		t.Fatalf("meeting settings not visible after signup: %v", err)
	}

	// First assignment: self.
	if _, err := page.Locator("#meeting_moderator_attendee_id").SelectOption(playwright.SelectOptionValues{
		Labels: playwright.StringSlice("Chair Person"),
	}); err != nil {
		t.Fatalf("select self as moderator: %v", err)
	}
	if err := page.Locator("#meeting-settings-container button:has-text('Set Moderator')").Click(); err != nil {
		t.Fatalf("click set moderator: %v", err)
	}
	if err := page.Locator("#meeting-settings-container p:has-text('Moderator:') strong:has-text('Chair Person')").WaitFor(); err != nil {
		t.Fatalf("expected self as moderator: %v", err)
	}

	// Clear moderator.
	if _, err := page.Locator("#meeting_moderator_attendee_id").SelectOption(playwright.SelectOptionValues{
		Values: playwright.StringSlice(""),
	}); err != nil {
		t.Fatalf("select none for moderator: %v", err)
	}
	if err := page.Locator("#meeting-settings-container button:has-text('Set Moderator')").Click(); err != nil {
		t.Fatalf("click set moderator to clear: %v", err)
	}
	if err := page.Locator("#meeting-settings-container p:has-text('Moderator:') strong:has-text('None')").WaitFor(); err != nil {
		t.Fatalf("expected moderator to be None after clear: %v", err)
	}

	// Re-assign self.
	if _, err := page.Locator("#meeting_moderator_attendee_id").SelectOption(playwright.SelectOptionValues{
		Labels: playwright.StringSlice("Chair Person"),
	}); err != nil {
		t.Fatalf("re-select self as moderator: %v", err)
	}
	if err := page.Locator("#meeting-settings-container button:has-text('Set Moderator')").Click(); err != nil {
		t.Fatalf("click set moderator (reassign): %v", err)
	}
	if err := page.Locator("#meeting-settings-container p:has-text('Moderator:') strong:has-text('Chair Person')").WaitFor(); err != nil {
		t.Fatalf("expected self as moderator after reassign: %v", err)
	}
}

// TestManagePage_CrossTab_AttendeeChangePropagates verifies that attendee
// changes in one tab are reflected in another tab of the same meeting.
func TestManagePage_CrossTab_AttendeeChangePropagates(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")

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

	if err := pageA.Locator("button:has-text('Sign yourself up')").Click(); err != nil {
		t.Fatalf("click sign yourself up in A: %v", err)
	}

	if err := pageB.Locator("#attendee-list-container td:has-text('Chair Person')").WaitFor(); err != nil {
		t.Fatalf("expected attendee propagated to B: %v", err)
	}
	if err := pageB.Locator("#meeting-settings-container").WaitFor(); err != nil {
		t.Fatalf("meeting settings missing on B: %v", err)
	}
	if err := pageB.Locator("#speakers-list-container").WaitFor(); err != nil {
		t.Fatalf("speakers list missing on B: %v", err)
	}
}

// TestManagePage_NoSelfEcho_PerTab verifies that the initiating tab does not
// end up with duplicate attendee rows after self-signup.
func TestManagePage_NoSelfEcho_PerTab(t *testing.T) {
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

	if err := page.Locator("button:has-text('Sign yourself up')").Click(); err != nil {
		t.Fatalf("click sign yourself up: %v", err)
	}
	if err := page.Locator("#attendee-list-container td:has-text('Chair Person')").WaitFor(); err != nil {
		t.Fatalf("expected chairperson in attendee table: %v", err)
	}

	rowCount, err := page.Locator("#attendee-list-container td:has-text('Chair Person')").Count()
	if err != nil {
		t.Fatalf("count chairperson attendee rows: %v", err)
	}
	if rowCount != 1 {
		t.Fatalf("expected exactly 1 chairperson row, got %d", rowCount)
	}
}

// TestManagePage_MeetingIsolation_SSE verifies attendee-change events are scoped
// to their meeting and do not update other meetings.
func TestManagePage_MeetingIsolation_SSE(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting A", "")
	ts.seedMeeting(t, "test-committee", "Board Meeting B", "")
	meetingA := ts.getMeetingID(t, "test-committee", "Board Meeting A")
	meetingB := ts.getMeetingID(t, "test-committee", "Board Meeting B")

	pageA := newPage(t)
	userLogin(t, pageA, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := pageA.Goto(manageURL(ts.URL, "test-committee", meetingA)); err != nil {
		t.Fatalf("goto manage page A: %v", err)
	}

	pageB := newPage(t)
	userLogin(t, pageB, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := pageB.Goto(manageURL(ts.URL, "test-committee", meetingB)); err != nil {
		t.Fatalf("goto manage page B: %v", err)
	}

	if err := pageA.Locator("button:has-text('Sign yourself up')").Click(); err != nil {
		t.Fatalf("click sign yourself up in A: %v", err)
	}
	if err := pageA.Locator("#attendee-list-container td:has-text('Chair Person')").WaitFor(); err != nil {
		t.Fatalf("expected attendee in A: %v", err)
	}

	if err := pageB.Locator("#attendee-list-container td:has-text('Chair Person')").WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(1500),
	}); err == nil {
		t.Fatalf("unexpected attendee propagation to different meeting")
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

// TestManagePage_GuestRecoveryLink verifies that each guest row provides a
// recovery-link page with a direct login URL and QR code.
func TestManagePage_GuestRecoveryLink(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	ts.seedAttendee(t, "test-committee", "Board Meeting", "Recoverable Guest", "secret-recoverable")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	row := page.Locator("tr:has(td:has-text('Recoverable Guest'))")
	if err := row.WaitFor(); err != nil {
		t.Fatalf("expected guest attendee row: %v", err)
	}
	if err := row.Locator("a:has-text('Recovery Link')").Click(); err != nil {
		t.Fatalf("click recovery link button: %v", err)
	}

	if err := page.Locator("#attendee-recovery-link").WaitFor(); err != nil {
		t.Fatalf("expected attendee recovery link on page: %v", err)
	}
	if err := page.Locator("#attendee-recovery-qr").WaitFor(); err != nil {
		t.Fatalf("expected attendee recovery QR on page: %v", err)
	}

	href, err := page.Locator("#attendee-recovery-link").GetAttribute("href")
	if err != nil {
		t.Fatalf("get attendee recovery href: %v", err)
	}
	if !strings.Contains(href, "/attendee-login?secret=secret-recoverable") {
		t.Fatalf("expected recovery href to contain attendee-login secret, got %q", href)
	}

	guestPage := newPage(t)
	if _, err := guestPage.Goto(href); err != nil {
		t.Fatalf("goto recovery href as guest: %v", err)
	}
	if err := guestPage.WaitForURL(liveURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("expected direct redirect to /live from recovery link: %v", err)
	}
	if err := guestPage.Locator("p:has-text('Recoverable Guest')").WaitFor(); err != nil {
		t.Fatalf("expected guest name on live page: %v", err)
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
