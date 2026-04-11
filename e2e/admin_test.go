//go:build e2e

package e2e_test

import (
	"context"
	"testing"

	"github.com/Y4shin/open-caucus/internal/repository/model"
	playwright "github.com/playwright-community/playwright-go"
)

func TestAdminLogin(t *testing.T) {
	ts := newTestServer(t)
	page := newPage(t)

	adminLogin(t, page, ts.URL)

	if err := page.Locator("#create-committee-form").WaitFor(); err != nil {
		t.Fatalf("expected admin dashboard create-committee form: %v", err)
	}
}

func TestAdminCreateCommittee(t *testing.T) {
	ts := newTestServer(t)
	page := newPage(t)

	adminLogin(t, page, ts.URL)

	// Note the URL before the HTMX form submit
	urlBefore := page.URL()

	if err := page.Locator("input[name=name]").Fill("Test Committee"); err != nil {
		t.Fatalf("fill name: %v", err)
	}
	if err := page.Locator("input[name=slug]").Fill("test-committee"); err != nil {
		t.Fatalf("fill slug: %v", err)
	}
	// The first submit button on the page is the "Create Committee" button
	if err := page.Locator("#create-committee-form button[type=submit]").Click(); err != nil {
		t.Fatalf("click submit: %v", err)
	}

	// HTMX should update the committee list in-place
	if err := page.Locator("td:has-text('test-committee')").WaitFor(); err != nil {
		t.Fatalf("expected new committee in list: %v", err)
	}

	// No full page navigation should have occurred
	if page.URL() != urlBefore {
		t.Errorf("HTMX swap caused unexpected navigation: before=%s after=%s", urlBefore, page.URL())
	}
}

func TestAdminDeleteCommittee(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Delete Me", "delete-me")

	page := newPage(t)
	adminLogin(t, page, ts.URL)

	// Verify committee is present
	if err := page.Locator("td:has-text('delete-me')").WaitFor(); err != nil {
		t.Fatalf("committee not visible before delete: %v", err)
	}

	// Handle the hx-confirm dialog before clicking
	page.OnDialog(func(d playwright.Dialog) {
		if err := d.Accept(); err != nil {
			t.Logf("accept dialog error: %v", err)
		}
	})

	// Click Delete for the specific committee row
	deleteBtn := page.Locator("tr").Filter(playwright.LocatorFilterOptions{HasText: "delete-me"}).Locator("button:has-text('Delete')")
	if err := deleteBtn.Click(); err != nil {
		t.Fatalf("click delete: %v", err)
	}

	// Committee row should be removed from the DOM
	if err := page.Locator("td:has-text('delete-me')").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected committee to be removed from list: %v", err)
	}
}

func TestAdminCreateAccount(t *testing.T) {
	ts := newTestServer(t)
	page := newPage(t)

	adminLogin(t, page, ts.URL)

	if err := page.Locator("a:has-text('Manage Accounts')").First().Click(); err != nil {
		t.Fatalf("click Manage Accounts: %v", err)
	}
	if err := page.WaitForURL(ts.URL + "/admin/accounts"); err != nil {
		t.Fatalf("wait for accounts page: %v", err)
	}

	urlBefore := page.URL()

	if err := page.Locator("input[name=username]").Fill("newaccount"); err != nil {
		t.Fatalf("fill username: %v", err)
	}
	if err := page.Locator("input[name=full_name]").Fill("New Account"); err != nil {
		t.Fatalf("fill full_name: %v", err)
	}
	if err := page.Locator("input[name=password]").Fill("password123"); err != nil {
		t.Fatalf("fill password: %v", err)
	}
	if err := page.Locator("#create-account-form button[type=submit]").Click(); err != nil {
		t.Fatalf("click submit: %v", err)
	}

	if err := page.Locator("td:has-text('newaccount')").WaitFor(); err != nil {
		t.Fatalf("expected new account in list: %v", err)
	}

	if page.URL() != urlBefore {
		t.Errorf("HTMX swap caused unexpected navigation: before=%s after=%s", urlBefore, page.URL())
	}
}

func TestAdminAssignAccount(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedAccount(t, "newuser", "password123", "New User")

	page := newPage(t)
	adminLogin(t, page, ts.URL)

	// Navigate to the committee membership page
	if err := page.Locator("a:has-text('Assign Accounts')").Click(); err != nil {
		t.Fatalf("click Assign Accounts: %v", err)
	}
	if err := page.WaitForURL(ts.URL + "/admin/committee/test-committee"); err != nil {
		t.Fatalf("wait for committee users page: %v", err)
	}

	urlBefore := page.URL()
	// Select account and assign it to the committee
	bitsSelectByID(t, page, "account_id", "newuser")
	bitsSelectByID(t, page, "role", "member")
	if err := page.Locator("#committee-users-container form button[type=submit]").First().Click(); err != nil {
		t.Fatalf("click submit: %v", err)
	}

	// HTMX should update the user list in-place
	if err := page.Locator("td:has-text('newuser')").WaitFor(); err != nil {
		t.Fatalf("expected new user in list: %v", err)
	}

	if page.URL() != urlBefore {
		t.Errorf("HTMX swap caused unexpected navigation: before=%s after=%s", urlBefore, page.URL())
	}
}

func TestAdminDeleteUser(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "todelete", "pass123", "To Delete", "member")

	page := newPage(t)
	adminLogin(t, page, ts.URL)

	// Navigate to the committee membership page
	if err := page.Locator("a:has-text('Assign Accounts')").Click(); err != nil {
		t.Fatalf("click Assign Accounts: %v", err)
	}
	if err := page.WaitForURL(ts.URL + "/admin/committee/test-committee"); err != nil {
		t.Fatalf("wait for committee users page: %v", err)
	}

	// Verify user is present
	if err := page.Locator("td:has-text('todelete')").WaitFor(); err != nil {
		t.Fatalf("user not visible before delete: %v", err)
	}

	// Handle the hx-confirm dialog before clicking
	page.OnDialog(func(d playwright.Dialog) {
		if err := d.Accept(); err != nil {
			t.Logf("accept dialog error: %v", err)
		}
	})

	// Click Delete for the specific user row.
	// The row has both "Save" and "Remove" submit buttons, so target by label.
	deleteBtn := page.Locator("tr").Filter(playwright.LocatorFilterOptions{HasText: "todelete"}).Locator("button:has-text('Remove')")
	if err := deleteBtn.Click(); err != nil {
		t.Fatalf("click delete: %v", err)
	}

	// User row should be removed from the DOM
	if err := page.Locator("td:has-text('todelete')").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected user to be removed from list: %v", err)
	}
}

func TestAdminUpdateMembershipRoleAndQuoted_ForManualMembership(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "editable", "pass123", "Editable User", "member")

	page := newPage(t)
	adminLogin(t, page, ts.URL)

	if err := page.Locator("a:has-text('Assign Accounts')").Click(); err != nil {
		t.Fatalf("click Assign Accounts: %v", err)
	}
	if err := page.WaitForURL(ts.URL + "/admin/committee/test-committee"); err != nil {
		t.Fatalf("wait for committee users page: %v", err)
	}

	row := page.Locator("tr").Filter(playwright.LocatorFilterOptions{HasText: "editable"})
	if err := row.WaitFor(); err != nil {
		t.Fatalf("expected editable user row: %v", err)
	}

	bitsSelectByLabel(t, page, row, "chairperson")
	if err := row.Locator("input[type='checkbox']").Check(); err != nil {
		t.Fatalf("check quoted checkbox: %v", err)
	}
	if err := row.Locator("button:has-text('Save')").Click(); err != nil {
		t.Fatalf("click save membership update: %v", err)
	}

	updated, err := ts.repo.GetUserByCommitteeAndUsername(context.Background(), "test-committee", "editable")
	if err != nil {
		t.Fatalf("load updated membership: %v", err)
	}
	if updated.Role != "chairperson" {
		t.Fatalf("expected updated role chairperson, got %q", updated.Role)
	}
	if !updated.Quoted {
		t.Fatalf("expected updated quoted=true")
	}
}

func TestAdminUpdateMembership_OAuthManagedRoleLockedButQuotedEditable(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")

	committeeID, err := ts.repo.GetCommitteeIDBySlug(context.Background(), "test-committee")
	if err != nil {
		t.Fatalf("get committee id: %v", err)
	}
	account, err := ts.repo.CreateOAuthAccount(context.Background(), "oidcuser", "OIDC User")
	if err != nil {
		t.Fatalf("create oauth account: %v", err)
	}
	if err := ts.repo.SyncOAuthCommitteeMemberships(context.Background(), account.ID, []model.OAuthDesiredMembership{
		{CommitteeID: committeeID, Role: "member"},
	}); err != nil {
		t.Fatalf("sync oauth membership: %v", err)
	}

	page := newPage(t)
	adminLogin(t, page, ts.URL)

	if err := page.Locator("a:has-text('Assign Accounts')").Click(); err != nil {
		t.Fatalf("click Assign Accounts: %v", err)
	}
	if err := page.WaitForURL(ts.URL + "/admin/committee/test-committee"); err != nil {
		t.Fatalf("wait for committee users page: %v", err)
	}

	row := page.Locator("tr").Filter(playwright.LocatorFilterOptions{HasText: "oidcuser"})
	if err := row.WaitFor(); err != nil {
		t.Fatalf("expected oidc user row: %v", err)
	}

	disabled, err := row.Locator("[role=combobox], button.input").First().IsDisabled()
	if err != nil {
		t.Fatalf("check role select disabled state: %v", err)
	}
	if !disabled {
		t.Fatalf("expected role select to be disabled for oauth-managed membership")
	}

	if err := row.Locator("input[type='checkbox']").Check(); err != nil {
		t.Fatalf("check quoted checkbox: %v", err)
	}
	if err := row.Locator("button:has-text('Save')").Click(); err != nil {
		t.Fatalf("click save membership update: %v", err)
	}

	updated, err := ts.repo.GetUserByCommitteeAndUsername(context.Background(), "test-committee", "oidcuser")
	if err != nil {
		t.Fatalf("load updated oauth-managed membership: %v", err)
	}
	if updated.Role != "member" {
		t.Fatalf("expected oauth-managed role to remain member, got %q", updated.Role)
	}
	if !updated.Quoted {
		t.Fatalf("expected quoted=true update to apply for oauth-managed membership")
	}
}
