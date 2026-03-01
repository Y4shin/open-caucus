package sqlite

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Y4shin/conference-tool/internal/repository"
	"github.com/Y4shin/conference-tool/internal/repository/model"
)

type votingFixture struct {
	meetingID          int64
	agendaPointID      int64
	otherMeetingID     int64
	attendeeID1        int64
	attendeeID2        int64
	otherMeetingUserID int64
}

func mustExecInsertID(t *testing.T, repo *Repository, query string, args ...any) int64 {
	t.Helper()
	res, err := repo.DB.Exec(query, args...)
	if err != nil {
		t.Fatalf("exec insert failed for %q: %v", query, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("read last insert id for %q: %v", query, err)
	}
	return id
}

func seedVotingFixture(t *testing.T, repo *Repository) votingFixture {
	t.Helper()

	committeeID := mustExecInsertID(t, repo, "INSERT INTO committees (name, slug) VALUES ('Committee', 'committee')")
	meetingID := mustExecInsertID(t, repo, "INSERT INTO meetings (committee_id, name, secret, signup_open) VALUES (?, 'Meeting', 'secret-m', 0)", committeeID)
	agendaPointID := mustExecInsertID(t, repo, "INSERT INTO agenda_points (meeting_id, position, title) VALUES (?, 1, 'Agenda')", meetingID)

	attendeeID1 := mustExecInsertID(t, repo, "INSERT INTO attendees (meeting_id, full_name, secret, quoted, attendee_number) VALUES (?, 'Alice', 's-alice', 0, 1)", meetingID)
	attendeeID2 := mustExecInsertID(t, repo, "INSERT INTO attendees (meeting_id, full_name, secret, quoted, attendee_number) VALUES (?, 'Bob', 's-bob', 0, 2)", meetingID)

	otherMeetingID := mustExecInsertID(t, repo, "INSERT INTO meetings (committee_id, name, secret, signup_open) VALUES (?, 'Meeting 2', 'secret-m2', 0)", committeeID)
	otherMeetingUserID := mustExecInsertID(t, repo, "INSERT INTO attendees (meeting_id, full_name, secret, quoted, attendee_number) VALUES (?, 'Mallory', 's-mal', 0, 1)", otherMeetingID)

	return votingFixture{
		meetingID:          meetingID,
		agendaPointID:      agendaPointID,
		otherMeetingID:     otherMeetingID,
		attendeeID1:        attendeeID1,
		attendeeID2:        attendeeID2,
		otherMeetingUserID: otherMeetingUserID,
	}
}

func createDraftVote(t *testing.T, repo *Repository, fx votingFixture, visibility string, minSelections, maxSelections int64) *model.VoteDefinition {
	t.Helper()
	vote, err := repo.CreateVoteDefinition(context.Background(), fx.meetingID, fx.agendaPointID, "Vote", visibility, minSelections, maxSelections)
	if err != nil {
		t.Fatalf("create vote definition: %v", err)
	}
	return vote
}

func TestOpenVoteWithEligibleVoters_TransactionRollback(t *testing.T) {
	repo := newTestRepo(t)
	if err := repo.MigrateUp(); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	fx := seedVotingFixture(t, repo)
	vote := createDraftVote(t, repo, fx, model.VoteVisibilityOpen, 1, 2)

	_, err := repo.OpenVoteWithEligibleVoters(context.Background(), vote.ID, []int64{fx.attendeeID1, fx.otherMeetingUserID})
	if err == nil {
		t.Fatalf("expected open vote to fail for mixed-meeting eligibility snapshot")
	}

	reloaded, err := repo.GetVoteDefinitionByID(context.Background(), vote.ID)
	if err != nil {
		t.Fatalf("reload vote definition: %v", err)
	}
	if reloaded.State != model.VoteStateDraft {
		t.Fatalf("expected vote to remain draft, got %s", reloaded.State)
	}

	eligible, err := repo.ListEligibleVoters(context.Background(), vote.ID)
	if err != nil {
		t.Fatalf("list eligible voters: %v", err)
	}
	if len(eligible) != 0 {
		t.Fatalf("expected no eligible voters after rollback, got %d", len(eligible))
	}
}

func TestVoteDraftOnlyMutability(t *testing.T) {
	repo := newTestRepo(t)
	if err := repo.MigrateUp(); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	fx := seedVotingFixture(t, repo)
	vote := createDraftVote(t, repo, fx, model.VoteVisibilityOpen, 1, 2)

	if err := repo.ReplaceVoteOptions(context.Background(), vote.ID, []repository.VoteOptionInput{
		{Label: "Yes", Position: 1},
		{Label: "No", Position: 2},
	}); err != nil {
		t.Fatalf("replace vote options in draft: %v", err)
	}

	if _, err := repo.OpenVoteWithEligibleVoters(context.Background(), vote.ID, []int64{fx.attendeeID1}); err != nil {
		t.Fatalf("open vote: %v", err)
	}

	if err := repo.ReplaceVoteOptions(context.Background(), vote.ID, []repository.VoteOptionInput{
		{Label: "Changed", Position: 1},
	}); err == nil {
		t.Fatalf("expected replace vote options to fail outside draft")
	}

	if _, err := repo.UpdateVoteDefinitionDraft(context.Background(), vote.ID, fx.meetingID, fx.agendaPointID, "Changed", model.VoteVisibilityOpen, 1, 1); err == nil {
		t.Fatalf("expected update vote definition draft to fail outside draft")
	}
}

func TestRegisterVoteCast_EligibilityAndDedupe(t *testing.T) {
	repo := newTestRepo(t)
	if err := repo.MigrateUp(); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	fx := seedVotingFixture(t, repo)
	vote := createDraftVote(t, repo, fx, model.VoteVisibilityOpen, 1, 2)
	if _, err := repo.OpenVoteWithEligibleVoters(context.Background(), vote.ID, []int64{fx.attendeeID1}); err != nil {
		t.Fatalf("open vote: %v", err)
	}

	if _, err := repo.RegisterVoteCast(context.Background(), vote.ID, fx.meetingID, fx.attendeeID1, model.VoteCastSourceSelfSubmission); err != nil {
		t.Fatalf("register eligible cast: %v", err)
	}
	if _, err := repo.RegisterVoteCast(context.Background(), vote.ID, fx.meetingID, fx.attendeeID1, model.VoteCastSourceSelfSubmission); err == nil {
		t.Fatalf("expected duplicate cast to fail")
	}
	if _, err := repo.RegisterVoteCast(context.Background(), vote.ID, fx.meetingID, fx.attendeeID2, model.VoteCastSourceSelfSubmission); err == nil {
		t.Fatalf("expected ineligible attendee cast to fail")
	}
}

func TestSubmitOpenBallot_BoundsAndVerification(t *testing.T) {
	repo := newTestRepo(t)
	if err := repo.MigrateUp(); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	fx := seedVotingFixture(t, repo)
	vote := createDraftVote(t, repo, fx, model.VoteVisibilityOpen, 1, 2)
	if err := repo.ReplaceVoteOptions(context.Background(), vote.ID, []repository.VoteOptionInput{
		{Label: "Yes", Position: 1},
		{Label: "No", Position: 2},
		{Label: "Abstain", Position: 3},
	}); err != nil {
		t.Fatalf("replace vote options: %v", err)
	}
	options, err := repo.ListVoteOptions(context.Background(), vote.ID)
	if err != nil {
		t.Fatalf("list vote options: %v", err)
	}
	if _, err := repo.OpenVoteWithEligibleVoters(context.Background(), vote.ID, []int64{fx.attendeeID1}); err != nil {
		t.Fatalf("open vote: %v", err)
	}

	_, err = repo.SubmitOpenBallot(context.Background(), repository.OpenBallotSubmission{
		VoteDefinitionID: vote.ID,
		MeetingID:        fx.meetingID,
		AttendeeID:       fx.attendeeID1,
		Source:           model.VoteCastSourceSelfSubmission,
		ReceiptToken:     "open-r1",
		OptionIDs:        []int64{},
	})
	if err == nil {
		t.Fatalf("expected selection-bounds violation for empty open ballot")
	}

	_, err = repo.SubmitOpenBallot(context.Background(), repository.OpenBallotSubmission{
		VoteDefinitionID: vote.ID,
		MeetingID:        fx.meetingID,
		AttendeeID:       fx.attendeeID1,
		Source:           model.VoteCastSourceSelfSubmission,
		ReceiptToken:     "open-r2",
		OptionIDs:        []int64{9999},
	})
	if err == nil {
		t.Fatalf("expected invalid option id to fail")
	}

	_, err = repo.SubmitOpenBallot(context.Background(), repository.OpenBallotSubmission{
		VoteDefinitionID: vote.ID,
		MeetingID:        fx.meetingID,
		AttendeeID:       fx.attendeeID1,
		Source:           model.VoteCastSourceSelfSubmission,
		ReceiptToken:     "open-ok",
		OptionIDs:        []int64{options[0].ID, options[1].ID},
	})
	if err != nil {
		t.Fatalf("submit valid open ballot: %v", err)
	}

	verification, err := repo.VerifyOpenBallotByReceipt(context.Background(), vote.ID, "open-ok")
	if err != nil {
		t.Fatalf("verify open receipt: %v", err)
	}
	if verification.AttendeeNumber != 1 {
		t.Fatalf("expected attendee number 1, got %d", verification.AttendeeNumber)
	}
	if len(verification.ChoiceOptionIDs) != 2 {
		t.Fatalf("expected 2 selected options, got %d", len(verification.ChoiceOptionIDs))
	}

	tallies, err := repo.GetVoteTallies(context.Background(), vote.ID)
	if err != nil {
		t.Fatalf("get vote tallies: %v", err)
	}
	if len(tallies) != 3 {
		t.Fatalf("expected 3 tally rows, got %d", len(tallies))
	}
}

func TestSubmitSecretBallot_CastLimitAndVerification(t *testing.T) {
	repo := newTestRepo(t)
	if err := repo.MigrateUp(); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	fx := seedVotingFixture(t, repo)
	vote := createDraftVote(t, repo, fx, model.VoteVisibilitySecret, 0, 2)
	if err := repo.ReplaceVoteOptions(context.Background(), vote.ID, []repository.VoteOptionInput{
		{Label: "A", Position: 1},
		{Label: "B", Position: 2},
	}); err != nil {
		t.Fatalf("replace vote options: %v", err)
	}
	options, err := repo.ListVoteOptions(context.Background(), vote.ID)
	if err != nil {
		t.Fatalf("list vote options: %v", err)
	}
	if _, err := repo.OpenVoteWithEligibleVoters(context.Background(), vote.ID, []int64{fx.attendeeID1, fx.attendeeID2}); err != nil {
		t.Fatalf("open vote: %v", err)
	}

	if _, err := repo.RegisterVoteCast(context.Background(), vote.ID, fx.meetingID, fx.attendeeID1, model.VoteCastSourceManualSubmission); err != nil {
		t.Fatalf("register cast 1: %v", err)
	}
	if _, err := repo.RegisterVoteCast(context.Background(), vote.ID, fx.meetingID, fx.attendeeID2, model.VoteCastSourceManualSubmission); err != nil {
		t.Fatalf("register cast 2: %v", err)
	}

	if _, err := repo.SubmitSecretBallot(context.Background(), repository.SecretBallotSubmission{
		VoteDefinitionID:    vote.ID,
		ReceiptToken:        "secret-1",
		EncryptedCommitment: []byte{0x01},
		CommitmentCipher:    "xchacha20poly1305",
		CommitmentVersion:   1,
		OptionIDs:           []int64{options[0].ID},
	}); err != nil {
		t.Fatalf("submit secret ballot 1: %v", err)
	}

	if _, err := repo.SubmitSecretBallot(context.Background(), repository.SecretBallotSubmission{
		VoteDefinitionID:    vote.ID,
		ReceiptToken:        "secret-2",
		EncryptedCommitment: []byte{0x02},
		CommitmentCipher:    "xchacha20poly1305",
		CommitmentVersion:   1,
		OptionIDs:           []int64{options[1].ID},
	}); err != nil {
		t.Fatalf("submit secret ballot 2: %v", err)
	}

	if _, err := repo.SubmitSecretBallot(context.Background(), repository.SecretBallotSubmission{
		VoteDefinitionID:    vote.ID,
		ReceiptToken:        "secret-3",
		EncryptedCommitment: []byte{0x03},
		CommitmentCipher:    "xchacha20poly1305",
		CommitmentVersion:   1,
		OptionIDs:           []int64{options[1].ID},
	}); err == nil {
		t.Fatalf("expected third secret ballot to fail when casts are exhausted")
	}

	verification, err := repo.VerifySecretBallotByReceipt(context.Background(), vote.ID, "secret-1")
	if err != nil {
		t.Fatalf("verify secret receipt: %v", err)
	}
	if verification.CommitmentCipher != "xchacha20poly1305" {
		t.Fatalf("unexpected commitment cipher: %s", verification.CommitmentCipher)
	}

	stats, err := repo.GetVoteSubmissionStats(context.Background(), vote.ID)
	if err != nil {
		t.Fatalf("get vote submission stats: %v", err)
	}
	if stats.CastCount != 2 || stats.SecretBallotCount != 2 || stats.OpenBallotCount != 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestCloseVote_SecretAdaptiveCountingLifecycle(t *testing.T) {
	repo := newTestRepo(t)
	if err := repo.MigrateUp(); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	fx := seedVotingFixture(t, repo)
	vote := createDraftVote(t, repo, fx, model.VoteVisibilitySecret, 0, 1)
	if err := repo.ReplaceVoteOptions(context.Background(), vote.ID, []repository.VoteOptionInput{
		{Label: "A", Position: 1},
		{Label: "B", Position: 2},
	}); err != nil {
		t.Fatalf("replace vote options: %v", err)
	}
	options, err := repo.ListVoteOptions(context.Background(), vote.ID)
	if err != nil {
		t.Fatalf("list vote options: %v", err)
	}
	if _, err := repo.OpenVoteWithEligibleVoters(context.Background(), vote.ID, []int64{fx.attendeeID1, fx.attendeeID2}); err != nil {
		t.Fatalf("open vote: %v", err)
	}
	if _, err := repo.RegisterVoteCast(context.Background(), vote.ID, fx.meetingID, fx.attendeeID1, model.VoteCastSourceManualSubmission); err != nil {
		t.Fatalf("register cast 1: %v", err)
	}
	if _, err := repo.RegisterVoteCast(context.Background(), vote.ID, fx.meetingID, fx.attendeeID2, model.VoteCastSourceManualSubmission); err != nil {
		t.Fatalf("register cast 2: %v", err)
	}
	if _, err := repo.SubmitSecretBallot(context.Background(), repository.SecretBallotSubmission{
		VoteDefinitionID:    vote.ID,
		ReceiptToken:        "secret-c1",
		EncryptedCommitment: []byte{0x11},
		CommitmentCipher:    "xchacha20poly1305",
		CommitmentVersion:   1,
		OptionIDs:           []int64{options[0].ID},
	}); err != nil {
		t.Fatalf("submit first secret ballot: %v", err)
	}

	closeResult, err := repo.CloseVote(context.Background(), vote.ID)
	if err != nil {
		t.Fatalf("close vote first call: %v", err)
	}
	if closeResult.Outcome != model.CloseVoteOutcomeEnteredCounting {
		t.Fatalf("expected entered_counting, got %s", closeResult.Outcome)
	}
	if closeResult.Vote.State != model.VoteStateCounting {
		t.Fatalf("expected counting state, got %s", closeResult.Vote.State)
	}

	stillCountingResult, err := repo.CloseVote(context.Background(), vote.ID)
	if err != nil {
		t.Fatalf("close vote second call: %v", err)
	}
	if stillCountingResult.Outcome != model.CloseVoteOutcomeStillCounting {
		t.Fatalf("expected still_counting, got %s", stillCountingResult.Outcome)
	}
	if stillCountingResult.Vote.State != model.VoteStateCounting {
		t.Fatalf("expected counting state, got %s", stillCountingResult.Vote.State)
	}

	if _, err := repo.RegisterVoteCast(context.Background(), vote.ID, fx.meetingID, fx.attendeeID1, model.VoteCastSourceManualSubmission); err == nil {
		t.Fatalf("expected register cast to fail in counting state")
	}

	if _, err := repo.SubmitSecretBallot(context.Background(), repository.SecretBallotSubmission{
		VoteDefinitionID:    vote.ID,
		ReceiptToken:        "secret-c2",
		EncryptedCommitment: []byte{0x12},
		CommitmentCipher:    "xchacha20poly1305",
		CommitmentVersion:   1,
		OptionIDs:           []int64{options[1].ID},
	}); err != nil {
		t.Fatalf("submit second secret ballot in counting: %v", err)
	}

	closedResult, err := repo.CloseVote(context.Background(), vote.ID)
	if err != nil {
		t.Fatalf("close vote third call: %v", err)
	}
	if closedResult.Outcome != model.CloseVoteOutcomeClosed {
		t.Fatalf("expected closed outcome, got %s", closedResult.Outcome)
	}
	if closedResult.Vote.State != model.VoteStateClosed {
		t.Fatalf("expected closed state, got %s", closedResult.Vote.State)
	}
	if closedResult.Vote.ClosedAt == nil {
		t.Fatalf("expected closed_at to be set")
	}

	_, err = repo.CloseVote(context.Background(), vote.ID)
	if err == nil {
		t.Fatalf("expected closing already-closed vote to fail")
	}
	var closeStateErr model.VoteCloseStateError
	if !errors.As(err, &closeStateErr) {
		t.Fatalf("expected VoteCloseStateError, got %T (%v)", err, err)
	}
}

func TestCloseVote_OpenVisibilityAlwaysCloses(t *testing.T) {
	repo := newTestRepo(t)
	if err := repo.MigrateUp(); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	fx := seedVotingFixture(t, repo)
	vote := createDraftVote(t, repo, fx, model.VoteVisibilityOpen, 0, 1)
	if err := repo.ReplaceVoteOptions(context.Background(), vote.ID, []repository.VoteOptionInput{
		{Label: "Yes", Position: 1},
		{Label: "No", Position: 2},
	}); err != nil {
		t.Fatalf("replace vote options: %v", err)
	}
	if _, err := repo.OpenVoteWithEligibleVoters(context.Background(), vote.ID, []int64{fx.attendeeID1, fx.attendeeID2}); err != nil {
		t.Fatalf("open vote: %v", err)
	}
	if _, err := repo.RegisterVoteCast(context.Background(), vote.ID, fx.meetingID, fx.attendeeID1, model.VoteCastSourceSelfSubmission); err != nil {
		t.Fatalf("register cast: %v", err)
	}

	result, err := repo.CloseVote(context.Background(), vote.ID)
	if err != nil {
		t.Fatalf("close vote: %v", err)
	}
	if result.Outcome != model.CloseVoteOutcomeClosed {
		t.Fatalf("expected closed outcome, got %s", result.Outcome)
	}
	if result.Vote.State != model.VoteStateClosed {
		t.Fatalf("expected closed state, got %s", result.Vote.State)
	}
}

func TestVoteReadsBlockedInCounting(t *testing.T) {
	repo := newTestRepo(t)
	if err := repo.MigrateUp(); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	fx := seedVotingFixture(t, repo)
	vote := createDraftVote(t, repo, fx, model.VoteVisibilitySecret, 0, 1)
	if err := repo.ReplaceVoteOptions(context.Background(), vote.ID, []repository.VoteOptionInput{
		{Label: "A", Position: 1},
	}); err != nil {
		t.Fatalf("replace vote options: %v", err)
	}
	options, err := repo.ListVoteOptions(context.Background(), vote.ID)
	if err != nil {
		t.Fatalf("list vote options: %v", err)
	}
	if _, err := repo.OpenVoteWithEligibleVoters(context.Background(), vote.ID, []int64{fx.attendeeID1, fx.attendeeID2}); err != nil {
		t.Fatalf("open vote: %v", err)
	}
	if _, err := repo.RegisterVoteCast(context.Background(), vote.ID, fx.meetingID, fx.attendeeID1, model.VoteCastSourceManualSubmission); err != nil {
		t.Fatalf("register cast 1: %v", err)
	}
	if _, err := repo.RegisterVoteCast(context.Background(), vote.ID, fx.meetingID, fx.attendeeID2, model.VoteCastSourceManualSubmission); err != nil {
		t.Fatalf("register cast 2: %v", err)
	}
	if _, err := repo.SubmitSecretBallot(context.Background(), repository.SecretBallotSubmission{
		VoteDefinitionID:    vote.ID,
		ReceiptToken:        "secret-read-1",
		EncryptedCommitment: []byte{0x21},
		CommitmentCipher:    "xchacha20poly1305",
		CommitmentVersion:   1,
		OptionIDs:           []int64{options[0].ID},
	}); err != nil {
		t.Fatalf("submit first secret ballot: %v", err)
	}
	result, err := repo.CloseVote(context.Background(), vote.ID)
	if err != nil {
		t.Fatalf("close vote first call: %v", err)
	}
	if result.Outcome != model.CloseVoteOutcomeEnteredCounting {
		t.Fatalf("expected entered_counting outcome, got %s", result.Outcome)
	}

	if _, err := repo.VerifyOpenBallotByReceipt(context.Background(), vote.ID, "does-not-matter"); err == nil || !strings.Contains(err.Error(), "counting") {
		t.Fatalf("expected VerifyOpenBallotByReceipt to fail in counting, got %v", err)
	}
	if _, err := repo.VerifySecretBallotByReceipt(context.Background(), vote.ID, "secret-read-1"); err == nil || !strings.Contains(err.Error(), "counting") {
		t.Fatalf("expected VerifySecretBallotByReceipt to fail in counting, got %v", err)
	}
	if _, err := repo.GetVoteTallies(context.Background(), vote.ID); err == nil || !strings.Contains(err.Error(), "counting") {
		t.Fatalf("expected GetVoteTallies to fail in counting, got %v", err)
	}
	if _, err := repo.GetVoteSubmissionStats(context.Background(), vote.ID); err == nil || !strings.Contains(err.Error(), "counting") {
		t.Fatalf("expected GetVoteSubmissionStats to fail in counting, got %v", err)
	}

	if _, err := repo.SubmitSecretBallot(context.Background(), repository.SecretBallotSubmission{
		VoteDefinitionID:    vote.ID,
		ReceiptToken:        "secret-read-2",
		EncryptedCommitment: []byte{0x22},
		CommitmentCipher:    "xchacha20poly1305",
		CommitmentVersion:   1,
		OptionIDs:           []int64{options[0].ID},
	}); err != nil {
		t.Fatalf("submit second secret ballot in counting: %v", err)
	}
	closed, err := repo.CloseVote(context.Background(), vote.ID)
	if err != nil {
		t.Fatalf("close vote second call: %v", err)
	}
	if closed.Outcome != model.CloseVoteOutcomeClosed {
		t.Fatalf("expected closed outcome, got %s", closed.Outcome)
	}

	if _, err := repo.VerifySecretBallotByReceipt(context.Background(), vote.ID, "secret-read-1"); err != nil {
		t.Fatalf("verify secret receipt after closing: %v", err)
	}
	if _, err := repo.GetVoteTallies(context.Background(), vote.ID); err != nil {
		t.Fatalf("get vote tallies after closing: %v", err)
	}
	if _, err := repo.GetVoteSubmissionStats(context.Background(), vote.ID); err != nil {
		t.Fatalf("get vote submission stats after closing: %v", err)
	}
}
