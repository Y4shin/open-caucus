//go:build e2e

package e2e_test

import (
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

func TestAdminLogin(t *testing.T) {
	ts := newTestServer(t)
	page := newPage(t)

	adminLogin(t, page, ts.URL)

	if err := page.Locator("h1:has-text('Admin Dashboard')").WaitFor(); err != nil {
		t.Fatalf("expected Admin Dashboard heading: %v", err)
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
	deleteBtn := page.Locator("tr").Filter(playwright.LocatorFilterOptions{HasText: "delete-me"}).Locator("button[type=submit]")
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

func TestAdminCreateUser(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")

	page := newPage(t)
	adminLogin(t, page, ts.URL)

	// Navigate to the committee users page
	if err := page.Locator("a:has-text('Manage Users')").Click(); err != nil {
		t.Fatalf("click Manage Users: %v", err)
	}
	if err := page.WaitForURL(ts.URL + "/admin/committee/test-committee"); err != nil {
		t.Fatalf("wait for committee users page: %v", err)
	}

	urlBefore := page.URL()

	// Fill the create user form
	if err := page.Locator("input[name=username]").Fill("newuser"); err != nil {
		t.Fatalf("fill username: %v", err)
	}
	if err := page.Locator("input[name=password]").Fill("password123"); err != nil {
		t.Fatalf("fill password: %v", err)
	}
	if err := page.Locator("input[name=full_name]").Fill("New User"); err != nil {
		t.Fatalf("fill full_name: %v", err)
	}
	roleValues := []string{"member"}
	if _, err := page.Locator("select[name=role]").SelectOption(playwright.SelectOptionValues{Values: &roleValues}); err != nil {
		t.Fatalf("select role: %v", err)
	}
	if err := page.Locator("button[type=submit]").First().Click(); err != nil {
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

	// Navigate to the committee users page
	if err := page.Locator("a:has-text('Manage Users')").Click(); err != nil {
		t.Fatalf("click Manage Users: %v", err)
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

	// Click Delete for the specific user row
	deleteBtn := page.Locator("tr").Filter(playwright.LocatorFilterOptions{HasText: "todelete"}).Locator("button[type=submit]")
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
