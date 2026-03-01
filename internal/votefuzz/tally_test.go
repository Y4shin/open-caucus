package votefuzz

import (
	"reflect"
	"testing"
)

func TestComputeExpectedTalliesSingleBallot(t *testing.T) {
	input := ComputeTallyInput{
		VoteOptionIDs: []int64{1, 2, 3},
		ActionResults: []ActionResult{
			{
				Action:           Action{Kind: ActionSubmitOpenBallot},
				Success:          true,
				AppliedOptionIDs: []int64{1},
			},
		},
	}
	got, err := ComputeExpectedTallies(input)
	if err != nil {
		t.Fatalf("compute tallies: %v", err)
	}
	want := map[int64]int64{1: 1, 2: 0, 3: 0}
	if !reflect.DeepEqual(got.ByOptionID, want) {
		t.Fatalf("unexpected tallies: got=%v want=%v", got.ByOptionID, want)
	}
	if got.BallotsCounted != 1 || got.SelectionsCounted != 1 {
		t.Fatalf("unexpected counters: %+v", got)
	}
}

func TestComputeExpectedTalliesIgnoresFailedActions(t *testing.T) {
	input := ComputeTallyInput{
		VoteOptionIDs: []int64{10, 20},
		ActionResults: []ActionResult{
			{Action: Action{Kind: ActionRegisterCast}, Success: true},
			{Action: Action{Kind: ActionSubmitSecretBallot}, Success: false, AppliedOptionIDs: []int64{10}},
			{Action: Action{Kind: ActionSubmitSecretBallot}, Success: true, AppliedOptionIDs: []int64{20}},
		},
	}
	got, err := ComputeExpectedTallies(input)
	if err != nil {
		t.Fatalf("compute tallies: %v", err)
	}
	want := map[int64]int64{10: 0, 20: 1}
	if !reflect.DeepEqual(got.ByOptionID, want) {
		t.Fatalf("unexpected tallies: got=%v want=%v", got.ByOptionID, want)
	}
	if got.BallotsCounted != 1 || got.SelectionsCounted != 1 {
		t.Fatalf("unexpected counters: %+v", got)
	}
}

func TestComputeExpectedTalliesUnknownOptionFails(t *testing.T) {
	_, err := ComputeExpectedTallies(ComputeTallyInput{
		VoteOptionIDs: []int64{1, 2},
		ActionResults: []ActionResult{{
			Action:           Action{Kind: ActionSubmitOpenBallot},
			Success:          true,
			AppliedOptionIDs: []int64{3},
		}},
	})
	if err == nil {
		t.Fatalf("expected unknown option error")
	}
}

func TestComputeExpectedTalliesDeterministic(t *testing.T) {
	input := ComputeTallyInput{
		VoteOptionIDs: []int64{1, 2, 3},
		ActionResults: []ActionResult{
			{Action: Action{Kind: ActionSubmitOpenBallot}, Success: true, AppliedOptionIDs: []int64{1, 3}},
			{Action: Action{Kind: ActionSubmitSecretBallot}, Success: true, AppliedOptionIDs: []int64{3}},
		},
	}
	a, err := ComputeExpectedTallies(input)
	if err != nil {
		t.Fatalf("first compute: %v", err)
	}
	b, err := ComputeExpectedTallies(input)
	if err != nil {
		t.Fatalf("second compute: %v", err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("expected deterministic tally computation")
	}
}
