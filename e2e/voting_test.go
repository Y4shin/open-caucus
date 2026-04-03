//go:build e2e

package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	connect "connectrpc.com/connect"
	playwright "github.com/playwright-community/playwright-go"

	votesv1 "github.com/Y4shin/conference-tool/gen/go/conference/votes/v1"
	votesv1connect "github.com/Y4shin/conference-tool/gen/go/conference/votes/v1/votesv1connect"
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
	accordion := panel.Locator("details").Filter(playwright.LocatorFilterOptions{HasText: voteName}).First()
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

func verifySecretReceiptViaConnect(t *testing.T, baseURL string, voteID int64, receiptToken string) (*votesv1.VerifySecretReceiptResponse, error) {
	t.Helper()
	client := votesv1connect.NewVoteServiceClient(&http.Client{}, baseURL+"/api")
	resp, err := client.VerifySecretReceipt(context.Background(), connect.NewRequest(&votesv1.VerifySecretReceiptRequest{
		VoteId:       strconv.FormatInt(voteID, 10),
		ReceiptToken: receiptToken,
	}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func createDraftVoteFromModeratorUI(t *testing.T, page playwright.Page, name, visibility string, minSelections, maxSelections int) {
	t.Helper()
	panel := openModerateVotesPanel(t, page)
	createDetails := panel.Locator("details").Filter(playwright.LocatorFilterOptions{HasText: "Create Vote"}).First()
	ensureDetailsOpen(t, createDetails)

	form := createDetails.Locator("form").First()
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
	if err := openModerateVotesPanel(t, page).Locator("details").Filter(playwright.LocatorFilterOptions{HasText: name}).First().WaitFor(); err != nil {
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
		count, err := page.Locator("#moderate-votes-panel details").
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
	form := accordion.Locator("[data-testid='manage-vote-register-cast-form']").First()
	if err := form.WaitFor(); err != nil {
		t.Fatalf("wait register-cast form for %q: %v", voteName, err)
	}
	if err := form.Locator("[data-testid='register-cast-attendee-query']").Fill(attendeeQuery); err != nil {
		t.Fatalf("fill attendee query for cast register: %v", err)
	}
	if err := form.Locator("[data-testid='register-cast-submit']").Click(); err != nil {
		t.Fatalf("submit cast register form: %v", err)
	}
}

func countSecretBallotFromModeratorUI(t *testing.T, page playwright.Page, voteName, receiptToken string, optionID int64) {
	t.Helper()
	accordion := moderatorVoteAccordion(t, page, voteName)
	form := accordion.Locator("[data-testid='manage-vote-count-secret-form']").First()
	if err := form.WaitFor(); err != nil {
		t.Fatalf("wait count secret ballot form: %v", err)
	}
	if receiptToken != "" {
		if err := form.Locator("[data-testid='count-secret-receipt-token']").Fill(receiptToken); err != nil {
			t.Fatalf("fill secret ballot receipt token: %v", err)
		}
	}
	optionSelector := fmt.Sprintf("input[value='%d']", optionID)
	if err := form.Locator(optionSelector).Check(); err != nil {
		t.Fatalf("check secret ballot option %d: %v", optionID, err)
	}
	if err := form.Locator("[data-testid='count-secret-submit']").Click(); err != nil {
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


// connectJSONCallViaPage calls a Connect JSON RPC endpoint from within the browser
// so that the browser's session cookies are included automatically.
// Returns the HTTP status and the JSON "code" field (empty string on success).
func connectJSONCallViaPage(t *testing.T, page playwright.Page, baseURL, method string, body any) (int, string) {
	t.Helper()
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal connect body for %s: %v", method, err)
	}
	raw, err := page.Evaluate(`async ({ url, body }) => {
		const res = await fetch(url, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			credentials: 'same-origin',
			body,
		});
		let code = '';
		try { const j = await res.json(); code = j.code || ''; } catch(e) {}
		return { status: res.status, code };
	}`, map[string]any{
		"url":  baseURL + "/api/conference.votes.v1.VoteService/" + method,
		"body": string(bodyBytes),
	})
	if err != nil {
		t.Fatalf("evaluate connect call %s: %v", method, err)
	}
	m, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("unexpected evaluate result type for %s: %T", method, raw)
	}
	var status int
	switch v := m["status"].(type) {
	case float64:
		status = int(v)
	case int:
		status = v
	case int64:
		status = int(v)
	default:
		t.Fatalf("unexpected status type for %s: %T = %v", method, m["status"], m["status"])
	}
	code, _ := m["code"].(string)
	return status, code
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

func TestVoting_OpenVote_ModeratorAndAttendeeHappyPath(t *testing.T) {
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

func TestVoting_DraftEditorStaysOpenAcrossVoteRefreshes(t *testing.T) {
	ts := newTestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, "test-committee", "Draft Vote Meeting", "")
	meetingID := ts.getMeetingID(t, "test-committee", "Draft Vote Meeting")
	apID := ts.seedAgendaPoint(t, "test-committee", "Draft Vote Meeting", "Budget")
	ts.activateAgendaPoint(t, "test-committee", "Draft Vote Meeting", apID)

	page := newPage(t)
	userLogin(t, page, ts.URL, "test-committee", "chair1", "pass123")
	if _, err := page.Goto(moderateURL(ts.URL, "test-committee", meetingID)); err != nil {
		t.Fatalf("goto moderate page: %v", err)
	}
	urlBefore := page.URL()

	createDraftVoteFromModeratorUI(t, page, "Persistent Draft Vote", "open", 1, 1)

	accordion := moderatorVoteAccordion(t, page, "Persistent Draft Vote")
	draftEditor := accordion.Locator("[data-vote-draft-editor]").First()
	ensureDetailsOpen(t, draftEditor)

	editorForm := draftEditor.Locator("form").First()
	if err := editorForm.WaitFor(); err != nil {
		t.Fatalf("wait draft editor form: %v", err)
	}
	optionInput := editorForm.Locator("input[name='option_label']").First()
	if err := optionInput.Fill("Approve"); err != nil {
		t.Fatalf("fill updated draft option label: %v", err)
	}
	if err := editorForm.Locator("button:has-text('Save Draft')").Click(); err != nil {
		t.Fatalf("save draft vote: %v", err)
	}

	waitUntil(t, 5*time.Second, func() (bool, error) {
		content, err := accordion.TextContent()
		if err != nil {
			return false, err
		}
		return strings.Contains(content, "Approve"), nil
	}, "updated draft vote option label to appear")

	if !detailsOpenState(t, accordion) {
		t.Fatalf("vote accordion closed immediately after saving the draft")
	}
	if !detailsOpenState(t, draftEditor) {
		t.Fatalf("draft editor closed immediately after saving the draft")
	}

	time.Sleep(2500 * time.Millisecond)

	if !detailsOpenState(t, accordion) {
		t.Fatalf("vote accordion closed after background refresh")
	}
	if !detailsOpenState(t, draftEditor) {
		t.Fatalf("draft editor closed after background refresh")
	}
	if page.URL() != urlBefore {
		t.Fatalf("moderator URL changed while editing draft vote: before=%s after=%s", urlBefore, page.URL())
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

	createStatus, createCode := connectJSONCallViaPage(t, page, ts.URL, "CreateVote", map[string]any{
		"committee_slug":  "test-committee",
		"meeting_id":      meetingID,
		"name":            "Should Fail",
		"visibility":      "open",
		"min_selections":  1,
		"max_selections":  1,
		"option_labels":   []string{"Yes", "No"},
	})
	if createStatus == 200 {
		t.Fatalf("expected create-vote to fail without active agenda, got 200")
	}
	if createCode != "invalid_argument" && createCode != "failed_precondition" {
		t.Fatalf("expected invalid_argument or failed_precondition, got code=%q status=%d", createCode, createStatus)
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

	// Empty name should be rejected.
	createStatus, createCode := connectJSONCallViaPage(t, page, ts.URL, "CreateVote", map[string]any{
		"committee_slug": "test-committee",
		"meeting_id":     meetingID,
		"name":           "",
		"visibility":     "open",
		"min_selections": 1,
		"max_selections": 1,
		"option_labels":  []string{"Yes", "No"},
	})
	if createStatus == 200 {
		t.Fatalf("expected create-vote with empty name to fail, got 200")
	}
	if createCode != "invalid_argument" {
		t.Fatalf("expected invalid_argument for empty-name create, got code=%q status=%d", createCode, createStatus)
	}

	// Invalid visibility should be rejected.
	createStatus2, createCode2 := connectJSONCallViaPage(t, page, ts.URL, "CreateVote", map[string]any{
		"committee_slug": "test-committee",
		"meeting_id":     meetingID,
		"name":           "Bad Visibility Vote",
		"visibility":     "invalid-vis",
		"min_selections": 1,
		"max_selections": 1,
		"option_labels":  []string{"Yes", "No"},
	})
	if createStatus2 == 200 {
		t.Fatalf("expected create-vote with invalid visibility to fail, got 200")
	}
	if createCode2 != "invalid_argument" {
		t.Fatalf("expected invalid_argument for bad visibility, got code=%q status=%d", createCode2, createStatus2)
	}

	// Non-parseable vote_id on update should be rejected.
	updateStatus, updateCode := connectJSONCallViaPage(t, page, ts.URL, "UpdateVoteDraft", map[string]any{
		"committee_slug": "test-committee",
		"meeting_id":     meetingID,
		"vote_id":        "not-a-number",
		"name":           "Valid Name",
		"visibility":     "open",
		"min_selections": 1,
		"max_selections": 1,
		"option_labels":  []string{"Yes", "No"},
	})
	if updateStatus == 200 {
		t.Fatalf("expected update-draft with invalid vote_id to fail, got 200")
	}
	if updateCode != "invalid_argument" {
		t.Fatalf("expected invalid_argument for bad vote_id update, got code=%q status=%d", updateCode, updateStatus)
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

	submitStatus, submitCode := connectJSONCallViaPage(t, lateAttendeePage, ts.URL, "SubmitBallot", map[string]any{
		"committee_slug":       "test-committee",
		"meeting_id":           meetingID,
		"vote_id":              strconv.FormatInt(voteID, 10),
		"selected_option_ids":  []string{strconv.FormatInt(optionID, 10)},
	})
	if submitStatus == 200 {
		t.Fatalf("expected ineligible ballot submission to fail, got 200")
	}
	if submitCode == "" {
		t.Fatalf("expected error code for ineligible ballot, got empty code (status=%d)", submitStatus)
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

	dupStatus, dupCode := connectJSONCallViaPage(t, attendeePage, ts.URL, "SubmitBallot", map[string]any{
		"committee_slug":       "test-committee",
		"meeting_id":           meetingID,
		"vote_id":              strconv.FormatInt(voteID, 10),
		"selected_option_ids":  []string{strconv.FormatInt(optionID, 10)},
	})
	if dupStatus == 200 {
		t.Fatalf("expected duplicate open submission to fail, got 200")
	}
	if dupCode == "" {
		t.Fatalf("expected error code for duplicate ballot, got empty code (status=%d)", dupStatus)
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
			Locator("#moderate-votes-panel details").
			Filter(playwright.LocatorFilterOptions{HasText: "Secret Vote"}).
			Locator("summary span:has-text('counting')").
			Count()
		return count > 0, err
	}, "secret vote to enter counting state after first close")

	accordion := moderatorVoteAccordion(t, moderatorPage, "Secret Vote")
	if err := accordion.Locator("text=Results are blocked while vote is in counting state.").WaitFor(); err != nil {
		t.Fatalf("expected counting-state results block message: %v", err)
	}

	aliceAttendeeIDStr := ts.getAttendeeIDForMeeting(t, "test-committee", "Secret Lifecycle Meeting", "Alice Member")
	registerStatus, registerCode := connectJSONCallViaPage(t, moderatorPage, ts.URL, "RegisterCast", map[string]any{
		"committee_slug": "test-committee",
		"meeting_id":     meetingID,
		"vote_id":        strconv.FormatInt(voteID, 10),
		"attendee_id":    aliceAttendeeIDStr,
	})
	if registerStatus == 200 {
		t.Fatalf("expected register-cast during counting to fail, got 200")
	}
	if registerCode == "" {
		t.Fatalf("expected error code for register-cast during counting, got empty code (status=%d)", registerStatus)
	}

	_, verifyErrCounting := verifySecretReceiptViaConnect(t, ts.URL, voteID, "secret-r1")
	if verifyErrCounting == nil {
		t.Fatal("expected verify secret receipt during counting to fail")
	}
	if connect.CodeOf(verifyErrCounting) != connect.CodeFailedPrecondition {
		t.Fatalf("expected verify secret receipt during counting to return failed precondition, got %v", connect.CodeOf(verifyErrCounting))
	}
	if !strings.Contains(strings.ToLower(verifyErrCounting.Error()), "counting") {
		t.Fatalf("expected counting error in verify secret receipt response, got: %v", verifyErrCounting)
	}

	closeVoteFromModeratorUI(t, moderatorPage, "Secret Vote")
	waitUntil(t, 5*time.Second, func() (bool, error) {
		count, err := moderatorPage.
			Locator("#moderate-votes-panel details").
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
			Locator("#moderate-votes-panel details").
			Filter(playwright.LocatorFilterOptions{HasText: "Secret Vote"}).
			Locator("summary span:has-text('closed')").
			Count()
		return count > 0, err
	}, "secret vote to close after counting completion")
	archiveVoteFromModeratorUI(t, moderatorPage, "Secret Vote")

	verifyClosed, verifyErrClosed := verifySecretReceiptViaConnect(t, ts.URL, voteID, "secret-r1")
	if verifyErrClosed != nil {
		t.Fatalf("expected verify secret receipt after close/archive to return success, got %v", verifyErrClosed)
	}
	if verifyClosed.GetReceiptToken() != "secret-r1" {
		t.Fatalf("expected receipt token in verify secret receipt success response, got: %q", verifyClosed.GetReceiptToken())
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

	createStatus, createCode := connectJSONCallViaPage(t, memberPage, ts.URL, "CreateVote", map[string]any{
		"committee_slug": "test-committee",
		"meeting_id":     meetingID,
		"name":           "Forbidden Vote",
		"visibility":     "open",
		"min_selections": 1,
		"max_selections": 1,
		"option_labels":  []string{"Yes", "No"},
	})
	if createStatus == 200 {
		t.Fatalf("expected member create-vote to fail, got 200")
	}
	if createCode != "permission_denied" {
		t.Fatalf("expected permission_denied for member create-vote, got code=%q status=%d", createCode, createStatus)
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

	dupStatus, dupCode := connectJSONCallViaPage(t, attendeePage, ts.URL, "SubmitBallot", map[string]any{
		"committee_slug":      "test-committee",
		"meeting_id":          meetingID,
		"vote_id":             strconv.FormatInt(voteID, 10),
		"selected_option_ids": []string{strconv.FormatInt(optionID, 10)},
	})
	if dupStatus == 200 {
		t.Fatalf("expected duplicate secret submission to fail, got 200")
	}
	if dupCode == "" {
		t.Fatalf("expected error code for duplicate secret ballot, got empty code (status=%d)", dupStatus)
	}

	stats, err := ts.repo.GetVoteSubmissionStatsLive(context.Background(), voteID)
	if err != nil {
		t.Fatalf("load live submission stats: %v", err)
	}
	if stats.CastCount != 1 || stats.SecretBallotCount != 1 {
		t.Fatalf("expected exactly one cast/secret ballot after duplicate attempt, got casts=%d secret_ballots=%d", stats.CastCount, stats.SecretBallotCount)
	}
}
