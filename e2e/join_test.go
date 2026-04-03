//go:build e2e

package e2e_test

import (
	"fmt"
	"strings"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

func joinURL(baseURL, slug, meetingID string) string {
	return fmt.Sprintf("%s/committee/%s/meeting/%s/join", baseURL, slug, meetingID)
}

func manageJoinQRURL(baseURL, slug, meetingID string) string {
	return fmt.Sprintf("%s/committee/%s/meeting/%s/moderate/join-qr", baseURL, slug, meetingID)
}

func waitForJoinGuestForm(t *testing.T, page playwright.Page) {
	t.Helper()
	if err := page.Locator("input[name=full_name]").WaitFor(); err != nil {
		t.Fatalf("wait for guest signup name field: %v", err)
	}
	if err := page.Locator("input[name=meeting_secret]").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateAttached,
	}); err != nil {
		t.Fatalf("wait for meeting secret field: %v", err)
	}
}

// TestJoinPage_RegisteredUserSeesSignupButton verifies that a logged-in committee
// member sees a sign-up button (not the guest form) on the join page.
func TestJoinPage_RegisteredUserSeesSignupButton(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Alice Member", "member")
	ts.seedMeeting(t, "test-committee", "Spring Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Spring Meeting")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "member1", "pass123")

	if _, err := page.Goto(joinURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto join page: %v", err)
	}

	if err := page.Locator("button:has-text('Sign Up')").WaitFor(); err != nil {
		t.Fatalf("expected sign-up button for logged-in user: %v", err)
	}

	// Guest form must not be shown to logged-in users
	visible, err := page.Locator("input[name=full_name]").IsVisible()
	if err != nil {
		t.Fatalf("check guest form visibility: %v", err)
	}
	if visible {
		t.Error("guest name input should not be visible for a logged-in user")
	}
}

// TestJoinPage_GuestSeesFormWhenSignupOpen verifies that an unauthenticated visitor
// sees the guest name-entry form when signup_open is true.
func TestJoinPage_GuestSeesFormWhenSignupOpen(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeetingOpen(t, "test-committee", "Open Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Open Meeting")

	page := newPage(t)
	if _, err := page.Goto(joinURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto join page: %v", err)
	}

	if err := page.Locator("input[name=full_name]").WaitFor(); err != nil {
		t.Fatalf("expected guest signup form when signup_open: %v", err)
	}
	if err := page.Locator("input[name=meeting_secret]").WaitFor(); err != nil {
		t.Fatalf("expected meeting secret field when no prefilled secret exists: %v", err)
	}
	if err := page.Locator("button:has-text('Sign Up as Guest')").WaitFor(); err != nil {
		t.Fatalf("expected guest sign-up button: %v", err)
	}
}

func TestJoinPage_GuestFormShowsFLINTALabel(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeetingOpen(t, "test-committee", "Open Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Open Meeting")

	page := newPage(t)
	if _, err := page.Goto(joinURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto join page: %v", err)
	}
	waitForJoinGuestForm(t, page)

	label := page.Locator("label[for='guest_gender_quoted']")
	if err := label.WaitFor(); err != nil {
		t.Fatalf("wait for FLINTA label: %v", err)
	}
	text, err := label.TextContent()
	if err != nil {
		t.Fatalf("read FLINTA label text: %v", err)
	}
	if strings.TrimSpace(text) != "FLINTA*" {
		t.Fatalf("expected FLINTA label text %q, got %q", "FLINTA*", strings.TrimSpace(text))
	}
}

// TestJoinPage_GuestSeesClosedMessageWhenSignupClosed verifies that an unauthenticated
// visitor sees a "signup is closed" message when signup_open is false.
func TestJoinPage_GuestSeesClosedMessageWhenSignupClosed(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeeting(t, "test-committee", "Closed Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Closed Meeting")

	page := newPage(t)
	if _, err := page.Goto(joinURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto join page: %v", err)
	}

	if err := page.Locator("p:has-text('closed')").WaitFor(); err != nil {
		t.Fatalf("expected signup closed message: %v", err)
	}

	visible, err := page.Locator("input[name=full_name]").IsVisible()
	if err != nil {
		t.Fatalf("check guest form visibility: %v", err)
	}
	if visible {
		t.Error("guest form must not be visible when signup is closed")
	}
}

// TestGuestSignup_Success verifies that a guest can submit their name and is
// redirected directly to the live page with an attendee session.
func TestGuestSignup_Success(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeetingOpen(t, "test-committee", "Open Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Open Meeting")

	page := newPage(t)
	if _, err := page.Goto(joinURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto join page: %v", err)
	}
	waitForJoinGuestForm(t, page)

	if err := page.Locator("input[name=full_name]").Fill("Bob Guest"); err != nil {
		t.Fatalf("fill name: %v", err)
	}
	if err := page.Locator("input[name=meeting_secret]").Fill("test-meeting-secret"); err != nil {
		t.Fatalf("fill meeting secret: %v", err)
	}
	if err := page.Locator("button:has-text('Sign Up as Guest')").Click(); err != nil {
		t.Fatalf("click submit: %v", err)
	}

	if err := page.WaitForURL(liveURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("expected redirect to /live after guest signup: %v", err)
	}
}

// TestMeetingJoinSubmit_RegisteredUser verifies that a logged-in committee member
// can sign up and is redirected to the live page with an attendee session.
func TestMeetingJoinSubmit_RegisteredUser(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Alice Member", "member")
	ts.seedMeeting(t, "test-committee", "Members Only", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Members Only")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "member1", "pass123")

	if _, err := page.Goto(joinURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto join page: %v", err)
	}

	if err := page.Locator("button:has-text('Sign Up')").Click(); err != nil {
		t.Fatalf("click sign up: %v", err)
	}

	// Should redirect to /live with an attendee session
	if err := page.WaitForURL(liveURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("expected redirect to /live after signup: %v", err)
	}
}

// TestMeetingJoinSubmit_Idempotent verifies that signing up twice does not
// create a duplicate attendee row: a second signup with a fresh user session
// reuses the existing attendee row and still redirects to /live.
func TestMeetingJoinSubmit_Idempotent(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Alice Member", "member")
	ts.seedMeeting(t, "test-committee", "Members Only", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Members Only")

	// First signup: user session → attendee row created → attendee session
	page1 := newPage(t)
	userLogin(t, page1, ts.URL, "test-committee", "member1", "pass123")
	if _, err := page1.Goto(joinURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto join page (first): %v", err)
	}
	if err := page1.Locator("button:has-text('Sign Up')").Click(); err != nil {
		t.Fatalf("click sign up (first): %v", err)
	}
	if err := page1.WaitForURL(liveURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("redirect after first signup: %v", err)
	}

	// Second visit: fresh browser context, log in again to get a user session.
	// The join page shows "already signed up" with an "Enter Meeting" button that
	// creates a new attendee session and redirects to /live.
	page2 := newPage(t)
	userLogin(t, page2, ts.URL, "test-committee", "member1", "pass123")
	if _, err := page2.Goto(joinURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto join page (second): %v", err)
	}
	if err := page2.Locator("button:has-text('Enter Meeting')").Click(); err != nil {
		t.Fatalf("click enter meeting (second): %v", err)
	}
	if err := page2.WaitForURL(liveURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("expected redirect to /live on second visit: %v", err)
	}
}

func TestGuestSignup_InvalidMeetingSecret(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeetingOpen(t, "test-committee", "Open Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Open Meeting")

	page := newPage(t)
	if _, err := page.Goto(joinURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto join page: %v", err)
	}
	waitForJoinGuestForm(t, page)

	if err := page.Locator("input[name=full_name]").Fill("Bob Guest"); err != nil {
		t.Fatalf("fill name: %v", err)
	}
	if err := page.Locator("input[name=meeting_secret]").Fill("wrong-secret"); err != nil {
		t.Fatalf("fill meeting secret: %v", err)
	}
	if err := page.Locator("button:has-text('Sign Up as Guest')").Click(); err != nil {
		t.Fatalf("click submit: %v", err)
	}

	if err := page.Locator("#app-notification-target .alert:has-text('Invalid meeting secret')").WaitFor(); err != nil {
		t.Fatalf("expected invalid meeting secret error: %v", err)
	}
}

func TestGuestSignup_PrefilledMeetingSecretFromQuery(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeetingOpen(t, "test-committee", "Open Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Open Meeting")

	page := newPage(t)
	urlWithSecret := joinURL(ts.URL, "test-committee", meetingID) + "?meeting_secret=test-meeting-secret"
	if _, err := page.Goto(urlWithSecret); err != nil {
		t.Fatalf("goto join page with prefilled secret: %v", err)
	}
	if err := page.Locator("input[name=full_name]").WaitFor(); err != nil {
		t.Fatalf("wait for guest signup name field: %v", err)
	}

	visibleSecretInput, err := page.Locator("#meeting_secret").IsVisible()
	if err != nil {
		t.Fatalf("check visible meeting secret input: %v", err)
	}
	if visibleSecretInput {
		t.Fatalf("expected meeting secret input to be hidden when prefilled via query")
	}

	if err := page.Locator("input[name=full_name]").Fill("Bob Guest"); err != nil {
		t.Fatalf("fill name: %v", err)
	}
	if err := page.Locator("button:has-text('Sign Up as Guest')").Click(); err != nil {
		t.Fatalf("click submit: %v", err)
	}

	if err := page.WaitForURL(liveURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("expected redirect to /live after guest signup: %v", err)
	}
}

// TestGuestSignup_PublishesManageSSE verifies that guest self-signup from the
// join page publishes attendee-change events consumed by the manage page.
func TestGuestSignup_PublishesManageSSE(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeetingOpen(t, "test-committee", "Open Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Open Meeting")

	managePage := newPage(t)
	userLogin(t, managePage, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := managePage.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	guestPage := newPage(t)
	if _, err := guestPage.Goto(joinURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto join page: %v", err)
	}
	waitForJoinGuestForm(t, guestPage)
	if err := guestPage.Locator("input[name=full_name]").Fill("Guest Via Join"); err != nil {
		t.Fatalf("fill name: %v", err)
	}
	if err := guestPage.Locator("input[name=meeting_secret]").Fill("test-meeting-secret"); err != nil {
		t.Fatalf("fill meeting secret: %v", err)
	}
	if err := guestPage.Locator("button:has-text('Sign Up as Guest')").Click(); err != nil {
		t.Fatalf("click submit: %v", err)
	}
	if err := guestPage.WaitForURL(liveURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("expected redirect to /live after guest signup: %v", err)
	}

	openModerateLeftTab(t, managePage, "attendees")
	if err := managePage.Locator("#attendee-list-container [data-testid='manage-attendee-card']:has-text('Guest Via Join')").WaitFor(); err != nil {
		t.Fatalf("expected attendee propagated to manage page via SSE: %v", err)
	}
}

// TestGuestSignup_Quoted_SetsSpeakerQuotedBadge verifies that quoted status
// chosen during guest signup is carried into the speaker entry chips.
func TestGuestSignup_Quoted_SetsSpeakerQuotedBadge(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeetingOpen(t, "test-committee", "Open Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Open Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Open Meeting", "Main Topic")
	ts.activateAgendaPoint(t, "test-committee", "Open Meeting", apID)

	page := newPage(t)
	if _, err := page.Goto(joinURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto join page: %v", err)
	}
	waitForJoinGuestForm(t, page)
	if err := page.Locator("input[name=full_name]").Fill("Quoted Via Join"); err != nil {
		t.Fatalf("fill name: %v", err)
	}
	if err := page.Locator("input[name=gender_quoted]").Check(); err != nil {
		t.Fatalf("check quoted during guest signup: %v", err)
	}
	if err := page.Locator("input[name=meeting_secret]").Fill("test-meeting-secret"); err != nil {
		t.Fatalf("fill meeting secret: %v", err)
	}
	if err := page.Locator("button:has-text('Sign Up as Guest')").Click(); err != nil {
		t.Fatalf("click guest sign-up submit: %v", err)
	}
	if err := page.WaitForURL(liveURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("expected redirect to /live after guest signup: %v", err)
	}

	if err := page.Locator("[data-testid='live-add-self-regular']").Click(); err != nil {
		t.Fatalf("click self-add regular: %v", err)
	}
	row := page.Locator("#attendee-speakers-list [data-testid='live-speakers-active-viewport'] [data-testid='live-speaker-item']").Filter(playwright.LocatorFilterOptions{
		HasText: "Quoted Via Join",
	})
	if err := row.WaitFor(); err != nil {
		t.Fatalf("expected quoted guest row in live speakers list: %v", err)
	}
	if err := row.Locator("[data-testid='live-speaker-quoted-badge']").WaitFor(); err != nil {
		t.Fatalf("expected quoted badge in guest speaker row: %v", err)
	}
}

func TestManageJoinQRPage_ContainsSecretJoinURL(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Alice Chair", "chairperson")
	ts.seedMeetingOpen(t, "test-committee", "Open Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Open Meeting")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(manageJoinQRURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage join qr page: %v", err)
	}

	if err := page.Locator("#join-qr-code").WaitFor(); err != nil {
		t.Fatalf("expected QR image: %v", err)
	}

	href, err := page.Locator("main a.plain-text-link").First().GetAttribute("href")
	if err != nil {
		t.Fatalf("read join URL href: %v", err)
	}
	if !strings.Contains(href, "meeting_secret=test-meeting-secret") {
		t.Fatalf("expected join URL with meeting_secret query param, got: %v", href)
	}
}

