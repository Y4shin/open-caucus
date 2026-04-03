//go:build e2e

package e2e_test

import (
	"strings"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

func openModeratePageForHelpTests(t *testing.T) (*testServer, playwright.Page, string) {
	t.Helper()

	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(moderateURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto moderate page: %v", err)
	}
	if err := page.Locator("#moderate-sse-root").WaitFor(); err != nil {
		t.Fatalf("wait moderate page root: %v", err)
	}

	return ts, page, meetingID
}

func assertHelpButtonOpensDocsPanel(
	t *testing.T,
	page playwright.Page,
	buttonAriaLabel string,
	expectedTitle string,
	expectedPathLine string,
) {
	t.Helper()

	button := page.Locator("button[aria-label='" + buttonAriaLabel + "']").First()
	if err := button.WaitFor(); err != nil {
		t.Fatalf("wait help button %q: %v", buttonAriaLabel, err)
	}
	if err := button.Click(); err != nil {
		t.Fatalf("click help button %q: %v", buttonAriaLabel, err)
	}

	panel := page.Locator("#app-docs-target[data-docs-open='1']")
	if err := panel.WaitFor(); err != nil {
		t.Fatalf("wait opened docs panel for %q: %v", buttonAriaLabel, err)
	}
	if err := panel.Locator("h2:has-text('" + expectedTitle + "')").WaitFor(); err != nil {
		t.Fatalf("expected docs title %q for help button %q: %v", expectedTitle, buttonAriaLabel, err)
	}
	if err := panel.Locator("p:has-text('Path: " + expectedPathLine + "')").WaitFor(); err != nil {
		t.Fatalf("expected docs path line %q for help button %q: %v", expectedPathLine, buttonAriaLabel, err)
	}
	if page.URL() == "" || page.URL() == "about:blank" {
		t.Fatalf("help button %q did not update the browser URL", buttonAriaLabel)
	}
	if strings.Contains(page.URL(), "/docs/") {
		t.Fatalf("help button %q replaced the current page with a docs route: %s", buttonAriaLabel, page.URL())
	}
	if !strings.Contains(page.URL(), "docs=") {
		t.Fatalf("help button %q did not open docs in overlay mode: %s", buttonAriaLabel, page.URL())
	}
}

func openHomePageForHelpTests(t *testing.T) (*testServer, playwright.Page) {
	t.Helper()

	ts := newTestServer(t)
	page := newPage(t)
	adminLogin(t, page, ts.URL)
	if _, err := page.Goto(ts.URL + "/home"); err != nil {
		t.Fatalf("goto home page: %v", err)
	}
	if err := page.Locator("footer button:has-text('Help')").WaitFor(); err != nil {
		t.Fatalf("wait home help button: %v", err)
	}
	return ts, page
}

func TestModerateHelpButton_OpensAgendaDocumentation(t *testing.T) {
	_, page, _ := openModeratePageForHelpTests(t)
	openModerateLeftTab(t, page, "agenda")

	assertHelpButtonOpensDocsPanel(
		t,
		page,
		"Open agenda help",
		"Agenda Management and Import",
		"Chairperson / Agenda Management and Import",
	)
}

func TestModerateHelpButton_OpensSpeakersDocumentation(t *testing.T) {
	_, page, _ := openModeratePageForHelpTests(t)

	assertHelpButtonOpensDocsPanel(
		t,
		page,
		"Open speakers help",
		"Speakers Moderator and Quotation",
		"Chairperson / Speakers Moderator and Quotation",
	)
}

func TestModerateHelpButton_OpensAddSpeakerDocumentation(t *testing.T) {
	_, page, _ := openModeratePageForHelpTests(t)

	assertHelpButtonOpensDocsPanel(
		t,
		page,
		"Open add-speaker help",
		"Speakers Moderator and Quotation",
		"Chairperson / Speakers Moderator and Quotation",
	)
}

func TestHelpOverlay_PreservesUnderlyingModerationPage(t *testing.T) {
	ts, page, meetingID := openModeratePageForHelpTests(t)
	baseURL := moderateURL(ts.URL, "test-committee", meetingID)

	if err := page.Locator("footer button:has-text('Help')").First().Click(); err != nil {
		t.Fatalf("click footer help: %v", err)
	}
	if err := page.Locator("#app-docs-target[data-docs-open='1']").WaitFor(); err != nil {
		t.Fatalf("wait help overlay: %v", err)
	}
	if err := page.Locator("#moderate-left-controls").WaitFor(); err != nil {
		t.Fatalf("moderation controls should remain visible under help overlay: %v", err)
	}
	if !strings.HasPrefix(page.URL(), baseURL) {
		t.Fatalf("help overlay should stay on the moderation page, got %s want prefix %s", page.URL(), baseURL)
	}
	if strings.Contains(page.URL(), "/docs/") {
		t.Fatalf("help overlay unexpectedly navigated to a docs route: %s", page.URL())
	}
}

func TestHelpOverlay_RendersMarkdownFormatting(t *testing.T) {
	ts, page := openHomePageForHelpTests(t)

	if _, err := page.Goto(ts.URL + "/home?docs=guides/01-capture"); err != nil {
		t.Fatalf("goto home with docs overlay params: %v", err)
	}
	if err := page.Locator("#app-docs-target .docs-markdown").WaitFor(); err != nil {
		t.Fatalf("wait docs markdown content: %v", err)
	}

	raw, err := page.Evaluate(`() => {
		const root = document.querySelector('#app-docs-target .docs-markdown');
		const heading = root?.querySelector('h1');
		const list = root?.querySelector('ul');
		const codeBlock = root?.querySelector('pre');
		if (!root || !heading || !list || !codeBlock) return null;
		const rootStyle = getComputedStyle(root);
		const headingStyle = getComputedStyle(heading);
		const listStyle = getComputedStyle(list);
		return {
			rootFontSize: rootStyle.fontSize,
			rootLineHeight: rootStyle.lineHeight,
			headingWeight: headingStyle.fontWeight,
			listStyleType: listStyle.listStyleType
		};
	}`, nil)
	if err != nil {
		t.Fatalf("evaluate markdown formatting styles: %v", err)
	}
	styles, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("unexpected markdown formatting evaluation result: %#v", raw)
	}
	if styles["headingWeight"] == "400" || styles["headingWeight"] == "500" {
		t.Fatalf("expected strong heading styling, got font-weight=%v", styles["headingWeight"])
	}
	if styles["rootFontSize"] == "16px" {
		t.Fatalf("expected markdown typography to change the base font size, got font-size=%v", styles["rootFontSize"])
	}
	if styles["rootLineHeight"] == "normal" {
		t.Fatalf("expected markdown typography to set an explicit line height, got line-height=%v", styles["rootLineHeight"])
	}
	if styles["listStyleType"] != "disc" {
		t.Fatalf("expected markdown unordered list bullets, got list-style-type=%v", styles["listStyleType"])
	}
}

func TestHelpOverlay_SearchStaysOnCurrentPageAndLoadsResults(t *testing.T) {
	ts, page := openHomePageForHelpTests(t)
	baseURL := ts.URL + "/home"

	if err := page.Locator("footer button:has-text('Help')").First().Click(); err != nil {
		t.Fatalf("click footer help: %v", err)
	}
	panel := page.Locator("#app-docs-target[data-docs-open='1']")
	if err := panel.WaitFor(); err != nil {
		t.Fatalf("wait docs overlay: %v", err)
	}
	if err := panel.Locator("input[type='search']").Fill("vote"); err != nil {
		t.Fatalf("fill docs search query: %v", err)
	}
	if err := panel.Locator("input[type='search']").Press("Enter"); err != nil {
		t.Fatalf("submit docs search query: %v", err)
	}
	results := panel.Locator("#docs-search-results li")
	if err := results.First().WaitFor(); err != nil {
		t.Fatalf("wait docs search results: %v", err)
	}
	if err := results.Filter(playwright.LocatorFilterOptions{HasText: "Vote Moderation Open and Secret"}).First().Locator("a").Click(); err != nil {
		t.Fatalf("open vote moderation search result: %v", err)
	}
	if err := panel.Locator("h2:has-text('Vote Moderation Open and Secret')").WaitFor(); err != nil {
		t.Fatalf("wait opened docs title after search result click: %v", err)
	}
	if !strings.HasPrefix(page.URL(), baseURL) {
		t.Fatalf("docs search should keep the current page path, got %s want prefix %s", page.URL(), baseURL)
	}
	if strings.Contains(page.URL(), "/docs/") {
		t.Fatalf("docs search unexpectedly navigated to a docs route: %s", page.URL())
	}
}
