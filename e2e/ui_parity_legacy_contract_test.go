//go:build e2e

// Package e2e_test contains end-to-end browser tests.
//
// This file contains legacy-fallback contract tests. These tests explicitly
// document route families that are still intentionally served by the legacy
// HTMX/Templ handler inside the new (SPA) application. For each such route
// family the tests verify that:
//
//  1. The new (SPA) application returns the same response as the standalone
//     legacy server — confirming the route is truly delegated to the legacy
//     handler rather than being re-implemented.
//  2. The response still contains the key structural elements that downstream
//     HTMX wiring depends on (e.g. hx-swap-oob attributes, expected containers).
//
// When a route family is ported away from the legacy handler, remove its test
// from this file and add direct SPA parity or behavioural coverage instead.
//
// Currently documented legacy-backed route families:
//   - /docs/oob/...        (HTMX hx-swap-oob doc-content fragments)
//   - /docs/search         (HTML search-results partial page)
package e2e_test

import (
	"strings"
	"testing"
)

// TestLegacyContract_DocsOOBFragment (A17) verifies that the docs OOB fragment
// endpoint (/docs/oob/<slug>) is still served by the legacy handler in the new
// app, producing identical HTML to the standalone legacy server.
func TestLegacyContract_DocsOOBFragment(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	adminLogin(t, newBrowserPage, newTS.URL)
	adminLogin(t, legacyBrowserPage, legacyTS.URL)

	// Navigate to the OOB docs index fragment on both servers.
	// The legacy handler returns a raw hx-swap-oob HTMX fragment (not a full page),
	// so page.Content() gives us the full raw response body.
	if _, err := newBrowserPage.Goto(newTS.URL + "/docs/oob/index"); err != nil {
		t.Fatalf("goto docs oob on new: %v", err)
	}
	if _, err := legacyBrowserPage.Goto(legacyTS.URL + "/docs/oob/index"); err != nil {
		t.Fatalf("goto docs oob on legacy: %v", err)
	}

	newContent, err := newBrowserPage.Content()
	if err != nil {
		t.Fatalf("get content on new: %v", err)
	}
	legacyContent, err := legacyBrowserPage.Content()
	if err != nil {
		t.Fatalf("get content on legacy: %v", err)
	}

	// Structural contract: the OOB fragment must contain the hx-swap-oob attribute
	// so that HTMX can splice it into the right DOM target.
	if !strings.Contains(newContent, "hx-swap-oob") {
		t.Errorf("new server /docs/oob/index: expected hx-swap-oob in response, got none")
	}
	if !strings.Contains(legacyContent, "hx-swap-oob") {
		t.Errorf("legacy server /docs/oob/index: expected hx-swap-oob in response, got none")
	}

	// Parity contract: both servers must return the same OOB fragment content.
	assertEqualHTML(t, "docs oob/index fragment", newContent, legacyContent)
}

// TestLegacyContract_DocsSearchPartial (A17) verifies that the docs search
// partial endpoint (/docs/search) is still served by the legacy handler in the
// new app, producing identical HTML to the standalone legacy server.
func TestLegacyContract_DocsSearchPartial(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	adminLogin(t, newBrowserPage, newTS.URL)
	adminLogin(t, legacyBrowserPage, legacyTS.URL)

	// Search for "receipt" — both servers embed the same docs content, so the
	// results must be identical.
	if _, err := newBrowserPage.Goto(newTS.URL + "/docs/search?q=receipt"); err != nil {
		t.Fatalf("goto docs search on new: %v", err)
	}
	if _, err := legacyBrowserPage.Goto(legacyTS.URL + "/docs/search?q=receipt"); err != nil {
		t.Fatalf("goto docs search on legacy: %v", err)
	}

	// Structural contract: the search response must include the results container.
	if err := newBrowserPage.Locator("#docs-search-results").WaitFor(); err != nil {
		t.Fatalf("new server: expected #docs-search-results: %v", err)
	}
	if err := legacyBrowserPage.Locator("#docs-search-results").WaitFor(); err != nil {
		t.Fatalf("legacy server: expected #docs-search-results: %v", err)
	}

	// Parity contract: both servers must return the same search results container.
	assertEqualHTML(t, "docs search results container",
		locatorOuterHTML(t, newBrowserPage, "#docs-search-results"),
		locatorOuterHTML(t, legacyBrowserPage, "#docs-search-results"),
	)
}
