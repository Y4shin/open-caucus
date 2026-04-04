//go:build e2e

package e2e_test

import (
	"strings"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

func TestShellLanguageSelector_SwitchesLocaleAndPersistsCookies(t *testing.T) {
	ts := newTestServer(t)

	page := newPage(t)
	adminLogin(t, page, ts.URL)
	gotoAndWaitForSelector(t, page, ts.URL+"/home", "footer button:has-text('DE')")

	// setLocale() triggers window.location.reload() — wait for that navigation to
	// settle before asserting anything about the post-switch state.
	if _, err := page.ExpectNavigation(func() error {
		return page.Locator("footer button:has-text('DE')").First().Click()
	}, playwright.PageExpectNavigationOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		t.Fatalf("click DE and wait for page reload: %v", err)
	}

	// Wait for the footer to be rendered in the reloaded page.
	if err := page.Locator("footer button:has-text('DE')").First().WaitFor(); err != nil {
		t.Fatalf("wait for footer after reload: %v", err)
	}

	// The server middleware replaces %paraglide.lang% in the HTML shell using the
	// PARAGLIDE_LOCALE cookie, so the html[lang] attribute is the most reliable
	// signal that the locale was applied.
	lang, err := page.Evaluate(`() => document.documentElement.lang`, nil)
	if err != nil {
		t.Fatalf("get html lang attribute: %v", err)
	}
	if lang != "de" {
		t.Fatalf("expected html[lang]=de after locale switch, got %q", lang)
	}

	// DE button should be marked active; EN should not.
	deClass, err := page.Locator("footer button:has-text('DE')").First().GetAttribute("class")
	if err != nil {
		t.Fatalf("get DE button class: %v", err)
	}
	if !strings.Contains(deClass, "btn-active") {
		t.Fatalf("expected DE button to have btn-active class, got %q", deClass)
	}

	enClass, err := page.Locator("footer button:has-text('EN')").First().GetAttribute("class")
	if err != nil {
		t.Fatalf("get EN button class: %v", err)
	}
	if strings.Contains(enClass, "btn-active") {
		t.Fatalf("expected EN button to NOT have btn-active class after switching to DE, got %q", enClass)
	}

	if got := page.URL(); got != ts.URL+"/home" {
		t.Fatalf("expected locale switch to stay on %q, got %q", ts.URL+"/home", got)
	}
}
