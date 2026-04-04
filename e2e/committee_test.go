//go:build e2e

package e2e_test

import (
	"context"
	"strconv"
	"testing"
	"time"

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

func TestMemberSeesActiveMeetingInfoAndJoinButton(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Regular Member", "member")
	ts.seedMeeting(t, "test-committee", "Active Meeting", "This one is active")
	activeMeetingID := ts.getMeetingID(t, "test-committee", "Active Meeting")
	activeMeetingIDInt, err := strconv.ParseInt(activeMeetingID, 10, 64)
	if err != nil {
		t.Fatalf("parse active meeting id: %v", err)
	}
	if err := ts.repo.SetActiveMeeting(context.Background(), "test-committee", &activeMeetingIDInt); err != nil {
		t.Fatalf("set active meeting: %v", err)
	}

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "member1", "pass123")

	if err := page.Locator("[data-testid='committee-active-meeting-card']").WaitFor(); err != nil {
		t.Fatalf("expected active meeting card for member: %v", err)
	}
	if err := page.Locator("[data-testid='committee-active-meeting-name']:has-text('Active Meeting')").WaitFor(); err != nil {
		t.Fatalf("expected active meeting name in card: %v", err)
	}
	if err := page.Locator("[data-testid='committee-join-active-meeting']").Click(); err != nil {
		t.Fatalf("click join active meeting button: %v", err)
	}
	if err := page.WaitForURL(ts.URL + "/committee/test-committee/meeting/" + activeMeetingID + "/join"); err != nil {
		t.Fatalf("expected navigation to join page: %v", err)
	}
}

func TestChairpersonCreateMeeting(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")

	// Click the "Create Meeting" button to open the wizard dialog
	createBtn := page.Locator("[data-testid='committee-create-form']")
	if err := createBtn.WaitFor(); err != nil {
		t.Fatalf("create meeting button not found: %v", err)
	}
	if err := createBtn.Click(); err != nil {
		t.Fatalf("click create meeting button: %v", err)
	}

	// Step 1: Fill meeting name and proceed
	if err := page.Locator("#wizard-name").Fill("Budget Meeting"); err != nil {
		t.Fatalf("fill meeting name: %v", err)
	}
	if err := page.Locator("dialog .modal-action button.btn-primary").Click(); err != nil {
		t.Fatalf("click next (step 1): %v", err)
	}

	// Step 2: Skip agenda, proceed
	if err := page.Locator("dialog .modal-action button.btn-primary").Click(); err != nil {
		t.Fatalf("click next (step 2): %v", err)
	}

	// Step 3: Skip participants, proceed
	if err := page.Locator("dialog .modal-action button.btn-primary").Click(); err != nil {
		t.Fatalf("click next (step 3): %v", err)
	}

	// Step 4: Review — click Create
	urlBefore := page.URL()
	if err := page.Locator("dialog .modal-action button.btn-primary").Click(); err != nil {
		t.Fatalf("click create (step 4): %v", err)
	}

	if err := meetingCard(page, "Budget Meeting").WaitFor(); err != nil {
		t.Fatalf("expected meeting card to appear: %v", err)
	}
	if page.URL() != urlBefore {
		t.Errorf("wizard caused unexpected navigation: before=%s after=%s", urlBefore, page.URL())
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
	waitUntil(t, 3*time.Second, func() (bool, error) {
		firstChecked, err = firstActive.IsChecked()
		if err != nil {
			return false, err
		}
		secondChecked, err := secondActive.IsChecked()
		if err != nil {
			return false, err
		}
		return !firstChecked && !secondChecked, nil
	}, "meeting active toggles to clear after deactivation")
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

	if err := card.Locator("button[aria-label='Delete']").Click(); err != nil {
		t.Fatalf("click delete: %v", err)
	}
	if err := meetingCard(page, "Old Meeting").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected meeting card to be removed: %v", err)
	}
}
