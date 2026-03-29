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
	content, err := page.Content()
	if err != nil {
		t.Fatalf("get docs page content: %v", err)
	}
	if !strings.Contains(content, "id=\"app-docs-target\"") {
		t.Fatalf("expected docs element payload, url=%s content=%s", page.URL(), content)
	}

	if _, err := page.Goto(ts.URL + "/docs/oob/index"); err != nil {
		t.Fatalf("goto docs oob index: %v", err)
	}
	content, err = page.Content()
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
	content, err := page.Content()
	if err != nil {
		t.Fatalf("get docs directory content: %v", err)
	}
	if !strings.Contains(content, "Chairperson") {
		t.Fatalf("expected chairperson docs title in payload, got: %s", content)
	}
	if !strings.Contains(content, "Path: Chairperson") {
		t.Fatalf("expected docs path display for directory index, got: %s", content)
	}
	if !strings.Contains(content, "Browse Documentation") {
		t.Fatalf("expected docs browse accordion label, got: %s", content)
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
