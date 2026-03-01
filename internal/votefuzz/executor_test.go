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

const modelVoteCastSourceSelf = "self_submission"
