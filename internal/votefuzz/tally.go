package votefuzz

import "fmt"

func ComputeExpectedTallies(input ComputeTallyInput) (ExpectedTallies, error) {
	tallies := ExpectedTallies{
		ByOptionID: make(map[int64]int64, len(input.VoteOptionIDs)),
	}
	allowed := make(map[int64]struct{}, len(input.VoteOptionIDs))
	for _, optionID := range input.VoteOptionIDs {
		allowed[optionID] = struct{}{}
		tallies.ByOptionID[optionID] = 0
	}

	for _, result := range input.ActionResults {
		if !result.Success {
			tallies.IgnoredActions++
			continue
		}
		if result.Action.Kind != ActionSubmitOpenBallot && result.Action.Kind != ActionSubmitSecretBallot {
			tallies.IgnoredActions++
			continue
		}

		tallies.BallotsCounted++
		for _, optionID := range result.AppliedOptionIDs {
			if _, ok := allowed[optionID]; !ok {
				return ExpectedTallies{}, fmt.Errorf("successful ballot references unknown option id %d", optionID)
			}
			tallies.ByOptionID[optionID]++
			tallies.SelectionsCounted++
		}
	}

	return tallies, nil
}
