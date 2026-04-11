//go:build e2e

package e2e_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

func TestDocsElementRendersOnNativeRoute(t *testing.T) {
	ts := newTestServer(t)
	page := newPage(t)

	// The SPA renders the DocsOverlay with id="app-docs-target" after JavaScript
	// executes, so we use a locator to wait for it rather than checking raw HTML.
	if _, err := page.Goto(ts.URL + "/docs/index"); err != nil {
		t.Fatalf("goto docs index: %v", err)
	}
	if err := page.Locator("#app-docs-target[data-docs-open]").First().WaitFor(
		playwright.LocatorWaitForOptions{State: playwright.WaitForSelectorStateAttached},
	); err != nil {
		t.Fatalf("expected docs element with id=app-docs-target and data-docs-open: %v", err)
	}
}

func TestDocsDirectoryPathResolvesIndexAndShowsExpectedPath(t *testing.T) {
	ts := newTestServer(t)
	page := newPage(t)

	if _, err := page.Goto(ts.URL + "/docs/03-chairperson"); err != nil {
		t.Fatalf("goto chairperson docs directory: %v", err)
	}
	// Wait for SPA to render the docs overlay — DocsOverlay renders h2 with the
	// page title, a "Path: <display>" paragraph, and the "Browse Documentation"
	// accordion summary.
	if err := page.Locator("h2:has-text('Chairperson')").First().WaitFor(); err != nil {
		t.Fatalf("expected chairperson docs title heading: %v", err)
	}
	if err := page.Locator("p:has-text('Path: Chairperson')").First().WaitFor(); err != nil {
		t.Fatalf("expected docs path display for directory index: %v", err)
	}
	if err := page.Locator("button:has-text('Browse Documentation')").First().WaitFor(); err != nil {
		t.Fatalf("expected docs browse accordion label: %v", err)
	}
}

func TestDocsAssetRouteServesEmbeddedCaptureDirectoryFile(t *testing.T) {
	ts := newTestServer(t)

	// Use a direct HTTP request instead of Playwright navigation because the SPA
	// client-side router intercepts /docs/assets/ URLs in the browser.
	resp, err := http.Get(ts.URL + "/docs/assets/captures/README.md")
	if err != nil {
		t.Fatalf("GET docs asset route: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected docs asset route status 200, got %d", resp.StatusCode)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read docs asset response body: %v", err)
	}
	body := string(bodyBytes)
	if !strings.Contains(body, "Documentation Captures") {
		t.Fatalf("expected embedded captures readme content, got: %s", body)
	}
}
