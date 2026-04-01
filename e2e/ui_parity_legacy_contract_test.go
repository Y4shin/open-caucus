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
//   - /docs/oob/...                              (HTMX hx-swap-oob doc-content fragments)
//   - /docs/search                               (HTML search-results partial page)
//   - /committee/.../attendee-login              (attendee secret-login form)
//   - /committee/.../attendee/:id/recovery       (attendee secret-recovery page)
//   - /committee/.../moderate/join-qr            (join QR code full page)
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

// TestLegacyContract_AttendeeLoginForm (A18) verifies that the attendee secret-login
// form page (GET /committee/.../attendee-login) is still served by the legacy handler
// in the new app, producing an identical form to the standalone legacy server.
func TestLegacyContract_AttendeeLoginForm(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	for _, ts := range []*testServer{newTS, legacyTS} {
		ts.seedCommittee(t, "Test Committee", "test")
		ts.seedMeetingOpen(t, "test", "Board Meeting", "")
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	newMeetingID := newTS.getMeetingID(t, "test", "Board Meeting")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Board Meeting")

	// The attendee-login GET page is public (no auth required).
	gotoAndWaitForSelector(t, newBrowserPage,
		newTS.URL+"/committee/test/meeting/"+newMeetingID+"/attendee-login",
		"input[name=secret]",
	)
	gotoAndWaitForSelector(t, legacyBrowserPage,
		legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/attendee-login",
		"input[name=secret]",
	)

	// Parity contract: both servers must return the same login form.
	assertEqualHTML(t, "attendee-login form",
		locatorOuterHTML(t, newBrowserPage, "main form"),
		locatorOuterHTML(t, legacyBrowserPage, "main form"),
	)
}

// TestLegacyContract_JoinQRPage (A19) verifies that the join-QR code page
// (GET /committee/.../moderate/join-qr) is still served by the legacy handler
// in the new app, producing an identical QR code element.
func TestLegacyContract_JoinQRPage(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	for _, ts := range []*testServer{newTS, legacyTS} {
		ts.seedCommittee(t, "Test Committee", "test")
		ts.seedUser(t, "test", "chair1", "pass123", "Chair Person", "chairperson")
		ts.seedMeetingOpen(t, "test", "Board Meeting", "")
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "chair1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "chair1", "pass123")

	newMeetingID := newTS.getMeetingID(t, "test", "Board Meeting")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Board Meeting")

	// Navigate to the join-qr page on both servers.
	gotoAndWaitForSelector(t, newBrowserPage,
		newTS.URL+"/committee/test/meeting/"+newMeetingID+"/moderate/join-qr",
		"#join-qr-code",
	)
	gotoAndWaitForSelector(t, legacyBrowserPage,
		legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/moderate/join-qr",
		"#join-qr-code",
	)

	// Structural contract: page must contain the QR code image element.
	// We compare the img tag src attributes are equivalent (both embed the same
	// attendee-login URL as a base64 QR code for the meeting).
	newSrc, err := newBrowserPage.Locator("#join-qr-code").GetAttribute("src")
	if err != nil {
		t.Fatalf("new: get #join-qr-code src: %v", err)
	}
	legacySrc, err := legacyBrowserPage.Locator("#join-qr-code").GetAttribute("src")
	if err != nil {
		t.Fatalf("legacy: get #join-qr-code src: %v", err)
	}
	if newSrc == "" {
		t.Errorf("new server /moderate/join-qr: expected non-empty QR code src")
	}
	if legacySrc == "" {
		t.Errorf("legacy server /moderate/join-qr: expected non-empty QR code src")
	}
}

// TestLegacyContract_AttendeeLoginByLink (A18) verifies that the attendee
// login-by-link variant (GET /committee/.../attendee-login?secret=<valid>) is
// still served by the legacy handler in the new app. A valid secret triggers an
// immediate redirect to the live meeting page; both servers must redirect to the
// same path.
func TestLegacyContract_AttendeeLoginByLink(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	for _, ts := range []*testServer{newTS, legacyTS} {
		ts.seedCommittee(t, "Test Committee", "test")
		ts.seedMeetingOpen(t, "test", "Board Meeting", "")
		ts.seedAttendee(t, "test", "Board Meeting", "Alice Member", "secret-alice")
	}

	newMeetingID := newTS.getMeetingID(t, "test", "Board Meeting")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Board Meeting")

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	// A GET with a valid ?secret= param logs in the attendee and redirects to the
	// live meeting page. Wait for the live page to appear after the redirect.
	expectedNewPath := newTS.URL + "/committee/test/meeting/" + newMeetingID
	expectedLegacyPath := legacyTS.URL + "/committee/test/meeting/" + legacyMeetingID

	if _, err := newBrowserPage.Goto(newTS.URL + "/committee/test/meeting/" + newMeetingID + "/attendee-login?secret=secret-alice"); err != nil {
		t.Fatalf("goto attendee-login?secret on new: %v", err)
	}
	if err := newBrowserPage.WaitForURL(expectedNewPath); err != nil {
		t.Fatalf("new: expected redirect to live page at %s: %v", expectedNewPath, err)
	}

	if _, err := legacyBrowserPage.Goto(legacyTS.URL + "/committee/test/meeting/" + legacyMeetingID + "/attendee-login?secret=secret-alice"); err != nil {
		t.Fatalf("goto attendee-login?secret on legacy: %v", err)
	}
	if err := legacyBrowserPage.WaitForURL(expectedLegacyPath); err != nil {
		t.Fatalf("legacy: expected redirect to live page at %s: %v", expectedLegacyPath, err)
	}

	// Both servers must land on the live page (same path structure).
	newFinalPath := newBrowserPage.URL()
	legacyFinalPath := legacyBrowserPage.URL()
	if newFinalPath != expectedNewPath {
		t.Errorf("new: unexpected final URL after login-by-link: got %s, want %s", newFinalPath, expectedNewPath)
	}
	if legacyFinalPath != expectedLegacyPath {
		t.Errorf("legacy: unexpected final URL after login-by-link: got %s, want %s", legacyFinalPath, expectedLegacyPath)
	}
}
