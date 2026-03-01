package votefuzz

import (
	"reflect"
	"testing"
)

func TestGenerateConfigDeterministic(t *testing.T) {
	cfg1, err := GenerateConfig(42)
	if err != nil {
		t.Fatalf("generate config #1: %v", err)
	}
	cfg2, err := GenerateConfig(42)
	if err != nil {
		t.Fatalf("generate config #2: %v", err)
	}
	if !reflect.DeepEqual(cfg1, cfg2) {
		t.Fatalf("expected deterministic config generation for same seed")
	}
}

func TestGenerateConfigBounds(t *testing.T) {
	for seed := uint64(1); seed <= 200; seed++ {
		cfg, err := GenerateConfig(seed)
		if err != nil {
			t.Fatalf("seed %d: generate config: %v", seed, err)
		}
		if cfg.UserCount < 1 || cfg.UserCount > 40 {
			t.Fatalf("seed %d: user_count out of range: %d", seed, cfg.UserCount)
		}
		if cfg.ParticipantCount < 5 || cfg.ParticipantCount > 30 {
			t.Fatalf("seed %d: participant_count out of range: %d", seed, cfg.ParticipantCount)
		}
		if cfg.ParticipantMemberCount < 0 || cfg.ParticipantMemberCount > minInt(cfg.ParticipantCount, cfg.UserCount) {
			t.Fatalf("seed %d: participant_member_count out of range: %d", seed, cfg.ParticipantMemberCount)
		}
		if cfg.EligibleCount < minInt(5, cfg.ParticipantCount) || cfg.EligibleCount > cfg.ParticipantCount {
			t.Fatalf("seed %d: eligible_count out of range: %d", seed, cfg.EligibleCount)
		}
		if cfg.MinSelections < 0 || cfg.MaxSelections < 1 || cfg.MaxSelections < cfg.MinSelections || cfg.MaxSelections > len(cfg.VoteOptions) {
			t.Fatalf("seed %d: invalid selection bounds min=%d max=%d options=%d", seed, cfg.MinSelections, cfg.MaxSelections, len(cfg.VoteOptions))
		}
		if cfg.Visibility != "open" && cfg.Visibility != "secret" {
			t.Fatalf("seed %d: invalid visibility: %s", seed, cfg.Visibility)
		}
		if cfg.Visibility == "open" && cfg.Scenario != ScenarioSelfSubmitted {
			t.Fatalf("seed %d: open visibility must force self scenario, got %s", seed, cfg.Scenario)
		}
		if cfg.Scenario == ScenarioManualSubmitted || cfg.Scenario == ScenarioHybrid {
			if closeCount(cfg.Actions) != 2 {
				t.Fatalf("seed %d: manual/hybrid scenario must have exactly 2 close actions", seed)
			}
		}
		if len(cfg.Actions) == 0 || cfg.Actions[len(cfg.Actions)-1].Kind != ActionCloseVote {
			t.Fatalf("seed %d: actions must end with close vote", seed)
		}
	}
}

func TestGeneratedBallotsAlwaysRespectVoteDefinition(t *testing.T) {
	for seed := uint64(1); seed <= 300; seed++ {
		cfg, err := GenerateConfig(seed)
		if err != nil {
			t.Fatalf("seed %d: generate config: %v", seed, err)
		}
		allowed := make(map[string]struct{}, len(cfg.VoteOptions))
		for _, option := range cfg.VoteOptions {
			allowed[option.Key] = struct{}{}
		}

		for i, action := range cfg.Actions {
			if action.Kind != ActionSubmitOpenBallot && action.Kind != ActionSubmitSecretBallot {
				continue
			}
			if len(action.OptionKeys) < cfg.MinSelections || len(action.OptionKeys) > cfg.MaxSelections {
				t.Fatalf("seed %d action %d: option count out of range: %d (min=%d max=%d)", seed, i, len(action.OptionKeys), cfg.MinSelections, cfg.MaxSelections)
			}
			seen := make(map[string]struct{}, len(action.OptionKeys))
			for _, key := range action.OptionKeys {
				if _, ok := allowed[key]; !ok {
					t.Fatalf("seed %d action %d: unknown option key %s", seed, i, key)
				}
				if _, dup := seen[key]; dup {
					t.Fatalf("seed %d action %d: duplicate option key %s", seed, i, key)
				}
				seen[key] = struct{}{}
			}
		}
	}
}

func TestManualOrderingStronglyFavorsCanonical(t *testing.T) {
	manualConfigs := 0
	canonicalConfigs := 0

	for seed := uint64(1); seed <= 8000; seed++ {
		cfg, err := GenerateConfig(seed)
		if err != nil {
			t.Fatalf("seed %d: generate config: %v", seed, err)
		}
		if cfg.Scenario != ScenarioManualSubmitted && cfg.Scenario != ScenarioHybrid {
			continue
		}
		manualConfigs++
		if HasCanonicalManualOrdering(cfg) {
			canonicalConfigs++
		}
		if manualConfigs >= 250 {
			break
		}
	}

	if manualConfigs < 100 {
		t.Fatalf("insufficient manual/hybrid configs generated for distribution check: %d", manualConfigs)
	}
	ratio := float64(canonicalConfigs) / float64(manualConfigs)
	if ratio < 0.60 {
		t.Fatalf("canonical ordering ratio too low: got %.2f (%d/%d)", ratio, canonicalConfigs, manualConfigs)
	}
}

func closeCount(actions []Action) int {
	count := 0
	for _, action := range actions {
		if action.Kind == ActionCloseVote {
			count++
		}
	}
	return count
}
