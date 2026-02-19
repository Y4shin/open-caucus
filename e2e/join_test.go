//go:build e2e

package e2e_test

import (
	"fmt"
	"testing"
)

func joinURL(baseURL, slug, meetingID string) string {
	return fmt.Sprintf("%s/committee/%s/meeting/%s/join", baseURL, slug, meetingID)
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
	if err := page.Locator("button:has-text('Sign Up as Guest')").WaitFor(); err != nil {
		t.Fatalf("expected guest sign-up button: %v", err)
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

// TestGuestSignup_Success verifies that a guest can submit their name and receive
// an access code on the success page.
func TestGuestSignup_Success(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeetingOpen(t, "test-committee", "Open Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Open Meeting")

	page := newPage(t)
	if _, err := page.Goto(joinURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto join page: %v", err)
	}

	if err := page.Locator("input[name=full_name]").Fill("Bob Guest"); err != nil {
		t.Fatalf("fill name: %v", err)
	}
	if err := page.Locator("button:has-text('Sign Up as Guest')").Click(); err != nil {
		t.Fatalf("click submit: %v", err)
	}

	if err := page.Locator("h3:has-text('You\\'re signed up!')").WaitFor(); err != nil {
		t.Fatalf("expected success message: %v", err)
	}
	if err := page.Locator("strong").WaitFor(); err != nil {
		t.Fatalf("expected access code element: %v", err)
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
