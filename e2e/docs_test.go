//go:build e2e

package e2e_test

import (
	"strings"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

func TestDocsElementAndOOBRoute(t *testing.T) {
	ts := newTestServer(t)
	page := newPage(t)
	adminLogin(t, page, ts.URL)

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

	// /docs/oob/ is served by the legacy handler and returns an hx-swap-oob fragment
	// directly in the response body (not JS-rendered), so page.Content() is correct here.
	if _, err := page.Goto(ts.URL + "/docs/oob/index"); err != nil {
		t.Fatalf("goto docs oob index: %v", err)
	}
	content, err := page.Content()
	if err != nil {
		t.Fatalf("get docs oob content: %v", err)
	}
	if !strings.Contains(content, "hx-swap-oob=\"outerHTML\"") {
		t.Fatalf("expected oob docs payload to include hx-swap-oob, got: %s", content)
	}
}

func TestDocsDirectoryPathResolvesIndexAndShowsExpectedPath(t *testing.T) {
	ts := newTestServer(t)
	page := newPage(t)
	adminLogin(t, page, ts.URL)

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
	if err := page.Locator("summary:has-text('Browse Documentation')").First().WaitFor(); err != nil {
		t.Fatalf("expected docs browse accordion label: %v", err)
	}
}

func TestDocsSearchReturnsEmbeddedDocsHit(t *testing.T) {
	ts := newTestServer(t)
	page := newPage(t)
	adminLogin(t, page, ts.URL)

	if _, err := page.Goto(ts.URL + "/docs/search?q=receipt"); err != nil {
		t.Fatalf("goto docs search route: %v", err)
	}
	if err := page.Locator("#docs-search-results").WaitFor(); err != nil {
		t.Fatalf("expected docs search results container: %v", err)
	}
	if err := page.Locator("a:has-text('Receipts Vault and Receipt Verification')").First().WaitFor(); err != nil {
		t.Fatalf("expected receipt-verification docs hit: %v", err)
	}
}

func TestDocsSearchResultNavigatesToDocumentationPage(t *testing.T) {
	ts := newTestServer(t)
	page := newPage(t)
	adminLogin(t, page, ts.URL)

	if _, err := page.Goto(ts.URL + "/docs/search?q=receipt"); err != nil {
		t.Fatalf("goto docs search route: %v", err)
	}
	resultLink := page.Locator("a:has-text('Receipts Vault and Receipt Verification')").First()
	if err := resultLink.WaitFor(); err != nil {
		t.Fatalf("wait docs search result link: %v", err)
	}
	if err := resultLink.Click(); err != nil {
		t.Fatalf("click docs search result link: %v", err)
	}
	if err := page.Locator("h1:has-text('Receipts Vault and Receipt Verification')").First().WaitFor(); err != nil {
		t.Fatalf("expected docs detail heading after search navigation: %v", err)
	}
	if !strings.Contains(page.URL(), "/docs/05-public-verification/01-receipts-vault-and-receipt-verification") {
		t.Fatalf("expected docs detail url after search navigation, got %s", page.URL())
	}
}

func TestDocsAssetRouteServesEmbeddedCaptureDirectoryFile(t *testing.T) {
	ts := newTestServer(t)
	page := newPage(t)
	adminLogin(t, page, ts.URL)

	resp, err := page.Goto(ts.URL + "/docs/assets/captures/README.md")
	if err != nil {
		t.Fatalf("goto docs asset route: %v", err)
	}
	if resp == nil {
		t.Fatalf("expected non-nil response for docs asset route")
	}
	if status := resp.Status(); status != 200 {
		t.Fatalf("expected docs asset route status 200, got %d", status)
	}
	body, err := resp.Text()
	if err != nil {
		t.Fatalf("read docs asset response body: %v", err)
	}
	if !strings.Contains(body, "Documentation Captures") {
		t.Fatalf("expected embedded captures readme content, got: %s", body)
	}
}
