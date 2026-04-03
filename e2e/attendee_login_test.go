//go:build e2e

package e2e_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/Y4shin/open-caucus/internal/repository/model"
)

func liveURL(baseURL, slug, meetingID string) string {
	return fmt.Sprintf("%s/committee/%s/meeting/%s", baseURL, slug, meetingID)
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
	a, err := ts.repo.CreateAttendee(context.Background(), mid, nil, fullName, secret, false)
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
	if err := page.Locator("input[name=secret]").Press("Enter"); err != nil {
		t.Fatalf("submit attendee login: %v", err)
	}

	if err := page.WaitForURL(liveURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("expected redirect to /live after login: %v", err)
	}
	if err := page.Locator("p.scaffold-auth-text:has-text('Carol Guest')").WaitFor(); err != nil {
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
	if err := page.Locator("input[name=secret]").Press("Enter"); err != nil {
		t.Fatalf("submit attendee login: %v", err)
	}

	if err := page.Locator("#app-notification-target .alert:has-text('Invalid access code')").WaitFor(); err != nil {
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
	activeMeetingID := int64(0)
	fmt.Sscanf(meetingID, "%d", &activeMeetingID)
	if err := ts.repo.SetActiveMeeting(context.Background(), "test-committee", &activeMeetingID); err != nil {
		t.Fatalf("set active meeting: %v", err)
	}

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
	if err := page.Locator("p.scaffold-auth-text:has-text('Alice Member')").WaitFor(); err != nil {
		t.Fatalf("expected full name on live page: %v", err)
	}
}

// TestLivePage_RequiresAttendeeSession verifies that /live redirects
// unauthenticated visitors (no session) to the login page.
func TestLivePage_RequiresAttendeeSession(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeeting(t, "test-committee", "Closed Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Closed Meeting")

	page := newPage(t)
	if _, err := page.Goto(liveURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto /live: %v", err)
	}
	if err := page.WaitForURL(joinURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("expected redirect to /join for unauthenticated /live: %v", err)
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
	if err := page.Locator("input[name=secret]").Press("Enter"); err != nil {
		t.Fatalf("submit attendee login: %v", err)
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

// TestGuestLive_LogoutButton verifies that guests can logout from the live page.
func TestGuestLive_LogoutButton(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeetingOpen(t, "test-committee", "Open Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Open Meeting")
	ts.seedAttendee(t, "test-committee", "Open Meeting", "Eve Guest", "secret-logout")

	page := newPage(t)
	if _, err := page.Goto(attendeeLoginURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto attendee-login: %v", err)
	}
	if err := page.Locator("input[name=secret]").Fill("secret-logout"); err != nil {
		t.Fatalf("fill secret: %v", err)
	}
	if err := page.Locator("input[name=secret]").Press("Enter"); err != nil {
		t.Fatalf("submit attendee login: %v", err)
	}
	if err := page.WaitForURL(liveURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("expected redirect to /live after login: %v", err)
	}

	if err := page.Locator("button:has-text('Logout')").Click(); err != nil {
		t.Fatalf("click logout on live page: %v", err)
	}
	if err := page.WaitForURL(ts.URL + "/login"); err != nil {
		t.Fatalf("expected redirect to /login after logout: %v", err)
	}

	resp, err := page.Goto(liveURL(ts.URL, "test-committee", meetingID))
	if err != nil {
		t.Fatalf("goto /live after logout: %v", err)
	}
	if err := page.WaitForURL(joinURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("expected redirect to /join for /live after logout: %v", err)
	}
	if resp.Status() != 200 {
		t.Fatalf("expected final join page response status 200 after redirect, got %d", resp.Status())
	}
}

// TestLivePage_AttendeeChair_SeesManageButtonAndCanOpenManage verifies that
// chair attendees can navigate from /live to /moderate.
func TestLivePage_AttendeeChair_SeesManageButtonAndCanOpenManage(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeetingOpen(t, "test-committee", "Open Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Open Meeting")
	attendee := ts.seedAttendee(t, "test-committee", "Open Meeting", "Chair Guest", "secret-chair")
	ts.setAttendeeChair(t, attendee.ID, true)

	page := newPage(t)
	if _, err := page.Goto(attendeeLoginURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto attendee-login: %v", err)
	}
	if err := page.Locator("input[name=secret]").Fill("secret-chair"); err != nil {
		t.Fatalf("fill secret: %v", err)
	}
	if err := page.Locator("input[name=secret]").Press("Enter"); err != nil {
		t.Fatalf("submit attendee login: %v", err)
	}
	if err := page.WaitForURL(liveURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("expected redirect to /live after login: %v", err)
	}

	manageLink := page.Locator("a.btn:has-text('Moderate')")
	if err := manageLink.WaitFor(); err != nil {
		t.Fatalf("expected manage button for chair attendee: %v", err)
	}
	if err := manageLink.Click(); err != nil {
		t.Fatalf("click manage button: %v", err)
	}
	if err := page.WaitForURL(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("expected navigation to /moderate for chair attendee: %v", err)
	}
}

// TestManagePage_AttendeeNonChair_Forbidden verifies non-chair attendees
// cannot open the manage page directly.
func TestManagePage_AttendeeNonChair_Forbidden(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeetingOpen(t, "test-committee", "Open Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Open Meeting")
	ts.seedAttendee(t, "test-committee", "Open Meeting", "Nonchair Guest", "secret-nonchair")

	page := newPage(t)
	if _, err := page.Goto(attendeeLoginURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto attendee-login: %v", err)
	}
	if err := page.Locator("input[name=secret]").Fill("secret-nonchair"); err != nil {
		t.Fatalf("fill secret: %v", err)
	}
	if err := page.Locator("input[name=secret]").Press("Enter"); err != nil {
		t.Fatalf("submit attendee login: %v", err)
	}
	if err := page.WaitForURL(liveURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("expected redirect to /live after login: %v", err)
	}

	manageCount, err := page.Locator("a.btn:has-text('Moderate')").Count()
	if err != nil {
		t.Fatalf("count manage links: %v", err)
	}
	if manageCount > 0 {
		t.Fatalf("did not expect manage button for non-chair attendee")
	}

	if _, err := page.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto /moderate as non-chair attendee: %v", err)
	}
	expectAlertContaining(t, page, "chairperson role required")
}
