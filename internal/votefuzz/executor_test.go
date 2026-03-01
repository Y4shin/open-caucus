package votefuzz

import (
	"context"
	"testing"
)

func TestExecuteConfigValidAcrossSampleSeeds(t *testing.T) {
	for seed := uint64(1); seed <= 50; seed++ {
		cfg, err := GenerateConfig(seed)
		if err != nil {
			t.Fatalf("seed %d: generate config: %v", seed, err)
		}
		res, err := ExecuteConfig(context.Background(), cfg, ExecuteOptions{})
		if err != nil {
			t.Fatalf("seed %d: execute config: %v", seed, err)
		}
		if !res.Passed {
			t.Fatalf("seed %d: expected pass, got failures: %v", seed, res.InvariantFailures)
		}
	}
}

func TestExecuteConfigNonCanonicalManualOrderingStillValid(t *testing.T) {
	var cfg Config
	found := false
	for seed := uint64(1); seed <= 20000; seed++ {
		candidate, err := GenerateConfig(seed)
		if err != nil {
			t.Fatalf("seed %d: generate config: %v", seed, err)
		}
		if (candidate.Scenario == ScenarioManualSubmitted || candidate.Scenario == ScenarioHybrid) && !HasCanonicalManualOrdering(candidate) {
			cfg = candidate
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("did not find non-canonical manual/hybrid config in search range")
	}

	res, err := ExecuteConfig(context.Background(), cfg, ExecuteOptions{})
	if err != nil {
		t.Fatalf("execute non-canonical config: %v", err)
	}
	if !res.Passed {
		t.Fatalf("expected non-canonical config to preserve DB validity, failures: %v", res.InvariantFailures)
	}
}

func TestExecuteConfigCountingFinalStateTallyCheck(t *testing.T) {
	var cfg Config
	found := false
	for seed := uint64(1); seed <= 5000; seed++ {
		candidate, err := GenerateConfig(seed)
		if err != nil {
			t.Fatalf("seed %d: generate config: %v", seed, err)
		}
		if candidate.Visibility == "secret" && len(candidate.EligibleParticipantKeys) > 0 {
			cfg = candidate
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("did not find secret config")
	}

	firstEligible := cfg.EligibleParticipantKeys[0]
	cfg.Scenario = ScenarioSelfSubmitted
	cfg.ManualVoterCount = 0
	cfg.ManualVoterParticipantKeys = nil
	cfg.Actions = []Action{
		{
			Kind:        ActionRegisterCast,
			AttendeeKey: &firstEligible,
			Source:      modelVoteCastSourceSelf,
		},
		{Kind: ActionCloseVote},
	}

	res, err := ExecuteConfig(context.Background(), cfg, ExecuteOptions{})
	if err != nil {
		t.Fatalf("execute counting config: %v", err)
	}
	if !res.Passed {
		t.Fatalf("expected counting config to pass invariants, failures: %v", res.InvariantFailures)
	}
	if res.FinalVoteState != "counting" {
		t.Fatalf("expected final state counting, got %s", res.FinalVoteState)
	}
	for _, count := range res.ActualTallies {
		if count != 0 {
			t.Fatalf("expected zero tallies in counting test, got %v", res.ActualTallies)
		}
	}
}

func TestExecuteConfigVerificationSpotChecksClosed(t *testing.T) {
	var cfg Config
	found := false
	for seed := uint64(1); seed <= 5000; seed++ {
		candidate, err := GenerateConfig(seed)
		if err != nil {
			t.Fatalf("seed %d: generate config: %v", seed, err)
		}
		if candidate.Visibility == "open" && len(candidate.EligibleParticipantKeys) > 0 {
			cfg = candidate
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("did not find open config")
	}

	firstEligible := cfg.EligibleParticipantKeys[0]
	cfg.Scenario = ScenarioSelfSubmitted
	cfg.ManualVoterCount = 0
	cfg.ManualVoterParticipantKeys = nil
	cfg.Actions = []Action{
		{
			Kind:         ActionSubmitOpenBallot,
			AttendeeKey:  &firstEligible,
			Source:       modelVoteCastSourceSelf,
			ReceiptToken: "open-check",
			OptionKeys:   validOptionKeysForTest(cfg),
		},
		{Kind: ActionCloseVote},
	}

	res, err := ExecuteConfig(context.Background(), cfg, ExecuteOptions{})
	if err != nil {
		t.Fatalf("execute config: %v", err)
	}
	if !res.Passed {
		t.Fatalf("expected pass, got failures: %v", res.InvariantFailures)
	}
	if len(res.VerificationChecks) == 0 {
		t.Fatalf("expected verification checks to be populated")
	}
	foundOpen := false
	for _, check := range res.VerificationChecks {
		if check.Kind != "open" {
			continue
		}
		foundOpen = true
		if check.ExpectedBlocked {
			t.Fatalf("open verification check should not be blocked in closed flow")
		}
		if !check.Passed {
			t.Fatalf("expected open verification check to pass: %+v", check)
		}
	}
	if !foundOpen {
		t.Fatalf("expected at least one open verification check")
	}
}

func TestExecuteConfigVerificationSpotChecksCountingBlocked(t *testing.T) {
	var cfg Config
	found := false
	for seed := uint64(1); seed <= 5000; seed++ {
		candidate, err := GenerateConfig(seed)
		if err != nil {
			t.Fatalf("seed %d: generate config: %v", seed, err)
		}
		if candidate.Visibility == "secret" && len(candidate.EligibleParticipantKeys) >= 2 {
			cfg = candidate
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("did not find secret config with >=2 eligible attendees")
	}

	firstEligible := cfg.EligibleParticipantKeys[0]
	secondEligible := cfg.EligibleParticipantKeys[1]
	cfg.Scenario = ScenarioSelfSubmitted
	cfg.ManualVoterCount = 0
	cfg.ManualVoterParticipantKeys = nil
	cfg.Actions = []Action{
		{Kind: ActionRegisterCast, AttendeeKey: &firstEligible, Source: modelVoteCastSourceSelf},
		{
			Kind:              ActionSubmitSecretBallot,
			ReceiptToken:      "secret-check",
			OptionKeys:        validOptionKeysForTest(cfg),
			EncryptedPayload:  []byte{0x01, 0x02},
			CommitmentCipher:  "xchacha20poly1305",
			CommitmentVersion: 1,
		},
		{Kind: ActionRegisterCast, AttendeeKey: &secondEligible, Source: modelVoteCastSourceSelf},
		{Kind: ActionCloseVote},
	}

	res, err := ExecuteConfig(context.Background(), cfg, ExecuteOptions{})
	if err != nil {
		t.Fatalf("execute config: %v", err)
	}
	if !res.Passed {
		t.Fatalf("expected pass, got failures: %v", res.InvariantFailures)
	}
	if res.FinalVoteState != "counting" {
		t.Fatalf("expected final state counting, got %s", res.FinalVoteState)
	}
	foundSecret := false
	for _, check := range res.VerificationChecks {
		if check.Kind != "secret" {
			continue
		}
		foundSecret = true
		if !check.ExpectedBlocked {
			t.Fatalf("expected secret verification to be blocked in counting: %+v", check)
		}
		if !check.Passed {
			t.Fatalf("expected blocked secret verification check to pass: %+v", check)
		}
	}
	if !foundSecret {
		t.Fatalf("expected at least one secret verification check")
	}
}

func validOptionKeysForTest(cfg Config) []string {
	if cfg.MinSelections == 0 {
		return nil
	}
	optionKeys := make([]string, 0, cfg.MinSelections)
	for i := 0; i < cfg.MinSelections && i < len(cfg.VoteOptions); i++ {
		optionKeys = append(optionKeys, cfg.VoteOptions[i].Key)
	}
	return optionKeys
}

const modelVoteCastSourceSelf = "self_submission"
