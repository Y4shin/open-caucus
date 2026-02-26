package handlers

import (
	"testing"

	"github.com/Y4shin/conference-tool/internal/repository/model"
)

func TestExtractAgendaImportLines_MarkdownSingleH1UsesH2H3(t *testing.T) {
	source := `
# Agenda
## Opening
### Sub Opening
## Reports
`
	lines, errKey := extractAgendaImportLines(source)
	if errKey != "" {
		t.Fatalf("unexpected parse error key: %s", errKey)
	}

	stateByText := map[string]string{}
	for _, line := range lines {
		stateByText[line.Text] = line.State
	}

	if got := stateByText["Agenda"]; got != importLineIgnore {
		t.Fatalf("expected 'Agenda' to be ignored in special markdown mode, got=%q", got)
	}
	if got := stateByText["Opening"]; got != importLineHeading {
		t.Fatalf("expected 'Opening' to be heading, got=%q", got)
	}
	if got := stateByText["Sub Opening"]; got != importLineSubheading {
		t.Fatalf("expected 'Sub Opening' to be subheading, got=%q", got)
	}
	if got := stateByText["Reports"]; got != importLineHeading {
		t.Fatalf("expected 'Reports' to be heading, got=%q", got)
	}
}

func TestExtractAgendaImportLines_PlainTextMixedNumberedIgnoresUnnumbered(t *testing.T) {
	source := `
TOP1 Opening
Notes line
TOP2 Reports
`
	lines, errKey := extractAgendaImportLines(source)
	if errKey != "" {
		t.Fatalf("unexpected parse error key: %s", errKey)
	}

	stateByText := map[string]string{}
	for _, line := range lines {
		stateByText[line.Text] = line.State
	}
	if got := stateByText["TOP1 Opening"]; got != importLineHeading {
		t.Fatalf("expected numbered line to be heading, got=%q", got)
	}
	if got := stateByText["TOP2 Reports"]; got != importLineHeading {
		t.Fatalf("expected numbered line to be heading, got=%q", got)
	}
	if got := stateByText["Notes line"]; got != importLineIgnore {
		t.Fatalf("expected unnumbered line to be ignored, got=%q", got)
	}
}

func TestBuildImportedAgendaPointsFromLines_SubheadingWithoutHeading(t *testing.T) {
	points, errKey := buildImportedAgendaPointsFromLines([]agendaImportLine{
		{LineNo: 1, Text: "Child only", State: importLineSubheading, DetectedState: importLineSubheading},
	})
	if errKey != "agenda_import.error_subheading_without_heading" {
		t.Fatalf("unexpected error key: %s", errKey)
	}
	if len(points) != 0 {
		t.Fatalf("expected no points, got=%d", len(points))
	}
}

func TestBuildAgendaDiff_InsertsBetweenExisting(t *testing.T) {
	existing := []*model.AgendaPoint{
		{ID: 1, Position: 1, Title: "A"},
		{ID: 2, Position: 2, Title: "C"},
	}
	imported := []importedAgendaPoint{
		{Key: "k1", Title: "A", Position: 1, DisplayCode: "TOP 1"},
		{Key: "k2", Title: "B", Position: 2, DisplayCode: "TOP 2"},
		{Key: "k3", Title: "C", Position: 3, DisplayCode: "TOP 3"},
	}

	diffItems, applyItems, deleteIDs := buildAgendaDiff(existing, imported)
	if len(applyItems) != 3 {
		t.Fatalf("expected 3 apply items, got=%d", len(applyItems))
	}
	if len(deleteIDs) != 0 {
		t.Fatalf("expected no deletes, got=%d", len(deleteIDs))
	}

	var foundInsert bool
	for _, item := range diffItems {
		if item.Operation == "insert" && item.AfterTitle == "B" {
			foundInsert = true
		}
	}
	if !foundInsert {
		t.Fatalf("expected insert diff item for title B")
	}
}

func TestBuildAgendaDiff_AlignedRowsIncludeGapForInsert(t *testing.T) {
	existing := []*model.AgendaPoint{
		{ID: 1, Position: 1, Title: "A"},
		{ID: 2, Position: 2, Title: "C"},
	}
	imported := []importedAgendaPoint{
		{Key: "k1", Title: "A", Position: 1, DisplayCode: "TOP 1"},
		{Key: "k2", Title: "B", Position: 2, DisplayCode: "TOP 2"},
		{Key: "k3", Title: "C", Position: 3, DisplayCode: "TOP 3"},
	}

	diffItems, _, _ := buildAgendaDiff(existing, imported)
	if len(diffItems) != 3 {
		t.Fatalf("expected 3 diff rows, got=%d", len(diffItems))
	}
	if diffItems[0].BeforeTitle != "A" || diffItems[0].AfterTitle != "A" {
		t.Fatalf("expected first row to align A, got before=%q after=%q", diffItems[0].BeforeTitle, diffItems[0].AfterTitle)
	}
	if diffItems[1].Operation != "insert" || diffItems[1].BeforeTitle != "" || diffItems[1].AfterTitle != "B" {
		t.Fatalf("expected second row to be insert gap, got op=%q before=%q after=%q", diffItems[1].Operation, diffItems[1].BeforeTitle, diffItems[1].AfterTitle)
	}
	if diffItems[1].AfterTone != "success" {
		t.Fatalf("expected insert row after tone to be success, got=%q", diffItems[1].AfterTone)
	}
	if diffItems[2].BeforeTitle != "C" || diffItems[2].AfterTitle != "C" {
		t.Fatalf("expected third row to align C, got before=%q after=%q", diffItems[2].BeforeTitle, diffItems[2].AfterTitle)
	}
}

func TestBuildAgendaDiff_MovedItemsRemainOrderedAndPaired(t *testing.T) {
	existing := []*model.AgendaPoint{
		{ID: 10, Position: 1, Title: "A"},
		{ID: 20, Position: 2, Title: "B"},
		{ID: 30, Position: 3, Title: "C"},
	}
	imported := []importedAgendaPoint{
		{Key: "b", Title: "B", Position: 1, DisplayCode: "TOP 1"},
		{Key: "a", Title: "A", Position: 2, DisplayCode: "TOP 2"},
		{Key: "c", Title: "C", Position: 3, DisplayCode: "TOP 3"},
	}

	diffItems, _, _ := buildAgendaDiff(existing, imported)
	if len(diffItems) != 3 {
		t.Fatalf("expected exactly 3 rows without move gaps, got=%d", len(diffItems))
	}

	if diffItems[0].BeforeTitle != "A" || diffItems[0].AfterTitle != "B" || diffItems[0].Operation != "move" {
		t.Fatalf("expected first row to show A->B move, got before=%q after=%q op=%q", diffItems[0].BeforeTitle, diffItems[0].AfterTitle, diffItems[0].Operation)
	}
	if diffItems[1].BeforeTitle != "B" || diffItems[1].AfterTitle != "A" || diffItems[1].Operation != "move" {
		t.Fatalf("expected second row to show B->A move, got before=%q after=%q op=%q", diffItems[1].BeforeTitle, diffItems[1].AfterTitle, diffItems[1].Operation)
	}
	if diffItems[2].BeforeTitle != "C" || diffItems[2].AfterTitle != "C" || diffItems[2].Operation != "unchanged" {
		t.Fatalf("expected third row to remain unchanged C, got before=%q after=%q op=%q", diffItems[2].BeforeTitle, diffItems[2].AfterTitle, diffItems[2].Operation)
	}
	if diffItems[0].BeforeTone != "warning" || diffItems[0].AfterTone != "warning" {
		t.Fatalf("expected first row move tones warning, got before=%q after=%q", diffItems[0].BeforeTone, diffItems[0].AfterTone)
	}
	if diffItems[0].BeforePairKey == "" || diffItems[0].AfterPairKey == "" || diffItems[0].BeforePairKey == diffItems[0].AfterPairKey {
		t.Fatalf("expected first row to carry distinct before/after pair keys, got before=%q after=%q", diffItems[0].BeforePairKey, diffItems[0].AfterPairKey)
	}
}

func TestBuildAgendaDiff_SwapFirstAndLastWithoutGaps(t *testing.T) {
	parent2 := int64(2)
	existing := []*model.AgendaPoint{
		{ID: 1, ParentID: nil, Position: 1, Title: "Test1"},
		{ID: 2, ParentID: nil, Position: 2, Title: "Test2"},
		{ID: 3, ParentID: &parent2, Position: 1, Title: "Test3"},
		{ID: 4, ParentID: &parent2, Position: 2, Title: "Test4"},
		{ID: 5, ParentID: nil, Position: 3, Title: "Test5"},
	}
	k2 := "k2"
	imported := []importedAgendaPoint{
		{Key: "k5", Title: "Test5", Position: 1, DisplayCode: "TOP 1"},
		{Key: k2, Title: "Test2", Position: 2, DisplayCode: "TOP 2"},
		{Key: "k3", Title: "Test3", ParentKey: &k2, Position: 1, DisplayCode: "TOP 2.1"},
		{Key: "k4", Title: "Test4", ParentKey: &k2, Position: 2, DisplayCode: "TOP 2.2"},
		{Key: "k1", Title: "Test1", Position: 3, DisplayCode: "TOP 3"},
	}

	diffItems, _, _ := buildAgendaDiff(existing, imported)
	if len(diffItems) != 5 {
		t.Fatalf("expected 5 rows without gaps, got=%d", len(diffItems))
	}
	for idx, item := range diffItems {
		if item.BeforeTitle == "" || item.AfterTitle == "" {
			t.Fatalf("unexpected gap at row %d: before=%q after=%q", idx, item.BeforeTitle, item.AfterTitle)
		}
	}
	if diffItems[0].BeforeTitle != "Test1" || diffItems[0].AfterTitle != "Test5" || diffItems[0].Operation != "move" {
		t.Fatalf("unexpected first row: before=%q after=%q op=%q", diffItems[0].BeforeTitle, diffItems[0].AfterTitle, diffItems[0].Operation)
	}
	if diffItems[4].BeforeTitle != "Test5" || diffItems[4].AfterTitle != "Test1" || diffItems[4].Operation != "move" {
		t.Fatalf("unexpected last row: before=%q after=%q op=%q", diffItems[4].BeforeTitle, diffItems[4].AfterTitle, diffItems[4].Operation)
	}
}
