//go:build e2e

package e2e_test

import (
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

func meetingCard(page playwright.Page, meetingName string) playwright.Locator {
	return page.Locator("#meeting-list-container article").Filter(playwright.LocatorFilterOptions{
		HasText: meetingName,
	})
}

func TestChairpersonSeesCreateMeetingForm(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")

	if err := page.Locator("#meeting-list-container .committee-create-layout").WaitFor(); err != nil {
		t.Fatalf("expected create meeting form for chairperson: %v", err)
	}
}

func TestMemberDoesNotSeeCreateMeetingForm(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Regular Member", "member")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "member1", "pass123")

	visible, err := page.Locator(".committee-create-layout").IsVisible()
	if err != nil {
		t.Fatalf("check create form visibility: %v", err)
	}
	if visible {
		t.Error("member should not see the create meeting form")
	}
}

func TestChairpersonCreateMeeting(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")

	if err := page.Locator("#meeting-list-container .committee-create-layout").WaitFor(); err != nil {
		t.Fatalf("create meeting form not found: %v", err)
	}

	urlBefore := page.URL()

	if err := page.Locator("#meeting-list-container input[name=name]").Fill("Budget Meeting"); err != nil {
		t.Fatalf("fill meeting name: %v", err)
	}
	if err := page.Locator("#meeting-list-container .committee-create-layout button[type=submit]").Click(); err != nil {
		t.Fatalf("submit create meeting: %v", err)
	}
	if err := meetingCard(page, "Budget Meeting").WaitFor(); err != nil {
		t.Fatalf("expected meeting card to appear: %v", err)
	}
	if page.URL() != urlBefore {
		t.Errorf("HTMX swap caused unexpected navigation: before=%s after=%s", urlBefore, page.URL())
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

	firstCard := meetingCard(page, "First Meeting")
	secondCard := meetingCard(page, "Second Meeting")
	if err := firstCard.WaitFor(); err != nil {
		t.Fatalf("first meeting card not visible: %v", err)
	}
	if err := secondCard.WaitFor(); err != nil {
		t.Fatalf("second meeting card not visible: %v", err)
	}

	firstActivateForm := firstCard.Locator("form[hx-post*='/activate']")
	if err := firstActivateForm.Locator("button[type=submit]").Click(); err != nil {
		t.Fatalf("activate first meeting: %v", err)
	}
	if err := firstActivateForm.WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected first activate form to disappear: %v", err)
	}

	secondActivateForm := secondCard.Locator("form[hx-post*='/activate']")
	if err := secondActivateForm.WaitFor(); err != nil {
		t.Fatalf("expected second activate form while first active: %v", err)
	}
	if err := secondActivateForm.Locator("button[type=submit]").Click(); err != nil {
		t.Fatalf("activate second meeting: %v", err)
	}
	if err := secondActivateForm.WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected second activate form to disappear: %v", err)
	}
	if err := firstCard.Locator("form[hx-post*='/activate']").WaitFor(); err != nil {
		t.Fatalf("expected first meeting to become activatable again: %v", err)
	}
}

func TestChairpersonDeleteMeeting(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Old Meeting", "")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")

	card := meetingCard(page, "Old Meeting")
	if err := card.WaitFor(); err != nil {
		t.Fatalf("meeting not visible before delete: %v", err)
	}

	page.OnDialog(func(d playwright.Dialog) {
		if err := d.Accept(); err != nil {
			t.Logf("accept dialog error: %v", err)
		}
	})

	if err := card.Locator("form[hx-post*='/delete'] button[type=submit]").Click(); err != nil {
		t.Fatalf("click delete: %v", err)
	}
	if err := meetingCard(page, "Old Meeting").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected meeting card to be removed: %v", err)
	}
}
