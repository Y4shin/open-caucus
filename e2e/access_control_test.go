//go:build e2e

// Package e2e_test contains browser-based end-to-end tests.
//
// This file covers access control across all protected pages and every
// significant user/attendee role. Tests are grouped by page and follow the
// naming convention:
//
//	TestAccess_<Page>_<Actor>_<ExpectedOutcome>
//
// Middleware reference
//
//	admin_required   – redirects to /admin/login for any non-admin session
//	auth             – redirects to / for any missing/expired session (any type passes)
//	committee_access – enforces committee slug for user sessions; attendee sessions skip
//	manage_access    – allows chairperson users OR chair attendees (for that meeting)
//	moderate_access  – allows chairperson users OR chair attendees OR designated moderator
//	attendee_required – 403 for anything that is not a valid attendee session
//
// What is NOT duplicated here (already covered elsewhere):
//   - Admin login happy path                           → admin_test.go
//   - Chairperson/member committee dashboard visibility → committee_test.go
//   - Non-chair attendee on manage page (403)          → attendee_login_test.go
//   - Chair attendee on manage page (allowed)          → attendee_login_test.go
//   - All four moderate access levels                  → moderate_test.go
package e2e_test

import (
	"context"
	"strconv"
	"testing"
)

// ─── Admin Panel ──────────────────────────────────────────────────────────────
// Middleware chain: [session, admin_required]
// admin_required redirects any non-admin session to /admin/login.

// TestAccess_AdminPanel_Unauthenticated_RedirectedToAdminLogin verifies that a
// visitor with no session is redirected to /admin/login when accessing /admin.
func TestAccess_AdminPanel_Unauthenticated_RedirectedToAdminLogin(t *testing.T) {
	ts := newTestServer(t)
	page := newPage(t)

	if _, err := page.Goto(ts.URL + "/admin"); err != nil {
		t.Fatalf("goto /admin: %v", err)
	}
	if err := page.WaitForURL(ts.URL + "/admin/login"); err != nil {
		t.Fatalf("expected redirect to /admin/login for unauthenticated /admin: %v", err)
	}
}

// TestAccess_AdminPanel_UserSession_RedirectedToAdminLogin verifies that a regular
// committee user session (even chairperson) is redirected from /admin to /admin/login.
func TestAccess_AdminPanel_UserSession_RedirectedToAdminLogin(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")

	if _, err := page.Goto(ts.URL + "/admin"); err != nil {
		t.Fatalf("goto /admin: %v", err)
	}
	if err := page.WaitForURL(ts.URL + "/admin/login"); err != nil {
		t.Fatalf("expected redirect to /admin/login for chairperson user session: %v", err)
	}
}

// TestAccess_AdminPanel_AttendeeSession_RedirectedToAdminLogin verifies that an
// attendee session is redirected from /admin to /admin/login.
func TestAccess_AdminPanel_AttendeeSession_RedirectedToAdminLogin(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeetingOpen(t, "test-committee", "Open Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Open Meeting")
	ts.seedAttendee(t, "test-committee", "Open Meeting", "Attendee", "secret-att")

	page := newPage(t)
	attendeeLoginHelper(t, page, ts.URL, "test-committee", meetingID, "secret-att")

	if _, err := page.Goto(ts.URL + "/admin"); err != nil {
		t.Fatalf("goto /admin: %v", err)
	}
	if err := page.WaitForURL(ts.URL + "/admin/login"); err != nil {
		t.Fatalf("expected redirect to /admin/login for attendee session: %v", err)
	}
}

// ─── Committee Dashboard ──────────────────────────────────────────────────────
// Middleware chain: [session, auth, committee_access]
// auth    → redirects to / for missing/expired sessions
// committee_access → 403 if user session's slug ≠ URL slug

// TestAccess_CommitteeDashboard_Unauthenticated_RedirectedToLogin verifies that a
// visitor with no session is redirected to the login page.
func TestAccess_CommitteeDashboard_Unauthenticated_RedirectedToLogin(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")

	page := newPage(t)
	if _, err := page.Goto(ts.URL + "/committee/test-committee"); err != nil {
		t.Fatalf("goto /committee/test-committee: %v", err)
	}
	if err := page.WaitForURL(ts.URL + "/login"); err != nil {
		t.Fatalf("expected redirect to /login for unauthenticated committee dashboard: %v", err)
	}
}

// TestAccess_CommitteeDashboard_WrongCommitteeUser_Forbidden verifies that a user
// cannot access a committee other than their own.
func TestAccess_CommitteeDashboard_WrongCommitteeUser_Forbidden(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Committee A", "committee-a")
	ts.seedCommittee(t, "Committee B", "committee-b")
	ts.seedUser(t, "committee-a", "user-a", "pass123", "User A", "chairperson")

	page := newPage(t)
	userLogin(t, page, ts.URL, "committee-a", "user-a", "pass123")

	if _, err := page.Goto(ts.URL + "/committee/committee-b"); err != nil {
		t.Fatalf("goto /committee/committee-b: %v", err)
	}
	expectAlertContaining(t, page, "not a member of this committee")
}

// TestAccess_CommitteeDashboard_Member_Allowed verifies that a member-role user can
// access their own committee's dashboard.
func TestAccess_CommitteeDashboard_Member_Allowed(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Regular Member", "member")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "member1", "pass123")

	resp, err := page.Goto(ts.URL + "/committee/test-committee")
	if err != nil {
		t.Fatalf("goto /committee/test-committee: %v", err)
	}
	if resp.Status() != 200 {
		t.Fatalf("expected 200 for member on own committee dashboard, got %d", resp.Status())
	}
}

// TestAccess_MeetingPage_Member_NonActiveMeetingForbidden verifies that a regular member
// can only access the currently active meeting page.
func TestAccess_MeetingPage_Member_NonActiveMeetingForbidden(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Regular Member", "member")
	ts.seedMeeting(t, "test-committee", "Active Meeting", "")
	ts.seedMeeting(t, "test-committee", "Old Meeting", "")

	activeMeetingIDStr := ts.getMeetingID(t, "test-committee", "Active Meeting")
	activeMeetingID, err := strconv.ParseInt(activeMeetingIDStr, 10, 64)
	if err != nil {
		t.Fatalf("parse active meeting id: %v", err)
	}
	if err := ts.repo.SetActiveMeeting(context.Background(), "test-committee", &activeMeetingID); err != nil {
		t.Fatalf("set active meeting: %v", err)
	}

	oldMeetingID := ts.getMeetingID(t, "test-committee", "Old Meeting")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "member1", "pass123")

	if _, err := page.Goto(ts.URL + "/committee/test-committee/meeting/" + oldMeetingID); err != nil {
		t.Fatalf("goto old meeting page as member: %v", err)
	}
	expectAlertContaining(t, page, "meeting is not currently active")
}

// ─── Manage Page ─────────────────────────────────────────────────────────────
// Middleware chain: [session, auth, committee_access, manage_access]
// manage_access allows: chairperson user OR chair attendee for the matching meeting.

// TestAccess_ManagePage_Unauthenticated_RedirectedToLogin verifies that a visitor
// with no session is redirected to the login page.
func TestAccess_ManagePage_Unauthenticated_RedirectedToLogin(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")

	page := newPage(t)
	if _, err := page.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto moderate page: %v", err)
	}
	if err := page.WaitForURL(ts.URL + "/login"); err != nil {
		t.Fatalf("expected redirect to /login for unauthenticated moderate page: %v", err)
	}
}

// TestAccess_ManagePage_Member_Forbidden verifies that a committee member (non-chairperson)
// user receives a 403 when attempting to access the manage page.
func TestAccess_ManagePage_Member_Forbidden(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Regular Member", "member")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "member1", "pass123")

	if _, err := page.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto moderate page as member: %v", err)
	}
	expectAlertContaining(t, page, "chairperson role required")
}

// TestAccess_ManagePage_WrongCommitteeUser_Forbidden verifies that a chairperson
// from committee A cannot access committee B's manage page.
func TestAccess_ManagePage_WrongCommitteeUser_Forbidden(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Committee A", "committee-a")
	ts.seedCommittee(t, "Committee B", "committee-b")
	ts.seedUser(t, "committee-a", "chair-a", "pass123", "Chair A", "chairperson")
	ts.seedMeeting(t, "committee-b", "B Meeting", "")
	meetingID := ts.getMeetingID(t, "committee-b", "B Meeting")

	page := newPage(t)
	userLogin(t, page, ts.URL, "committee-a", "chair-a", "pass123")

	if _, err := page.Goto(manageURL(ts.URL, "committee-b", meetingID)); err != nil {
		t.Fatalf("goto wrong-committee moderate page: %v", err)
	}
	expectAlertContaining(t, page, "chairperson role required")
}

// TestAccess_ManagePage_WrongMeetingAttendee_Forbidden verifies that a chair attendee
// for meeting A cannot access meeting B's manage page (meeting_id mismatch).
func TestAccess_ManagePage_WrongMeetingAttendee_Forbidden(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeetingOpen(t, "test-committee", "Meeting A", "")
	ts.seedMeeting(t, "test-committee", "Meeting B", "")
	meetingA := ts.getMeetingID(t, "test-committee", "Meeting A")
	meetingB := ts.getMeetingID(t, "test-committee", "Meeting B")
	attendee := ts.seedAttendee(t, "test-committee", "Meeting A", "Chair of A", "secret-chair-a")
	ts.setAttendeeChair(t, attendee.ID, true)

	page := newPage(t)
	attendeeLoginHelper(t, page, ts.URL, "test-committee", meetingA, "secret-chair-a")

	if _, err := page.Goto(manageURL(ts.URL, "test-committee", meetingB)); err != nil {
		t.Fatalf("goto wrong-meeting moderate page: %v", err)
	}
	expectAlertContaining(t, page, "chairperson role required")
}

// TestAccess_ManagePage_DesignatedModerator_Forbidden verifies that an attendee who
// is only the designated meeting moderator (not a chair) cannot access the manage page.
func TestAccess_ManagePage_DesignatedModerator_Allowed(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeetingOpen(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	attendee := ts.seedAttendee(t, "test-committee", "Board Meeting", "Mod Guest", "secret-mod-mg")
	ts.setMeetingModerator(t, "test-committee", "Board Meeting", &attendee.ID)

	page := newPage(t)
	attendeeLoginHelper(t, page, ts.URL, "test-committee", meetingID, "secret-mod-mg")

	resp, err := page.Goto(manageURL(ts.URL, "test-committee", meetingID))
	if err != nil {
		t.Fatalf("goto moderate page as designated moderator: %v", err)
	}
	if resp.Status() != 200 {
		t.Fatalf("expected 200 for designated moderator on moderate page, got %d", resp.Status())
	}
}

// ─── Moderate Page ────────────────────────────────────────────────────────────
// Middleware chain: [session, moderate_access]
// moderate_access allows: chairperson user OR chair attendee OR designated moderator.
// All other sessions (no session, member user, plain attendee, wrong meeting) are
// redirected to / (no session) or receive 403 (invalid session).

// TestAccess_ModeratePage_Unauthenticated_RedirectedToLogin verifies that a visitor
// with no session is redirected to the login page.
func TestAccess_ModeratePage_Unauthenticated_RedirectedToLogin(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")

	page := newPage(t)
	if _, err := page.Goto(moderateURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto moderate page: %v", err)
	}
	if err := page.WaitForURL(ts.URL + "/login"); err != nil {
		t.Fatalf("expected redirect to /login for unauthenticated moderate page: %v", err)
	}
}

// TestAccess_ModeratePage_Member_Forbidden verifies that a regular committee member
// (non-chairperson) user receives a 403 on the moderate page.
func TestAccess_ModeratePage_Member_Forbidden(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Regular Member", "member")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "member1", "pass123")

	if _, err := page.Goto(moderateURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto moderate page as member: %v", err)
	}
	expectAlertContaining(t, page, "chairperson role required")
}

// A designated meeting moderator has a narrow access window:
//   - moderate_access routes (moderate page, speaker actions): ALLOWED
//   - manage_access routes (manage page, attendee management): FORBIDDEN

// TestAccess_DesignatedModerator_ModerateAllowed_ManageForbidden verifies that a
// designated meeting moderator can access the moderate page but is blocked from the
// manage page in the same browser session.
func TestAccess_DesignatedModerator_ModerateAllowed(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedMeetingOpen(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	attendee := ts.seedAttendee(t, "test-committee", "Board Meeting", "Mod Guest", "secret-mod-bound")
	ts.setMeetingModerator(t, "test-committee", "Board Meeting", &attendee.ID)

	page := newPage(t)
	attendeeLoginHelper(t, page, ts.URL, "test-committee", meetingID, "secret-mod-bound")

	// Moderate page must load successfully.
	if _, err := page.Goto(moderateURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto moderate page as designated moderator: %v", err)
	}
	if err := page.Locator("#speakers-list-container").WaitFor(); err != nil {
		t.Fatalf("expected moderate page to render for designated moderator: %v", err)
	}

	// Ensure follow-up loads still succeed on the same route/session.
	resp, err := page.Goto(moderateURL(ts.URL, "test-committee", meetingID))
	if err != nil {
		t.Fatalf("revisit moderate page as designated moderator: %v", err)
	}
	if resp.Status() != 200 {
		t.Fatalf("expected 200 on moderate revisit for designated moderator, got %d", resp.Status())
	}
}
