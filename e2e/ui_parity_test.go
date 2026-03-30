//go:build e2e

package e2e_test

import (
	"context"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	playwright "github.com/playwright-community/playwright-go"
)

func normalizeWhitespace(raw string) string {
	return strings.Join(strings.Fields(raw), " ")
}

var betweenTagsWhitespace = regexp.MustCompile(`>\s+<`)
var svelteCommentNodes = regexp.MustCompile(`<!---->`)

func normalizeHTML(raw string) string {
	trimmed := strings.TrimSpace(raw)
	trimmed = svelteCommentNodes.ReplaceAllString(trimmed, "")
	trimmed = betweenTagsWhitespace.ReplaceAllString(trimmed, "><")
	return trimmed
}

const canonicalOuterHTMLJS = `el => {
	function normalize(node) {
		if (node.nodeType === Node.ELEMENT_NODE) {
			const attrs = Array.from(node.attributes).map((attr) => [attr.name, attr.value]);
			attrs.sort((a, b) => a[0].localeCompare(b[0]) || a[1].localeCompare(b[1]));
			for (const attr of Array.from(node.attributes)) {
				node.removeAttribute(attr.name);
			}
			for (const [name, value] of attrs) {
				node.setAttribute(name, value);
			}
			for (const child of Array.from(node.childNodes)) {
				normalize(child);
			}
		} else if (node.nodeType === Node.COMMENT_NODE) {
			node.remove();
		}
	}
	const clone = el.cloneNode(true);
	normalize(clone);
	return clone.outerHTML;
}`

func locatorText(t *testing.T, page playwright.Page, selector string) string {
	t.Helper()
	text, err := page.Locator(selector).First().TextContent()
	if err != nil {
		t.Fatalf("read text for %s: %v", selector, err)
	}
	return normalizeWhitespace(text)
}

func locatorAllTexts(t *testing.T, page playwright.Page, selector string) []string {
	t.Helper()
	values, err := page.Locator(selector).AllTextContents()
	if err != nil {
		t.Fatalf("read texts for %s: %v", selector, err)
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, normalizeWhitespace(value))
	}
	return out
}

func locatorClassTokens(t *testing.T, page playwright.Page, selector string) []string {
	t.Helper()
	className, err := page.Locator(selector).First().GetAttribute("class")
	if err != nil {
		t.Fatalf("read class for %s: %v", selector, err)
	}
	tokens := strings.Fields(strings.TrimSpace(className))
	sort.Strings(tokens)
	return tokens
}

func locatorCount(t *testing.T, page playwright.Page, selector string) int {
	t.Helper()
	count, err := page.Locator(selector).Count()
	if err != nil {
		t.Fatalf("read count for %s: %v", selector, err)
	}
	return count
}

func locatorOuterHTML(t *testing.T, page playwright.Page, selector string) string {
	t.Helper()
	if err := page.Locator(selector).First().WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(defaultE2ETimeoutMs),
	}); err != nil {
		currentURL := page.URL()
		bodyHTML := "<body unavailable>"
		if value, contentErr := page.Locator("body").Evaluate(canonicalOuterHTMLJS, nil); contentErr == nil {
			if raw, ok := value.(string); ok {
				bodyHTML = normalizeHTML(raw)
			}
		}
		t.Fatalf("wait for %s before outerHTML: %v (current URL: %s, body: %s)", selector, err, currentURL, bodyHTML)
	}
	value, err := page.Locator(selector).First().Evaluate(canonicalOuterHTMLJS, nil)
	if err != nil {
		t.Fatalf("read outerHTML for %s: %v", selector, err)
	}
	raw, ok := value.(string)
	if !ok {
		t.Fatalf("outerHTML for %s was %T, want string", selector, value)
	}
	return normalizeHTML(raw)
}

func locatorAllOuterHTML(t *testing.T, page playwright.Page, selector string) []string {
	t.Helper()
	if err := page.Locator(selector).First().WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(defaultE2ETimeoutMs),
	}); err != nil {
		t.Fatalf("wait for %s before outerHTML list: %v", selector, err)
	}
	value, err := page.Locator(selector).EvaluateAll(`(elements) => {
		function normalize(node) {
			if (node.nodeType === Node.ELEMENT_NODE) {
				const attrs = Array.from(node.attributes).map((attr) => [attr.name, attr.value]);
				attrs.sort((a, b) => a[0].localeCompare(b[0]) || a[1].localeCompare(b[1]));
				for (const attr of Array.from(node.attributes)) {
					node.removeAttribute(attr.name);
				}
				for (const [name, value] of attrs) {
					node.setAttribute(name, value);
				}
				for (const child of Array.from(node.childNodes)) {
					normalize(child);
				}
			} else if (node.nodeType === Node.COMMENT_NODE) {
				node.remove();
			}
		}
		return elements.map((el) => {
			const clone = el.cloneNode(true);
			normalize(clone);
			return clone.outerHTML;
		});
	}`, nil)
	if err != nil {
		t.Fatalf("read outerHTML list for %s: %v", selector, err)
	}
	items, ok := value.([]interface{})
	if !ok {
		t.Fatalf("outerHTML list for %s was %T, want []interface{}", selector, value)
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		raw, ok := item.(string)
		if !ok {
			t.Fatalf("outerHTML list item for %s was %T, want string", selector, item)
		}
		out = append(out, normalizeHTML(raw))
	}
	return out
}

// locatorAllInnerText returns the trimmed innerText of every element matching
// selector. Useful when comparing visible text content across SPA and legacy
// pages that render the same data with different HTML structure.
func locatorAllInnerText(t *testing.T, page playwright.Page, selector string) []string {
	t.Helper()
	if err := page.Locator(selector).First().WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(defaultE2ETimeoutMs),
	}); err != nil {
		t.Fatalf("wait for %s before innerText list: %v", selector, err)
	}
	value, err := page.Locator(selector).EvaluateAll(`(elements) => elements.map(el => (el.innerText || '').trim())`, nil)
	if err != nil {
		t.Fatalf("read innerText list for %s: %v", selector, err)
	}
	items, ok := value.([]interface{})
	if !ok {
		t.Fatalf("innerText list for %s was %T, want []interface{}", selector, value)
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		raw, ok := item.(string)
		if !ok {
			t.Fatalf("innerText list item for %s was %T, want string", selector, item)
		}
		out = append(out, raw)
	}
	return out
}

func locatorClosestOuterHTML(t *testing.T, page playwright.Page, selector, closest string) string {
	t.Helper()
	if err := page.Locator(selector).First().WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(defaultE2ETimeoutMs),
	}); err != nil {
		t.Fatalf("wait for %s before closest outerHTML: %v", selector, err)
	}
	value, err := page.Locator(selector).First().Evaluate(`(el, closestSelector) => {
		function normalize(node) {
			if (node.nodeType === Node.ELEMENT_NODE) {
				const attrs = Array.from(node.attributes).map((attr) => [attr.name, attr.value]);
				attrs.sort((a, b) => a[0].localeCompare(b[0]) || a[1].localeCompare(b[1]));
				for (const attr of Array.from(node.attributes)) {
					node.removeAttribute(attr.name);
				}
				for (const [name, value] of attrs) {
					node.setAttribute(name, value);
				}
				for (const child of Array.from(node.childNodes)) {
					normalize(child);
				}
			} else if (node.nodeType === Node.COMMENT_NODE) {
				node.remove();
			}
		}
		const target = el.closest(closestSelector);
		if (!target) return null;
		const clone = target.cloneNode(true);
		normalize(clone);
		return clone.outerHTML;
	}`, closest)
	if err != nil {
		t.Fatalf("read closest outerHTML for %s -> %s: %v", selector, closest, err)
	}
	raw, ok := value.(string)
	if !ok {
		t.Fatalf("closest outerHTML for %s -> %s was %T, want string", selector, closest, value)
	}
	return normalizeHTML(raw)
}

func gotoAndWaitForSelector(t *testing.T, page playwright.Page, url, selector string) {
	t.Helper()
	if _, err := page.Goto(url); err != nil {
		t.Fatalf("goto %s: %v", url, err)
	}
	if err := page.Locator(selector).First().WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(defaultE2ETimeoutMs),
	}); err != nil {
		currentURL := page.URL()
		bodyHTML := "<body unavailable>"
		if value, contentErr := page.Locator("body").Evaluate(canonicalOuterHTMLJS, nil); contentErr == nil {
			if raw, ok := value.(string); ok {
				bodyHTML = normalizeHTML(raw)
			}
		}
		t.Fatalf("wait for %s on %s: %v (current URL: %s, body: %s)", selector, url, err, currentURL, bodyHTML)
	}
}

func assertEqualStrings(t *testing.T, label, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s mismatch:\nnew:    %q\nlegacy: %q", label, got, want)
	}
}

func assertEqualStringSlices(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s length mismatch:\nnew:    %#v\nlegacy: %#v", label, got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%s mismatch:\nnew:    %#v\nlegacy: %#v", label, got, want)
		}
	}
}

func assertEqualHTML(t *testing.T, label, got, want string) {
	t.Helper()
	if got != want {
		diffAt := -1
		limit := len(got)
		if len(want) < limit {
			limit = len(want)
		}
		for i := 0; i < limit; i++ {
			if got[i] != want[i] {
				diffAt = i
				break
			}
		}
		if diffAt == -1 && len(got) != len(want) {
			diffAt = limit
		}
		start := diffAt - 80
		if start < 0 {
			start = 0
		}
		gotEnd := diffAt + 80
		if gotEnd > len(got) {
			gotEnd = len(got)
		}
		wantEnd := diffAt + 80
		if wantEnd > len(want) {
			wantEnd = len(want)
		}
		t.Fatalf("%s HTML mismatch at %d:\nnew ctx:    %s\nlegacy ctx: %s\n\nnew:    %s\nlegacy: %s", label, diffAt, got[start:gotEnd], want[start:wantEnd], got, want)
	}
}

func compareFragmentAfterAction(
	t *testing.T,
	label string,
	newPage, legacyPage playwright.Page,
	fragmentSelector string,
	action func(page playwright.Page) error,
) {
	t.Helper()
	if err := action(newPage); err != nil {
		t.Fatalf("%s action on new: %v", label, err)
	}
	if err := action(legacyPage); err != nil {
		t.Fatalf("%s action on legacy: %v", label, err)
	}
	assertEqualHTML(t, label,
		locatorOuterHTML(t, newPage, fragmentSelector),
		locatorOuterHTML(t, legacyPage, fragmentSelector),
	)
}

func TestLoginPage_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	gotoAndWaitForInput(t, newBrowserPage, newTS.URL+"/login", "input[name=username]")
	gotoAndWaitForInput(t, legacyBrowserPage, legacyTS.URL+"/login", "input[name=username]")

	assertEqualHTML(t, "login fieldset", locatorClosestOuterHTML(t, newBrowserPage, "input[name=username]", "fieldset"), locatorClosestOuterHTML(t, legacyBrowserPage, "input[name=username]", "fieldset"))
	if !strings.Contains(locatorText(t, newBrowserPage, "nav"), "Conference Tool") && !strings.Contains(locatorText(t, newBrowserPage, "nav"), "Conference-Tool") {
		t.Fatal("new login shell is missing the app name")
	}
	if !strings.Contains(locatorText(t, legacyBrowserPage, "nav"), "Conference Tool") && !strings.Contains(locatorText(t, legacyBrowserPage, "nav"), "Conference-Tool") {
		t.Fatal("legacy login shell is missing the app name")
	}
}

func TestAdminLoginPage_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	gotoAndWaitForInput(t, newBrowserPage, newTS.URL+"/admin/login", "input[name=username]")
	gotoAndWaitForInput(t, legacyBrowserPage, legacyTS.URL+"/admin/login", "input[name=username]")

	assertEqualHTML(t, "admin login form", locatorClosestOuterHTML(t, newBrowserPage, "input[name=username]", "form"), locatorClosestOuterHTML(t, legacyBrowserPage, "input[name=username]", "form"))
	const oauthSelector = `a[href^="/oauth/start"]`
	legacyOAuthButtonCount := locatorCount(t, legacyBrowserPage, oauthSelector)
	assertEqualStrings(t, "admin login oauth button count", strconv.Itoa(locatorCount(t, newBrowserPage, oauthSelector)), strconv.Itoa(legacyOAuthButtonCount))
	if legacyOAuthButtonCount > 0 {
		assertEqualHTML(t, "admin login oauth button", locatorOuterHTML(t, newBrowserPage, oauthSelector), locatorOuterHTML(t, legacyBrowserPage, oauthSelector))
	}
}

func TestHomePage_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	newTS.seedCommittee(t, "Budget Committee", "budget")
	newTS.seedCommittee(t, "Rules Committee", "rules")
	newTS.seedUser(t, "budget", "member1", "pass123", "Regular Member", "member")
	newTS.seedUser(t, "rules", "member1", "pass123", "Regular Member", "member")

	legacyTS.seedCommittee(t, "Budget Committee", "budget")
	legacyTS.seedCommittee(t, "Rules Committee", "rules")
	legacyTS.seedUser(t, "budget", "member1", "pass123", "Regular Member", "member")
	legacyTS.seedUser(t, "rules", "member1", "pass123", "Regular Member", "member")

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "budget", "member1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "budget", "member1", "pass123")

	if _, err := newBrowserPage.Goto(newTS.URL + "/home"); err != nil {
		t.Fatalf("goto new /home: %v", err)
	}
	if _, err := legacyBrowserPage.Goto(legacyTS.URL + "/home"); err != nil {
		t.Fatalf("goto legacy /home: %v", err)
	}

	assertEqualHTML(t, "home committee card", locatorOuterHTML(t, newBrowserPage, ".rounded-box.bg-base-200"), locatorOuterHTML(t, legacyBrowserPage, ".rounded-box.bg-base-200"))
}

func TestCommitteeChairPage_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	newTS.seedCommittee(t, "Test Committee", "test-committee")
	newTS.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	newTS.seedMeeting(t, "test-committee", "Budget Meeting", "Annual budget")
	newTS.seedMeeting(t, "test-committee", "Rules Meeting", "")

	legacyTS.seedCommittee(t, "Test Committee", "test-committee")
	legacyTS.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	legacyTS.seedMeeting(t, "test-committee", "Budget Meeting", "Annual budget")
	legacyTS.seedMeeting(t, "test-committee", "Rules Meeting", "")

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test-committee", "chair1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test-committee", "chair1", "pass123")

	assertEqualHTML(
		t,
		"committee create form",
		locatorOuterHTML(t, newBrowserPage, "[data-testid='committee-create-form']"),
		locatorOuterHTML(t, legacyBrowserPage, "[data-testid='committee-create-form']"),
	)

	newRows := locatorAllOuterHTML(t, newBrowserPage, "[data-testid='committee-meeting-row']")
	legacyRows := locatorAllOuterHTML(t, legacyBrowserPage, "[data-testid='committee-meeting-row']")
	sort.Strings(newRows)
	sort.Strings(legacyRows)
	assertEqualStringSlices(t, "committee meeting rows", newRows, legacyRows)
}

func TestAdminDashboard_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	newTS.seedCommittee(t, "Rules Committee", "rules")
	legacyTS.seedCommittee(t, "Rules Committee", "rules")

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	adminLogin(t, newBrowserPage, newTS.URL)
	adminLogin(t, legacyBrowserPage, legacyTS.URL)

	assertEqualHTML(t, "admin dashboard first section", locatorClosestOuterHTML(t, newBrowserPage, "#create-committee-form", "section"), locatorClosestOuterHTML(t, legacyBrowserPage, "#create-committee-form", "section"))
	assertEqualStringSlices(t, "admin committee rows html", locatorAllOuterHTML(t, newBrowserPage, "table tbody tr"), locatorAllOuterHTML(t, legacyBrowserPage, "table tbody tr"))
}

func TestAdminAccounts_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	newTS.seedAccount(t, "alice", "pass123", "Alice Example")
	legacyTS.seedAccount(t, "alice", "pass123", "Alice Example")

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	adminLogin(t, newBrowserPage, newTS.URL)
	adminLogin(t, legacyBrowserPage, legacyTS.URL)

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/admin/accounts", "#create-account-form")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/admin/accounts", "#create-account-form")

	assertEqualHTML(t, "admin accounts first section", locatorClosestOuterHTML(t, newBrowserPage, "#create-account-form", "section"), locatorClosestOuterHTML(t, legacyBrowserPage, "#create-account-form", "section"))
	assertEqualStringSlices(t, "admin accounts rows html", locatorAllOuterHTML(t, newBrowserPage, "table tbody tr"), locatorAllOuterHTML(t, legacyBrowserPage, "table tbody tr"))
}

func TestAdminCommitteeUsers_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	newTS.seedCommittee(t, "Budget Committee", "budget")
	newTS.seedAccount(t, "assignable", "pass123", "Assignable Person")
	newTS.seedUser(t, "budget", "member1", "pass123", "Member One", "member")

	legacyTS.seedCommittee(t, "Budget Committee", "budget")
	legacyTS.seedAccount(t, "assignable", "pass123", "Assignable Person")
	legacyTS.seedUser(t, "budget", "member1", "pass123", "Member One", "member")

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	adminLogin(t, newBrowserPage, newTS.URL)
	adminLogin(t, legacyBrowserPage, legacyTS.URL)

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/admin/committee/budget", "#committee-users-container")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/admin/committee/budget", "#committee-users-container")

	assertEqualStringSlices(t, "admin committee sections html", locatorAllOuterHTML(t, newBrowserPage, "#committee-users-container section"), locatorAllOuterHTML(t, legacyBrowserPage, "#committee-users-container section"))
	assertEqualStringSlices(t, "admin committee user rows html", locatorAllOuterHTML(t, newBrowserPage, "table tbody tr"), locatorAllOuterHTML(t, legacyBrowserPage, "table tbody tr"))
}

func TestMeetingJoinAndAttendeeLogin_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	newTS.seedCommittee(t, "Test Committee", "test")
	newTS.seedUser(t, "test", "member1", "pass123", "Member One", "member")
	newTS.seedMeetingOpen(t, "test", "Open Meeting", "Join flow")
	meetingID := newTS.getMeetingID(t, "test", "Open Meeting")

	legacyTS.seedCommittee(t, "Test Committee", "test")
	legacyTS.seedUser(t, "test", "member1", "pass123", "Member One", "member")
	legacyTS.seedMeetingOpen(t, "test", "Open Meeting", "Join flow")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Open Meeting")

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "member1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "member1", "pass123")

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID+"/join", "main h3")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/join", "main h3")

	assertEqualHTML(t, "join section heading", locatorOuterHTML(t, newBrowserPage, "main h3"), locatorOuterHTML(t, legacyBrowserPage, "main h3"))
	assertEqualHTML(t, "join primary button", locatorOuterHTML(t, newBrowserPage, "main button"), locatorOuterHTML(t, legacyBrowserPage, "main button"))

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+meetingID+"/attendee-login", "form")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/attendee-login", "form")

	assertEqualHTML(t, "attendee login heading", locatorOuterHTML(t, newBrowserPage, "main h3"), locatorOuterHTML(t, legacyBrowserPage, "main h3"))
	assertEqualHTML(t, "attendee login form", locatorOuterHTML(t, newBrowserPage, "main form"), locatorOuterHTML(t, legacyBrowserPage, "main form"))
}

func TestMeetingLive_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	newTS.seedCommittee(t, "Test Committee", "test")
	newTS.seedUser(t, "test", "member1", "pass123", "Member One", "member")
	newTS.seedMeetingOpen(t, "test", "Open Meeting", "Join flow")
	newAgendaID := newTS.seedAgendaPoint(t, "test", "Open Meeting", "Main Topic")
	newTS.seedAgendaPoint(t, "test", "Open Meeting", "Second Topic")
	newTS.activateAgendaPoint(t, "test", "Open Meeting", newAgendaID)
	newMeetingID := newTS.getMeetingID(t, "test", "Open Meeting")
	newMeetingIDInt, err := strconv.ParseInt(newMeetingID, 10, 64)
	if err != nil {
		t.Fatalf("parse new meeting id: %v", err)
	}
	if err := newTS.repo.SetActiveMeeting(context.Background(), "test", &newMeetingIDInt); err != nil {
		t.Fatalf("set new active meeting: %v", err)
	}

	legacyTS.seedCommittee(t, "Test Committee", "test")
	legacyTS.seedUser(t, "test", "member1", "pass123", "Member One", "member")
	legacyTS.seedMeetingOpen(t, "test", "Open Meeting", "Join flow")
	legacyAgendaID := legacyTS.seedAgendaPoint(t, "test", "Open Meeting", "Main Topic")
	legacyTS.seedAgendaPoint(t, "test", "Open Meeting", "Second Topic")
	legacyTS.activateAgendaPoint(t, "test", "Open Meeting", legacyAgendaID)
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Open Meeting")
	legacyMeetingIDInt, err := strconv.ParseInt(legacyMeetingID, 10, 64)
	if err != nil {
		t.Fatalf("parse legacy meeting id: %v", err)
	}
	if err := legacyTS.repo.SetActiveMeeting(context.Background(), "test", &legacyMeetingIDInt); err != nil {
		t.Fatalf("set legacy active meeting: %v", err)
	}

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "member1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "member1", "pass123")

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+newMeetingID+"/join", "main button")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/join", "main button")

	if err := newBrowserPage.Locator("main button").First().Click(); err != nil {
		t.Fatalf("self-signup on new live flow: %v", err)
	}
	if err := legacyBrowserPage.Locator("main button").First().Click(); err != nil {
		t.Fatalf("self-signup on legacy live flow: %v", err)
	}

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+newMeetingID, "#live-votes-panel")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID, "#live-votes-panel")

	assertEqualHTML(t, "live navbar start", locatorOuterHTML(t, newBrowserPage, "nav .flex-1"), locatorOuterHTML(t, legacyBrowserPage, "nav .flex-1"))
	assertEqualHTML(t, "live agenda stack", locatorOuterHTML(t, newBrowserPage, "#live-agenda-main-stack"), locatorOuterHTML(t, legacyBrowserPage, "#live-agenda-main-stack"))
	assertEqualHTML(t, "live speakers container", locatorOuterHTML(t, newBrowserPage, "#attendee-speakers-list"), locatorOuterHTML(t, legacyBrowserPage, "#attendee-speakers-list"))
	assertEqualHTML(t, "live self-add regular button", locatorOuterHTML(t, newBrowserPage, "[data-testid='live-add-self-regular']"), locatorOuterHTML(t, legacyBrowserPage, "[data-testid='live-add-self-regular']"))
	assertEqualHTML(t, "live votes panel", locatorOuterHTML(t, newBrowserPage, "#live-votes-panel"), locatorOuterHTML(t, legacyBrowserPage, "#live-votes-panel"))
}

func TestMeetingModerate_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	newTS.seedCommittee(t, "Test Committee", "test")
	newTS.seedUser(t, "test", "chair1", "pass123", "Chair Person", "chairperson")
	newTS.seedMeetingOpen(t, "test", "Board Meeting", "Moderation flow")
	newTS.seedAgendaPoint(t, "test", "Board Meeting", "Main Topic")
	newTS.seedAttendee(t, "test", "Board Meeting", "Alice Member", "secret-alice-moderate")
	newMeetingID := newTS.getMeetingID(t, "test", "Board Meeting")

	legacyTS.seedCommittee(t, "Test Committee", "test")
	legacyTS.seedUser(t, "test", "chair1", "pass123", "Chair Person", "chairperson")
	legacyTS.seedMeetingOpen(t, "test", "Board Meeting", "Moderation flow")
	legacyTS.seedAgendaPoint(t, "test", "Board Meeting", "Main Topic")
	legacyTS.seedAttendee(t, "test", "Board Meeting", "Alice Member", "secret-alice-moderate")
	legacyMeetingID := legacyTS.getMeetingID(t, "test", "Board Meeting")

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	userLogin(t, newBrowserPage, newTS.URL, "test", "chair1", "pass123")
	userLogin(t, legacyBrowserPage, legacyTS.URL, "test", "chair1", "pass123")

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/committee/test/meeting/"+newMeetingID+"/moderate", "#moderate-left-controls")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/committee/test/meeting/"+legacyMeetingID+"/moderate", "#moderate-left-controls")

	assertEqualHTML(t, "moderate left controls", locatorOuterHTML(t, newBrowserPage, "#moderate-left-controls"), locatorOuterHTML(t, legacyBrowserPage, "#moderate-left-controls"))
	assertEqualHTML(t, "moderate speakers container", locatorOuterHTML(t, newBrowserPage, "#speakers-list-container"), locatorOuterHTML(t, legacyBrowserPage, "#speakers-list-container"))
}

func TestDocsAndReceipts_UIParityWithLegacy(t *testing.T) {
	newTS := newTestServer(t)
	legacyTS := newLegacyTestServer(t)

	newBrowserPage := newPage(t)
	legacyBrowserPage := newPage(t)

	adminLogin(t, newBrowserPage, newTS.URL)
	adminLogin(t, legacyBrowserPage, legacyTS.URL)

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/home", "footer button:has-text('Help')")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/home", "footer button:has-text('Help')")

	if err := newBrowserPage.Locator("footer button:has-text('Help')").First().Click(); err != nil {
		t.Fatalf("open docs on new home: %v", err)
	}
	if err := legacyBrowserPage.Locator("footer button:has-text('Help')").First().Click(); err != nil {
		t.Fatalf("open docs on legacy home: %v", err)
	}

	if err := newBrowserPage.Locator("#app-docs-target details").First().WaitFor(); err != nil {
		t.Fatalf("wait for new docs overlay: %v", err)
	}
	if err := legacyBrowserPage.Locator("#app-docs-target details").First().WaitFor(); err != nil {
		t.Fatalf("wait for legacy docs overlay: %v", err)
	}

	assertEqualHTML(t, "docs title", locatorOuterHTML(t, newBrowserPage, "#app-docs-target h2"), locatorOuterHTML(t, legacyBrowserPage, "#app-docs-target h2"))
	assertEqualHTML(t, "docs browse details", locatorOuterHTML(t, newBrowserPage, "#app-docs-target details"), locatorOuterHTML(t, legacyBrowserPage, "#app-docs-target details"))

	if err := newBrowserPage.Locator("#app-docs-target input[type='search']").Fill("agenda"); err != nil {
		t.Fatalf("fill new docs search: %v", err)
	}
	if err := legacyBrowserPage.Locator("#app-docs-target input[type='search']").Fill("agenda"); err != nil {
		t.Fatalf("fill legacy docs search: %v", err)
	}
	if err := newBrowserPage.Locator("#app-docs-target input[type='search']").Press("Enter"); err != nil {
		t.Fatalf("submit new docs search: %v", err)
	}
	if err := legacyBrowserPage.Locator("#app-docs-target input[type='search']").Press("Enter"); err != nil {
		t.Fatalf("submit legacy docs search: %v", err)
	}

	if err := newBrowserPage.Locator("#app-docs-target #docs-search-results").WaitFor(); err != nil {
		t.Fatalf("wait for new docs search results: %v", err)
	}
	if err := legacyBrowserPage.Locator("#app-docs-target #docs-search-results").WaitFor(); err != nil {
		t.Fatalf("wait for legacy docs search results: %v", err)
	}
	// Wait for HTMX transitional classes to settle on the legacy page before comparing.
	waitUntil(t, 3*time.Second, func() (bool, error) {
		className, err := legacyBrowserPage.Locator("#app-docs-target #docs-search-results").First().GetAttribute("class")
		if err != nil {
			return false, err
		}
		return !strings.Contains(className, "htmx-swapping") &&
			!strings.Contains(className, "htmx-added") &&
			!strings.Contains(className, "htmx-settling"), nil
	}, "legacy docs search results HTMX classes to settle")
	assertEqualHTML(t, "docs search container", locatorOuterHTML(t, newBrowserPage, "#app-docs-target #docs-search-results"), locatorOuterHTML(t, legacyBrowserPage, "#app-docs-target #docs-search-results"))

	gotoAndWaitForSelector(t, newBrowserPage, newTS.URL+"/receipts", "#receipts-vault-content")
	gotoAndWaitForSelector(t, legacyBrowserPage, legacyTS.URL+"/receipts", "#receipts-vault-content")

	assertEqualHTML(t, "receipts vault content", locatorOuterHTML(t, newBrowserPage, "#receipts-vault-content"), locatorOuterHTML(t, legacyBrowserPage, "#receipts-vault-content"))
}
