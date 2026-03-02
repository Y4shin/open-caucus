//go:build e2e

package e2e_test

import (
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

	urlBefore := page.URL()
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
	if page.URL() != urlBefore {
		t.Fatalf("help button %q triggered navigation: before=%s after=%s", buttonAriaLabel, urlBefore, page.URL())
	}
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
