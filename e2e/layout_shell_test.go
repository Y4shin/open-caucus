//go:build e2e

package e2e_test

import (
	"strings"
	"testing"
	"time"
)

func TestShellLanguageSelector_SwitchesLocaleAndPersistsCookies(t *testing.T) {
	ts := newTestServer(t)

	page := newPage(t)
	adminLogin(t, page, ts.URL)
	gotoAndWaitForSelector(t, page, ts.URL+"/home", "footer button:has-text('DE')")

	if err := page.Locator("footer button:has-text('DE')").First().Click(); err != nil {
		t.Fatalf("click DE language button: %v", err)
	}

	waitUntil(t, 5*time.Second, func() (bool, error) {
		raw, err := page.Evaluate(`() => document.cookie`, nil)
		if err != nil {
			return false, err
		}
		cookie, ok := raw.(string)
		if !ok {
			t.Fatalf("language selector cookie payload was %T, want string", raw)
		}
		activeClass, err := page.Locator("footer button:has-text('DE')").First().GetAttribute("class")
		if err != nil {
			return false, err
		}
		return strings.Contains(cookie, "PARAGLIDE_LOCALE=de") &&
			strings.Contains(cookie, "locale=de") &&
			strings.Contains(activeClass, "btn-active"), nil
	}, "language selector to switch to de")

	if got := page.URL(); got != ts.URL+"/home" {
		t.Fatalf("expected locale switch to stay on %q, got %q", ts.URL+"/home", got)
	}
}
