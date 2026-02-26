package handlers

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/Y4shin/conference-tool/internal/repository"
	"github.com/Y4shin/conference-tool/internal/repository/model"
	"github.com/Y4shin/conference-tool/internal/routes"
	"github.com/Y4shin/conference-tool/internal/templates"
	"github.com/invopop/ctxi18n/i18n"
)

const agendaImportMaxBytes = 1 << 20

const (
	importLineIgnore     = "ignore"
	importLineHeading    = "heading"
	importLineSubheading = "subheading"
)

type agendaImportLine struct {
	LineNo        int
	Text          string
	State         string
	DetectedState string
}

type importedAgendaPoint struct {
	Key         string
	Title       string
	ParentKey   *string
	Position    int64
	DisplayCode string
}

type agendaDiffBuildResult struct {
	Preview    templates.AgendaImportPreview
	ApplyItems []repository.AgendaApplyPoint
	DeleteIDs  []int64
}

func (h *Handler) ManageAgendaPointMoveUp(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.AgendaPointListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	agendaPointID, err := strconv.ParseInt(params.AgendaPointId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid agenda point ID")
	}
	if err := h.Repository.MoveAgendaPointUp(ctx, meetingID, agendaPointID); err != nil {
		return nil, nil, fmt.Errorf("failed to move agenda point up: %w", err)
	}
	partial, err := h.loadAgendaPointListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

func (h *Handler) ManageAgendaPointMoveDown(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.AgendaPointListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	agendaPointID, err := strconv.ParseInt(params.AgendaPointId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid agenda point ID")
	}
	if err := h.Repository.MoveAgendaPointDown(ctx, meetingID, agendaPointID); err != nil {
		return nil, nil, fmt.Errorf("failed to move agenda point down: %w", err)
	}
	partial, err := h.loadAgendaPointListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

func (h *Handler) ManageAgendaImportExtract(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.AgendaPointListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	_ = r.ParseForm()
	sourceText := strings.TrimSpace(r.FormValue("source_text"))

	partial, err := h.loadAgendaPointListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	if err != nil {
		return nil, nil, err
	}

	lines, parseErr := extractAgendaImportLines(sourceText)
	if parseErr != "" {
		partial.Error = i18n.T(ctx, parseErr)
		partial.Import = templates.AgendaImportPreview{
			SourceText: sourceText,
		}
		return partial, nil, nil
	}

	partial.Import = templates.AgendaImportPreview{
		SourceText: sourceText,
		Lines:      toTemplateImportLines(lines),
		Stage:      "correction",
	}
	return partial, nil, nil
}

func (h *Handler) ManageAgendaImportDiff(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.AgendaPointListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	_ = r.ParseForm()

	partial, err := h.loadAgendaPointListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	if err != nil {
		return nil, nil, err
	}

	sourceText := strings.TrimSpace(r.FormValue("source_text"))
	lines, lineErr := parseImportLinesFromForm(r)
	if lineErr != "" {
		partial.Error = i18n.T(ctx, lineErr)
		return partial, nil, nil
	}
	diffResult, diffErr := h.buildAgendaDiffResult(ctx, meetingID, sourceText, lines)
	if diffErr != "" {
		partial.Error = i18n.T(ctx, diffErr)
		partial.Import = templates.AgendaImportPreview{
			SourceText: sourceText,
			Lines:      toTemplateImportLines(lines),
			Stage:      "correction",
		}
		return partial, nil, nil
	}

	partial.Import = diffResult.Preview
	return partial, nil, nil
}

func (h *Handler) ManageAgendaImportApply(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.AgendaPointListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	_ = r.ParseForm()

	sourceText := strings.TrimSpace(r.FormValue("source_text"))
	lines, lineErr := parseImportLinesFromForm(r)
	if lineErr != "" {
		partial, loadErr := h.loadAgendaPointListPartial(ctx, params.Slug, params.MeetingId, meetingID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		partial.Error = i18n.T(ctx, lineErr)
		return partial, nil, nil
	}

	currentFingerprint, err := h.computeAgendaFingerprint(ctx, meetingID)
	if err != nil {
		return nil, nil, err
	}
	suppliedFingerprint := strings.TrimSpace(r.FormValue("fingerprint"))

	diffResult, diffErr := h.buildAgendaDiffResult(ctx, meetingID, sourceText, lines)
	if diffErr != "" {
		partial, loadErr := h.loadAgendaPointListPartial(ctx, params.Slug, params.MeetingId, meetingID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		partial.Error = i18n.T(ctx, diffErr)
		partial.Import = templates.AgendaImportPreview{
			SourceText: sourceText,
			Lines:      toTemplateImportLines(lines),
			Stage:      "correction",
		}
		return partial, nil, nil
	}

	if suppliedFingerprint == "" || suppliedFingerprint != currentFingerprint {
		partial, loadErr := h.loadAgendaPointListPartial(ctx, params.Slug, params.MeetingId, meetingID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		diffResult.Preview.Warning = i18n.T(ctx, "agenda_import.warning_stale_diff")
		partial.Import = diffResult.Preview
		return partial, nil, nil
	}

	if err := h.Repository.ApplyAgendaPoints(ctx, meetingID, diffResult.ApplyItems, diffResult.DeleteIDs); err != nil {
		return nil, nil, fmt.Errorf("failed to apply agenda import: %w", err)
	}
	h.publishSpeakersUpdated(meetingID)

	partial, err := h.loadAgendaPointListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	if err != nil {
		return nil, nil, err
	}
	return partial, nil, nil
}

func (h *Handler) buildAgendaDiffResult(ctx context.Context, meetingID int64, sourceText string, lines []agendaImportLine) (*agendaDiffBuildResult, string) {
	importedPoints, errKey := buildImportedAgendaPointsFromLines(lines)
	if errKey != "" {
		return nil, errKey
	}

	topLevel, err := h.Repository.ListAgendaPointsForMeeting(ctx, meetingID)
	if err != nil {
		return nil, "agenda_import.error_load_existing"
	}
	children, err := h.Repository.ListSubAgendaPointsForMeeting(ctx, meetingID)
	if err != nil {
		return nil, "agenda_import.error_load_existing"
	}
	existing := flattenAgendaPoints(topLevel, children)
	fingerprint, err := h.computeAgendaFingerprint(ctx, meetingID)
	if err != nil {
		return nil, "agenda_import.error_load_existing"
	}

	diffItems, applyItems, deleteIDs := buildAgendaDiff(existing, importedPoints)
	return &agendaDiffBuildResult{
		Preview: templates.AgendaImportPreview{
			SourceText:  sourceText,
			Lines:       toTemplateImportLines(lines),
			Diff:        diffItems,
			Fingerprint: fingerprint,
			Stage:       "diff",
		},
		ApplyItems: applyItems,
		DeleteIDs:  deleteIDs,
	}, ""
}

func (h *Handler) computeAgendaFingerprint(ctx context.Context, meetingID int64) (string, error) {
	topLevel, err := h.Repository.ListAgendaPointsForMeeting(ctx, meetingID)
	if err != nil {
		return "", fmt.Errorf("failed to load agenda points: %w", err)
	}
	children, err := h.Repository.ListSubAgendaPointsForMeeting(ctx, meetingID)
	if err != nil {
		return "", fmt.Errorf("failed to load sub-agenda points: %w", err)
	}

	var b strings.Builder
	for _, ap := range flattenAgendaPoints(topLevel, children) {
		parentID := int64(-1)
		if ap.ParentID != nil {
			parentID = *ap.ParentID
		}
		fmt.Fprintf(&b, "%d|%d|%d|%s\n", ap.ID, parentID, ap.Position, strings.TrimSpace(ap.Title))
	}
	sum := sha256.Sum256([]byte(b.String()))
	return fmt.Sprintf("%x", sum), nil
}

func toTemplateImportLines(lines []agendaImportLine) []templates.AgendaImportLineItem {
	items := make([]templates.AgendaImportLineItem, 0, len(lines))
	for _, line := range lines {
		items = append(items, templates.AgendaImportLineItem{
			LineNo:        line.LineNo,
			Text:          line.Text,
			State:         line.State,
			DetectedState: line.DetectedState,
		})
	}
	return items
}

func parseImportLinesFromForm(r *http.Request) ([]agendaImportLine, string) {
	lineNos := r.Form["line_no"]
	lineTexts := r.Form["line_text"]
	lineStates := r.Form["line_state"]
	lineDetected := r.Form["line_detected_state"]

	if len(lineNos) == 0 || len(lineTexts) == 0 || len(lineStates) == 0 {
		return nil, "agenda_import.error_missing_lines"
	}
	if len(lineNos) != len(lineTexts) || len(lineNos) != len(lineStates) {
		return nil, "agenda_import.error_missing_lines"
	}
	if len(lineDetected) > 0 && len(lineDetected) != len(lineNos) {
		return nil, "agenda_import.error_missing_lines"
	}

	lines := make([]agendaImportLine, 0, len(lineNos))
	for idx := range lineNos {
		lineNo, err := strconv.Atoi(strings.TrimSpace(lineNos[idx]))
		if err != nil || lineNo <= 0 {
			return nil, "agenda_import.error_missing_lines"
		}
		state := normalizeImportLineState(lineStates[idx])
		if state == "" {
			return nil, "agenda_import.error_invalid_line_state"
		}
		detected := importLineIgnore
		if len(lineDetected) > idx {
			detected = normalizeImportLineState(lineDetected[idx])
			if detected == "" {
				detected = importLineIgnore
			}
		}
		lines = append(lines, agendaImportLine{
			LineNo:        lineNo,
			Text:          strings.TrimSpace(lineTexts[idx]),
			State:         state,
			DetectedState: detected,
		})
	}
	slices.SortFunc(lines, func(a, b agendaImportLine) int { return a.LineNo - b.LineNo })
	return lines, ""
}

func normalizeImportLineState(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case importLineIgnore:
		return importLineIgnore
	case importLineHeading:
		return importLineHeading
	case importLineSubheading:
		return importLineSubheading
	default:
		return ""
	}
}

func extractAgendaImportLines(source string) ([]agendaImportLine, string) {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil, "agenda_import.error_empty_source"
	}
	if len(source) > agendaImportMaxBytes {
		return nil, "agenda_import.error_source_too_large"
	}

	lines := splitNonEmptyLines(source)
	if len(lines) == 0 {
		return nil, "agenda_import.error_empty_source"
	}

	if markdownLines := detectMarkdownHeadings(lines); len(markdownLines) > 0 {
		return markdownLines, ""
	}
	if plainLines := detectPlainTextLines(lines); len(plainLines) > 0 {
		return plainLines, ""
	}
	return nil, "agenda_import.error_parse_failed"
}

func splitNonEmptyLines(source string) []agendaImportLine {
	rawLines := strings.Split(source, "\n")
	result := make([]agendaImportLine, 0, len(rawLines))
	for idx, raw := range rawLines {
		text := strings.TrimSpace(strings.ReplaceAll(raw, "\r", ""))
		if text == "" {
			continue
		}
		result = append(result, agendaImportLine{
			LineNo:        idx + 1,
			Text:          text,
			State:         importLineIgnore,
			DetectedState: importLineIgnore,
		})
	}
	return result
}

var markdownHeadingPattern = regexp.MustCompile(`^\s{0,3}(#{1,6})\s+(.+?)\s*#*\s*$`)

func detectMarkdownHeadings(lines []agendaImportLine) []agendaImportLine {
	result := make([]agendaImportLine, len(lines))
	copy(result, lines)

	type headingInfo struct {
		idx   int
		level int
		title string
	}
	headings := make([]headingInfo, 0)
	inCodeFence := false
	for idx, line := range lines {
		text := strings.TrimSpace(line.Text)
		if strings.HasPrefix(text, "```") || strings.HasPrefix(text, "~~~") {
			inCodeFence = !inCodeFence
			continue
		}
		if inCodeFence {
			continue
		}
		matches := markdownHeadingPattern.FindStringSubmatch(text)
		if len(matches) != 3 {
			continue
		}
		headings = append(headings, headingInfo{
			idx:   idx,
			level: len(matches[1]),
			title: strings.TrimSpace(matches[2]),
		})
	}
	if len(headings) == 0 {
		return nil
	}

	h1Count := 0
	h23Count := 0
	for _, h := range headings {
		if h.level == 1 {
			h1Count++
		}
		if h.level == 2 || h.level == 3 {
			h23Count++
		}
	}
	useSpecial := h1Count == 1 && h23Count >= 2

	for _, h := range headings {
		result[h.idx].Text = h.title
		switch {
		case useSpecial && h.level == 2:
			result[h.idx].State = importLineHeading
			result[h.idx].DetectedState = importLineHeading
		case useSpecial && h.level == 3:
			result[h.idx].State = importLineSubheading
			result[h.idx].DetectedState = importLineSubheading
		case !useSpecial && h.level == 1:
			result[h.idx].State = importLineHeading
			result[h.idx].DetectedState = importLineHeading
		case !useSpecial && h.level == 2:
			result[h.idx].State = importLineSubheading
			result[h.idx].DetectedState = importLineSubheading
		}
	}
	return result
}

type numberedLine struct {
	prefix    string
	segments  []int
	titleText string
}

var numberedLinePattern = regexp.MustCompile(`^(?i)([a-z]+)?\s*(\d+(?:[.\)]\d+)*)[\):.\-]?\s+(.+)$`)

func parseNumberedLine(line string) *numberedLine {
	matches := numberedLinePattern.FindStringSubmatch(strings.TrimSpace(line))
	if len(matches) != 4 {
		return nil
	}
	segmentsRaw := strings.ReplaceAll(matches[2], ")", ".")
	parts := strings.Split(segmentsRaw, ".")
	segments := make([]int, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) == "" {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return nil
		}
		segments = append(segments, n)
	}
	if len(segments) == 0 {
		return nil
	}
	return &numberedLine{
		prefix:    strings.ToUpper(strings.TrimSpace(matches[1])),
		segments:  segments,
		titleText: strings.TrimSpace(matches[3]),
	}
}

func isContinuousNumbering(lines []*numberedLine) bool {
	if len(lines) == 0 {
		return false
	}
	expectedTop := lines[0].segments[0]
	subExpected := 0
	currentTop := -1

	for _, line := range lines {
		if len(line.segments) == 1 {
			if line.segments[0] != expectedTop {
				return false
			}
			currentTop = line.segments[0]
			expectedTop++
			subExpected = 1
			continue
		}
		if len(line.segments) >= 2 {
			if currentTop == -1 || line.segments[0] != currentTop {
				return false
			}
			if line.segments[1] != subExpected {
				return false
			}
			subExpected++
		}
	}
	return true
}

func detectPlainTextLines(lines []agendaImportLine) []agendaImportLine {
	result := make([]agendaImportLine, len(lines))
	copy(result, lines)

	numbered := make([]*numberedLine, len(lines))
	numberedCount := 0
	for idx, line := range lines {
		num := parseNumberedLine(line.Text)
		numbered[idx] = num
		if num != nil {
			numberedCount++
		}
	}

	usedNumbering := false
	if numberedCount > 0 {
		filtered := make([]*numberedLine, 0, numberedCount)
		for _, line := range numbered {
			if line != nil {
				filtered = append(filtered, line)
			}
		}

		prefix := ""
		prefixOK := true
		for _, line := range filtered {
			if line.prefix == "" {
				continue
			}
			if prefix == "" {
				prefix = line.prefix
				continue
			}
			if line.prefix != prefix {
				prefixOK = false
				break
			}
		}
		if prefixOK && isContinuousNumbering(filtered) {
			usedNumbering = true
			for idx := range result {
				num := numbered[idx]
				if num == nil {
					result[idx].State = importLineIgnore
					result[idx].DetectedState = importLineIgnore
					continue
				}
				if len(num.segments) == 1 {
					result[idx].State = importLineHeading
					result[idx].DetectedState = importLineHeading
				} else {
					result[idx].State = importLineSubheading
					result[idx].DetectedState = importLineSubheading
				}
			}
		}
	}

	if !usedNumbering {
		indents := make([]int, len(lines))
		minIndent := -1
		maxIndent := 0
		for idx, line := range lines {
			indent := 0
			for _, r := range line.Text {
				if r != ' ' && r != '\t' {
					break
				}
				indent++
			}
			indents[idx] = indent
			if minIndent == -1 || indent < minIndent {
				minIndent = indent
			}
			if indent > maxIndent {
				maxIndent = indent
			}
		}
		useIndentation := maxIndent > minIndent
		if useIndentation {
			for idx := range result {
				if indents[idx] == minIndent {
					result[idx].State = importLineHeading
					result[idx].DetectedState = importLineHeading
				} else {
					result[idx].State = importLineSubheading
					result[idx].DetectedState = importLineSubheading
				}
			}
		} else {
			for idx := range result {
				result[idx].State = importLineHeading
				result[idx].DetectedState = importLineHeading
			}
		}
	}

	return result
}

func buildImportedAgendaPointsFromLines(lines []agendaImportLine) ([]importedAgendaPoint, string) {
	points := make([]importedAgendaPoint, 0)
	topLevelPosition := int64(0)
	childPositionByParent := make(map[string]int64)
	var currentParentKey *string

	for _, line := range lines {
		state := normalizeImportLineState(line.State)
		if state == importLineIgnore {
			continue
		}
		title := strings.TrimSpace(line.Text)
		if numbered := parseNumberedLine(title); numbered != nil && line.DetectedState != importLineIgnore {
			title = strings.TrimSpace(numbered.titleText)
		}
		if title == "" {
			continue
		}
		key := fmt.Sprintf("line-%d", line.LineNo)
		switch state {
		case importLineHeading:
			topLevelPosition++
			parent := (*string)(nil)
			code := fmt.Sprintf("TOP %d", topLevelPosition)
			points = append(points, importedAgendaPoint{
				Key:         key,
				Title:       title,
				ParentKey:   parent,
				Position:    topLevelPosition,
				DisplayCode: code,
			})
			currentParentKey = &points[len(points)-1].Key
		case importLineSubheading:
			if currentParentKey == nil {
				return nil, "agenda_import.error_subheading_without_heading"
			}
			childPositionByParent[*currentParentKey]++
			code := fmt.Sprintf("TOP %d.%d", topLevelPosition, childPositionByParent[*currentParentKey])
			parentKey := *currentParentKey
			points = append(points, importedAgendaPoint{
				Key:         key,
				Title:       title,
				ParentKey:   &parentKey,
				Position:    childPositionByParent[parentKey],
				DisplayCode: code,
			})
		}
	}
	if len(points) == 0 {
		return nil, "agenda_import.error_no_headings_after_correction"
	}
	return points, ""
}

func buildAgendaDiff(existing []*model.AgendaPoint, imported []importedAgendaPoint) ([]templates.AgendaImportDiffItem, []repository.AgendaApplyPoint, []int64) {
	existingByID := make(map[int64]*model.AgendaPoint, len(existing))
	existingLabels := make(map[int64]string, len(existing))
	for _, ap := range existing {
		existingByID[ap.ID] = ap
		existingLabels[ap.ID] = agendaLabelForExisting(existingByID, ap)
	}

	type agendaPairing struct {
		ExistingID  int64
		ImportedKey string
		Tag         string
	}

	pairByExisting := make(map[int64]*agendaPairing)
	pairByImported := make(map[string]*agendaPairing)
	importedByKey := make(map[string]importedAgendaPoint, len(imported))
	for _, point := range imported {
		importedByKey[point.Key] = point
	}

	addPair := func(existingID int64, importedKey string, tag string) {
		pair := &agendaPairing{
			ExistingID:  existingID,
			ImportedKey: importedKey,
			Tag:         tag,
		}
		pairByExisting[existingID] = pair
		pairByImported[importedKey] = pair
	}

	// Step 1: match by title (exact first, then fuzzy) without tags.
	initialMatches := matchImportedToExisting(existing, imported)
	for importedKey, existingID := range initialMatches {
		addPair(existingID, importedKey, "")
	}

	unmatchedExistingByLabel := make(map[string]int64)
	for _, ap := range existing {
		if _, paired := pairByExisting[ap.ID]; paired {
			continue
		}
		unmatchedExistingByLabel[existingLabels[ap.ID]] = ap.ID
	}

	// Step 2: if both sides are unmatched at the same position label, tag as rename.
	for _, point := range imported {
		if _, paired := pairByImported[point.Key]; paired {
			continue
		}
		existingID, ok := unmatchedExistingByLabel[point.DisplayCode]
		if !ok {
			continue
		}
		addPair(existingID, point.Key, "rename")
		delete(unmatchedExistingByLabel, point.DisplayCode)
	}

	// Build the imported->existing mapping from current pairings for move detection.
	matches := make(map[string]int64, len(pairByImported))
	for importedKey, pair := range pairByImported {
		matches[importedKey] = pair.ExistingID
	}

	// Step 3: tag remaining pairings that do not have a tag yet.
	for _, pair := range pairByImported {
		if pair.Tag != "" {
			continue
		}
		existingPoint := existingByID[pair.ExistingID]
		importedPoint := importedByKey[pair.ImportedKey]
		renamed := normalizeAgendaTitle(existingPoint.Title) != normalizeAgendaTitle(importedPoint.Title)
		moved := hasMoved(existingPoint, importedPoint, matches, existingByID)
		switch {
		case moved:
			pair.Tag = "move"
		case renamed:
			pair.Tag = "rename"
		default:
			pair.Tag = "unchanged"
		}
	}

	pairKeyForExisting := func(existingID int64) string {
		return fmt.Sprintf("existing-%d", existingID)
	}
	toneForTag := func(tag string) (string, string) {
		switch tag {
		case "insert":
			return "neutral", "success"
		case "delete":
			return "error", "neutral"
		case "move", "rename":
			return "warning", "warning"
		default:
			return "neutral", "neutral"
		}
	}

	// Build apply operations using final pairings.
	applyItems := make([]repository.AgendaApplyPoint, 0, len(imported))
	for _, point := range imported {
		var existingID *int64
		if pair, ok := pairByImported[point.Key]; ok {
			id := pair.ExistingID
			existingID = &id
		}
		applyItems = append(applyItems, repository.AgendaApplyPoint{
			Key:        point.Key,
			ExistingID: existingID,
			ParentKey:  point.ParentKey,
			Title:      point.Title,
			Position:   point.Position,
		})
	}

	// Step 4: render by list position; only insert/delete produce gaps.
	diffItems := make([]templates.AgendaImportDiffItem, 0, maxInt(len(existing), len(imported)))
	existingIndex := 0
	importedIndex := 0
	for existingIndex < len(existing) || importedIndex < len(imported) {
		if existingIndex < len(existing) && importedIndex < len(imported) {
			existingPoint := existing[existingIndex]
			importedPoint := imported[importedIndex]
			existingPair, existingHasPair := pairByExisting[existingPoint.ID]
			importedPair, importedHasPair := pairByImported[importedPoint.Key]

			if !existingHasPair {
				beforeTone, afterTone := toneForTag("delete")
				diffItems = append(diffItems, templates.AgendaImportDiffItem{
					Operation:   "delete",
					BeforeLabel: existingLabels[existingPoint.ID],
					BeforeTitle: existingPoint.Title,
					BeforeTone:  beforeTone,
					AfterTone:   afterTone,
				})
				existingIndex++
				continue
			}
			if !importedHasPair {
				beforeTone, afterTone := toneForTag("insert")
				diffItems = append(diffItems, templates.AgendaImportDiffItem{
					Operation:  "insert",
					AfterLabel: importedPoint.DisplayCode,
					AfterTitle: importedPoint.Title,
					BeforeTone: beforeTone,
					AfterTone:  afterTone,
				})
				importedIndex++
				continue
			}

			operation := "move"
			if existingPair.ImportedKey == importedPoint.Key && importedPair.ExistingID == existingPoint.ID {
				operation = existingPair.Tag
			}
			beforeTone, afterTone := toneForTag(operation)
			diffItems = append(diffItems, templates.AgendaImportDiffItem{
				Operation:     operation,
				BeforeLabel:   existingLabels[existingPoint.ID],
				BeforeTitle:   existingPoint.Title,
				AfterLabel:    importedPoint.DisplayCode,
				AfterTitle:    importedPoint.Title,
				BeforeTone:    beforeTone,
				AfterTone:     afterTone,
				BeforePairKey: pairKeyForExisting(existingPair.ExistingID),
				AfterPairKey:  pairKeyForExisting(importedPair.ExistingID),
			})
			existingIndex++
			importedIndex++
			continue
		}

		if existingIndex < len(existing) {
			existingPoint := existing[existingIndex]
			existingPair, existingHasPair := pairByExisting[existingPoint.ID]
			operation := "delete"
			beforePairKey := ""
			if existingHasPair {
				operation = "move"
				beforePairKey = pairKeyForExisting(existingPair.ExistingID)
			}
			beforeTone, afterTone := toneForTag(operation)
			diffItems = append(diffItems, templates.AgendaImportDiffItem{
				Operation:     operation,
				BeforeLabel:   existingLabels[existingPoint.ID],
				BeforeTitle:   existingPoint.Title,
				BeforeTone:    beforeTone,
				AfterTone:     afterTone,
				BeforePairKey: beforePairKey,
			})
			existingIndex++
			continue
		}

		importedPoint := imported[importedIndex]
		importedPair, importedHasPair := pairByImported[importedPoint.Key]
		operation := "insert"
		afterPairKey := ""
		if importedHasPair {
			operation = "move"
			afterPairKey = pairKeyForExisting(importedPair.ExistingID)
		}
		beforeTone, afterTone := toneForTag(operation)
		diffItems = append(diffItems, templates.AgendaImportDiffItem{
			Operation:    operation,
			AfterLabel:   importedPoint.DisplayCode,
			AfterTitle:   importedPoint.Title,
			BeforeTone:   beforeTone,
			AfterTone:    afterTone,
			AfterPairKey: afterPairKey,
		})
		importedIndex++
	}

	deleteIDs := make([]int64, 0, len(existing))
	for _, ap := range existing {
		if _, ok := pairByExisting[ap.ID]; ok {
			continue
		}
		deleteIDs = append(deleteIDs, ap.ID)
	}

	return diffItems, applyItems, deleteIDs
}

func hasMoved(existing *model.AgendaPoint, point importedAgendaPoint, matches map[string]int64, existingByID map[int64]*model.AgendaPoint) bool {
	if point.ParentKey == nil && existing.ParentID != nil {
		return true
	}
	if point.ParentKey != nil {
		matchedParentID, ok := matches[*point.ParentKey]
		if !ok {
			return true
		}
		if existing.ParentID == nil || *existing.ParentID != matchedParentID {
			return true
		}
	}
	return existing.Position != point.Position
}

func agendaLabelForExisting(existingByID map[int64]*model.AgendaPoint, ap *model.AgendaPoint) string {
	if ap.ParentID == nil {
		return fmt.Sprintf("TOP %d", ap.Position)
	}
	parent := existingByID[*ap.ParentID]
	if parent == nil {
		return fmt.Sprintf("TOP ?.%d", ap.Position)
	}
	return fmt.Sprintf("TOP %d.%d", parent.Position, ap.Position)
}

func matchImportedToExisting(existing []*model.AgendaPoint, imported []importedAgendaPoint) map[string]int64 {
	matches := make(map[string]int64)
	used := make(map[int64]struct{})

	normalizedExisting := make(map[int64]string, len(existing))
	for _, ap := range existing {
		normalizedExisting[ap.ID] = normalizeAgendaTitle(ap.Title)
	}

	// First pass: exact normalized title matches.
	for _, point := range imported {
		target := normalizeAgendaTitle(point.Title)
		var candidateID int64
		count := 0
		for _, ap := range existing {
			if _, already := used[ap.ID]; already {
				continue
			}
			if normalizedExisting[ap.ID] != target {
				continue
			}
			candidateID = ap.ID
			count++
			if count > 1 {
				break
			}
		}
		if count == 1 {
			matches[point.Key] = candidateID
			used[candidateID] = struct{}{}
		}
	}

	// Second pass: fuzzy rename matching with positional proximity.
	for idx, point := range imported {
		if _, ok := matches[point.Key]; ok {
			continue
		}
		bestScore := 0.0
		bestID := int64(0)
		for exIdx, ap := range existing {
			if _, already := used[ap.ID]; already {
				continue
			}
			score := titleSimilarity(point.Title, ap.Title)
			score -= float64(absInt(idx-exIdx)) * 0.02
			if score > bestScore {
				bestScore = score
				bestID = ap.ID
			}
		}
		if bestID != 0 && bestScore >= 0.62 {
			matches[point.Key] = bestID
			used[bestID] = struct{}{}
		}
	}

	return matches
}

func normalizeAgendaTitle(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			return r
		}
		return -1
	}, s)
	return strings.Join(strings.Fields(s), " ")
}

func titleSimilarity(a, b string) float64 {
	na := normalizeAgendaTitle(a)
	nb := normalizeAgendaTitle(b)
	if na == "" || nb == "" {
		return 0
	}
	if na == nb {
		return 1
	}
	ar := []rune(na)
	br := []rune(nb)
	dist := levenshteinDistance(ar, br)
	maxLen := maxInt(len(ar), len(br))
	if maxLen == 0 {
		return 0
	}
	return 1 - (float64(dist) / float64(maxLen))
}

func levenshteinDistance(a, b []rune) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := 0; j <= len(b); j++ {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			curr[j] = minInt(
				curr[j-1]+1,
				prev[j]+1,
				prev[j-1]+cost,
			)
		}
		copy(prev, curr)
	}
	return prev[len(b)]
}

func minInt(values ...int) int {
	if len(values) == 0 {
		return 0
	}
	m := values[0]
	for _, v := range values[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
