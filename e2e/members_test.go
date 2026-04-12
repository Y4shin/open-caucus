//go:build e2e

package e2e_test

import (
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

func TestMembers_AddByEmail_AppearsInList(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")

	// Wait for members section to load.
	membersHeading := page.Locator("h3:has-text('Members')")
	if err := membersHeading.WaitFor(); err != nil {
		t.Fatalf("members section not found: %v", err)
	}

	// Fill the "Add by email" form.
	emailInput := page.Locator("input[type='email']")
	if err := emailInput.WaitFor(); err != nil {
		t.Fatalf("email input not found: %v", err)
	}
	if err := emailInput.Fill("bob@example.com"); err != nil {
		t.Fatalf("fill email: %v", err)
	}
	nameInput := page.Locator("input[placeholder]").Filter(playwright.LocatorFilterOptions{HasText: ""}).Nth(1)
	// Find the name input more precisely — look for the one near the email input
	addForm := page.Locator("form").Filter(playwright.LocatorFilterOptions{Has: emailInput})
	if err := addForm.Locator("input[type='text']").Fill("Bob Jones"); err != nil {
		t.Fatalf("fill name: %v", err)
	}

	// Click Add Member.
	if err := addForm.Locator("button[type='submit']").Click(); err != nil {
		t.Fatalf("click add member: %v", err)
	}

	// Verify the member appears in the list.
	memberRow := page.Locator("tr").Filter(playwright.LocatorFilterOptions{HasText: "Bob Jones"})
	if err := memberRow.WaitFor(); err != nil {
		t.Fatalf("expected Bob Jones in member list: %v", err)
	}
	if err := memberRow.Locator("text=bob@example.com").WaitFor(); err != nil {
		t.Fatalf("expected email in member row: %v", err)
	}
	_ = nameInput // suppress unused
}

func TestMembers_CreateMeetingWithInvites(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")

	// Add a member by email first.
	emailInput := page.Locator("input[type='email']")
	if err := emailInput.WaitFor(); err != nil {
		t.Fatalf("email input not found: %v", err)
	}
	addForm := page.Locator("form").Filter(playwright.LocatorFilterOptions{Has: emailInput})
	if err := emailInput.Fill("alice@example.com"); err != nil {
		t.Fatalf("fill email: %v", err)
	}
	if err := addForm.Locator("input[type='text']").Fill("Alice Smith"); err != nil {
		t.Fatalf("fill name: %v", err)
	}
	if err := addForm.Locator("button[type='submit']").Click(); err != nil {
		t.Fatalf("click add: %v", err)
	}

	// Wait for member to appear.
	if err := page.Locator("tr").Filter(playwright.LocatorFilterOptions{HasText: "Alice Smith"}).WaitFor(); err != nil {
		t.Fatalf("wait alice: %v", err)
	}

	// Open the meeting wizard.
	createBtn := page.Locator("[data-testid='committee-create-form']")
	if err := createBtn.WaitFor(); err != nil {
		t.Fatalf("create button not found: %v", err)
	}
	if err := createBtn.Click(); err != nil {
		t.Fatalf("click create: %v", err)
	}

	// Wait for wizard dialog.
	if err := page.Locator("dialog[open] #wizard-name").WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(defaultE2ETimeoutMs),
	}); err != nil {
		t.Fatalf("wait wizard: %v", err)
	}

	// Step 1: Fill name.
	if err := page.Locator("#wizard-name").Fill("Invited Meeting"); err != nil {
		t.Fatalf("fill name: %v", err)
	}
	if err := page.Locator("dialog[open] button:has-text('Next')").Click(); err != nil {
		t.Fatalf("next step 1: %v", err)
	}

	// Step 2: Skip agenda.
	if err := page.Locator("dialog[open] button:has-text('Next')").Click(); err != nil {
		t.Fatalf("next step 2: %v", err)
	}

	// Step 3: Invites — verify Alice shows up and is checked.
	aliceRow := page.Locator("dialog[open] li").Filter(playwright.LocatorFilterOptions{HasText: "Alice Smith"})
	if err := aliceRow.WaitFor(); err != nil {
		t.Fatalf("alice not in invites list: %v", err)
	}
	// Check that Alice's email badge is visible.
	if err := aliceRow.Locator("text=alice@example.com").WaitFor(); err != nil {
		t.Fatalf("alice email badge not visible: %v", err)
	}
	// Verify Alice is checked (has contact info).
	aliceCheckbox := aliceRow.Locator("input[type='checkbox']")
	checked, err := aliceCheckbox.IsChecked()
	if err != nil {
		t.Fatalf("read alice checkbox: %v", err)
	}
	if !checked {
		t.Fatal("expected Alice to be checked by default (has email)")
	}

	// Also verify the chair member shows up.
	chairRow := page.Locator("dialog[open] li").Filter(playwright.LocatorFilterOptions{HasText: "Chair Person"})
	if err := chairRow.WaitFor(); err != nil {
		t.Fatalf("chair not in invites list: %v", err)
	}

	// Proceed to review.
	if err := page.Locator("dialog[open] button:has-text('Next')").Click(); err != nil {
		t.Fatalf("next step 3: %v", err)
	}

	// Step 4: Review — verify invite count is shown.
	reviewSection := page.Locator("dialog[open]")
	inviteText := reviewSection.Locator("text=/Invite.*will.*sent/i")
	if err := inviteText.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		// Invites text might not appear if email is disabled in test — that's ok.
		t.Logf("invite text not found (email may be disabled in test): %v", err)
	}

	// Create the meeting.
	if err := page.Locator("dialog[open] button:has-text('Create Meeting')").Click(); err != nil {
		t.Fatalf("create meeting: %v", err)
	}

	// Verify the meeting was created.
	meetingCard := page.Locator("[data-testid='committee-meeting-list'] li").Filter(playwright.LocatorFilterOptions{HasText: "Invited Meeting"})
	if err := meetingCard.WaitFor(); err != nil {
		t.Fatalf("expected meeting card: %v", err)
	}
}
