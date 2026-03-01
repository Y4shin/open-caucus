package votefuzz

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
)

func GenerateConfig(seed uint64) (Config, error) {
	subSeeds := map[string]uint64{
		"users":        DeriveSeed(seed, "users", 0),
		"participants": DeriveSeed(seed, "participants", 0),
		"eligibility":  DeriveSeed(seed, "eligibility", 0),
		"visibility":   DeriveSeed(seed, "visibility", 0),
		"vote_kind":    DeriveSeed(seed, "vote_kind", 0),
		"scenario":     DeriveSeed(seed, "scenario", 0),
		"actions":      DeriveSeed(seed, "actions", 0),
	}

	cfg := Config{
		Seed:     seed,
		SubSeeds: subSeeds,
	}

	userRand := rng(subSeeds["users"])
	cfg.UserCount = randomIntInclusive(userRand, 1, 40)
	cfg.Users = make([]UserSpec, cfg.UserCount)
	for i := range cfg.Users {
		cfg.Users[i] = UserSpec{
			Key:      fmt.Sprintf("U%02d", i+1),
			Username: fmt.Sprintf("user%02d", i+1),
			FullName: fmt.Sprintf("User %02d", i+1),
		}
	}

	participantRand := rng(subSeeds["participants"])
	cfg.ParticipantCount = randomIntInclusive(participantRand, 5, 30)
	cfg.ParticipantMemberCount = randomIntInclusive(participantRand, 0, minInt(cfg.ParticipantCount, cfg.UserCount))

	memberUserIndices := chooseDistinctIndices(participantRand, cfg.UserCount, cfg.ParticipantMemberCount)
	cfg.ParticipantMemberUserKeys = make([]string, 0, len(memberUserIndices))
	for _, idx := range memberUserIndices {
		cfg.ParticipantMemberUserKeys = append(cfg.ParticipantMemberUserKeys, cfg.Users[idx].Key)
	}
	sort.Strings(cfg.ParticipantMemberUserKeys)

	participantMemberSlotIndices := chooseDistinctIndices(participantRand, cfg.ParticipantCount, cfg.ParticipantMemberCount)
	memberSlotToUser := make(map[int]string, cfg.ParticipantMemberCount)
	for i, slotIdx := range participantMemberSlotIndices {
		if i < len(cfg.ParticipantMemberUserKeys) {
			memberSlotToUser[slotIdx] = cfg.ParticipantMemberUserKeys[i]
		}
	}

	cfg.Participants = make([]ParticipantSpec, cfg.ParticipantCount)
	for i := range cfg.Participants {
		participant := ParticipantSpec{
			Key:      fmt.Sprintf("A%02d", i+1),
			FullName: fmt.Sprintf("Attendee %02d", i+1),
			Secret:   fmt.Sprintf("attendee-secret-%02d", i+1),
		}
		if userKey, ok := memberSlotToUser[i]; ok {
			key := userKey
			participant.MemberKey = &key
		}
		cfg.Participants[i] = participant
	}

	eligibilityRand := rng(subSeeds["eligibility"])
	cfg.EligibleCount = randomIntInclusive(eligibilityRand, minInt(5, cfg.ParticipantCount), cfg.ParticipantCount)
	eligibleIndices := chooseDistinctIndices(eligibilityRand, cfg.ParticipantCount, cfg.EligibleCount)
	cfg.EligibleParticipantKeys = make([]string, 0, len(eligibleIndices))
	for _, idx := range eligibleIndices {
		cfg.EligibleParticipantKeys = append(cfg.EligibleParticipantKeys, cfg.Participants[idx].Key)
	}
	sort.Strings(cfg.EligibleParticipantKeys)

	visibilityRand := rng(subSeeds["visibility"])
	if visibilityRand.Intn(2) == 0 {
		cfg.Visibility = "open"
	} else {
		cfg.Visibility = "secret"
	}

	kindRand := rng(subSeeds["vote_kind"])
	if kindRand.Intn(2) == 0 {
		cfg.VoteKind = VoteKindYesNoAbstain
		cfg.MinSelections = 1
		cfg.MaxSelections = 1
		cfg.VoteOptions = []VoteOptionSpec{
			{Key: "O01", Label: "Yes", Position: 1},
			{Key: "O02", Label: "No", Position: 2},
			{Key: "O03", Label: "Abstain", Position: 3},
		}
	} else {
		cfg.VoteKind = VoteKindNOfM
		m := randomIntInclusive(kindRand, 2, 10)
		// Keep compatibility with DB constraint max_selections >= 1 while still
		// allowing 0-min ballots when desired.
		cfg.MaxSelections = randomIntInclusive(kindRand, 1, m)
		cfg.MinSelections = randomIntInclusive(kindRand, 0, cfg.MaxSelections)
		cfg.VoteOptions = make([]VoteOptionSpec, m)
		for i := range cfg.VoteOptions {
			cfg.VoteOptions[i] = VoteOptionSpec{
				Key:      fmt.Sprintf("O%02d", i+1),
				Label:    fmt.Sprintf("Choice %02d", i+1),
				Position: i + 1,
			}
		}
	}

	scenarioRand := rng(subSeeds["scenario"])
	if cfg.Visibility == "open" {
		cfg.Scenario = ScenarioSelfSubmitted
	} else {
		switch scenarioRand.Intn(3) {
		case 0:
			cfg.Scenario = ScenarioSelfSubmitted
		case 1:
			cfg.Scenario = ScenarioManualSubmitted
		default:
			cfg.Scenario = ScenarioHybrid
		}
	}

	switch cfg.Scenario {
	case ScenarioManualSubmitted:
		cfg.ManualVoterParticipantKeys = append([]string(nil), cfg.EligibleParticipantKeys...)
	case ScenarioHybrid:
		manualCount := randomIntInclusive(scenarioRand, 1, len(cfg.EligibleParticipantKeys)-1)
		manualIndices := chooseDistinctIndices(scenarioRand, len(cfg.EligibleParticipantKeys), manualCount)
		cfg.ManualVoterParticipantKeys = make([]string, 0, len(manualIndices))
		for _, idx := range manualIndices {
			cfg.ManualVoterParticipantKeys = append(cfg.ManualVoterParticipantKeys, cfg.EligibleParticipantKeys[idx])
		}
		sort.Strings(cfg.ManualVoterParticipantKeys)
	default:
		cfg.ManualVoterParticipantKeys = nil
	}
	cfg.ManualVoterCount = len(cfg.ManualVoterParticipantKeys)

	actions, err := generateActions(subSeeds["actions"], cfg)
	if err != nil {
		return Config{}, err
	}
	cfg.Actions = actions
	return cfg, nil
}

func ExplainConfig(seed uint64) (Config, error) {
	return GenerateConfig(seed)
}

func generateActions(seed uint64, cfg Config) ([]Action, error) {
	r := rng(seed)
	manualSet := make(map[string]struct{}, len(cfg.ManualVoterParticipantKeys))
	for _, key := range cfg.ManualVoterParticipantKeys {
		manualSet[key] = struct{}{}
	}

	manualCasts := make([]Action, 0, len(cfg.ManualVoterParticipantKeys))
	manualBallots := make([]Action, 0, len(cfg.ManualVoterParticipantKeys))
	for i, key := range cfg.ManualVoterParticipantKeys {
		attendeeKey := key
		manualCasts = append(manualCasts, Action{
			Kind:        ActionRegisterCast,
			AttendeeKey: &attendeeKey,
			Source:      "manual_submission",
		})
		manualBallots = append(manualBallots, Action{
			Kind:              ActionSubmitSecretBallot,
			ReceiptToken:      fmt.Sprintf("m-%02d", i+1),
			OptionKeys:        pickValidOptionKeys(r, cfg),
			EncryptedPayload:  []byte{byte((i + 1) % 255), byte((i + 17) % 255)},
			CommitmentCipher:  "xchacha20poly1305",
			CommitmentVersion: 1,
		})
	}

	selfEligibleKeys := make([]string, 0, len(cfg.EligibleParticipantKeys))
	for _, key := range cfg.EligibleParticipantKeys {
		if _, isManual := manualSet[key]; !isManual {
			selfEligibleKeys = append(selfEligibleKeys, key)
		}
	}

	selfActions := make([]Action, 0, len(selfEligibleKeys)*2)
	for i, key := range selfEligibleKeys {
		if r.Intn(100) >= 85 {
			continue
		}
		attendeeKey := key
		if cfg.Visibility == "open" {
			selfActions = append(selfActions, Action{
				Kind:         ActionSubmitOpenBallot,
				AttendeeKey:  &attendeeKey,
				Source:       "self_submission",
				OptionKeys:   pickValidOptionKeys(r, cfg),
				ReceiptToken: fmt.Sprintf("o-%02d", i+1),
			})
			continue
		}
		selfActions = append(selfActions,
			Action{Kind: ActionRegisterCast, AttendeeKey: &attendeeKey, Source: "self_submission"},
			Action{
				Kind:              ActionSubmitSecretBallot,
				ReceiptToken:      fmt.Sprintf("s-%02d", i+1),
				OptionKeys:        pickValidOptionKeys(r, cfg),
				EncryptedPayload:  []byte{byte((i + 33) % 255), byte((i + 71) % 255)},
				CommitmentCipher:  "xchacha20poly1305",
				CommitmentVersion: 1,
			},
		)
	}

	if len(selfActions) == 0 && cfg.Scenario == ScenarioSelfSubmitted && len(cfg.EligibleParticipantKeys) > 0 {
		firstKey := cfg.EligibleParticipantKeys[0]
		attendeeKey := firstKey
		if cfg.Visibility == "open" {
			selfActions = append(selfActions, Action{
				Kind:         ActionSubmitOpenBallot,
				AttendeeKey:  &attendeeKey,
				Source:       "self_submission",
				OptionKeys:   pickValidOptionKeys(r, cfg),
				ReceiptToken: "o-fallback",
			})
		} else {
			selfActions = append(selfActions,
				Action{Kind: ActionRegisterCast, AttendeeKey: &attendeeKey, Source: "self_submission"},
				Action{
					Kind:              ActionSubmitSecretBallot,
					ReceiptToken:      "s-fallback",
					OptionKeys:        pickValidOptionKeys(r, cfg),
					EncryptedPayload:  []byte{0xAA, 0x55},
					CommitmentCipher:  "xchacha20poly1305",
					CommitmentVersion: 1,
				},
			)
		}
	}

	noise := make([]Action, 0, 4)
	if len(cfg.Participants) > 0 && r.Intn(100) < 40 {
		rawIdx := r.Intn(len(cfg.Participants))
		attendeeKey := cfg.Participants[rawIdx].Key
		if cfg.Visibility == "open" {
			noise = append(noise, Action{
				Kind:         ActionSubmitOpenBallot,
				AttendeeKey:  &attendeeKey,
				Source:       "self_submission",
				OptionKeys:   pickValidOptionKeys(r, cfg),
				ReceiptToken: fmt.Sprintf("o-noise-%02d", rawIdx+1),
			})
		} else {
			noise = append(noise,
				Action{Kind: ActionRegisterCast, AttendeeKey: &attendeeKey, Source: "self_submission"},
				Action{
					Kind:              ActionSubmitSecretBallot,
					ReceiptToken:      fmt.Sprintf("s-noise-%02d", rawIdx+1),
					OptionKeys:        pickValidOptionKeys(r, cfg),
					EncryptedPayload:  []byte{0x0F, byte(rawIdx)},
					CommitmentCipher:  "xchacha20poly1305",
					CommitmentVersion: 1,
				},
			)
		}
	}

	if cfg.Scenario == ScenarioManualSubmitted || cfg.Scenario == ScenarioHybrid {
		preClose := make([]Action, 0, len(manualCasts)+len(selfActions)+len(noise))
		betweenClose := make([]Action, 0, len(manualBallots)+len(noise))

		canonical := r.Intn(100) < 80
		if canonical {
			preClose = append(preClose, manualCasts...)
			betweenClose = append(betweenClose, manualBallots...)
		} else {
			mode := r.Intn(3)
			switch mode {
			case 0:
				preClose = append(preClose, manualCasts...)
				betweenClose = append(betweenClose, manualBallots...)
				if len(manualBallots) > 0 {
					moveCount := randomIntInclusive(r, 1, maxInt(1, len(manualBallots)/2))
					idxs := chooseDistinctIndices(r, len(manualBallots), moveCount)
					idxSet := make(map[int]struct{}, len(idxs))
					for _, idx := range idxs {
						idxSet[idx] = struct{}{}
					}
					newBetween := make([]Action, 0, len(betweenClose))
					for i := range manualBallots {
						if _, ok := idxSet[i]; ok {
							preClose = append(preClose, manualBallots[i])
							continue
						}
						newBetween = append(newBetween, manualBallots[i])
					}
					betweenClose = newBetween
				}
			case 1:
				preClose = append(preClose, manualCasts...)
				betweenClose = append(betweenClose, manualBallots...)
				if len(manualCasts) > 0 {
					moveCount := randomIntInclusive(r, 1, maxInt(1, len(manualCasts)/2))
					idxs := chooseDistinctIndices(r, len(manualCasts), moveCount)
					idxSet := make(map[int]struct{}, len(idxs))
					for _, idx := range idxs {
						idxSet[idx] = struct{}{}
					}
					newPre := make([]Action, 0, len(preClose))
					for i := range manualCasts {
						if _, ok := idxSet[i]; ok {
							betweenClose = append(betweenClose, manualCasts[i])
							continue
						}
						newPre = append(newPre, manualCasts[i])
					}
					preClose = newPre
				}
			default:
				for i := range manualCasts {
					if r.Intn(100) < 70 {
						preClose = append(preClose, manualCasts[i])
					} else {
						betweenClose = append(betweenClose, manualCasts[i])
					}
				}
				for i := range manualBallots {
					if r.Intn(100) < 70 {
						betweenClose = append(betweenClose, manualBallots[i])
					} else {
						preClose = append(preClose, manualBallots[i])
					}
				}
			}
		}

		for _, act := range selfActions {
			if act.Kind == ActionRegisterCast {
				if r.Intn(100) < 85 {
					preClose = append(preClose, act)
				} else {
					betweenClose = append(betweenClose, act)
				}
				continue
			}
			if r.Intn(100) < 60 {
				preClose = append(preClose, act)
			} else {
				betweenClose = append(betweenClose, act)
			}
		}

		for _, act := range noise {
			if r.Intn(2) == 0 {
				preClose = append(preClose, act)
			} else {
				betweenClose = append(betweenClose, act)
			}
		}

		r.Shuffle(len(preClose), func(i, j int) { preClose[i], preClose[j] = preClose[j], preClose[i] })
		r.Shuffle(len(betweenClose), func(i, j int) { betweenClose[i], betweenClose[j] = betweenClose[j], betweenClose[i] })

		actions := make([]Action, 0, len(preClose)+len(betweenClose)+2)
		actions = append(actions, preClose...)
		actions = append(actions, Action{Kind: ActionCloseVote})
		actions = append(actions, betweenClose...)
		actions = append(actions, Action{Kind: ActionCloseVote})
		return actions, nil
	}

	actions := make([]Action, 0, len(selfActions)+len(noise)+1)
	actions = append(actions, selfActions...)
	actions = append(actions, noise...)
	r.Shuffle(len(actions), func(i, j int) { actions[i], actions[j] = actions[j], actions[i] })
	actions = append(actions, Action{Kind: ActionCloseVote})
	return actions, nil
}

func pickValidOptionKeys(r *rand.Rand, cfg Config) []string {
	if len(cfg.VoteOptions) == 0 {
		return nil
	}
	count := cfg.MinSelections
	if cfg.MaxSelections > cfg.MinSelections {
		count += r.Intn(cfg.MaxSelections - cfg.MinSelections + 1)
	}
	if count <= 0 {
		return nil
	}
	indices := chooseDistinctIndices(r, len(cfg.VoteOptions), count)
	optionKeys := make([]string, 0, len(indices))
	for _, idx := range indices {
		optionKeys = append(optionKeys, cfg.VoteOptions[idx].Key)
	}
	sort.Strings(optionKeys)
	return optionKeys
}

func HasCanonicalManualOrdering(cfg Config) bool {
	if cfg.Scenario != ScenarioManualSubmitted && cfg.Scenario != ScenarioHybrid {
		return true
	}
	firstClose := -1
	secondClose := -1
	for i, action := range cfg.Actions {
		if action.Kind != ActionCloseVote {
			continue
		}
		if firstClose < 0 {
			firstClose = i
			continue
		}
		secondClose = i
		break
	}
	if firstClose < 0 || secondClose < 0 {
		return false
	}

	manualSet := make(map[string]struct{}, len(cfg.ManualVoterParticipantKeys))
	for _, key := range cfg.ManualVoterParticipantKeys {
		manualSet[key] = struct{}{}
	}

	for i, action := range cfg.Actions {
		switch action.Kind {
		case ActionRegisterCast:
			if action.Source != "manual_submission" || action.AttendeeKey == nil {
				continue
			}
			if _, ok := manualSet[*action.AttendeeKey]; !ok {
				continue
			}
			if i >= firstClose {
				return false
			}
		case ActionSubmitSecretBallot:
			if !strings.HasPrefix(action.ReceiptToken, "m-") {
				continue
			}
			if i <= firstClose || i >= secondClose {
				return false
			}
		}
	}
	return true
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
