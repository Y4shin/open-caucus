package votefuzz

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Y4shin/conference-tool/internal/repository"
	"github.com/Y4shin/conference-tool/internal/repository/model"
	reposqlite "github.com/Y4shin/conference-tool/internal/repository/sqlite"
)

type executionFixture struct {
	meetingID        int64
	agendaPointID    int64
	voteDefinitionID int64
	attendeeIDByKey  map[string]int64
	optionIDByKey    map[string]int64
	optionIDs        []int64
}

func InspectSeed(ctx context.Context, seed uint64, opts ExecuteOptions) (ExecutionResult, error) {
	cfg, err := GenerateConfig(seed)
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("generate config: %w", err)
	}
	return ExecuteConfig(ctx, cfg, opts)
}

func ExecuteConfig(ctx context.Context, cfg Config, opts ExecuteOptions) (ExecutionResult, error) {
	result := ExecutionResult{
		Seed:      cfg.Seed,
		StartedAt: time.Now().UTC(),
		Config:    cfg,
	}

	dbPath := opts.DBPath
	if dbPath == "" {
		dbPath = ":memory:"
	}

	repo, err := reposqlite.New(dbPath)
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("open repository: %w", err)
	}
	defer repo.Close()

	if err := repo.MigrateUp(); err != nil {
		return ExecutionResult{}, fmt.Errorf("migrate up: %w", err)
	}

	fixture, err := seedExecutionFixture(ctx, repo, cfg)
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("seed execution fixture: %w", err)
	}

	result.ActionResults = make([]ActionResult, 0, len(cfg.Actions))
	result.VerificationChecks = make([]VerificationCheck, 0)
	result.InvariantFailures = make([]string, 0)

	for i, action := range cfg.Actions {
		actionResult := ActionResult{Index: i, Action: action}
		execErr := executeAction(ctx, repo, fixture, action, &actionResult)
		if execErr != nil {
			actionResult.Success = false
			actionResult.Error = execErr.Error()
		} else {
			actionResult.Success = true
		}

		vote, voteErr := repo.GetVoteDefinitionByID(ctx, fixture.voteDefinitionID)
		if voteErr == nil {
			actionResult.VoteStateAfter = vote.State
			result.FinalVoteState = vote.State
		}

		result.ActionResults = append(result.ActionResults, actionResult)

		invariantFailures, invErr := validateVoteInvariants(repo.DB, fixture.voteDefinitionID)
		if invErr != nil {
			result.InvariantFailures = append(result.InvariantFailures, fmt.Sprintf("invariant query error: %v", invErr))
			continue
		}
		result.InvariantFailures = append(result.InvariantFailures, invariantFailures...)
	}

	expected, err := ComputeExpectedTallies(ComputeTallyInput{
		VoteOptionIDs: fixture.optionIDs,
		ActionResults: result.ActionResults,
	})
	if err != nil {
		result.InvariantFailures = append(result.InvariantFailures, fmt.Sprintf("compute expected tallies: %v", err))
	} else {
		result.ExpectedTallies = expected.ByOptionID
	}

	actualTallies, err := loadActualTallies(repo.DB, fixture.voteDefinitionID)
	if err != nil {
		result.InvariantFailures = append(result.InvariantFailures, fmt.Sprintf("load actual tallies: %v", err))
	} else {
		result.ActualTallies = actualTallies
	}

	if result.ExpectedTallies != nil && result.ActualTallies != nil {
		for _, failure := range compareTallies(result.ExpectedTallies, result.ActualTallies) {
			result.InvariantFailures = append(result.InvariantFailures, failure)
		}
	}

	verificationChecks, verificationFailures, err := runVerificationSpotChecks(ctx, repo, fixture, result.ActionResults)
	if err != nil {
		result.InvariantFailures = append(result.InvariantFailures, fmt.Sprintf("run verification spot checks: %v", err))
	} else {
		result.VerificationChecks = verificationChecks
		result.InvariantFailures = append(result.InvariantFailures, verificationFailures...)
	}

	if result.FinalVoteState == "" {
		vote, voteErr := repo.GetVoteDefinitionByID(ctx, fixture.voteDefinitionID)
		if voteErr == nil {
			result.FinalVoteState = vote.State
		}
	}

	result.FinishedAt = time.Now().UTC()
	if len(result.InvariantFailures) == 0 {
		result.Passed = true
		result.StatusMessage = "ok"
	} else {
		result.Passed = false
		result.StatusMessage = result.InvariantFailures[0]
	}
	return result, nil
}

func executeAction(ctx context.Context, repo *reposqlite.Repository, fx executionFixture, action Action, actionResult *ActionResult) error {
	switch action.Kind {
	case ActionRegisterCast:
		if action.AttendeeKey == nil {
			return fmt.Errorf("missing attendee_key")
		}
		attendeeID, ok := fx.attendeeIDByKey[*action.AttendeeKey]
		if !ok {
			return fmt.Errorf("unknown attendee key %q", *action.AttendeeKey)
		}
		source := action.Source
		if source == "" {
			source = model.VoteCastSourceSelfSubmission
		}
		_, err := repo.RegisterVoteCast(ctx, fx.voteDefinitionID, fx.meetingID, attendeeID, source)
		return err
	case ActionSubmitOpenBallot:
		if action.AttendeeKey == nil {
			return fmt.Errorf("missing attendee_key")
		}
		attendeeID, ok := fx.attendeeIDByKey[*action.AttendeeKey]
		if !ok {
			return fmt.Errorf("unknown attendee key %q", *action.AttendeeKey)
		}
		optionIDs, err := optionIDsFromKeys(fx.optionIDByKey, action.OptionKeys)
		if err != nil {
			return err
		}
		receiptToken := action.ReceiptToken
		if receiptToken == "" {
			receiptToken = fmt.Sprintf("open-%d", actionResult.Index+1)
		}
		actionResult.ReceiptToken = receiptToken
		source := action.Source
		if source == "" {
			source = model.VoteCastSourceSelfSubmission
		}
		_, err = repo.SubmitOpenBallot(ctx, repository.OpenBallotSubmission{
			VoteDefinitionID: fx.voteDefinitionID,
			MeetingID:        fx.meetingID,
			AttendeeID:       attendeeID,
			Source:           source,
			ReceiptToken:     receiptToken,
			OptionIDs:        optionIDs,
		})
		if err == nil {
			actionResult.AppliedOptionIDs = append([]int64(nil), optionIDs...)
		}
		return err
	case ActionSubmitSecretBallot:
		optionIDs, err := optionIDsFromKeys(fx.optionIDByKey, action.OptionKeys)
		if err != nil {
			return err
		}
		receiptToken := action.ReceiptToken
		if receiptToken == "" {
			receiptToken = fmt.Sprintf("secret-%d", actionResult.Index+1)
		}
		actionResult.ReceiptToken = receiptToken
		payload := append([]byte(nil), action.EncryptedPayload...)
		if len(payload) == 0 {
			payload = []byte{byte((actionResult.Index + 1) % 255), 0xAB}
		}
		cipher := action.CommitmentCipher
		if cipher == "" {
			cipher = "xchacha20poly1305"
		}
		version := action.CommitmentVersion
		if version == 0 {
			version = 1
		}
		_, err = repo.SubmitSecretBallot(ctx, repository.SecretBallotSubmission{
			VoteDefinitionID:    fx.voteDefinitionID,
			ReceiptToken:        receiptToken,
			EncryptedCommitment: payload,
			CommitmentCipher:    cipher,
			CommitmentVersion:   version,
			OptionIDs:           optionIDs,
		})
		if err == nil {
			actionResult.AppliedOptionIDs = append([]int64(nil), optionIDs...)
		}
		return err
	case ActionCloseVote:
		closeResult, err := repo.CloseVote(ctx, fx.voteDefinitionID)
		if err != nil {
			return err
		}
		actionResult.CloseOutcome = string(closeResult.Outcome)
		if closeResult.Vote != nil {
			actionResult.VoteStateAfter = closeResult.Vote.State
		}
		return nil
	default:
		return fmt.Errorf("unsupported action kind %q", action.Kind)
	}
}

func seedExecutionFixture(ctx context.Context, repo *reposqlite.Repository, cfg Config) (executionFixture, error) {
	committeeSlug := fmt.Sprintf("fuzz-%d", cfg.Seed)
	if err := repo.CreateCommitteeWithSlug(ctx, fmt.Sprintf("Committee %d", cfg.Seed), committeeSlug); err != nil {
		return executionFixture{}, fmt.Errorf("create committee: %w", err)
	}
	committeeID, err := repo.GetCommitteeIDBySlug(ctx, committeeSlug)
	if err != nil {
		return executionFixture{}, fmt.Errorf("get committee id: %w", err)
	}

	adminAccount, err := repo.CreateAccount(ctx, "admin", "Admin", "admin-hash")
	if err != nil {
		return executionFixture{}, fmt.Errorf("create admin account: %w", err)
	}
	if err := repo.SetAccountIsAdmin(ctx, adminAccount.ID, true); err != nil {
		return executionFixture{}, fmt.Errorf("set admin flag: %w", err)
	}
	if err := repo.AssignAccountToCommittee(ctx, committeeID, adminAccount.ID, false, "chairperson"); err != nil {
		return executionFixture{}, fmt.Errorf("assign admin to committee: %w", err)
	}

	if err := repo.CreateMeeting(ctx, committeeID, fmt.Sprintf("Meeting %d", cfg.Seed), "", fmt.Sprintf("meeting-secret-%d", cfg.Seed), false); err != nil {
		return executionFixture{}, fmt.Errorf("create meeting: %w", err)
	}
	var meetingID int64
	if err := repo.DB.QueryRow("SELECT id FROM meetings WHERE committee_id = ? ORDER BY id DESC LIMIT 1", committeeID).Scan(&meetingID); err != nil {
		return executionFixture{}, fmt.Errorf("lookup meeting id: %w", err)
	}

	agendaPointID, err := insertID(repo.DB, "INSERT INTO agenda_points (meeting_id, position, title) VALUES (?, 1, ?)", meetingID, "Fuzz Agenda")
	if err != nil {
		return executionFixture{}, fmt.Errorf("insert agenda point: %w", err)
	}
	userIDByKey := make(map[string]int64, len(cfg.Users))
	for _, user := range cfg.Users {
		account, err := repo.CreateAccount(ctx, user.Username, user.FullName, "member-hash")
		if err != nil {
			return executionFixture{}, fmt.Errorf("create account %s: %w", user.Key, err)
		}
		if err := repo.AssignAccountToCommittee(ctx, committeeID, account.ID, false, "member"); err != nil {
			return executionFixture{}, fmt.Errorf("assign account %s to committee: %w", user.Key, err)
		}
		var userID int64
		if err := repo.DB.QueryRow("SELECT id FROM users WHERE committee_id = ? AND account_id = ? LIMIT 1", committeeID, account.ID).Scan(&userID); err != nil {
			return executionFixture{}, fmt.Errorf("lookup user id %s: %w", user.Key, err)
		}
		userIDByKey[user.Key] = userID
	}

	attendeeIDByKey := make(map[string]int64, len(cfg.Participants))
	for _, participant := range cfg.Participants {
		var userID *int64
		if participant.MemberKey != nil {
			mappedUserID, ok := userIDByKey[*participant.MemberKey]
			if !ok {
				return executionFixture{}, fmt.Errorf("participant %s references unknown member key %s", participant.Key, *participant.MemberKey)
			}
			copyUserID := mappedUserID
			userID = &copyUserID
		}
		attendee, err := repo.CreateAttendee(ctx, meetingID, userID, participant.FullName, participant.Secret, false)
		if err != nil {
			return executionFixture{}, fmt.Errorf("create attendee %s: %w", participant.Key, err)
		}
		attendeeIDByKey[participant.Key] = attendee.ID
	}

	voteDefinition, err := repo.CreateVoteDefinition(
		ctx,
		meetingID,
		agendaPointID,
		fmt.Sprintf("Vote %d", cfg.Seed),
		cfg.Visibility,
		int64(cfg.MinSelections),
		int64(cfg.MaxSelections),
	)
	if err != nil {
		return executionFixture{}, fmt.Errorf("create vote definition: %w", err)
	}

	voteInputs := make([]repository.VoteOptionInput, 0, len(cfg.VoteOptions))
	optionKeyByPosition := make(map[int]string, len(cfg.VoteOptions))
	for _, option := range cfg.VoteOptions {
		voteInputs = append(voteInputs, repository.VoteOptionInput{Label: option.Label, Position: int64(option.Position)})
		optionKeyByPosition[option.Position] = option.Key
	}
	if err := repo.ReplaceVoteOptions(ctx, voteDefinition.ID, voteInputs); err != nil {
		return executionFixture{}, fmt.Errorf("replace vote options: %w", err)
	}

	options, err := repo.ListVoteOptions(ctx, voteDefinition.ID)
	if err != nil {
		return executionFixture{}, fmt.Errorf("list vote options: %w", err)
	}
	optionIDByKey := make(map[string]int64, len(options))
	optionIDs := make([]int64, 0, len(options))
	for _, option := range options {
		key, ok := optionKeyByPosition[int(option.Position)]
		if !ok {
			return executionFixture{}, fmt.Errorf("missing config key for option position %d", option.Position)
		}
		optionIDByKey[key] = option.ID
		optionIDs = append(optionIDs, option.ID)
	}
	sort.Slice(optionIDs, func(i, j int) bool { return optionIDs[i] < optionIDs[j] })

	eligibleIDs := make([]int64, 0, len(cfg.EligibleParticipantKeys))
	for _, key := range cfg.EligibleParticipantKeys {
		attendeeID, ok := attendeeIDByKey[key]
		if !ok {
			return executionFixture{}, fmt.Errorf("eligible participant key %s not found", key)
		}
		eligibleIDs = append(eligibleIDs, attendeeID)
	}
	if _, err := repo.OpenVoteWithEligibleVoters(ctx, voteDefinition.ID, eligibleIDs); err != nil {
		return executionFixture{}, fmt.Errorf("open vote with eligible voters: %w", err)
	}

	return executionFixture{
		meetingID:        meetingID,
		agendaPointID:    agendaPointID,
		voteDefinitionID: voteDefinition.ID,
		attendeeIDByKey:  attendeeIDByKey,
		optionIDByKey:    optionIDByKey,
		optionIDs:        optionIDs,
	}, nil
}

func insertID(db *sql.DB, query string, args ...any) (int64, error) {
	res, err := db.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

func optionIDsFromKeys(optionIDByKey map[string]int64, optionKeys []string) ([]int64, error) {
	optionIDs := make([]int64, 0, len(optionKeys))
	for _, key := range optionKeys {
		id, ok := optionIDByKey[key]
		if !ok {
			return nil, fmt.Errorf("unknown option key %q", key)
		}
		optionIDs = append(optionIDs, id)
	}
	return optionIDs, nil
}

func validateVoteInvariants(db *sql.DB, voteDefinitionID int64) ([]string, error) {
	failures := make([]string, 0)

	allowedStates := map[string]struct{}{
		model.VoteStateDraft:    {},
		model.VoteStateOpen:     {},
		model.VoteStateCounting: {},
		model.VoteStateClosed:   {},
		model.VoteStateArchived: {},
	}
	var state string
	if err := db.QueryRow("SELECT state FROM vote_definitions WHERE id = ?", voteDefinitionID).Scan(&state); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []string{"vote definition missing"}, nil
		}
		return nil, err
	}
	if _, ok := allowedStates[state]; !ok {
		failures = append(failures, fmt.Sprintf("invalid vote state value %q", state))
	}

	invalidOpen, err := scalarInt64(db, `SELECT COUNT(*)
FROM vote_ballots
WHERE vote_definition_id = ?
  AND attendee_id IS NOT NULL
  AND (
      cast_id IS NULL
      OR encrypted_commitment IS NOT NULL
      OR commitment_cipher IS NOT NULL
      OR commitment_version IS NOT NULL
  )`, voteDefinitionID)
	if err != nil {
		return nil, err
	}
	if invalidOpen > 0 {
		failures = append(failures, fmt.Sprintf("%d invalid open ballot rows", invalidOpen))
	}

	invalidSecret, err := scalarInt64(db, `SELECT COUNT(*)
FROM vote_ballots
WHERE vote_definition_id = ?
  AND attendee_id IS NULL
  AND (
      cast_id IS NOT NULL
      OR encrypted_commitment IS NULL
      OR commitment_cipher IS NULL
      OR commitment_version IS NULL
  )`, voteDefinitionID)
	if err != nil {
		return nil, err
	}
	if invalidSecret > 0 {
		failures = append(failures, fmt.Sprintf("%d invalid secret ballot rows", invalidSecret))
	}

	invalidSelections, err := scalarInt64(db, `SELECT COUNT(*)
FROM vote_ballot_selections vbs
LEFT JOIN vote_options vo
    ON vo.id = vbs.option_id AND vo.vote_definition_id = vbs.vote_definition_id
WHERE vbs.vote_definition_id = ?
  AND vo.id IS NULL`, voteDefinitionID)
	if err != nil {
		return nil, err
	}
	if invalidSelections > 0 {
		failures = append(failures, fmt.Sprintf("%d ballot selections reference missing options", invalidSelections))
	}

	secretBallots, err := scalarInt64(db, "SELECT COUNT(*) FROM vote_ballots WHERE vote_definition_id = ? AND attendee_id IS NULL", voteDefinitionID)
	if err != nil {
		return nil, err
	}
	casts, err := scalarInt64(db, "SELECT COUNT(*) FROM vote_casts WHERE vote_definition_id = ?", voteDefinitionID)
	if err != nil {
		return nil, err
	}
	if secretBallots > casts {
		failures = append(failures, fmt.Sprintf("secret ballots (%d) exceed casts (%d)", secretBallots, casts))
	}

	return failures, nil
}

func scalarInt64(db *sql.DB, query string, args ...any) (int64, error) {
	var value int64
	if err := db.QueryRow(query, args...).Scan(&value); err != nil {
		return 0, err
	}
	return value, nil
}

func loadActualTallies(db *sql.DB, voteDefinitionID int64) (map[int64]int64, error) {
	rows, err := db.Query(`SELECT vo.id, CAST(COALESCE(COUNT(vbs.option_id), 0) AS INTEGER) AS tally_count
FROM vote_options vo
LEFT JOIN vote_ballot_selections vbs
    ON vbs.option_id = vo.id AND vbs.vote_definition_id = vo.vote_definition_id
WHERE vo.vote_definition_id = ?
GROUP BY vo.id, vo.position
ORDER BY vo.position ASC, vo.id ASC`, voteDefinitionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tallies := make(map[int64]int64)
	for rows.Next() {
		var optionID int64
		var count int64
		if err := rows.Scan(&optionID, &count); err != nil {
			return nil, err
		}
		tallies[optionID] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tallies, nil
}

func compareTallies(expected, actual map[int64]int64) []string {
	failures := make([]string, 0)
	keys := make([]int64, 0, len(expected))
	for optionID := range expected {
		keys = append(keys, optionID)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	for _, optionID := range keys {
		expectedCount := expected[optionID]
		actualCount := actual[optionID]
		if expectedCount != actualCount {
			failures = append(failures, fmt.Sprintf("tally mismatch for option %d: expected=%d actual=%d", optionID, expectedCount, actualCount))
		}
	}
	return failures
}

func runVerificationSpotChecks(
	ctx context.Context,
	repo *reposqlite.Repository,
	fx executionFixture,
	actionResults []ActionResult,
) ([]VerificationCheck, []string, error) {
	vote, err := repo.GetVoteDefinitionByID(ctx, fx.voteDefinitionID)
	if err != nil {
		return nil, nil, fmt.Errorf("load vote definition: %w", err)
	}

	openCandidates := make([]ActionResult, 0)
	secretCandidates := make([]ActionResult, 0)
	for _, result := range actionResults {
		if !result.Success || result.ReceiptToken == "" {
			continue
		}
		switch result.Action.Kind {
		case ActionSubmitOpenBallot:
			openCandidates = append(openCandidates, result)
		case ActionSubmitSecretBallot:
			secretCandidates = append(secretCandidates, result)
		}
	}

	limit := 2
	if len(openCandidates) > limit {
		openCandidates = openCandidates[:limit]
	}
	if len(secretCandidates) > limit {
		secretCandidates = secretCandidates[:limit]
	}

	checks := make([]VerificationCheck, 0, len(openCandidates)+len(secretCandidates))
	failures := make([]string, 0)

	expectBlocked := vote.State == model.VoteStateCounting
	for _, candidate := range openCandidates {
		check := VerificationCheck{
			Index:           candidate.Index,
			Kind:            "open",
			ReceiptToken:    candidate.ReceiptToken,
			ExpectedBlocked: expectBlocked,
		}
		verification, verifyErr := repo.VerifyOpenBallotByReceipt(ctx, fx.voteDefinitionID, candidate.ReceiptToken)
		if expectBlocked {
			if verifyErr == nil || !strings.Contains(verifyErr.Error(), "counting") {
				check.Passed = false
				check.Message = fmt.Sprintf("expected blocked open verification in counting, got: %v", verifyErr)
				failures = append(failures, fmt.Sprintf("open verification not blocked for receipt %s", candidate.ReceiptToken))
			} else {
				check.Passed = true
			}
			checks = append(checks, check)
			continue
		}
		if verifyErr != nil {
			check.Passed = false
			check.Message = verifyErr.Error()
			failures = append(failures, fmt.Sprintf("open verification failed for receipt %s: %v", candidate.ReceiptToken, verifyErr))
			checks = append(checks, check)
			continue
		}
		expectedAttendeeID := int64(0)
		if candidate.Action.AttendeeKey != nil {
			expectedAttendeeID = fx.attendeeIDByKey[*candidate.Action.AttendeeKey]
		}
		if verification.AttendeeID != expectedAttendeeID {
			check.Passed = false
			check.Message = fmt.Sprintf("attendee mismatch expected=%d actual=%d", expectedAttendeeID, verification.AttendeeID)
			failures = append(failures, fmt.Sprintf("open verification attendee mismatch for receipt %s", candidate.ReceiptToken))
			checks = append(checks, check)
			continue
		}
		if verification.VoteName != vote.Name {
			check.Passed = false
			check.Message = fmt.Sprintf("vote name mismatch expected=%q actual=%q", vote.Name, verification.VoteName)
			failures = append(failures, fmt.Sprintf("open verification vote name mismatch for receipt %s", candidate.ReceiptToken))
			checks = append(checks, check)
			continue
		}
		if !equalInt64Sets(candidate.AppliedOptionIDs, verification.ChoiceOptionIDs) {
			check.Passed = false
			check.Message = fmt.Sprintf("choice mismatch expected=%v actual=%v", candidate.AppliedOptionIDs, verification.ChoiceOptionIDs)
			failures = append(failures, fmt.Sprintf("open verification choices mismatch for receipt %s", candidate.ReceiptToken))
			checks = append(checks, check)
			continue
		}
		check.Passed = true
		checks = append(checks, check)
	}

	for _, candidate := range secretCandidates {
		check := VerificationCheck{
			Index:           candidate.Index,
			Kind:            "secret",
			ReceiptToken:    candidate.ReceiptToken,
			ExpectedBlocked: expectBlocked,
		}
		verification, verifyErr := repo.VerifySecretBallotByReceipt(ctx, fx.voteDefinitionID, candidate.ReceiptToken)
		if expectBlocked {
			if verifyErr == nil || !strings.Contains(verifyErr.Error(), "counting") {
				check.Passed = false
				check.Message = fmt.Sprintf("expected blocked secret verification in counting, got: %v", verifyErr)
				failures = append(failures, fmt.Sprintf("secret verification not blocked for receipt %s", candidate.ReceiptToken))
			} else {
				check.Passed = true
			}
			checks = append(checks, check)
			continue
		}
		if verifyErr != nil {
			check.Passed = false
			check.Message = verifyErr.Error()
			failures = append(failures, fmt.Sprintf("secret verification failed for receipt %s: %v", candidate.ReceiptToken, verifyErr))
			checks = append(checks, check)
			continue
		}
		if verification.VoteName != vote.Name {
			check.Passed = false
			check.Message = fmt.Sprintf("vote name mismatch expected=%q actual=%q", vote.Name, verification.VoteName)
			failures = append(failures, fmt.Sprintf("secret verification vote name mismatch for receipt %s", candidate.ReceiptToken))
			checks = append(checks, check)
			continue
		}
		if verification.ReceiptToken != candidate.ReceiptToken {
			check.Passed = false
			check.Message = fmt.Sprintf("receipt mismatch expected=%q actual=%q", candidate.ReceiptToken, verification.ReceiptToken)
			failures = append(failures, fmt.Sprintf("secret verification receipt mismatch for receipt %s", candidate.ReceiptToken))
			checks = append(checks, check)
			continue
		}
		if len(verification.EncryptedCommitment) == 0 || verification.CommitmentCipher == "" || verification.CommitmentVersion <= 0 {
			check.Passed = false
			check.Message = "secret verification returned incomplete commitment metadata"
			failures = append(failures, fmt.Sprintf("secret verification commitment incomplete for receipt %s", candidate.ReceiptToken))
			checks = append(checks, check)
			continue
		}
		check.Passed = true
		checks = append(checks, check)
	}

	return checks, failures, nil
}

func equalInt64Sets(expected, actual []int64) bool {
	if len(expected) != len(actual) {
		return false
	}
	expectedCopy := append([]int64(nil), expected...)
	actualCopy := append([]int64(nil), actual...)
	sort.Slice(expectedCopy, func(i, j int) bool { return expectedCopy[i] < expectedCopy[j] })
	sort.Slice(actualCopy, func(i, j int) bool { return actualCopy[i] < actualCopy[j] })
	for i := range expectedCopy {
		if expectedCopy[i] != actualCopy[i] {
			return false
		}
	}
	return true
}
