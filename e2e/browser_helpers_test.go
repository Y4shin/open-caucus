//go:build e2e

package e2e_test

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

const (
	defaultE2ETimeoutMs           = 10000
	defaultE2ENavigationTimeoutMs = 15000
)

// newPage launches a Chromium browser, creates an isolated browser context, and
// returns a new page. All resources are registered with t.Cleanup.
// If the Playwright driver or Chromium browser is not installed the test is
// skipped — run 'task playwright:install' to enable E2E tests.
func newPage(t *testing.T) playwright.Page {
	t.Helper()

	if pw == nil {
		t.Skip(fmt.Sprintf("playwright driver not installed — run 'task playwright:install'\n%s", pwErr))
	}

	launchOpts := playwright.BrowserTypeLaunchOptions{}
	if os.Getenv("PLAYWRIGHT_HEADED") == "1" {
		launchOpts.Headless = playwright.Bool(false)
	}
	if slowMoMs, err := strconv.ParseFloat(os.Getenv("PLAYWRIGHT_SLOW_MO_MS"), 64); err == nil && slowMoMs > 0 {
		launchOpts.SlowMo = playwright.Float(slowMoMs)
	}

	browser, err := pw.Chromium.Launch(launchOpts)
	if err != nil {
		t.Skipf("chromium not installed (run 'task playwright:install'): %v", err)
	}
	t.Cleanup(func() { browser.Close() })

	ctx, err := browser.NewContext()
	if err != nil {
		t.Fatalf("new browser context: %v", err)
	}
	t.Cleanup(func() { ctx.Close() })
	ctx.SetDefaultTimeout(defaultE2ETimeoutMs)
	ctx.SetDefaultNavigationTimeout(defaultE2ENavigationTimeoutMs)

	page, err := ctx.NewPage()
	if err != nil {
		t.Fatalf("new page: %v", err)
	}
	return page
}

func gotoAndWaitForInput(t *testing.T, page playwright.Page, url, selector string) {
	t.Helper()

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if _, err := page.Goto(url); err != nil {
			lastErr = fmt.Errorf("goto %s: %w", url, err)
			continue
		}
		if err := page.WaitForURL(url); err != nil {
			lastErr = fmt.Errorf("wait for %s: %w", url, err)
			continue
		}
		if err := page.Locator(selector).First().WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(defaultE2ETimeoutMs),
		}); err != nil {
			lastErr = fmt.Errorf("wait for %s on %s: %w", selector, url, err)
			continue
		}
		return
	}

	currentURL := page.URL()
	if currentURL == "" {
		currentURL = "<empty>"
	}
	t.Fatalf("open %s with %s: %v (current URL: %s)", url, selector, lastErr, currentURL)
}

// adminLogin navigates to /admin/login and authenticates with the test admin credentials.
func adminLogin(t *testing.T, page playwright.Page, baseURL string) {
	t.Helper()
	gotoAndWaitForInput(t, page, baseURL+"/admin/login", "input[name=username]")
	if err := page.Locator("input[name=username]").Fill(testAdminUsername); err != nil {
		t.Fatalf("fill username: %v", err)
	}
	if err := page.Locator("input[name=password]").Fill(testAdminPassword); err != nil {
		t.Fatalf("fill password: %v", err)
	}
	if err := page.Locator("input[name=password]").Press("Enter"); err != nil {
		t.Fatalf("submit admin login: %v", err)
	}
	if err := page.WaitForURL(baseURL + "/admin"); err != nil {
		t.Fatalf("wait for /admin: %v", err)
	}
}

// userLogin navigates to /login and authenticates as the given user, then navigates to
// /committee/{committee}. After login the app redirects to /home; this helper
// proceeds to the requested committee page so callers land where they expect.
func userLogin(t *testing.T, page playwright.Page, baseURL, committee, username, password string) {
	t.Helper()
	gotoAndWaitForInput(t, page, baseURL+"/login", "input[name=username]")
	if err := page.Locator("input[name=username]").Fill(username); err != nil {
		t.Fatalf("fill username: %v", err)
	}
	if err := page.Locator("input[name=password]").Fill(password); err != nil {
		t.Fatalf("fill password: %v", err)
	}
	if err := page.Locator("input[name=password]").Press("Enter"); err != nil {
		t.Fatalf("submit user login: %v", err)
	}
	if err := page.WaitForURL(baseURL + "/home"); err != nil {
		t.Fatalf("wait for /home after login: %v", err)
	}
	if _, err := page.Goto(baseURL + "/committee/" + committee); err != nil {
		t.Fatalf("goto /committee/%s: %v", committee, err)
	}
}

func openModerateLeftTab(t *testing.T, page playwright.Page, tabName string) {
	t.Helper()
	controls := page.Locator("#moderate-left-controls")
	if err := controls.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(30000)}); err != nil {
		t.Fatalf("wait moderate left controls: %v", err)
	}
	tab := page.Locator("#moderate-left-controls [data-moderate-left-tab='" + tabName + "']")
	if err := tab.First().WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(30000)}); err != nil {
		t.Fatalf("wait moderate left tab %q: %v", tabName, err)
	}
	panel := page.Locator("#moderate-left-panel-" + tabName)
	if visible, err := panel.IsVisible(); err == nil && visible {
		return
	}
	if err := tab.First().Click(); err != nil {
		if _, evalErr := tab.First().Evaluate("el => { el.click(); return true; }", nil); evalErr != nil {
			t.Fatalf("click moderate left tab %q: %v (eval fallback: %v)", tabName, err, evalErr)
		}
	}
	if err := panel.WaitFor(); err != nil {
		t.Fatalf("wait moderate left panel %q: %v", tabName, err)
	}
}

func openModerateAgendaEditor(t *testing.T, page playwright.Page) {
	t.Helper()
	openModerateLeftTab(t, page, "agenda")
	if err := page.Locator("#agenda-point-list-container").First().WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(defaultE2ETimeoutMs),
	}); err == nil {
		return
	}
	isOpen, err := page.Locator("#moderate-agenda-edit-dialog[open]").IsVisible()
	if err == nil && isOpen {
		if err := page.Locator("#agenda-point-list-container").WaitFor(); err != nil {
			t.Fatalf("wait agenda-point-list-container in dialog: %v", err)
		}
		return
	}
	openButton := page.Locator("#moderate-left-panel-agenda [data-manage-dialog-open][aria-controls='moderate-agenda-edit-dialog']")
	if err := openButton.First().Click(); err != nil {
		t.Fatalf("open moderate agenda editor: %v", err)
	}
	if err := page.Locator("#moderate-agenda-edit-dialog[open]").WaitFor(); err != nil {
		t.Fatalf("wait moderate agenda dialog open: %v", err)
	}
	if err := page.Locator("#agenda-point-list-container").WaitFor(); err != nil {
		t.Fatalf("wait agenda-point-list-container in dialog: %v", err)
	}
}

func expectAlertContaining(t *testing.T, page playwright.Page, text string) {
	t.Helper()
	if err := page.Locator("[role=alert]").Filter(playwright.LocatorFilterOptions{HasText: text}).First().WaitFor(); err != nil {
		t.Fatalf("expected alert containing %q: %v", text, err)
	}
}
