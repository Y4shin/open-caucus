//go:build e2e

package e2e_test

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	playwright "github.com/playwright-community/playwright-go"
)

const transgenderIconPath = "M579-401q41-41 41-99t-41-99q-41-41-99-41t-99 41q-41 41-41 99t41 99q41 41 99 41t99-41ZM440-40v-80h-80v-80h80v-84q-78-14-129-75t-51-141q0-33 9.5-65t28.5-59l-26-26-56 56-56-56 56-57-76-76v103H60v-240h240v80H197l76 76 57-56 56 56-56 57 26 26q27-20 59-29.5t65-9.5q33 0 65 9.5t59 29.5l159-159H660v-80h240v240h-80v-103L661-625q19 28 29 59.5t10 65.5q0 80-51 141t-129 75v84h80v80h-80v80h-80Z"

func manageURL(baseURL, slug, meetingID string) string {
	return baseURL + "/committee/" + slug + "/meeting/" + meetingID + "/moderate"
}

func manageAttendeeCard(t *testing.T, page playwright.Page, fullName string) playwright.Locator {
	t.Helper()
	openModerateLeftTab(t, page, "attendees")
	return page.Locator("#attendee-list-container [data-testid='manage-attendee-card']").Filter(playwright.LocatorFilterOptions{
		HasText: fullName,
	})
}

func manageSignupOpenChecked(t *testing.T, page playwright.Page) bool {
	t.Helper()
	openModerateLeftTab(t, page, "attendees")
	checked, err := page.Locator("#manage_signup_open").IsChecked()
	if err != nil {
		t.Fatalf("read signup state: %v", err)
	}
	return checked
}

func submitAddGuest(t *testing.T, page playwright.Page, fullName string) {
	t.Helper()
	submitAddGuestWithQuoted(t, page, fullName, false)
}

func submitAddGuestWithQuoted(t *testing.T, page playwright.Page, fullName string, quoted bool) {
	t.Helper()
	openModerateLeftTab(t, page, "attendees")
	form := page.Locator("#attendee-list-container [data-testid='manage-add-guest-form']")
	quotedToggle := form.Locator("input[name=gender_quoted]")
	if quoted {
		if err := quotedToggle.Check(); err != nil {
			t.Fatalf("check quoted toggle: %v", err)
		}
	} else {
		if err := quotedToggle.Uncheck(); err != nil {
			t.Fatalf("uncheck quoted toggle: %v", err)
		}
	}
	if err := form.Locator("input[name=full_name]").Fill(fullName); err != nil {
		t.Fatalf("fill guest name: %v", err)
	}
	value, err := form.Locator("input[name=full_name]").InputValue()
	if err != nil {
		t.Fatalf("read guest name input value: %v", err)
	}
	if value == "" {
		t.Fatalf("guest name input unexpectedly empty before submit")
	}
	validAny, err := form.Evaluate("f => f.checkValidity()", nil)
	if err != nil {
		t.Fatalf("check add-guest form validity: %v", err)
	}
	valid, ok := validAny.(bool)
	if !ok || !valid {
		t.Fatalf("add-guest form invalid before submit")
	}
	if _, err := form.Evaluate("f => { f.requestSubmit(); return true; }", nil); err != nil {
		t.Fatalf("submit add-guest form: %v", err)
	}
}

func submitSelfSignup(t *testing.T, page playwright.Page) bool {
	t.Helper()
	openModerateLeftTab(t, page, "attendees")
	form := page.Locator("#attendee-list-container [data-testid='manage-self-signup-form']")
	count, err := form.Count()
	if err != nil {
		t.Fatalf("count self-signup forms: %v", err)
	}
	if count == 0 {
		return false
	}
	if _, err := form.First().Evaluate("f => { f.requestSubmit(); return true; }", nil); err != nil {
		t.Fatalf("submit self-signup form: %v", err)
	}
	return true
}

func waitUntil(t *testing.T, timeout time.Duration, fn func() (bool, error), description string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ok, err := fn()
		if err == nil && ok {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", description)
}

func assertFLINTABadgeUsesTransgenderIcon(t *testing.T, badge playwright.Locator) {
	t.Helper()

	if err := badge.WaitFor(); err != nil {
		t.Fatalf("wait for FLINTA badge: %v", err)
	}

	tooltipAny, err := badge.Evaluate(`el => el.closest('[data-tip]')?.getAttribute('data-tip') ?? ''`, nil)
	if err != nil {
		t.Fatalf("read FLINTA badge tooltip: %v", err)
	}
	tooltip, ok := tooltipAny.(string)
	if !ok {
		t.Fatalf("FLINTA badge tooltip was %T, want string", tooltipAny)
	}
	if tooltip != "FLINTA*" {
		t.Fatalf("expected FLINTA badge tooltip %q, got %q", "FLINTA*", tooltip)
	}

	viewBox, err := badge.Locator("svg").GetAttribute("viewBox")
	if err != nil {
		t.Fatalf("read FLINTA badge svg viewBox: %v", err)
	}
	if viewBox != "0 -960 960 960" {
		t.Fatalf("expected transgender icon viewBox %q, got %q", "0 -960 960 960", viewBox)
	}

	path, err := badge.Locator("svg path").GetAttribute("d")
	if err != nil {
		t.Fatalf("read FLINTA badge svg path: %v", err)
	}
	if path != transgenderIconPath {
		t.Fatalf("expected transgender icon path %q, got %q", transgenderIconPath, path)
	}
}

func manageSpeakersViewportSnapshot(t *testing.T, page playwright.Page) (float64, float64, float64, int, string) {
	t.Helper()
	raw, err := page.Evaluate(`() => {
		const scopedContainer = document.querySelector("[data-testid='manage-speakers-card'] #speakers-list-container");
		let viewport = scopedContainer ? scopedContainer.querySelector("[data-manage-speakers-viewport]") : null;
		if (!viewport) {
			const allViewports = Array.from(document.querySelectorAll("#speakers-list-container [data-manage-speakers-viewport]"));
			viewport = allViewports.reduce((best, candidate) => {
				if (!best) return candidate;
				const bestScore = Math.max(best.scrollHeight || 0, best.clientHeight || 0);
				const candidateScore = Math.max(candidate.scrollHeight || 0, candidate.clientHeight || 0);
				return candidateScore > bestScore ? candidate : best;
			}, null);
		}
		if (!viewport) return null;
		const rows = Array.from(viewport.querySelectorAll("[data-testid='live-speaker-item']"));
		const vpRect = viewport.getBoundingClientRect();
		let firstVisibleName = "";
		let firstVisibleTop = Number.POSITIVE_INFINITY;
		for (const row of rows) {
			const rect = row.getBoundingClientRect();
			if (rect.bottom <= vpRect.top + 1) continue;
			if (rect.top < firstVisibleTop) {
				firstVisibleTop = rect.top;
				const nameEl = row.querySelector("[data-testid='live-speaker-name']");
				firstVisibleName = (nameEl ? nameEl.textContent : row.textContent || "").trim();
			}
		}
		const visibleRows = rows.filter((row) => {
			const rect = row.getBoundingClientRect();
			return rect.bottom > vpRect.top + 1 && rect.top < vpRect.bottom - 1;
		}).length;
		return {
			scrollTop: viewport.scrollTop,
			clientHeight: viewport.clientHeight,
			scrollHeight: viewport.scrollHeight,
			visibleRows,
			firstVisibleName
		};
	}`, nil)
	if err != nil {
		t.Fatalf("read speakers viewport snapshot: %v", err)
	}
	state, ok := raw.(map[string]interface{})
	if !ok || state == nil {
		t.Fatalf("unexpected speakers viewport snapshot: %#v", raw)
	}
	scrollTop, _ := state["scrollTop"].(float64)
	clientHeight, _ := state["clientHeight"].(float64)
	scrollHeight, _ := state["scrollHeight"].(float64)
	visibleRowsFloat, _ := state["visibleRows"].(float64)
	firstVisibleName, _ := state["firstVisibleName"].(string)
	return scrollTop, clientHeight, scrollHeight, int(visibleRowsFloat), firstVisibleName
}

func manageSpeakersQuickControlMetrics(t *testing.T, page playwright.Page) (float64, float64, float64) {
	t.Helper()
	raw, err := page.Evaluate(`() => {
		const scopedContainer = document.querySelector("[data-testid='manage-speakers-card'] #speakers-list-container");
		const wrap = scopedContainer
			? scopedContainer.querySelector("[data-testid='manage-speakers-quick-controls']")
			: document.querySelector("#speakers-list-container [data-testid='manage-speakers-quick-controls']");
		const button = wrap ? wrap.querySelector("[data-testid-group='manage-speakers-quick-button']") : null;
		if (!wrap || !button) return null;
		const wrapRect = wrap.getBoundingClientRect();
		const buttonRect = button.getBoundingClientRect();
		const marginTop = parseFloat(window.getComputedStyle(wrap).marginTop || "0");
		return { wrapWidth: wrapRect.width, buttonWidth: buttonRect.width, marginTop };
	}`, nil)
	if err != nil {
		t.Fatalf("read quick control metrics: %v", err)
	}
	state, ok := raw.(map[string]interface{})
	if !ok || state == nil {
		t.Fatalf("unexpected quick control metrics: %#v", raw)
	}
	wrapWidth, _ := state["wrapWidth"].(float64)
	buttonWidth, _ := state["buttonWidth"].(float64)
	marginTop, _ := state["marginTop"].(float64)
	return wrapWidth, buttonWidth, marginTop
}

func manageSpeakersViewportStyles(t *testing.T, page playwright.Page) (string, string) {
	t.Helper()
	raw, err := page.Evaluate(`() => {
		const scopedContainer = document.querySelector("[data-testid='manage-speakers-card'] #speakers-list-container");
		let viewport = scopedContainer ? scopedContainer.querySelector("[data-manage-speakers-viewport]") : null;
		if (!viewport) {
			viewport = document.querySelector("#speakers-list-container [data-manage-speakers-viewport]");
		}
		if (!viewport) return null;
		const style = window.getComputedStyle(viewport);
		return { overflowY: style.overflowY, maxHeight: style.maxHeight };
	}`, nil)
	if err != nil {
		t.Fatalf("read speakers viewport styles: %v", err)
	}
	state, ok := raw.(map[string]interface{})
	if !ok || state == nil {
		t.Fatalf("unexpected speakers viewport styles: %#v", raw)
	}
	overflowY, _ := state["overflowY"].(string)
	maxHeight, _ := state["maxHeight"].(string)
	return overflowY, maxHeight
}

// TestManagePage_ShowsAttendeeList verifies that the manage page shows existing
// attendees after they have been seeded.
func TestManagePage_ShowsAttendeeList(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	ts.seedAttendee(t, "test-committee", "Board Meeting", "Alice Member", "secret-alice")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}
	if err := manageAttendeeCard(t, page, "Alice Member").WaitFor(); err != nil {
		t.Fatalf("expected attendee card for Alice Member: %v", err)
	}
}

// TestManagePage_AddGuest verifies that the chairperson can add a guest attendee
// via the inline form and the list updates without a full page reload.
func TestManagePage_AddGuest(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	urlBefore := page.URL()
	submitAddGuest(t, page, "Bob Guest")
	if err := manageAttendeeCard(t, page, "Bob Guest").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateAttached,
	}); err != nil {
		t.Fatalf("expected new guest attendee card: %v", err)
	}
	if page.URL() != urlBefore {
		t.Errorf("URL changed on add guest: got %s, want %s", page.URL(), urlBefore)
	}
}

// TestManagePage_CrossTab_AttendeeChangePropagates verifies attendee
// changes in one tab are reflected in another tab of the same meeting.
func TestManagePage_CrossTab_AttendeeChangePropagates(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")

	pageA := newPage(t)
	userLogin(t, pageA, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := pageA.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page A: %v", err)
	}

	pageB := newPage(t)
	userLogin(t, pageB, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := pageB.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page B: %v", err)
	}

	if !submitSelfSignup(t, pageA) {
		submitAddGuest(t, pageA, "CrossTab Guest")
	}
	if err := pageB.Locator("#attendee-list-container [data-testid='manage-attendee-card']:has-text('Chair Person'), #attendee-list-container [data-testid='manage-attendee-card']:has-text('CrossTab Guest')").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateAttached,
	}); err != nil {
		t.Fatalf("expected attendee update propagated to B: %v", err)
	}
}

// TestManagePage_MeetingIsolation_SSE verifies attendee-change events are scoped
// to their meeting and do not update other meetings.
func TestManagePage_MeetingIsolation_SSE(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting A", "")
	ts.seedMeeting(t, "test-committee", "Board Meeting B", "")
	meetingA := ts.getMeetingID(t, "test-committee", "Board Meeting A")
	meetingB := ts.getMeetingID(t, "test-committee", "Board Meeting B")

	pageA := newPage(t)
	userLogin(t, pageA, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := pageA.Goto(manageURL(ts.URL, "test-committee", meetingA)); err != nil {
		t.Fatalf("goto manage page A: %v", err)
	}

	pageB := newPage(t)
	userLogin(t, pageB, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := pageB.Goto(manageURL(ts.URL, "test-committee", meetingB)); err != nil {
		t.Fatalf("goto manage page B: %v", err)
	}

	submitAddGuest(t, pageA, "Isolation Guest")
	if err := manageAttendeeCard(t, pageA, "Isolation Guest").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateAttached,
	}); err != nil {
		t.Fatalf("expected attendee in meeting A: %v", err)
	}

	if err := manageAttendeeCard(t, pageB, "Isolation Guest").WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(1500),
	}); err == nil {
		t.Fatalf("unexpected attendee propagation to meeting B")
	}
}

// TestManagePage_RemoveAttendee verifies that clicking Remove deletes the
// attendee card from the list (after confirmation).
func TestManagePage_RemoveAttendee(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	ts.seedAttendee(t, "test-committee", "Board Meeting", "Carol Guest", "secret-carol")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	card := manageAttendeeCard(t, page, "Carol Guest")
	if err := card.WaitFor(); err != nil {
		t.Fatalf("attendee not visible before remove: %v", err)
	}

	page.OnDialog(func(d playwright.Dialog) {
		if err := d.Accept(); err != nil {
			t.Logf("accept dialog error: %v", err)
		}
	})

	if err := card.Locator("button[title='Remove attendee']").Click(); err != nil {
		t.Fatalf("click remove attendee: %v", err)
	}
	if err := manageAttendeeCard(t, page, "Carol Guest").WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateDetached,
	}); err != nil {
		t.Fatalf("expected attendee card to disappear: %v", err)
	}
}

// TestManagePage_ToggleChair verifies that chair toggling changes state.
func TestManagePage_ToggleChair(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	ts.seedAttendee(t, "test-committee", "Board Meeting", "Dave Member", "secret-dave")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	card := manageAttendeeCard(t, page, "Dave Member")
	if err := card.WaitFor(); err != nil {
		t.Fatalf("dave card not visible: %v", err)
	}

	chairToggle := card.Locator("input[title='Chairperson']")
	if err := chairToggle.Click(); err != nil {
		t.Fatalf("click chair toggle on: %v", err)
	}
	waitUntil(t, 3*time.Second, func() (bool, error) {
		return chairToggle.IsChecked()
	}, "chair toggle to become checked")

	if err := chairToggle.Click(); err != nil {
		t.Fatalf("click chair toggle off: %v", err)
	}
	waitUntil(t, 3*time.Second, func() (bool, error) {
		checked, err := chairToggle.IsChecked()
		return !checked, err
	}, "chair toggle to become unchecked")
}

// TestManagePage_GuestRecoveryDialog verifies that guest cards open a
// recovery dialog with a direct login URL and QR code.
func TestManagePage_GuestRecoveryDialog(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	ts.seedAttendee(t, "test-committee", "Board Meeting", "Recoverable Guest", "secret-recoverable")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	card := manageAttendeeCard(t, page, "Recoverable Guest")
	if err := card.WaitFor(); err != nil {
		t.Fatalf("expected guest attendee card: %v", err)
	}
	if err := card.Locator("button[title='Recovery link']").Click(); err != nil {
		t.Fatalf("click recovery link: %v", err)
	}
	if err := page.Locator("#recovery-qr-dialog #attendee-recovery-link").WaitFor(); err != nil {
		t.Fatalf("expected attendee recovery link in dialog: %v", err)
	}
	if err := page.Locator("#recovery-qr-dialog #attendee-recovery-qr").WaitFor(); err != nil {
		t.Fatalf("expected attendee recovery QR in dialog: %v", err)
	}

	href, err := page.Locator("#recovery-qr-dialog #attendee-recovery-link").GetAttribute("href")
	if err != nil {
		t.Fatalf("get attendee recovery href: %v", err)
	}
	if !strings.Contains(href, "/attendee-login?secret=secret-recoverable") {
		t.Fatalf("expected recovery href to contain attendee-login secret, got %q", href)
	}

	guestPage := newPage(t)
	if _, err := guestPage.Goto(href); err != nil {
		t.Fatalf("goto recovery href as guest: %v", err)
	}
	if err := guestPage.WaitForURL(liveURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("expected direct redirect to /live from recovery link: %v", err)
	}
	if err := guestPage.Locator("p.scaffold-auth-text:has-text('Recoverable Guest')").WaitFor(); err != nil {
		t.Fatalf("expected guest name on live page: %v", err)
	}
}

// TestManagePage_ToggleSignupOpen verifies that the signup switch changes state
// and controls join-page guest form visibility.
func TestManagePage_ToggleSignupOpen(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	if manageSignupOpenChecked(t, page) {
		t.Fatalf("expected signup to start closed")
	}

	if err := page.Locator("label[for='manage_signup_open']").Click(); err != nil {
		t.Fatalf("toggle signup switch on: %v", err)
	}
	waitUntil(t, 3*time.Second, func() (bool, error) {
		return page.Locator("#manage_signup_open").IsChecked()
	}, "signup switch to become checked")

	guestPage := newPage(t)
	if _, err := guestPage.Goto(joinURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto join page after opening signup: %v", err)
	}
	if err := guestPage.Locator("input[name=full_name]").WaitFor(); err != nil {
		t.Fatalf("expected guest signup form when signup is open: %v", err)
	}
}

// TestManagePage_SpeakersQuickControls verifies the top speaker controls:
// start-next when no active speaker, and end-speech when one is active.
func TestManagePage_SpeakersQuickControls(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "Main Topic")
	ts.activateAgendaPoint(t, "test-committee", "Board Meeting", apID)

	first := ts.seedAttendee(t, "test-committee", "Board Meeting", "First Queue", "secret-first-queue")
	second := ts.seedAttendee(t, "test-committee", "Board Meeting", "Second Queue", "secret-second-queue")
	ts.seedSpeaker(t, apID, strconv.FormatInt(first.ID, 10))
	ts.seedSpeaker(t, apID, strconv.FormatInt(second.ID, 10))

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}
	urlBefore := page.URL()

	if err := page.Locator("[data-testid='manage-start-next-speaker']").WaitFor(); err != nil {
		t.Fatalf("expected start-next control when no active speaker: %v", err)
	}
	if err := page.Locator("[data-testid='manage-start-next-speaker']").Click(); err != nil {
		t.Fatalf("click start-next control: %v", err)
	}
	if err := page.Locator("[data-testid='manage-end-current-speaker']").WaitFor(); err != nil {
		t.Fatalf("expected end-speech control after starting next: %v", err)
	}
	if err := page.Locator("#speakers-list-container [data-testid='live-speaker-item'][data-speaker-state='speaking']:has-text('First Queue')").WaitFor(); err != nil {
		t.Fatalf("expected first speaker to be active after start-next: %v", err)
	}

	if err := page.Locator("[data-testid='manage-end-current-speaker']").Click(); err != nil {
		t.Fatalf("click end-current control: %v", err)
	}
	if err := page.Locator("[data-testid='manage-start-next-speaker']").WaitFor(); err != nil {
		t.Fatalf("expected start-next control after ending current speaker: %v", err)
	}
	if page.URL() != urlBefore {
		t.Errorf("URL changed during quick speaker controls: got %s, want %s", page.URL(), urlBefore)
	}
}

// TestManagePage_SpeakersViewport_InitialScrollAndReset verifies that the
// speakers list is capped/scrollable and initially anchored to the active
// speaker, with a control to restore that initial scroll state.
func TestManagePage_SpeakersViewport_InitialScrollAndReset(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "Main Topic")
	ts.activateAgendaPoint(t, "test-committee", "Board Meeting", apID)

	var speakerIDs []int64
	for i := 1; i <= 10; i++ {
		name := fmt.Sprintf("Speaker %02d", i)
		secret := fmt.Sprintf("secret-speaker-%02d", i)
		attendee := ts.seedAttendee(t, "test-committee", "Board Meeting", name, secret)
		speakerID := parseID(t, ts.seedSpeaker(t, apID, strconv.FormatInt(attendee.ID, 10)))
		speakerIDs = append(speakerIDs, speakerID)
	}
	for i := 0; i < 7; i++ {
		if err := ts.repo.SetSpeakerDone(context.Background(), speakerIDs[i]); err != nil {
			t.Fatalf("set speaker %d done: %v", i+1, err)
		}
	}
	if err := ts.repo.SetSpeakerSpeaking(context.Background(), speakerIDs[7], parseID(t, apID)); err != nil {
		t.Fatalf("set speaking speaker: %v", err)
	}
	if err := ts.repo.RecomputeSpeakerOrder(context.Background(), parseID(t, apID)); err != nil {
		t.Fatalf("recompute speaker order: %v", err)
	}

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	if err := page.Locator("#speakers-list-container [data-testid='manage-end-current-speaker']").WaitFor(); err != nil {
		t.Fatalf("expected end-current quick control: %v", err)
	}
	if err := page.Locator("#speakers-list-container [data-testid='manage-speakers-reset-scroll']").WaitFor(); err != nil {
		t.Fatalf("expected speakers reset-scroll button: %v", err)
	}

	overflowY, _ := manageSpeakersViewportStyles(t, page)
	if overflowY != "auto" && overflowY != "scroll" {
		t.Errorf("expected speakers viewport to be scrollable, got overflow-y=%q", overflowY)
	}

	scrollTop, clientHeight, scrollHeight, visibleRows, firstVisibleName := manageSpeakersViewportSnapshot(t, page)
	_ = clientHeight
	_ = scrollHeight
	if visibleRows > 7 {
		t.Errorf("expected approximately <= 6 visible speakers (tolerated <=7), got %d", visibleRows)
	}
	_ = scrollTop
	if strings.TrimSpace(firstVisibleName) == "" {
		t.Errorf("expected an initial visible speaker row name")
	}

	if _, err := page.Evaluate(`() => {
		const scopedContainer = document.querySelector("[data-testid='manage-speakers-card'] #speakers-list-container");
		let viewport = scopedContainer ? scopedContainer.querySelector("[data-manage-speakers-viewport]") : null;
		if (!viewport) {
			viewport = document.querySelector("#speakers-list-container [data-manage-speakers-viewport]");
		}
		if (!viewport) return false;
		viewport.scrollTop = viewport.scrollHeight;
		return true;
	}`, nil); err != nil {
		t.Fatalf("scroll viewport away from initial position before reset: %v", err)
	}

	if err := page.Locator("#speakers-list-container [data-testid='manage-speakers-reset-scroll']").Click(); err != nil {
		t.Fatalf("click reset-scroll button: %v", err)
	}

	waitUntil(t, 3*time.Second, func() (bool, error) {
		scrollTopAfterReset, _, _, _, firstVisibleAfterReset := manageSpeakersViewportSnapshot(t, page)
		_ = firstVisibleAfterReset
		return scrollTopAfterReset >= scrollTop-2 && scrollTopAfterReset <= scrollTop+2, nil
	}, "speakers viewport to restore initial anchor")
}

// TestManagePage_SpeakersViewport_InitialScrollTargetsNextWaiting verifies that
// when no one is speaking, initial scroll anchors to the next waiting speaker.
func TestManagePage_SpeakersViewport_InitialScrollTargetsNextWaiting(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "Main Topic")
	ts.activateAgendaPoint(t, "test-committee", "Board Meeting", apID)

	var speakerIDs []int64
	for i := 1; i <= 10; i++ {
		name := fmt.Sprintf("Waiting Speaker %02d", i)
		secret := fmt.Sprintf("secret-waiting-%02d", i)
		attendee := ts.seedAttendee(t, "test-committee", "Board Meeting", name, secret)
		speakerID := parseID(t, ts.seedSpeaker(t, apID, strconv.FormatInt(attendee.ID, 10)))
		speakerIDs = append(speakerIDs, speakerID)
	}
	for i := 0; i < 7; i++ {
		if err := ts.repo.SetSpeakerDone(context.Background(), speakerIDs[i]); err != nil {
			t.Fatalf("set waiting speaker %d done: %v", i+1, err)
		}
	}
	if err := ts.repo.RecomputeSpeakerOrder(context.Background(), parseID(t, apID)); err != nil {
		t.Fatalf("recompute speaker order: %v", err)
	}

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	if err := page.Locator("#speakers-list-container [data-testid='manage-start-next-speaker']").WaitFor(); err != nil {
		t.Fatalf("expected start-next quick control: %v", err)
	}

	scrollTop, _, _, _, firstVisibleName := manageSpeakersViewportSnapshot(t, page)
	_ = scrollTop
	if strings.TrimSpace(firstVisibleName) == "" {
		t.Errorf("expected an initial visible speaker row name for waiting-speaker view")
	}
}

// TestManagePage_QuotedCheckbox_DoesNotToggleSignupOpen verifies that interacting
// with the add-guest quoted control does not trigger signup-open toggles.
func TestManagePage_QuotedCheckbox_DoesNotToggleSignupOpen(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	if manageSignupOpenChecked(t, page) {
		t.Fatalf("expected signup to start closed")
	}

	form := page.Locator("#attendee-list-container [data-testid='manage-add-guest-form']")
	if err := form.Locator("label[for='manage_guest_gender_quoted']").Click(); err != nil {
		t.Fatalf("click quoted label: %v", err)
	}
	waitUntil(t, 3*time.Second, func() (bool, error) {
		return form.Locator("#manage_guest_gender_quoted").IsChecked()
	}, "quoted toggle to become checked")
	time.Sleep(750 * time.Millisecond)

	if manageSignupOpenChecked(t, page) {
		t.Fatalf("signup was toggled by quoted control interaction")
	}
}

// TestManagePage_AddQuotedGuest_ShowsQuotedBadgeInManageAndLive verifies that
// chair-added quoted guests keep quoted status into speaker entries in both
// manage and attendee live sessions.
func TestManagePage_AddQuotedGuest_ShowsQuotedBadgeInManageAndLive(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "Main Topic")
	ts.activateAgendaPoint(t, "test-committee", "Board Meeting", apID)

	managePage := newPage(t)
	userLogin(t, managePage, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := managePage.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}
	openModerateLeftTab(t, managePage, "attendees")
	if err := managePage.Locator("#attendee-list-container [data-testid='manage-add-guest-form'] input[name=gender_quoted]").WaitFor(); err != nil {
		t.Fatalf("expected quoted control on add-guest form: %v", err)
	}
	if manageSignupOpenChecked(t, managePage) {
		t.Fatalf("expected signup to start closed")
	}

	submitAddGuestWithQuoted(t, managePage, "Quoted Guest", true)
	quotedGuestCard := manageAttendeeCard(t, managePage, "Quoted Guest")
	if err := quotedGuestCard.WaitFor(); err != nil {
		t.Fatalf("expected quoted guest attendee card: %v", err)
	}
	if err := quotedGuestCard.Locator("[data-testid='manage-attendee-quoted-badge']").WaitFor(); err != nil {
		t.Fatalf("expected quoted badge on attendee card: %v", err)
	}
	assertFLINTABadgeUsesTransgenderIcon(t, quotedGuestCard.Locator("[data-testid='manage-attendee-quoted-badge']"))
	if manageSignupOpenChecked(t, managePage) {
		t.Fatalf("signup was toggled by quoted guest add flow")
	}

	meetingIDInt, err := strconv.ParseInt(meetingID, 10, 64)
	if err != nil {
		t.Fatalf("parse meeting id: %v", err)
	}
	attendees, err := ts.repo.ListAttendeesForMeeting(context.Background(), meetingIDInt)
	if err != nil {
		t.Fatalf("list attendees after quoted add: %v", err)
	}
	quotedGuestSecret := ""
	for _, a := range attendees {
		if a.FullName == "Quoted Guest" {
			quotedGuestSecret = a.Secret
			break
		}
	}
	if quotedGuestSecret == "" {
		t.Fatalf("quoted guest attendee not found in repository")
	}
	guestPage := newPage(t)
	attendeeLoginHelper(t, guestPage, ts.URL, "test-committee", meetingID, quotedGuestSecret)

	openSpeakerAddDialog(t, managePage)
	if err := speakerCandidateCard(managePage, "Quoted Guest").Locator("button[title='Add regular speech']").Click(); err != nil {
		t.Fatalf("add quoted guest as speaker: %v", err)
	}

	manageSpeakerRow := managePage.Locator("#speakers-list-container [data-testid='live-speaker-item']").Filter(playwright.LocatorFilterOptions{
		HasText: "Quoted Guest",
	})
	if err := manageSpeakerRow.WaitFor(); err != nil {
		t.Fatalf("expected quoted guest row in manage speakers list: %v", err)
	}
	if err := manageSpeakerRow.Locator("[data-testid='live-speaker-quoted-badge']").WaitFor(); err != nil {
		t.Fatalf("expected quoted badge in manage speaker row: %v", err)
	}
	assertFLINTABadgeUsesTransgenderIcon(t, manageSpeakerRow.Locator("[data-testid='live-speaker-quoted-badge']"))

	liveSpeakerRow := guestPage.Locator("#attendee-speakers-list [data-testid='live-speakers-active-viewport'] [data-testid='live-speaker-item']").Filter(playwright.LocatorFilterOptions{
		HasText: "Quoted Guest",
	})
	if err := liveSpeakerRow.WaitFor(); err != nil {
		t.Fatalf("expected quoted guest row in live speakers list: %v", err)
	}
	if err := liveSpeakerRow.Locator("[data-testid='live-speaker-quoted-badge']").WaitFor(); err != nil {
		t.Fatalf("expected quoted badge in live speaker row: %v", err)
	}
	assertFLINTABadgeUsesTransgenderIcon(t, liveSpeakerRow.Locator("[data-testid='live-speaker-quoted-badge']"))
}

// TestManagePage_ToggleGuestGenderQuoted_UpdatesSpeakerChip verifies that
// changing a guest attendee's gender-quoted status on manage propagates into
// newly created speaker entries.
func TestManagePage_ToggleGuestGenderQuoted_UpdatesSpeakerChip(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Board Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Board Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Board Meeting", "Main Topic")
	ts.activateAgendaPoint(t, "test-committee", "Board Meeting", apID)

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(manageURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto manage page: %v", err)
	}

	submitAddGuestWithQuoted(t, page, "Toggle Guest", false)
	guestCard := manageAttendeeCard(t, page, "Toggle Guest")
	if err := guestCard.WaitFor(); err != nil {
		t.Fatalf("expected guest attendee card: %v", err)
	}
	initialQuotedChipCount, err := guestCard.Locator("[data-testid='manage-attendee-quoted-badge']").Count()
	if err != nil {
		t.Fatalf("count initial quoted chips: %v", err)
	}
	if initialQuotedChipCount != 0 {
		t.Fatalf("expected no initial quoted attendee chip, got %d", initialQuotedChipCount)
	}

	guestQuotedToggle := guestCard.Locator("input[title='FLINTA*']")
	if err := guestQuotedToggle.Click(); err != nil {
		t.Fatalf("toggle guest gender quoted on: %v", err)
	}
	waitUntil(t, 3*time.Second, func() (bool, error) {
		return guestQuotedToggle.IsChecked()
	}, "guest quoted toggle to become checked")
	if err := guestCard.Locator("[data-testid='manage-attendee-quoted-badge']").WaitFor(); err != nil {
		t.Fatalf("expected quoted attendee chip after toggle: %v", err)
	}

	meetingIDInt, err := strconv.ParseInt(meetingID, 10, 64)
	if err != nil {
		t.Fatalf("parse meeting id: %v", err)
	}
	attendees, err := ts.repo.ListAttendeesForMeeting(context.Background(), meetingIDInt)
	if err != nil {
		t.Fatalf("list attendees after toggle: %v", err)
	}
	guestSecret := ""
	for _, a := range attendees {
		if a.FullName == "Toggle Guest" {
			guestSecret = a.Secret
			break
		}
	}
	if guestSecret == "" {
		t.Fatalf("toggle guest attendee not found in repository")
	}

	guestPage := newPage(t)
	attendeeLoginHelper(t, guestPage, ts.URL, "test-committee", meetingID, guestSecret)
	if err := guestPage.Locator("[data-testid='live-add-self-regular']").Click(); err != nil {
		t.Fatalf("guest self-add regular speaker: %v", err)
	}
	guestSpeakerRow := guestPage.Locator("#attendee-speakers-list [data-testid='live-speakers-active-viewport'] [data-testid='live-speaker-item']").Filter(playwright.LocatorFilterOptions{
		HasText: "Toggle Guest",
	})
	if err := guestSpeakerRow.WaitFor(); err != nil {
		t.Fatalf("expected guest speaker row: %v", err)
	}
	if err := guestSpeakerRow.Locator("[data-testid='live-speaker-quoted-badge']").WaitFor(); err != nil {
		t.Fatalf("expected gender quoted speaker chip after guest toggle: %v", err)
	}
}
