package votefuzz

import "time"

type Scenario string

const (
	ScenarioSelfSubmitted   Scenario = "self_submitted"
	ScenarioManualSubmitted Scenario = "manual_submitted"
	ScenarioHybrid          Scenario = "hybrid"
)

type VoteKind string

const (
	VoteKindYesNoAbstain VoteKind = "yes_no_abstain"
	VoteKindNOfM         VoteKind = "n_of_m"
)

type ActionKind string

const (
	ActionRegisterCast       ActionKind = "register_cast"
	ActionSubmitOpenBallot   ActionKind = "submit_open_ballot"
	ActionSubmitSecretBallot ActionKind = "submit_secret_ballot"
	ActionCloseVote          ActionKind = "close_vote"
)

type UserSpec struct {
	Key      string `yaml:"key"`
	Username string `yaml:"username"`
	FullName string `yaml:"full_name"`
}

type ParticipantSpec struct {
	Key       string  `yaml:"key"`
	FullName  string  `yaml:"full_name"`
	Secret    string  `yaml:"secret"`
	MemberKey *string `yaml:"member_key,omitempty"`
}

type VoteOptionSpec struct {
	Key      string `yaml:"key"`
	Label    string `yaml:"label"`
	Position int    `yaml:"position"`
}

type Action struct {
	Kind              ActionKind `yaml:"kind"`
	AttendeeKey       *string    `yaml:"attendee_key,omitempty"`
	Source            string     `yaml:"source,omitempty"`
	OptionKeys        []string   `yaml:"option_keys,omitempty"`
	ReceiptToken      string     `yaml:"receipt_token,omitempty"`
	EncryptedPayload  []byte     `yaml:"encrypted_payload,omitempty"`
	CommitmentCipher  string     `yaml:"commitment_cipher,omitempty"`
	CommitmentVersion int64      `yaml:"commitment_version,omitempty"`
}

type Config struct {
	Seed                       uint64            `yaml:"seed"`
	SubSeeds                   map[string]uint64 `yaml:"sub_seeds"`
	UserCount                  int               `yaml:"user_count"`
	Users                      []UserSpec        `yaml:"users"`
	ParticipantCount           int               `yaml:"participant_count"`
	ParticipantMemberCount     int               `yaml:"participant_member_count"`
	ParticipantMemberUserKeys  []string          `yaml:"participant_member_user_keys"`
	Participants               []ParticipantSpec `yaml:"participants"`
	EligibleCount              int               `yaml:"eligible_count"`
	EligibleParticipantKeys    []string          `yaml:"eligible_participant_keys"`
	Visibility                 string            `yaml:"visibility"`
	VoteKind                   VoteKind          `yaml:"vote_kind"`
	MinSelections              int               `yaml:"min_selections"`
	MaxSelections              int               `yaml:"max_selections"`
	VoteOptions                []VoteOptionSpec  `yaml:"vote_options"`
	Scenario                   Scenario          `yaml:"scenario"`
	ManualVoterCount           int               `yaml:"manual_voter_count"`
	ManualVoterParticipantKeys []string          `yaml:"manual_voter_participant_keys"`
	Actions                    []Action          `yaml:"actions"`
}

type ActionResult struct {
	Index            int     `yaml:"index"`
	Action           Action  `yaml:"action"`
	Success          bool    `yaml:"success"`
	Error            string  `yaml:"error,omitempty"`
	VoteStateAfter   string  `yaml:"vote_state_after,omitempty"`
	CloseOutcome     string  `yaml:"close_outcome,omitempty"`
	ReceiptToken     string  `yaml:"receipt_token,omitempty"`
	AppliedOptionIDs []int64 `yaml:"applied_option_ids,omitempty"`
}

type ComputeTallyInput struct {
	VoteOptionIDs []int64        `yaml:"vote_option_ids"`
	ActionResults []ActionResult `yaml:"action_results"`
}

type ExpectedTallies struct {
	ByOptionID        map[int64]int64 `yaml:"by_option_id"`
	BallotsCounted    int64           `yaml:"ballots_counted"`
	SelectionsCounted int64           `yaml:"selections_counted"`
	IgnoredActions    int64           `yaml:"ignored_actions"`
	Diagnostics       []string        `yaml:"diagnostics,omitempty"`
}

type ExecuteOptions struct {
	DBPath string
}

type ExecutionResult struct {
	Seed               uint64              `yaml:"seed"`
	StartedAt          time.Time           `yaml:"started_at"`
	FinishedAt         time.Time           `yaml:"finished_at"`
	Config             Config              `yaml:"config"`
	ActionResults      []ActionResult      `yaml:"action_results"`
	VerificationChecks []VerificationCheck `yaml:"verification_checks"`
	InvariantFailures  []string            `yaml:"invariant_failures"`
	ExpectedTallies    map[int64]int64     `yaml:"expected_tallies"`
	ActualTallies      map[int64]int64     `yaml:"actual_tallies"`
	FinalVoteState     string              `yaml:"final_vote_state"`
	Passed             bool                `yaml:"passed"`
	StatusMessage      string              `yaml:"status_message"`
}

type VerificationCheck struct {
	Index           int    `yaml:"index"`
	Kind            string `yaml:"kind"`
	ReceiptToken    string `yaml:"receipt_token"`
	ExpectedBlocked bool   `yaml:"expected_blocked"`
	Passed          bool   `yaml:"passed"`
	Message         string `yaml:"message,omitempty"`
}

type FuzzInvocation struct {
	InitialSeed uint64    `yaml:"initial_seed"`
	Count       int       `yaml:"count"`
	Jobs        int       `yaml:"jobs"`
	Timestamp   time.Time `yaml:"timestamp"`
}

type FuzzSummary struct {
	Passed     int   `yaml:"passed"`
	Failed     int   `yaml:"failed"`
	DurationMS int64 `yaml:"duration_ms"`
}

type FuzzFailure struct {
	Seed    uint64 `yaml:"seed"`
	Status  string `yaml:"status"`
	Message string `yaml:"message"`
}

type FuzzReport struct {
	Invocation FuzzInvocation `yaml:"invocation"`
	Summary    FuzzSummary    `yaml:"summary"`
	Failures   []FuzzFailure  `yaml:"failures"`
}

type RegressionSeeds struct {
	Seeds []uint64 `yaml:"seeds"`
}
