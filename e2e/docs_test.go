//go:build e2e

package e2e_test

import (
	"strings"
	"testing"
)

func TestDocsElementAndOOBRoute(t *testing.T) {
	ts := newTestServer(t)
	page := newPage(t)
	adminLogin(t, page, ts.URL)

	if _, err := page.Goto(ts.URL + "/docs/index"); err != nil {
		t.Fatalf("goto docs index: %v", err)
	}
	// SPA renders the docs overlay as a section with id="app-docs-target"
	if err := page.Locator("#app-docs-target").WaitFor(); err != nil {
		t.Fatalf("expected docs overlay to render with id=app-docs-target: %v", err)
	}

	// Navigate to a different docs path within the same session; the overlay
	// should remain in the DOM (no full-page reload).
	urlBefore := page.URL()
	if _, err := page.Goto(ts.URL + "/docs/index"); err != nil {
		t.Fatalf("second goto docs index: %v", err)
	}
	if err := page.Locator("#app-docs-target").WaitFor(); err != nil {
		t.Fatalf("expected docs overlay present after re-navigation: %v", err)
	}
	_ = urlBefore
}

func TestDocsDirectoryPathResolvesIndexAndShowsExpectedPath(t *testing.T) {
	ts := newTestServer(t)
	page := newPage(t)
	adminLogin(t, page, ts.URL)

	if _, err := page.Goto(ts.URL + "/docs/03-chairperson"); err != nil {
		t.Fatalf("goto chairperson docs directory: %v", err)
	}
	if err := page.Locator("text=Chairperson").First().WaitFor(); err != nil {
		t.Fatalf("expected chairperson docs title in overlay: %v", err)
	}
	if err := page.Locator("text=Path: Chairperson").WaitFor(); err != nil {
		t.Fatalf("expected docs path display for directory index: %v", err)
	}
	if err := page.Locator("text=Browse Documentation").WaitFor(); err != nil {
		t.Fatalf("expected docs browse accordion label: %v", err)
	}
	content, err := page.Content()
	if err != nil {
		t.Fatalf("get docs directory content: %v", err)
	}
	if !strings.Contains(content, "Chairperson") {
		t.Fatalf("expected chairperson docs title in payload, got: %s", content)
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
	if err := page.Locator("h1:has-text('Receipts Vault and Receipt Verification')").WaitFor(); err != nil {
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
