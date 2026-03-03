//go:build e2e

package e2e_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// attendeeHTTPClient logs an attendee in via a plain HTTP request (no browser) and
// returns an *http.Client whose cookie jar holds the resulting session cookie.
// This avoids spinning up a browser context for every concurrent voter.
func attendeeHTTPClient(t *testing.T, baseURL, slug, meetingID, secret string) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create cookie jar: %v", err)
	}
	client := &http.Client{Jar: jar}
	loginURL := fmt.Sprintf("%s/committee/%s/meeting/%s/attendee-login", baseURL, slug, meetingID)
	resp, err := client.PostForm(loginURL, url.Values{"secret": {secret}})
	if err != nil {
		t.Fatalf("HTTP login for attendee secret %q: %v", secret, err)
	}
	resp.Body.Close()
	return client
}

// TestVoting_Concurrent20Attendees_TallyIsCorrect verifies that when 20 attendees
// submit their open ballots simultaneously the server serialises the writes
// correctly: every ballot is counted exactly once and the final per-option tallies
// add up to the expected totals.
func TestVoting_Concurrent20Attendees_TallyIsCorrect(t *testing.T) {
	const (
		numAttendees  = 20
		yesVoterCount = 13
		noVoterCount  = numAttendees - yesVoterCount
		slug          = "concurrent-vote-committee"
		meetingName   = "Concurrent Vote Meeting"
		voteName      = "Big Decision"
	)

	ts := newTestServer(t)
	ts.seedCommittee(t, "Concurrent Vote Committee", slug)
	ts.seedUser(t, slug, "chair1", "pass123", "Chair Person", "chairperson")
	ts.seedMeeting(t, slug, meetingName, "")
	meetingID := ts.getMeetingID(t, slug, meetingName)
	apID := ts.seedAgendaPoint(t, slug, meetingName, "Big Decision Agenda")
	ts.activateAgendaPoint(t, slug, meetingName, apID)

	// Seed all 20 attendees upfront so they are all captured in the eligibility
	// snapshot when the vote is opened.
	secrets := make([]string, numAttendees)
	for i := range secrets {
		secrets[i] = fmt.Sprintf("secret-concurrent-%02d", i+1)
		ts.seedAttendee(t, slug, meetingName, fmt.Sprintf("Member %02d", i+1), secrets[i])
	}

	// --- Moderator opens the vote via the browser UI ---
	moderatorPage := newPage(t)
	userLogin(t, moderatorPage, ts.URL, slug, "chair1", "pass123")
	if _, err := moderatorPage.Goto(moderateURL(ts.URL, slug, meetingID)); err != nil {
		t.Fatalf("goto moderate page: %v", err)
	}
	moderatorURLBefore := moderatorPage.URL()

	createDraftVoteFromModeratorUI(t, moderatorPage, voteName, "open", 1, 1)
	openDraftVoteFromModeratorUI(t, moderatorPage, voteName)

	voteID := voteIDByName(t, ts, apID, voteName)
	yesOptionID := voteOptionIDByLabel(t, ts, voteID, "Yes")
	noOptionID := voteOptionIDByLabel(t, ts, voteID, "No")

	// Verify all 20 attendees are in the eligibility snapshot.
	stats, err := ts.repo.GetVoteSubmissionStatsLive(context.Background(), voteID)
	if err != nil {
		t.Fatalf("get live stats after open: %v", err)
	}
	if stats.EligibleCount != int64(numAttendees) {
		t.Fatalf("eligible count after open: got %d, want %d", stats.EligibleCount, numAttendees)
	}

	// --- Authenticate every attendee via plain HTTP (no browser per attendee) ---
	clients := make([]*http.Client, numAttendees)
	for i, secret := range secrets {
		clients[i] = attendeeHTTPClient(t, ts.URL, slug, meetingID, secret)
	}

	// Assign vote choices: first yesVoterCount vote Yes, the rest vote No.
	optionIDs := make([]int64, numAttendees)
	for i := range optionIDs {
		if i < yesVoterCount {
			optionIDs[i] = yesOptionID
		} else {
			optionIDs[i] = noOptionID
		}
	}

	submitURL := voteOpenSubmitURL(ts.URL, slug, meetingID, voteID)

	// --- Fire all ballot submissions concurrently ---
	type submitResult struct {
		status int
		body   string
		err    error
	}
	results := make([]submitResult, numAttendees)
	var wg sync.WaitGroup
	for i := range clients {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			form := url.Values{
				"option_id":     {strconv.FormatInt(optionIDs[i], 10)},
				"receipt_token": {fmt.Sprintf("receipt-concurrent-%02d", i+1)},
			}
			req, err := http.NewRequest(http.MethodPost, submitURL, strings.NewReader(form.Encode()))
			if err != nil {
				results[i] = submitResult{err: fmt.Errorf("build request: %w", err)}
				return
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")
			resp, err := clients[i].Do(req)
			if err != nil {
				results[i] = submitResult{err: fmt.Errorf("do request: %w", err)}
				return
			}
			defer resp.Body.Close()
			bodyBytes, _ := io.ReadAll(resp.Body)
			results[i] = submitResult{status: resp.StatusCode, body: string(bodyBytes)}
		}()
	}
	wg.Wait()

	// All submissions must succeed with an HTTP 200 and must not be rejected.
	for i, r := range results {
		if r.err != nil {
			t.Errorf("attendee %d request error: %v", i+1, r.err)
			continue
		}
		if r.status != http.StatusOK {
			t.Errorf("attendee %d: got HTTP %d, body=%q", i+1, r.status, r.body)
		}
		if strings.Contains(r.body, "rejected") {
			t.Errorf("attendee %d ballot was rejected: %s", i+1, r.body)
		}
	}

	// --- Wait for all 20 ballots to be persisted ---
	waitUntil(t, 10*time.Second, func() (bool, error) {
		s, err := ts.repo.GetVoteSubmissionStatsLive(context.Background(), voteID)
		if err != nil {
			return false, err
		}
		return s.BallotCount == int64(numAttendees), nil
	}, fmt.Sprintf("all %d concurrent ballots to be persisted", numAttendees))

	// Confirm cast count also matches (one cast registration per voter).
	stats, err = ts.repo.GetVoteSubmissionStatsLive(context.Background(), voteID)
	if err != nil {
		t.Fatalf("get live stats after concurrent submissions: %v", err)
	}
	if stats.CastCount != int64(numAttendees) {
		t.Errorf("cast count: got %d, want %d", stats.CastCount, numAttendees)
	}
	if stats.BallotCount != int64(numAttendees) {
		t.Errorf("ballot count: got %d, want %d", stats.BallotCount, numAttendees)
	}

	// --- Moderator closes the vote via the browser UI ---
	closeVoteFromModeratorUI(t, moderatorPage, voteName)

	// Moderator panel should show final tallies.
	accordion := moderatorVoteAccordion(t, moderatorPage, voteName)
	if err := accordion.Locator("text=Final Tallies").WaitFor(); err != nil {
		t.Fatalf("wait for Final Tallies in moderator UI: %v", err)
	}

	// --- Verify per-option tallies are exactly correct ---
	tallies, err := ts.repo.GetVoteTallies(context.Background(), voteID)
	if err != nil {
		t.Fatalf("get vote tallies: %v", err)
	}

	tallyMap := make(map[int64]int64, len(tallies))
	for _, row := range tallies {
		tallyMap[row.OptionID] = row.Count
	}

	if got := tallyMap[yesOptionID]; got != int64(yesVoterCount) {
		t.Errorf("Yes tally: got %d, want %d", got, yesVoterCount)
	}
	if got := tallyMap[noOptionID]; got != int64(noVoterCount) {
		t.Errorf("No tally: got %d, want %d", got, noVoterCount)
	}

	var totalVotes int64
	for _, count := range tallyMap {
		totalVotes += count
	}
	if totalVotes != int64(numAttendees) {
		t.Errorf("total tally: got %d, want %d (expected no double-counts or missing votes)", totalVotes, numAttendees)
	}

	// Moderator page must not have done a full navigation at any point.
	if moderatorPage.URL() != moderatorURLBefore {
		t.Errorf("moderator URL changed during vote lifecycle: before=%s after=%s", moderatorURLBefore, moderatorPage.URL())
	}
}
