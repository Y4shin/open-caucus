//go:build e2e

package e2e_test

import (
	"fmt"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

func assertNoLegacyUIAttrs(t *testing.T, page playwright.Page, label string) {
	t.Helper()

	raw, err := page.Evaluate(`() => {
		const selectors = [
			"[hx-get]",
			"[hx-post]",
			"[hx-put]",
			"[hx-patch]",
			"[hx-delete]",
			"[hx-target]",
			"[hx-swap]",
			"[hx-swap-oob]",
			"[hx-trigger]",
			"[hx-vals]",
			"[hx-include]",
			"[hx-confirm]",
			"[hx-ext]",
			"[sse-swap]",
			"[data-hx-get]",
			"[data-hx-post]",
			"[data-hx-put]",
			"[data-hx-patch]",
			"[data-hx-delete]",
			"[data-hx-target]",
			"[data-hx-swap]",
			"[data-hx-swap-oob]",
			"[data-hx-trigger]",
			"[data-hx-vals]",
			"[data-hx-include]",
			"[data-hx-confirm]",
			"[data-hx-ext]",
			"[data-sse-swap]"
		];

		const matches = [];
		for (const selector of selectors) {
			for (const node of document.querySelectorAll(selector)) {
				if (matches.length >= 25) break;
				matches.push({
					selector,
					tag: node.tagName.toLowerCase(),
					id: node.id || "",
					className: typeof node.className === "string" ? node.className : ""
				});
			}
		}

		return {
			hasWindowHtmx: typeof window.htmx !== "undefined",
			htmxScriptSources: Array.from(document.scripts)
				.map((script) => script.src || "")
				.filter((src) => src.toLowerCase().includes("htmx")),
			matches
		};
	}`, nil)
	if err != nil {
		t.Fatalf("evaluate legacy ui attrs on %s: %v", label, err)
	}

	result, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("unexpected legacy ui attr result on %s: %#v", label, raw)
	}

	if hasWindowHtmx, _ := result["hasWindowHtmx"].(bool); hasWindowHtmx {
		t.Fatalf("%s exposed window.htmx after the SPA migration", label)
	}

	if sources, ok := result["htmxScriptSources"].([]any); ok && len(sources) > 0 {
		t.Fatalf("%s still loads HTMX script sources: %#v", label, sources)
	}

	if matches, ok := result["matches"].([]any); ok && len(matches) > 0 {
		t.Fatalf("%s still rendered legacy UI attributes: %#v", label, matches)
	}
}

func TestSPA_PagesDoNotRenderLegacyHXAttributes(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")

	adminLoginPage := newPage(t)
	if _, err := adminLoginPage.Goto(ts.URL + "/admin/login"); err != nil {
		t.Fatalf("goto admin login: %v", err)
	}
	if err := adminLoginPage.Locator("input[name='username']").WaitFor(); err != nil {
		t.Fatalf("wait admin login form: %v", err)
	}
	assertNoLegacyUIAttrs(t, adminLoginPage, "admin login")

	adminPage := newPage(t)
	adminLogin(t, adminPage, ts.URL)
	if err := adminPage.Locator("#create-committee-form").WaitFor(); err != nil {
		t.Fatalf("wait admin dashboard: %v", err)
	}
	assertNoLegacyUIAttrs(t, adminPage, "admin dashboard")

	memberPage := newPage(t)
	userLogin(t, memberPage, ts.URL, "test-committee", "chair1", "pass123")
	if err := memberPage.Locator("[data-testid='committee-create-form']").WaitFor(); err != nil {
		t.Fatalf("wait committee page: %v", err)
	}
	assertNoLegacyUIAttrs(t, memberPage, "committee page")

	if _, err := memberPage.Goto(moderateURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto moderate page: %v", err)
	}
	if err := memberPage.Locator("#moderate-sse-root").WaitFor(); err != nil {
		t.Fatalf("wait moderate page: %v", err)
	}
	assertNoLegacyUIAttrs(t, memberPage, "moderate page")

	guestPage := newPage(t)
	joinPageURL := joinURL(ts.URL, "test-committee", meetingID)
	if _, err := guestPage.Goto(joinPageURL); err != nil {
		t.Fatalf("goto join page: %v", err)
	}
	if err := guestPage.Locator("main").WaitFor(); err != nil {
		t.Fatalf("wait join page: %v", err)
	}
	assertNoLegacyUIAttrs(t, guestPage, fmt.Sprintf("join page %s", joinPageURL))
}
