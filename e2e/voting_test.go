//go:build e2e

package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	playwright "github.com/playwright-community/playwright-go"
)

func openModerateVotesPanel(t *testing.T, page playwright.Page) playwright.Locator {
	t.Helper()
	openModerateLeftTab(t, page, "tools")
	panel := page.Locator("#moderate-votes-panel")
	if err := panel.WaitFor(); err != nil {
		t.Fatalf("wait moderate votes panel: %v", err)
	}
	return panel
}

func detailsOpenState(t *testing.T, details playwright.Locator) bool {
	t.Helper()
	raw, err := details.Evaluate("el => el.hasAttribute('open')", nil)
	if err != nil {
		t.Fatalf("read details open state: %v", err)
	}
	open, ok := raw.(bool)
	if !ok {
		t.Fatalf("unexpected details open state value: %#v", raw)
	}
	return open
}

func ensureDetailsOpen(t *testing.T, details playwright.Locator) {
	t.Helper()
	if err := details.WaitFor(); err != nil {
		t.Fatalf("wait details: %v", err)
	}
	if detailsOpenState(t, details) {
		return
	}
	if err := details.Locator("summary").First().Click(); err != nil {
		t.Fatalf("open details via summary click: %v", err)
	}
	waitUntil(t, 3*time.Second, func() (bool, error) {
		raw, err := details.Evaluate("el => el.hasAttribute('open')", nil)
		if err != nil {
			return false, err
		}
		open, ok := raw.(bool)
		return ok && open, nil
	}, "details to open")
}

func moderatorVoteAccordion(t *testing.T, page playwright.Page, voteName string) playwright.Locator {
	t.Helper()
	panel := openModerateVotesPanel(t, page)
	accordion := panel.Locator("details.collapse").Filter(playwright.LocatorFilterOptions{HasText: voteName}).First()
	if err := accordion.WaitFor(); err != nil {
		t.Fatalf("wait vote accordion %q: %v", voteName, err)
	}
	ensureDetailsOpen(t, accordion)
	return accordion
}

func waitPageContainsText(t *testing.T, page playwright.Page, contains string) {
	t.Helper()
	waitUntil(t, 5*time.Second, func() (bool, error) {
		text, err := page.Locator("body").TextContent()
		if err != nil {
			return false, err
		}
		return strings.Contains(text, contains), nil
	}, fmt.Sprintf("page to contain text %q", contains))
}

func createDraftVoteFromModeratorUI(t *testing.T, page playwright.Page, name, visibility string, minSelections, maxSelections int) {
	t.Helper()
	panel := openModerateVotesPanel(t, page)
	createDetails := panel.Locator("details").Filter(playwright.LocatorFilterOptions{HasText: "Create Vote"}).First()
	ensureDetailsOpen(t, createDetails)

	form := createDetails.Locator("form[hx-post$='/votes/create']").First()
	if err := form.WaitFor(); err != nil {
		t.Fatalf("wait create vote form: %v", err)
	}
	if err := form.Locator("input[name='name']").Fill(name); err != nil {
		t.Fatalf("fill vote name: %v", err)
	}
	visibilityValues := []string{visibility}
	if _, err := form.Locator("select[name='visibility']").SelectOption(playwright.SelectOptionValues{Values: &visibilityValues}); err != nil {
		t.Fatalf("select vote visibility: %v", err)
	}
	if err := form.Locator("input[name='min_selections']").Fill(strconv.Itoa(minSelections)); err != nil {
		t.Fatalf("fill min selections: %v", err)
	}
	if err := form.Locator("input[name='max_selections']").Fill(strconv.Itoa(maxSelections)); err != nil {
		t.Fatalf("fill max selections: %v", err)
	}
	if err := form.Locator("button:has-text('Create Draft Vote')").Click(); err != nil {
		t.Fatalf("submit create draft vote form: %v", err)
	}
	if err := openModerateVotesPanel(t, page).Locator("details.collapse").Filter(playwright.LocatorFilterOptions{HasText: name}).First().WaitFor(); err != nil {
		t.Fatalf("wait newly created draft vote %q in panel: %v", name, err)
	}
}

func openDraftVoteFromModeratorUI(t *testing.T, page playwright.Page, voteName string) {
	t.Helper()
	accordion := moderatorVoteAccordion(t, page, voteName)
	if err := accordion.Locator("button:has-text('Open Vote')").Click(); err != nil {
		t.Fatalf("open draft vote %q: %v", voteName, err)
	}
	waitUntil(t, 5*time.Second, func() (bool, error) {
		count, err := page.Locator("#moderate-votes-panel details.collapse").
			Filter(playwright.LocatorFilterOptions{HasText: voteName}).
			Locator("button:has-text('Close Vote')").
			Count()
		return count > 0, err
	}, fmt.Sprintf("vote %q to enter open/counting controls", voteName))
}

func closeVoteFromModeratorUI(t *testing.T, page playwright.Page, voteName string) {
	t.Helper()
	accordion := moderatorVoteAccordion(t, page, voteName)
	if err := accordion.Locator("button:has-text('Close Vote')").Click(); err != nil {
		t.Fatalf("close vote %q: %v", voteName, err)
	}
	if err := openModerateVotesPanel(t, page).WaitFor(); err != nil {
		t.Fatalf("wait votes panel after close for %q: %v", voteName, err)
	}
}

func archiveVoteFromModeratorUI(t *testing.T, page playwright.Page, voteName string) {
	t.Helper()
	accordion := moderatorVoteAccordion(t, page, voteName)
	if err := accordion.Locator("button:has-text('Archive Vote')").Click(); err != nil {
		t.Fatalf("archive vote %q: %v", voteName, err)
	}
	waitPageContainsText(t, page, "Vote archived")
}

func registerSecretCastFromModeratorUI(t *testing.T, page playwright.Page, voteName, attendeeQuery string) {
	t.Helper()
	accordion := moderatorVoteAccordion(t, page, voteName)
	form := accordion.Locator("form[hx-post*='/cast/register']").First()
	if err := form.WaitFor(); err != nil {
		t.Fatalf("wait register-cast form for %q: %v", voteName, err)
	}
	if err := form.Locator("input[name='attendee_query']").Fill(attendeeQuery); err != nil {
		t.Fatalf("fill attendee query for cast register: %v", err)
	}
	if err := form.Locator("button:has-text('Register Cast')").Click(); err != nil {
		t.Fatalf("submit cast register form: %v", err)
	}
}

func countSecretBallotFromModeratorUI(t *testing.T, page playwright.Page, voteName, receiptToken string, optionID int64) {
	t.Helper()
	accordion := moderatorVoteAccordion(t, page, voteName)
	form := accordion.Locator("form[hx-post*='/ballot/secret']").First()
	if err := form.WaitFor(); err != nil {
		t.Fatalf("wait count secret ballot form: %v", err)
	}
	if receiptToken != "" {
		if err := form.Locator("input[name='receipt_token']").Fill(receiptToken); err != nil {
			t.Fatalf("fill secret ballot receipt token: %v", err)
		}
	}
	optionSelector := fmt.Sprintf("input[name='option_id'][value='%d']", optionID)
	if err := form.Locator(optionSelector).Check(); err != nil {
		t.Fatalf("check secret ballot option %d: %v", optionID, err)
	}
	if err := form.Locator("button:has-text('Count Ballot')").Click(); err != nil {
		t.Fatalf("submit count secret ballot form: %v", err)
	}
}

func voteIDByName(t *testing.T, ts *testServer, agendaPointIDStr, voteName string) int64 {
	t.Helper()
	agendaPointID := parseID(t, agendaPointIDStr)
	votes, err := ts.repo.ListVoteDefinitionsForAgendaPoint(context.Background(), agendaPointID)
	if err != nil {
		t.Fatalf("list vote definitions for agenda point %d: %v", agendaPointID, err)
	}
	for _, vote := range votes {
		if vote.Name == voteName {
			return vote.ID
		}
	}
	t.Fatalf("vote %q not found for agenda point %d", voteName, agendaPointID)
	return 0
}

func voteOptionIDByLabel(t *testing.T, ts *testServer, voteID int64, label string) int64 {
	t.Helper()
	options, err := ts.repo.ListVoteOptions(context.Background(), voteID)
	if err != nil {
		t.Fatalf("list vote options for vote %d: %v", voteID, err)
	}
	for _, option := range options {
		if option.Label == label {
			return option.ID
		}
	}
	t.Fatalf("option %q not found for vote %d", label, voteID)
	return 0
}

func firstVoteOptionID(t *testing.T, ts *testServer, voteID int64) int64 {
	t.Helper()
	options, err := ts.repo.ListVoteOptions(context.Background(), voteID)
	if err != nil {
		t.Fatalf("list vote options for vote %d: %v", voteID, err)
	}
	if len(options) == 0 {
		t.Fatalf("vote %d has no options", voteID)
	}
	return options[0].ID
}

func voteOpenSubmitURL(baseURL, slug, meetingID string, voteID int64) string {
	return fmt.Sprintf("%s/committee/%s/meeting/%s/votes/%d/submit/open", baseURL, slug, meetingID, voteID)
}

func voteSecretSubmitURL(baseURL, slug, meetingID string, voteID int64) string {
	return fmt.Sprintf("%s/committee/%s/meeting/%s/votes/%d/submit/secret", baseURL, slug, meetingID, voteID)
}

func postFormFromPage(t *testing.T, page playwright.Page, url string, form map[string]any) (int, string) {
	t.Helper()
	raw, err := page.Evaluate(`async ({ url, form }) => {
		const params = new URLSearchParams();
		for (const [key, value] of Object.entries(form || {})) {
			if (Array.isArray(value)) {
				for (const item of value) {
					params.append(key, String(item));
				}
				continue;
			}
			if (value !== undefined && value !== null) {
				params.append(key, String(value));
			}
		}
		const response = await fetch(url, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/x-www-form-urlencoded; charset=UTF-8',
				'HX-Request': 'true'
			},
			credentials: 'same-origin',
			body: params.toString(),
		});
		return { status: response.status, body: await response.text() };
	}`, map[string]any{
		"url":  url,
		"form": form,
	})
	if err != nil {
		t.Fatalf("execute in-page form POST to %s: %v", url, err)
	}
	result, ok := raw.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected form POST result shape: %#v", raw)
	}
	status := parseJSStatusCode(t, result["status"])
	body, _ := result["body"].(string)
	return status, body
}

func postJSONFromPage(t *testing.T, page playwright.Page, url string, payload map[string]any) (int, string) {
	t.Helper()
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal JSON payload for %s: %v", url, err)
	}
	raw, err := page.Evaluate(`async ({ url, payload }) => {
		const response = await fetch(url, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			credentials: 'same-origin',
			body: payload,
		});
		return { status: response.status, body: await response.text() };
	}`, map[string]any{
		"url":     url,
		"payload": string(payloadJSON),
	})
	if err != nil {
		t.Fatalf("execute in-page JSON POST to %s: %v", url, err)
	}
	result, ok := raw.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected JSON POST result shape: %#v", raw)
	}
	status := parseJSStatusCode(t, result["status"])
	body, _ := result["body"].(string)
	return status, body
}

func parseJSStatusCode(t *testing.T, raw any) int {
	t.Helper()
	switch value := raw.(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			t.Fatalf("parse JS status code %q: %v", value, err)
		}
		return parsed
	default:
		t.Fatalf("unexpected JS status code type: %#v", raw)
		return 0
	}
}

func TestVoting_OpenVote_ModeratorAndAttendeeHappyPath_HTMX(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Voting Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Voting Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Voting Meeting", "Budget")
	ts.activateAgendaPoint(t, "test-committee", "Voting Meeting", apID)
	ts.seedAttendee(t, "test-committee", "Voting Meeting", "Alice Member", "secret-alice-vote")

	moderatorPage := newPage(t)
	userLogin(t, moderatorPage, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := moderatorPage.Goto(moderateURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto moderate page: %v", err)
	}
	moderatorURLBefore := moderatorPage.URL()

	createDraftVoteFromModeratorUI(t, moderatorPage, "Budget Vote", "open", 1, 1)
	openDraftVoteFromModeratorUI(t, moderatorPage, "Budget Vote")

	voteID := voteIDByName(t, ts, apID, "Budget Vote")
	yesOptionID := voteOptionIDByLabel(t, ts, voteID, "Yes")

	attendeePage := newPage(t)
	attendeeLoginHelper(t, attendeePage, ts.URL, "test-committee", meetingID, "secret-alice-vote")
	attendeeURLBefore := attendeePage.URL()
	voteCard := attendeePage.Locator("#live-votes-panel [data-vote-card]").Filter(playwright.LocatorFilterOptions{HasText: "Budget Vote"})
	if err := voteCard.WaitFor(); err != nil {
		t.Fatalf("wait live vote card for Budget Vote: %v", err)
	}
	if err := voteCard.Locator(fmt.Sprintf("input[name='option_id'][value='%d']", yesOptionID)).Check(); err != nil {
		t.Fatalf("select Yes option: %v", err)
	}
	if err := voteCard.Locator("button:has-text('Submit Open Ballot')").Click(); err != nil {
		t.Fatalf("submit open ballot from live page: %v", err)
	}
	waitUntil(t, 5*time.Second, func() (bool, error) {
		stats, err := ts.repo.GetVoteSubmissionStatsLive(context.Background(), voteID)
		if err != nil {
			return false, err
		}
		return stats.CastCount == 1 && stats.BallotCount == 1, nil
	}, "first open ballot to be persisted")

	if attendeePage.URL() != attendeeURLBefore {
		t.Fatalf("attendee URL changed during ballot submission: before=%s after=%s", attendeeURLBefore, attendeePage.URL())
	}

	closeVoteFromModeratorUI(t, moderatorPage, "Budget Vote")

	accordion := moderatorVoteAccordion(t, moderatorPage, "Budget Vote")
	if err := accordion.Locator("text=Final Tallies").WaitFor(); err != nil {
		t.Fatalf("wait final tallies in moderator panel: %v", err)
	}

	waitUntil(t, 3*time.Second, func() (bool, error) {
		count, err := voteCard.Locator("text=Final Results (visible for 30s)").Count()
		return count > 0, err
	}, "live timed results after vote close")

	if moderatorPage.URL() != moderatorURLBefore {
		t.Fatalf("moderator URL changed during vote lifecycle: before=%s after=%s", moderatorURLBefore, moderatorPage.URL())
	}
}

func TestVoting_CreateRejectedWithoutActiveAgenda(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "No Agenda Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "No Agenda Meeting")

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(moderateURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto moderate page: %v", err)
	}

	openModerateLeftTab(t, page, "tools")
	toolsPanel := page.Locator("#moderate-left-panel-tools")
	if err := toolsPanel.WaitFor(); err != nil {
		t.Fatalf("wait tools panel: %v", err)
	}
	if err := toolsPanel.Locator("text=No active agenda point.").WaitFor(); err != nil {
		t.Fatalf("expected no-active-agenda state in votes panel: %v", err)
	}

	status, body := postFormFromPage(t, page,
		fmt.Sprintf("%s/committee/%s/meeting/%s/votes/create", ts.URL, "test-committee", meetingID),
		map[string]any{
			"name":           "Should Fail",
			"visibility":     "open",
			"min_selections": "1",
			"max_selections": "1",
			"option_label":   []string{"Yes", "No"},
		},
	)
	if status != 200 {
		t.Fatalf("expected create-vote POST to return 200 with inline error, got %d", status)
	}
	if !strings.Contains(body, "No active agenda point.") {
		t.Fatalf("expected no-active-agenda error in response body, got: %s", body)
	}
}

func TestVoting_InvalidCreateAndUpdateDraftPayloadsRejected(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Validation Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Validation Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Validation Meeting", "Main Item")
	ts.activateAgendaPoint(t, "test-committee", "Validation Meeting", apID)

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(moderateURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto moderate page: %v", err)
	}

	createStatus, createBody := postFormFromPage(t, page,
		fmt.Sprintf("%s/committee/%s/meeting/%s/votes/create", ts.URL, "test-committee", meetingID),
		map[string]any{
			"name":           "Bad Vote",
			"visibility":     "open",
			"min_selections": "1",
			"max_selections": "1",
			"option_label":   []string{"OnlyOneChoice"},
		},
	)
	if createStatus != 200 {
		t.Fatalf("expected invalid create-vote POST to return 200 with inline error, got %d", createStatus)
	}
	if !strings.Contains(createBody, "Provide valid bounds and at least two non-empty options.") {
		t.Fatalf("expected invalid-create error in response body, got: %s", createBody)
	}

	createDraftVoteFromModeratorUI(t, page, "Draft To Validate", "open", 1, 1)
	voteID := voteIDByName(t, ts, apID, "Draft To Validate")

	updateStatus, updateBody := postFormFromPage(t, page,
		fmt.Sprintf("%s/committee/%s/meeting/%s/votes/%d/update-draft", ts.URL, "test-committee", meetingID, voteID),
		map[string]any{
			"name":           "",
			"visibility":     "open",
			"min_selections": "1",
			"max_selections": "1",
			"option_label":   []string{"OnlyOneChoice"},
		},
	)
	if updateStatus != 200 {
		t.Fatalf("expected invalid update-draft POST to return 200 with inline error, got %d", updateStatus)
	}
	if !strings.Contains(updateBody, "Provide valid draft fields and at least two non-empty options.") {
		t.Fatalf("expected invalid-update error in response body, got: %s", updateBody)
	}
}

func TestVoting_IneligibleAttendeeCannotSubmit(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Eligibility Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Eligibility Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Eligibility Meeting", "Main Item")
	ts.activateAgendaPoint(t, "test-committee", "Eligibility Meeting", apID)
	ts.seedAttendee(t, "test-committee", "Eligibility Meeting", "Eligible Member", "secret-eligible")

	moderatorPage := newPage(t)
	userLogin(t, moderatorPage, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := moderatorPage.Goto(moderateURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto moderate page: %v", err)
	}

	createDraftVoteFromModeratorUI(t, moderatorPage, "Eligibility Vote", "open", 1, 1)
	openDraftVoteFromModeratorUI(t, moderatorPage, "Eligibility Vote")

	voteID := voteIDByName(t, ts, apID, "Eligibility Vote")
	optionID := firstVoteOptionID(t, ts, voteID)

	// Added after vote open => not part of eligibility snapshot.
	ts.seedAttendee(t, "test-committee", "Eligibility Meeting", "Late Member", "secret-late")

	lateAttendeePage := newPage(t)
	attendeeLoginHelper(t, lateAttendeePage, ts.URL, "test-committee", meetingID, "secret-late")
	card := lateAttendeePage.Locator("#live-votes-panel [data-vote-card]").Filter(playwright.LocatorFilterOptions{HasText: "Eligibility Vote"})
	if err := card.WaitFor(); err != nil {
		t.Fatalf("wait eligibility vote card for late attendee: %v", err)
	}
	if err := card.Locator("span:has-text('not eligible')").WaitFor(); err != nil {
		t.Fatalf("expected not-eligible badge for late attendee: %v", err)
	}
	isDisabled, err := card.Locator("button:has-text('Submit Open Ballot')").IsDisabled()
	if err != nil {
		t.Fatalf("read live submit button disabled state: %v", err)
	}
	if !isDisabled {
		t.Fatalf("expected submit button to be disabled for ineligible attendee")
	}

	status, body := postFormFromPage(t, lateAttendeePage,
		voteOpenSubmitURL(ts.URL, "test-committee", meetingID, voteID),
		map[string]any{
			"option_id":     strconv.FormatInt(optionID, 10),
			"receipt_token": "late-forged-receipt",
		},
	)
	if status != 200 {
		t.Fatalf("expected forged ineligible submission to return 200 with inline error, got %d", status)
	}
	if !strings.Contains(body, "Open ballot rejected") {
		t.Fatalf("expected open ballot rejection for ineligible attendee, got: %s", body)
	}

	stats, err := ts.repo.GetVoteSubmissionStatsLive(context.Background(), voteID)
	if err != nil {
		t.Fatalf("load live submission stats: %v", err)
	}
	if stats.CastCount != 0 || stats.BallotCount != 0 {
		t.Fatalf("expected no casts/ballots after ineligible submission, got casts=%d ballots=%d", stats.CastCount, stats.BallotCount)
	}
}

func TestVoting_DuplicateOpenSubmissionRejected(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Duplicate Open Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Duplicate Open Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Duplicate Open Meeting", "Main Item")
	ts.activateAgendaPoint(t, "test-committee", "Duplicate Open Meeting", apID)
	ts.seedAttendee(t, "test-committee", "Duplicate Open Meeting", "Alice Member", "secret-alice-open")

	moderatorPage := newPage(t)
	userLogin(t, moderatorPage, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := moderatorPage.Goto(moderateURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto moderate page: %v", err)
	}
	createDraftVoteFromModeratorUI(t, moderatorPage, "Duplicate Open Vote", "open", 1, 1)
	openDraftVoteFromModeratorUI(t, moderatorPage, "Duplicate Open Vote")

	voteID := voteIDByName(t, ts, apID, "Duplicate Open Vote")
	optionID := firstVoteOptionID(t, ts, voteID)

	attendeePage := newPage(t)
	attendeeLoginHelper(t, attendeePage, ts.URL, "test-committee", meetingID, "secret-alice-open")
	card := attendeePage.Locator("#live-votes-panel [data-vote-card]").Filter(playwright.LocatorFilterOptions{HasText: "Duplicate Open Vote"})
	if err := card.WaitFor(); err != nil {
		t.Fatalf("wait duplicate-open vote card: %v", err)
	}
	if err := card.Locator(fmt.Sprintf("input[name='option_id'][value='%d']", optionID)).Check(); err != nil {
		t.Fatalf("check open vote option: %v", err)
	}
	if err := card.Locator("button:has-text('Submit Open Ballot')").Click(); err != nil {
		t.Fatalf("submit first open ballot: %v", err)
	}
	waitUntil(t, 5*time.Second, func() (bool, error) {
		stats, err := ts.repo.GetVoteSubmissionStatsLive(context.Background(), voteID)
		if err != nil {
			return false, err
		}
		return stats.CastCount == 1 && stats.BallotCount == 1, nil
	}, "first open ballot to be persisted")

	status, body := postFormFromPage(t, attendeePage,
		voteOpenSubmitURL(ts.URL, "test-committee", meetingID, voteID),
		map[string]any{
			"option_id":     strconv.FormatInt(optionID, 10),
			"receipt_token": "duplicate-open-2",
		},
	)
	if status != 200 {
		t.Fatalf("expected duplicate open submission to return 200 with inline error, got %d", status)
	}
	if !strings.Contains(body, "Open ballot rejected") {
		t.Fatalf("expected duplicate open ballot rejection, got: %s", body)
	}

	stats, err := ts.repo.GetVoteSubmissionStatsLive(context.Background(), voteID)
	if err != nil {
		t.Fatalf("load live submission stats: %v", err)
	}
	if stats.CastCount != 1 || stats.BallotCount != 1 {
		t.Fatalf("expected exactly one cast/ballot after duplicate submission attempt, got casts=%d ballots=%d", stats.CastCount, stats.BallotCount)
	}
}

func TestVoting_SecretVoteLifecycle_CountingAndVerificationGuards(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Secret Lifecycle Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Secret Lifecycle Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Secret Lifecycle Meeting", "Main Item")
	ts.activateAgendaPoint(t, "test-committee", "Secret Lifecycle Meeting", apID)
	ts.seedAttendee(t, "test-committee", "Secret Lifecycle Meeting", "Alice Member", "secret-alice-secret")
	ts.seedAttendee(t, "test-committee", "Secret Lifecycle Meeting", "Bob Member", "secret-bob-secret")

	moderatorPage := newPage(t)
	userLogin(t, moderatorPage, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := moderatorPage.Goto(moderateURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto moderate page: %v", err)
	}

	createDraftVoteFromModeratorUI(t, moderatorPage, "Secret Vote", "secret", 1, 1)
	openDraftVoteFromModeratorUI(t, moderatorPage, "Secret Vote")

	voteID := voteIDByName(t, ts, apID, "Secret Vote")
	yesOptionID := voteOptionIDByLabel(t, ts, voteID, "Yes")
	noOptionID := voteOptionIDByLabel(t, ts, voteID, "No")

	registerSecretCastFromModeratorUI(t, moderatorPage, "Secret Vote", "Alice Member")
	registerSecretCastFromModeratorUI(t, moderatorPage, "Secret Vote", "Bob Member")
	waitUntil(t, 5*time.Second, func() (bool, error) {
		stats, err := ts.repo.GetVoteSubmissionStatsLive(context.Background(), voteID)
		if err != nil {
			return false, err
		}
		return stats.CastCount == 2, nil
	}, "two registered casts for secret vote")
	countSecretBallotFromModeratorUI(t, moderatorPage, "Secret Vote", "secret-r1", yesOptionID)
	waitUntil(t, 5*time.Second, func() (bool, error) {
		stats, err := ts.repo.GetVoteSubmissionStatsLive(context.Background(), voteID)
		if err != nil {
			return false, err
		}
		return stats.SecretBallotCount == 1, nil
	}, "first secret ballot to be counted")

	closeVoteFromModeratorUI(t, moderatorPage, "Secret Vote")
	waitUntil(t, 5*time.Second, func() (bool, error) {
		count, err := moderatorPage.
			Locator("#moderate-votes-panel details.collapse").
			Filter(playwright.LocatorFilterOptions{HasText: "Secret Vote"}).
			Locator("summary span:has-text('counting')").
			Count()
		return count > 0, err
	}, "secret vote to enter counting state after first close")

	accordion := moderatorVoteAccordion(t, moderatorPage, "Secret Vote")
	if err := accordion.Locator("text=Results are blocked while vote is in counting state.").WaitFor(); err != nil {
		t.Fatalf("expected counting-state results block message: %v", err)
	}

	registerStatus, registerBody := postFormFromPage(t, moderatorPage,
		fmt.Sprintf("%s/committee/%s/meeting/%s/votes/%d/cast/register", ts.URL, "test-committee", meetingID, voteID),
		map[string]any{"attendee_query": "Alice Member"},
	)
	if registerStatus != 200 {
		t.Fatalf("expected register-cast during counting to return 200 with inline error, got %d", registerStatus)
	}
	if !strings.Contains(registerBody, "Failed to register cast") {
		t.Fatalf("expected register-cast rejection during counting, got: %s", registerBody)
	}

	verifyStatusCounting, verifyBodyCounting := postJSONFromPage(t, moderatorPage,
		ts.URL+"/api/votes/verify/secret",
		map[string]any{"vote_id": voteID, "receipt_token": "secret-r1"},
	)
	if verifyStatusCounting != 409 {
		t.Fatalf("expected verify/secret during counting to return 409, got %d (body=%s)", verifyStatusCounting, verifyBodyCounting)
	}
	if !strings.Contains(strings.ToLower(verifyBodyCounting), "counting") {
		t.Fatalf("expected counting error in verify/secret response, got: %s", verifyBodyCounting)
	}

	closeVoteFromModeratorUI(t, moderatorPage, "Secret Vote")
	waitUntil(t, 5*time.Second, func() (bool, error) {
		count, err := moderatorPage.
			Locator("#moderate-votes-panel details.collapse").
			Filter(playwright.LocatorFilterOptions{HasText: "Secret Vote"}).
			Locator("summary span:has-text('counting')").
			Count()
		return count > 0, err
	}, "secret vote to remain counting on second close")

	countSecretBallotFromModeratorUI(t, moderatorPage, "Secret Vote", "secret-r2", noOptionID)
	waitUntil(t, 5*time.Second, func() (bool, error) {
		stats, err := ts.repo.GetVoteSubmissionStatsLive(context.Background(), voteID)
		if err != nil {
			return false, err
		}
		return stats.SecretBallotCount == 2, nil
	}, "second secret ballot to be counted")
	closeVoteFromModeratorUI(t, moderatorPage, "Secret Vote")
	waitUntil(t, 5*time.Second, func() (bool, error) {
		count, err := moderatorPage.
			Locator("#moderate-votes-panel details.collapse").
			Filter(playwright.LocatorFilterOptions{HasText: "Secret Vote"}).
			Locator("summary span:has-text('closed')").
			Count()
		return count > 0, err
	}, "secret vote to close after counting completion")
	archiveVoteFromModeratorUI(t, moderatorPage, "Secret Vote")

	verifyStatusClosed, verifyBodyClosed := postJSONFromPage(t, moderatorPage,
		ts.URL+"/api/votes/verify/secret",
		map[string]any{"vote_id": voteID, "receipt_token": "secret-r1"},
	)
	if verifyStatusClosed != 200 {
		t.Fatalf("expected verify/secret after close/archive to return 200, got %d (body=%s)", verifyStatusClosed, verifyBodyClosed)
	}
	if !strings.Contains(verifyBodyClosed, "secret-r1") {
		t.Fatalf("expected receipt token in verify/secret success response, got: %s", verifyBodyClosed)
	}
}

func TestVoting_ModeratorEndpoints_ForbiddenForMember(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Member User", "member")
	ts.seedMeeting(t, "test-committee", "Forbidden Vote Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Forbidden Vote Meeting")

	memberPage := newPage(t)
	userLogin(t, memberPage, ts.URL, "test-committee", "member1", "pass123")

	status, _ := postFormFromPage(t, memberPage,
		fmt.Sprintf("%s/committee/%s/meeting/%s/votes/create", ts.URL, "test-committee", meetingID),
		map[string]any{
			"name":           "Forbidden Vote",
			"visibility":     "open",
			"min_selections": "1",
			"max_selections": "1",
			"option_label":   []string{"Yes", "No"},
		},
	)
	if status != 403 {
		t.Fatalf("expected member POST to moderator votes endpoint to return 403, got %d", status)
	}
}

func TestVoting_LivePanelUpdatesViaSSEOnVoteOpen(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "SSE Vote Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "SSE Vote Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "SSE Vote Meeting", "Main Item")
	ts.activateAgendaPoint(t, "test-committee", "SSE Vote Meeting", apID)
	ts.seedAttendee(t, "test-committee", "SSE Vote Meeting", "Alice Member", "secret-alice-sse-vote")

	attendeePage := newPage(t)
	attendeeLoginHelper(t, attendeePage, ts.URL, "test-committee", meetingID, "secret-alice-sse-vote")
	attendeeURLBefore := attendeePage.URL()
	if err := attendeePage.Locator("#live-votes-panel").Locator("text=No open or recently closed votes right now.").WaitFor(); err != nil {
		t.Fatalf("expected empty live votes state before moderator actions: %v", err)
	}
	// Allow the attendee SSE connection to establish before publishing updates.
	time.Sleep(800 * time.Millisecond)

	moderatorPage := newPage(t)
	userLogin(t, moderatorPage, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := moderatorPage.Goto(moderateURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto moderate page: %v", err)
	}

	createDraftVoteFromModeratorUI(t, moderatorPage, "SSE Vote", "open", 1, 1)
	openDraftVoteFromModeratorUI(t, moderatorPage, "SSE Vote")

	if err := attendeePage.Locator("#live-votes-panel [data-vote-card]").Filter(playwright.LocatorFilterOptions{HasText: "SSE Vote"}).WaitFor(); err != nil {
		t.Fatalf("expected attendee live votes panel to update via SSE after vote open: %v", err)
	}
	if attendeePage.URL() != attendeeURLBefore {
		t.Fatalf("attendee URL changed during SSE vote update: before=%s after=%s", attendeeURLBefore, attendeePage.URL())
	}
}

func TestVoting_DuplicateSecretSubmissionRejected(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Duplicate Secret Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Duplicate Secret Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Duplicate Secret Meeting", "Main Item")
	ts.activateAgendaPoint(t, "test-committee", "Duplicate Secret Meeting", apID)
	ts.seedAttendee(t, "test-committee", "Duplicate Secret Meeting", "Alice Member", "secret-alice-dup-secret")

	moderatorPage := newPage(t)
	userLogin(t, moderatorPage, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := moderatorPage.Goto(moderateURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto moderate page: %v", err)
	}
	createDraftVoteFromModeratorUI(t, moderatorPage, "Duplicate Secret Vote", "secret", 1, 1)
	openDraftVoteFromModeratorUI(t, moderatorPage, "Duplicate Secret Vote")

	voteID := voteIDByName(t, ts, apID, "Duplicate Secret Vote")
	optionID := firstVoteOptionID(t, ts, voteID)

	attendeePage := newPage(t)
	attendeeLoginHelper(t, attendeePage, ts.URL, "test-committee", meetingID, "secret-alice-dup-secret")
	card := attendeePage.Locator("#live-votes-panel [data-vote-card]").Filter(playwright.LocatorFilterOptions{HasText: "Duplicate Secret Vote"})
	if err := card.WaitFor(); err != nil {
		t.Fatalf("wait duplicate-secret vote card: %v", err)
	}
	if err := card.Locator(fmt.Sprintf("input[name='option_id'][value='%d']", optionID)).Check(); err != nil {
		t.Fatalf("check secret vote option: %v", err)
	}
	if err := card.Locator("button:has-text('Submit Secret Ballot')").Click(); err != nil {
		t.Fatalf("submit first secret ballot: %v", err)
	}
	waitUntil(t, 5*time.Second, func() (bool, error) {
		stats, err := ts.repo.GetVoteSubmissionStatsLive(context.Background(), voteID)
		if err != nil {
			return false, err
		}
		return stats.CastCount == 1 && stats.SecretBallotCount == 1, nil
	}, "first secret ballot to be persisted")

	status, body := postFormFromPage(t, attendeePage,
		voteSecretSubmitURL(ts.URL, "test-committee", meetingID, voteID),
		map[string]any{
			"option_id":                strconv.FormatInt(optionID, 10),
			"receipt_token":            "dup-secret-2",
			"nonce":                    "nonce-dup-secret-2",
			"encrypted_commitment_b64": "YQ==",
			"commitment_cipher":        "xchacha20poly1305",
			"commitment_version":       "1",
		},
	)
	if status != 200 {
		t.Fatalf("expected duplicate secret submission to return 200 with inline error, got %d", status)
	}
	if !strings.Contains(body, "Secret ballot rejected") {
		t.Fatalf("expected duplicate secret ballot rejection, got: %s", body)
	}

	stats, err := ts.repo.GetVoteSubmissionStatsLive(context.Background(), voteID)
	if err != nil {
		t.Fatalf("load live submission stats: %v", err)
	}
	if stats.CastCount != 1 || stats.SecretBallotCount != 1 {
		t.Fatalf("expected exactly one cast/secret ballot after duplicate attempt, got casts=%d secret_ballots=%d", stats.CastCount, stats.SecretBallotCount)
	}
}
