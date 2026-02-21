//go:build e2e

package e2e_test

import (
	"strings"
	"testing"
	"time"

	playwright "github.com/playwright-community/playwright-go"
)

func manageURL(baseURL, slug, meetingID string) string {
	return baseURL + "/committee/" + slug + "/meeting/" + meetingID + "/manage"
}

func manageAttendeeCard(page playwright.Page, fullName string) playwright.Locator {
	return page.Locator("#attendee-list-container .manage-attendee-card").Filter(playwright.LocatorFilterOptions{
		HasText: fullName,
	})
}

func submitAddGuest(t *testing.T, page playwright.Page, fullName string) {
	t.Helper()
	form := page.Locator("#attendee-list-container .manage-attendee-add-guest-form")
	if err := form.Locator("input[name=full_name]").Fill(fullName); err != nil {
		t.Fatalf("fill guest name: %v", err)
	}
	if _, err := form.Evaluate("f => { f.requestSubmit(); return true; }", nil); err != nil {
		t.Fatalf("submit add-guest form: %v", err)
	}
}

func submitSelfSignup(t *testing.T, page playwright.Page) bool {
	t.Helper()
	form := page.Locator("#attendee-list-container form[hx-post*='/attendee/self-signup']")
	count, err := form.Count()
	if err != nil {
		t.Fatalf("count self-signup forms: %v", err)
	}
	if count == 0 {
		return false
	}
	if _, err := form.First().Evaluate("f => { f.requestSubmit(); return true; }", nil); err != nil {
		t.Fatalf("submit self-signup form: %v", err)
	}
	return true
}

func waitUntil(t *testing.T, timeout time.Duration, fn func() (bool, error), description string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ok, err := fn()
		if err == nil && ok {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", description)
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
	if err := manageAttendeeCard(page, "Alice Member").WaitFor(); err != nil {
		t.Fatalf("expected attendee card for Alice Member: %v", err)
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
	submitAddGuest(t, page, "Bob Guest")
	if err := manageAttendeeCard(page, "Bob Guest").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateAttached,
	}); err != nil {
		t.Fatalf("expected new guest attendee card: %v", err)
	}
	if page.URL() != urlBefore {
		t.Errorf("URL changed on add guest: got %s, want %s", page.URL(), urlBefore)
	}
}

// TestManagePage_CrossTab_AttendeeChangePropagates verifies attendee
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

	if !submitSelfSignup(t, pageA) {
		submitAddGuest(t, pageA, "CrossTab Guest")
	}
	if err := pageB.Locator("#attendee-list-container .manage-attendee-card:has-text('Chair Person'), #attendee-list-container .manage-attendee-card:has-text('CrossTab Guest')").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateAttached,
	}); err != nil {
		t.Fatalf("expected attendee update propagated to B: %v", err)
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

	submitAddGuest(t, pageA, "Isolation Guest")
	if err := manageAttendeeCard(pageA, "Isolation Guest").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateAttached,
	}); err != nil {
		t.Fatalf("expected attendee in meeting A: %v", err)
	}

	if err := manageAttendeeCard(pageB, "Isolation Guest").WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(1500),
	}); err == nil {
		t.Fatalf("unexpected attendee propagation to meeting B")
	}
}

// TestManagePage_RemoveAttendee verifies that clicking Remove deletes the
// attendee card from the list (after confirmation).
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

	card := manageAttendeeCard(page, "Carol Guest")
	if err := card.WaitFor(); err != nil {
		t.Fatalf("attendee not visible before remove: %v", err)
	}

	page.OnDialog(func(d playwright.Dialog) {
		if err := d.Accept(); err != nil {
			t.Logf("accept dialog error: %v", err)
		}
	})

	if err := card.Locator("button[title='Remove attendee']").Click(); err != nil {
		t.Fatalf("click remove attendee: %v", err)
	}
	if err := manageAttendeeCard(page, "Carol Guest").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected attendee card to disappear: %v", err)
	}
}

// TestManagePage_ToggleChair verifies that chair toggling switches button label.
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

	card := manageAttendeeCard(page, "Dave Member")
	if err := card.WaitFor(); err != nil {
		t.Fatalf("dave card not visible: %v", err)
	}

	if err := card.Locator("button[title='Make chair']").Click(); err != nil {
		t.Fatalf("click make chair: %v", err)
	}
	if err := card.Locator("button[title='Demote chair']").WaitFor(); err != nil {
		t.Fatalf("expected demote chair button after promote: %v", err)
	}

	if err := card.Locator("button[title='Demote chair']").Click(); err != nil {
		t.Fatalf("click demote chair: %v", err)
	}
	if err := card.Locator("button[title='Make chair']").WaitFor(); err != nil {
		t.Fatalf("expected make chair button after demote: %v", err)
	}
}

// TestManagePage_GuestRecoveryLink verifies that guest cards provide a
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

	card := manageAttendeeCard(page, "Recoverable Guest")
	if err := card.WaitFor(); err != nil {
		t.Fatalf("expected guest attendee card: %v", err)
	}
	if err := card.Locator("a[title='Recovery link']").Click(); err != nil {
		t.Fatalf("click recovery link: %v", err)
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
	if err := guestPage.Locator("p.scaffold-auth-text:has-text('Recoverable Guest')").WaitFor(); err != nil {
		t.Fatalf("expected guest name on live page: %v", err)
	}
}

// TestManagePage_ToggleSignupOpen verifies that the signup switch changes state
// and controls join-page guest form visibility.
func TestManagePage_ToggleSignupOpen(t *testing.T) {
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

	checked, err := page.Locator("#manage_signup_open").IsChecked()
	if err != nil {
		t.Fatalf("read initial signup state: %v", err)
	}
	if checked {
		t.Fatalf("expected signup to start closed")
	}

	if err := page.Locator("label[for='manage_signup_open']").Click(); err != nil {
		t.Fatalf("toggle signup switch on: %v", err)
	}
	waitUntil(t, 3*time.Second, func() (bool, error) {
		return page.Locator("#manage_signup_open").IsChecked()
	}, "signup switch to become checked")

	guestPage := newPage(t)
	if _, err := guestPage.Goto(joinURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto join page after opening signup: %v", err)
	}
	if err := guestPage.Locator("input[name=full_name]").WaitFor(); err != nil {
		t.Fatalf("expected guest signup form when signup is open: %v", err)
	}
}
