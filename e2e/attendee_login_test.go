//go:build e2e

package e2e_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/Y4shin/conference-tool/internal/repository/model"
)

func liveURL(baseURL, slug, meetingID string) string {
	return fmt.Sprintf("%s/committee/%s/meeting/%s/live", baseURL, slug, meetingID)
}

func attendeeLoginURL(baseURL, slug, meetingID string) string {
	return fmt.Sprintf("%s/committee/%s/meeting/%s/attendee-login", baseURL, slug, meetingID)
}

// seedAttendee creates a guest attendee directly in the DB and returns the secret.
func (ts *testServer) seedAttendee(t *testing.T, slug, meetingName, fullName, secret string) *model.Attendee {
	t.Helper()
	meetingID := ts.getMeetingID(t, slug, meetingName)
	mid := int64(0)
	fmt.Sscanf(meetingID, "%d", &mid)
	a, err := ts.repo.CreateAttendee(context.Background(), mid, nil, fullName, secret)
	if err != nil {
		t.Fatalf("seed attendee %q: %v", fullName, err)
	}
	return a
}

// TestAttendeeLogin_ValidSecret verifies that a guest can log in with their access
// code and is redirected to the live page.
func TestAttendeeLogin_ValidSecret(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeetingOpen(t, "test-committee", "Open Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Open Meeting")
	ts.seedAttendee(t, "test-committee", "Open Meeting", "Carol Guest", "secret-abc123")

	page := newPage(t)
	if _, err := page.Goto(attendeeLoginURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto attendee-login: %v", err)
	}

	if err := page.Locator("input[name=secret]").Fill("secret-abc123"); err != nil {
		t.Fatalf("fill secret: %v", err)
	}
	if err := page.Locator("button[type=submit]").Click(); err != nil {
		t.Fatalf("click submit: %v", err)
	}

	if err := page.WaitForURL(liveURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("expected redirect to /live after login: %v", err)
	}
	if err := page.Locator("p:has-text('Carol Guest')").WaitFor(); err != nil {
		t.Fatalf("expected attendee name on live page: %v", err)
	}
}

// TestAttendeeLogin_InvalidSecret verifies that a wrong access code shows an error.
func TestAttendeeLogin_InvalidSecret(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeetingOpen(t, "test-committee", "Open Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Open Meeting")

	page := newPage(t)
	if _, err := page.Goto(attendeeLoginURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto attendee-login: %v", err)
	}

	urlBefore := page.URL()

	if err := page.Locator("input[name=secret]").Fill("wrong-secret"); err != nil {
		t.Fatalf("fill secret: %v", err)
	}
	if err := page.Locator("button[type=submit]").Click(); err != nil {
		t.Fatalf("click submit: %v", err)
	}

	if err := page.Locator("p.error:has-text('Invalid')").WaitFor(); err != nil {
		t.Fatalf("expected error message for invalid code: %v", err)
	}
	if page.URL() != urlBefore {
		t.Errorf("URL changed on bad login: got %s, want %s", page.URL(), urlBefore)
	}
}

// TestMeetingJoinSubmit_CreatesAttendeeSession verifies that a registered user who
// signs up for a meeting is redirected to /live with an attendee session.
func TestMeetingJoinSubmit_CreatesAttendeeSession(t *testing.T) {
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

	if err := page.WaitForURL(liveURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("expected redirect to /live after signup: %v", err)
	}
	if err := page.Locator("p:has-text('Alice Member')").WaitFor(); err != nil {
		t.Fatalf("expected full name on live page: %v", err)
	}
}

// TestLivePage_RequiresAttendeeSession verifies that /live redirects unauthenticated
// visitors (no attendee session) with a 403.
func TestLivePage_RequiresAttendeeSession(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeeting(t, "test-committee", "Closed Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Closed Meeting")

	page := newPage(t)
	resp, err := page.Goto(liveURL(ts.URL, "test-committee", meetingID))
	if err != nil {
		t.Fatalf("goto /live: %v", err)
	}
	if resp.Status() != 403 {
		t.Errorf("expected 403 for unauthenticated /live, got %d", resp.Status())
	}
}

// TestAttendeeLoginPage_RedirectsIfAlreadyLoggedIn verifies that the attendee-login
// page redirects to /live when an active attendee session is already held.
func TestAttendeeLoginPage_RedirectsIfAlreadyLoggedIn(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeetingOpen(t, "test-committee", "Open Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Open Meeting")
	ts.seedAttendee(t, "test-committee", "Open Meeting", "Dan Guest", "secret-xyz789")

	page := newPage(t)
	// Log in once
	if _, err := page.Goto(attendeeLoginURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto attendee-login: %v", err)
	}
	if err := page.Locator("input[name=secret]").Fill("secret-xyz789"); err != nil {
		t.Fatalf("fill secret: %v", err)
	}
	if err := page.Locator("button[type=submit]").Click(); err != nil {
		t.Fatalf("click submit: %v", err)
	}
	if err := page.WaitForURL(liveURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("first login did not redirect to /live: %v", err)
	}

	// Visiting the login page again should redirect straight back to /live
	if _, err := page.Goto(attendeeLoginURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto attendee-login (second visit): %v", err)
	}
	if err := page.WaitForURL(liveURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("expected redirect to /live on second visit to login page: %v", err)
	}
}
