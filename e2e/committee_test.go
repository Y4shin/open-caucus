//go:build e2e

package e2e_test

import (
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

func TestChairpersonSeesCreateMeetingForm(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")

	if err := page.Locator("h3:has-text('Create New Meeting')").WaitFor(); err != nil {
		t.Fatalf("expected create meeting form to be visible for chairperson: %v", err)
	}
}

func TestMemberDoesNotSeeCreateMeetingForm(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Regular Member", "member")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "member1", "pass123")

	// The committee page should load successfully
	if err := page.Locator("h2:has-text('Committee Dashboard')").WaitFor(); err != nil {
		t.Fatalf("committee dashboard not loaded: %v", err)
	}

	// Members must not see the create meeting form
	visible, err := page.Locator("h3:has-text('Create New Meeting')").IsVisible()
	if err != nil {
		t.Fatalf("check visibility: %v", err)
	}
	if visible {
		t.Error("member should not see the Create New Meeting form")
	}
}

func TestChairpersonCreateMeeting(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")

	// Wait for the create meeting form to appear
	if err := page.Locator("h3:has-text('Create New Meeting')").WaitFor(); err != nil {
		t.Fatalf("create meeting form not found: %v", err)
	}

	urlBefore := page.URL()

	if err := page.Locator("input[name=name]").Fill("Budget Meeting"); err != nil {
		t.Fatalf("fill meeting name: %v", err)
	}
	if err := page.Locator("button:has-text('Create Meeting')").Click(); err != nil {
		t.Fatalf("click submit: %v", err)
	}

	// HTMX should update the meeting list in-place
	if err := page.Locator("td:has-text('Budget Meeting')").WaitFor(); err != nil {
		t.Fatalf("expected meeting to appear in list: %v", err)
	}

	if page.URL() != urlBefore {
		t.Errorf("HTMX swap caused unexpected navigation: before=%s after=%s", urlBefore, page.URL())
	}
}

func TestSetActiveMeeting_MarksAsActiveAndHidesButton(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Annual Meeting", "")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")

	if err := page.Locator("td:has-text('Annual Meeting')").WaitFor(); err != nil {
		t.Fatalf("meeting not visible: %v", err)
	}

	row := page.Locator("tr").Filter(playwright.LocatorFilterOptions{HasText: "Annual Meeting"})
	setActiveBtn := row.Locator("button:has-text('Set Active')")

	// Initially not active: button must be visible.
	if err := setActiveBtn.WaitFor(); err != nil {
		t.Fatalf("expected 'Set Active' button before activation: %v", err)
	}

	urlBefore := page.URL()

	if err := setActiveBtn.Click(); err != nil {
		t.Fatalf("click Set Active: %v", err)
	}

	// Button must disappear after HTMX swap re-renders the list.
	if err := setActiveBtn.WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected 'Set Active' button to be removed after activation: %v", err)
	}

	// No full navigation should have occurred.
	if page.URL() != urlBefore {
		t.Errorf("HTMX swap caused unexpected navigation: before=%s after=%s", urlBefore, page.URL())
	}

	// Active column (4th td) must now contain "Yes".
	if err := row.Locator("td:nth-child(4):has-text('Yes')").WaitFor(); err != nil {
		t.Fatalf("expected Active column to show 'Yes': %v", err)
	}
}

func TestSetActiveMeeting_DoesNotChangeSignupOpen(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Spring Meeting", "")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")

	if err := page.Locator("td:has-text('Spring Meeting')").WaitFor(); err != nil {
		t.Fatalf("meeting not visible: %v", err)
	}

	row := page.Locator("tr").Filter(playwright.LocatorFilterOptions{HasText: "Spring Meeting"})

	// Confirm signup_open column shows "No" before activation (seeded with signup_open=false).
	if err := row.Locator("td:nth-child(3):has-text('No')").WaitFor(); err != nil {
		t.Fatalf("expected Signup Open column to show 'No' before activation: %v", err)
	}

	setActiveBtn := row.Locator("button:has-text('Set Active')")
	if err := setActiveBtn.Click(); err != nil {
		t.Fatalf("click Set Active: %v", err)
	}
	if err := setActiveBtn.WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("wait for activation to complete: %v", err)
	}

	// Signup column must still show "No" — activation must not modify signup_open.
	if err := row.Locator("td:nth-child(3):has-text('No')").WaitFor(); err != nil {
		t.Fatalf("signup_open changed after activation — expected 'No' to remain: %v", err)
	}
}

func TestSetActiveMeeting_TransfersActiveStatus(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "First Meeting", "")
	ts.seedMeeting(t, "test-committee", "Second Meeting", "")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")

	if err := page.Locator("td:has-text('First Meeting')").WaitFor(); err != nil {
		t.Fatalf("first meeting not visible: %v", err)
	}
	if err := page.Locator("td:has-text('Second Meeting')").WaitFor(); err != nil {
		t.Fatalf("second meeting not visible: %v", err)
	}

	firstRow := page.Locator("tr").Filter(playwright.LocatorFilterOptions{HasText: "First Meeting"})
	secondRow := page.Locator("tr").Filter(playwright.LocatorFilterOptions{HasText: "Second Meeting"})

	// Activate the first meeting.
	firstBtn := firstRow.Locator("button:has-text('Set Active')")
	if err := firstBtn.Click(); err != nil {
		t.Fatalf("click Set Active for first meeting: %v", err)
	}
	if err := firstBtn.WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("wait for first meeting activation: %v", err)
	}

	// First is active, second is not.
	if err := firstRow.Locator("td:nth-child(4):has-text('Yes')").WaitFor(); err != nil {
		t.Fatalf("first meeting Active column should be 'Yes': %v", err)
	}
	if err := secondRow.Locator("button:has-text('Set Active')").WaitFor(); err != nil {
		t.Fatalf("second meeting should still show 'Set Active' button: %v", err)
	}

	// Now activate the second meeting.
	secondBtn := secondRow.Locator("button:has-text('Set Active')")
	if err := secondBtn.Click(); err != nil {
		t.Fatalf("click Set Active for second meeting: %v", err)
	}
	if err := secondBtn.WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("wait for second meeting activation: %v", err)
	}

	// Second is now active, first is not.
	if err := secondRow.Locator("td:nth-child(4):has-text('Yes')").WaitFor(); err != nil {
		t.Fatalf("second meeting Active column should be 'Yes': %v", err)
	}
	if err := firstRow.Locator("td:nth-child(4):has-text('No')").WaitFor(); err != nil {
		t.Fatalf("first meeting Active column should be 'No' after second is activated: %v", err)
	}
	if err := firstRow.Locator("button:has-text('Set Active')").WaitFor(); err != nil {
		t.Fatalf("first meeting should have 'Set Active' button back after being deactivated: %v", err)
	}
}

func TestChairpersonDeleteMeeting(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Old Meeting", "")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")

	// Wait for the meeting to appear in the list
	if err := page.Locator("td:has-text('Old Meeting')").WaitFor(); err != nil {
		t.Fatalf("meeting not visible before delete: %v", err)
	}

	// Handle the hx-confirm dialog before clicking
	page.OnDialog(func(d playwright.Dialog) {
		if err := d.Accept(); err != nil {
			t.Logf("accept dialog error: %v", err)
		}
	})

	// Click the Delete button in the specific meeting row.
	// Each row has "Set Active" and "Delete" buttons; filter by button text.
	deleteBtn := page.Locator("tr").Filter(playwright.LocatorFilterOptions{HasText: "Old Meeting"}).
		Locator("button:has-text('Delete')")
	if err := deleteBtn.Click(); err != nil {
		t.Fatalf("click delete: %v", err)
	}

	// Meeting row should be removed from the DOM
	if err := page.Locator("td:has-text('Old Meeting')").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected meeting to be removed from list: %v", err)
	}
}
