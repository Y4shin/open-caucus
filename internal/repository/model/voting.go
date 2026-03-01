package model

import (
	"fmt"
	"time"
)

const (
	VoteVisibilityOpen   = "open"
	VoteVisibilitySecret = "secret"
)

const (
	VoteStateDraft    = "draft"
	VoteStateOpen     = "open"
	VoteStateCounting = "counting"
	VoteStateClosed   = "closed"
	VoteStateArchived = "archived"
)

type CloseVoteOutcome string

const (
	CloseVoteOutcomeEnteredCounting CloseVoteOutcome = "entered_counting"
	CloseVoteOutcomeClosed          CloseVoteOutcome = "closed"
	CloseVoteOutcomeStillCounting   CloseVoteOutcome = "still_counting"
)

const (
	VoteCastSourceSelfSubmission   = "self_submission"
	VoteCastSourceManualSubmission = "manual_submission"
)

// VoteDefinition describes one configurable vote.
type VoteDefinition struct {
	ID            int64
	MeetingID     int64
	AgendaPointID int64
	MotionID      *int64
	Name          string
	Visibility    string
	State         string
	MinSelections int64
	MaxSelections int64
	OpenedAt      *time.Time
	ClosedAt      *time.Time
	ArchivedAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// VoteOption is one selectable choice in a vote definition.
type VoteOption struct {
	ID               int64
	VoteDefinitionID int64
	Label            string
	Position         int64
	CreatedAt        time.Time
}

// EligibleVoter captures one attendee in the eligibility snapshot for a vote.
type EligibleVoter struct {
	VoteDefinitionID int64
	MeetingID        int64
	AttendeeID       int64
	CreatedAt        time.Time
}

// VoteCast tracks that one attendee submitted or handed in a vote.
type VoteCast struct {
	ID               int64
	VoteDefinitionID int64
	MeetingID        int64
	AttendeeID       int64
	Source           string
	CreatedAt        time.Time
}

// VoteBallot stores the recorded ballot data.
// Open ballots have AttendeeID/CastID set and no encrypted commitment.
// Secret ballots have neither AttendeeID nor CastID and carry encrypted commitment fields.
type VoteBallot struct {
	ID                  int64
	VoteDefinitionID    int64
	CastID              *int64
	AttendeeID          *int64
	ReceiptToken        string
	EncryptedCommitment *[]byte
	CommitmentCipher    *string
	CommitmentVersion   *int64
	CreatedAt           time.Time
}

// VoteBallotSelection links one ballot to one selected option.
type VoteBallotSelection struct {
	BallotID         int64
	VoteDefinitionID int64
	OptionID         int64
	CreatedAt        time.Time
}

// VoteOpenVerification contains data returned for open vote receipt verification.
type VoteOpenVerification struct {
	VoteDefinitionID int64
	VoteName         string
	AttendeeID       int64
	AttendeeNumber   int64
	ReceiptToken     string
	ChoiceLabels     []string
	ChoiceOptionIDs  []int64
}

// VoteSecretVerification contains data returned for secret vote receipt verification.
type VoteSecretVerification struct {
	VoteDefinitionID    int64
	VoteName            string
	ReceiptToken        string
	EncryptedCommitment []byte
	CommitmentCipher    string
	CommitmentVersion   int64
}

// VoteTallyRow aggregates count per option for one vote.
type VoteTallyRow struct {
	OptionID int64
	Label    string
	Count    int64
}

// VoteSubmissionStats reports counts across vote tracking tables.
type VoteSubmissionStats struct {
	EligibleCount     int64
	CastCount         int64
	BallotCount       int64
	OpenBallotCount   int64
	SecretBallotCount int64
}

// CloseVoteResult describes the outcome of one CloseVote call.
type CloseVoteResult struct {
	Vote    *VoteDefinition
	Outcome CloseVoteOutcome
}

// VoteCloseStateError is returned when CloseVote is called in a disallowed state.
type VoteCloseStateError struct {
	State string
}

func (e VoteCloseStateError) Error() string {
	return fmt.Sprintf("cannot close vote in state %q", e.State)
}
