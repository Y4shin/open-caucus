package templates

import (
	"reflect"
	"testing"
)

func TestFilteredSpeakerCandidates_NumberQueryPrioritizesExactMatch(t *testing.T) {
	input := SpeakersListPartialInput{
		SearchQuery: "2",
		Attendees: []AttendeeItem{
			{ID: 1, AttendeeNumber: 12, FullName: "Twelve"},
			{ID: 2, AttendeeNumber: 2, FullName: "Two"},
			{ID: 3, AttendeeNumber: 20, FullName: "Twenty"},
		},
	}

	got := input.FilteredSpeakerCandidates()
	var gotNumbers []int64
	for _, attendee := range got {
		gotNumbers = append(gotNumbers, attendee.AttendeeNumber)
	}

	want := []int64{2, 20, 12}
	if !reflect.DeepEqual(gotNumbers, want) {
		t.Fatalf("unexpected candidate ordering for numeric query: got=%v want=%v", gotNumbers, want)
	}
}

func TestFilteredSpeakerCandidates_TextQueryRanksBestNameMatch(t *testing.T) {
	input := SpeakersListPartialInput{
		SearchQuery: "alice",
		Attendees: []AttendeeItem{
			{ID: 1, AttendeeNumber: 1, FullName: "Malice Harper"},
			{ID: 2, AttendeeNumber: 2, FullName: "Alicia Stone"},
			{ID: 3, AttendeeNumber: 3, FullName: "Alice Johnson"},
		},
	}

	got := input.FilteredSpeakerCandidates()
	if len(got) < 3 {
		t.Fatalf("expected at least 3 candidates, got=%d", len(got))
	}

	if got[0].FullName != "Alice Johnson" {
		t.Fatalf("expected best match first, got first=%q", got[0].FullName)
	}
}

func TestFilteredSpeakerCandidates_TextQuerySupportsFuzzySubsequence(t *testing.T) {
	input := SpeakersListPartialInput{
		SearchQuery: "asj",
		Attendees: []AttendeeItem{
			{ID: 1, AttendeeNumber: 1, FullName: "Alice Stone Johnson"},
			{ID: 2, AttendeeNumber: 2, FullName: "Brian Cole"},
		},
	}

	got := input.FilteredSpeakerCandidates()
	if len(got) == 0 {
		t.Fatalf("expected fuzzy match candidate for query %q", input.SearchQuery)
	}
	if got[0].FullName != "Alice Stone Johnson" {
		t.Fatalf("unexpected fuzzy match result first=%q", got[0].FullName)
	}
}

