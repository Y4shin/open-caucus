//go:build e2e

package e2e_test

import (
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

func meetingCard(page playwright.Page, meetingName string) playwright.Locator {
	return page.Locator("#meeting-list-container [data-testid='committee-meeting-row']").Filter(playwright.LocatorFilterOptions{
		HasText: meetingName,
	})
}

func meetingActiveToggle(page playwright.Page, meetingName string) playwright.Locator {
	return meetingCard(page, meetingName).Locator("input[data-testid='committee-toggle-active']")
}

func TestChairpersonSeesCreateMeetingForm(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")

	if err := page.Locator("#meeting-list-container [data-testid='committee-create-form']").WaitFor(); err != nil {
		t.Fatalf("expected create meeting form for chairperson: %v", err)
	}
}

func TestMemberDoesNotSeeCreateMeetingForm(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Regular Member", "member")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "member1", "pass123")

	visible, err := page.Locator("[data-testid='committee-create-form']").IsVisible()
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

	if err := page.Locator("#meeting-list-container [data-testid='committee-create-form']").WaitFor(); err != nil {
		t.Fatalf("create meeting form not found: %v", err)
	}

	urlBefore := page.URL()

	if err := page.Locator("#meeting-list-container input[name=name]").Fill("Budget Meeting"); err != nil {
		t.Fatalf("fill meeting name: %v", err)
	}
	if err := page.Locator("#meeting-list-container [data-testid='committee-create-form'] button[type=submit]").Click(); err != nil {
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

	firstActive := meetingActiveToggle(page, "First Meeting")
	secondActive := meetingActiveToggle(page, "Second Meeting")

	if err := firstActive.Click(); err != nil {
		t.Fatalf("activate first meeting: %v", err)
	}
	firstChecked, err := firstActive.IsChecked()
	if err != nil {
		t.Fatalf("read first active toggle: %v", err)
	}
	if !firstChecked {
		t.Fatalf("expected first meeting to be active")
	}
	secondChecked, err := secondActive.IsChecked()
	if err != nil {
		t.Fatalf("read second active toggle: %v", err)
	}
	if secondChecked {
		t.Fatalf("expected second meeting to be inactive while first is active")
	}

	if err := secondActive.Click(); err != nil {
		t.Fatalf("activate second meeting: %v", err)
	}
	firstChecked, err = firstActive.IsChecked()
	if err != nil {
		t.Fatalf("read first active toggle after activating second: %v", err)
	}
	if firstChecked {
		t.Fatalf("expected first meeting to become inactive after activating second")
	}
	secondChecked, err = secondActive.IsChecked()
	if err != nil {
		t.Fatalf("read second active toggle after activation: %v", err)
	}
	if !secondChecked {
		t.Fatalf("expected second meeting to be active")
	}
}

func TestUnsetActiveMeeting_WhenClickingActiveToggleAgain(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "First Meeting", "")
	ts.seedMeeting(t, "test-committee", "Second Meeting", "")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")

	firstActive := meetingActiveToggle(page, "First Meeting")
	secondActive := meetingActiveToggle(page, "Second Meeting")

	if err := firstActive.Click(); err != nil {
		t.Fatalf("activate first meeting: %v", err)
	}
	firstChecked, err := firstActive.IsChecked()
	if err != nil {
		t.Fatalf("read first active toggle after activation: %v", err)
	}
	if !firstChecked {
		t.Fatalf("expected first meeting to be active")
	}

	if err := firstActive.Click(); err != nil {
		t.Fatalf("deactivate active meeting: %v", err)
	}
	firstChecked, err = firstActive.IsChecked()
	if err != nil {
		t.Fatalf("read first active toggle after deactivation: %v", err)
	}
	if firstChecked {
		t.Fatalf("expected first meeting to be inactive after toggling active off")
	}
	secondChecked, err := secondActive.IsChecked()
	if err != nil {
		t.Fatalf("read second active toggle after deactivation: %v", err)
	}
	if secondChecked {
		t.Fatalf("expected no active meetings after deactivating the active one")
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

