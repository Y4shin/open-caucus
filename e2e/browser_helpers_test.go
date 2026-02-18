//go:build e2e

package e2e_test

import (
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

// newPage launches a Chromium browser, creates an isolated browser context, and
// returns a new page. All resources are registered with t.Cleanup.
// If the Playwright driver or Chromium browser is not installed the test is
// skipped — run 'task playwright:install' to enable E2E tests.
func newPage(t *testing.T) playwright.Page {
	t.Helper()

	if pw == nil {
		t.Skip("playwright driver not installed — run 'task playwright:install'")
	}

	browser, err := pw.Chromium.Launch()
	if err != nil {
		t.Skipf("chromium not installed (run 'task playwright:install'): %v", err)
	}
	t.Cleanup(func() { browser.Close() })

	ctx, err := browser.NewContext()
	if err != nil {
		t.Fatalf("new browser context: %v", err)
	}
	t.Cleanup(func() { ctx.Close() })

	page, err := ctx.NewPage()
	if err != nil {
		t.Fatalf("new page: %v", err)
	}
	return page
}

// adminLogin navigates to /admin/login and authenticates with the test admin key.
func adminLogin(t *testing.T, page playwright.Page, baseURL string) {
	t.Helper()
	if _, err := page.Goto(baseURL + "/admin/login"); err != nil {
		t.Fatalf("goto /admin/login: %v", err)
	}
	if err := page.Locator("input[name=admin_key]").Fill(testAdminKey); err != nil {
		t.Fatalf("fill admin_key: %v", err)
	}
	if err := page.Locator("button[type=submit]").Click(); err != nil {
		t.Fatalf("click submit: %v", err)
	}
	if err := page.WaitForURL(baseURL + "/admin"); err != nil {
		t.Fatalf("wait for /admin: %v", err)
	}
}

// userLogin navigates to / and authenticates as the given user in the given committee.
// It waits for the redirect to /committee/{slug}.
func userLogin(t *testing.T, page playwright.Page, baseURL, committee, username, password string) {
	t.Helper()
	if _, err := page.Goto(baseURL + "/"); err != nil {
		t.Fatalf("goto /: %v", err)
	}
	if err := page.Locator("input[name=committee]").Fill(committee); err != nil {
		t.Fatalf("fill committee: %v", err)
	}
	if err := page.Locator("input[name=username]").Fill(username); err != nil {
		t.Fatalf("fill username: %v", err)
	}
	if err := page.Locator("input[name=password]").Fill(password); err != nil {
		t.Fatalf("fill password: %v", err)
	}
	if err := page.Locator("button[type=submit]").Click(); err != nil {
		t.Fatalf("click submit: %v", err)
	}
	if err := page.WaitForURL(baseURL + "/committee/" + committee); err != nil {
		t.Fatalf("wait for /committee/%s: %v", committee, err)
	}
}
