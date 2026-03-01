package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Y4shin/conference-tool/internal/broker"
	"github.com/Y4shin/conference-tool/internal/repository"
	"github.com/Y4shin/conference-tool/internal/repository/model"
	"github.com/Y4shin/conference-tool/internal/routes"
	"github.com/Y4shin/conference-tool/internal/session"
	"github.com/Y4shin/conference-tool/internal/templates"
)

const meetingVotesChangedEvent = "meeting-votes-changed"

func (h *Handler) publishMeetingVotesChanged(meetingID int64) {
	mid := meetingID
	h.Broker.Publish(broker.SSEEvent{Event: meetingVotesChangedEvent, Data: []byte("{}"), MeetingID: &mid})
}

func parseVoteOptionsText(raw string) []repository.VoteOptionInput {
	lines := strings.Split(raw, "\n")
	options := make([]repository.VoteOptionInput, 0, len(lines))
	position := int64(1)
	for _, line := range lines {
		label := strings.TrimSpace(line)
		if label == "" {
			continue
		}
		options = append(options, repository.VoteOptionInput{Label: label, Position: position})
		position++
	}
	return options
}

func parseVoteOptionsFromLabels(labels []string) []repository.VoteOptionInput {
	options := make([]repository.VoteOptionInput, 0, len(labels))
	position := int64(1)
	for _, label := range labels {
		trimmed := strings.TrimSpace(label)
		if trimmed == "" {
			continue
		}
		options = append(options, repository.VoteOptionInput{Label: trimmed, Position: position})
		position++
	}
	return options
}

func parseVoteOptionsFromForm(r *http.Request) []repository.VoteOptionInput {
	if labels, ok := r.Form["option_label"]; ok {
		options := parseVoteOptionsFromLabels(labels)
		if len(options) > 0 {
			return options
		}
	}
	return parseVoteOptionsText(r.FormValue("options_text"))
}

func joinVoteOptionsText(options []templates.VoteOptionDisplay) string {
	lines := make([]string, 0, len(options))
	for _, opt := range options {
		lines = append(lines, opt.Label)
	}
	return strings.Join(lines, "\n")
}

func parseInt64FormValue(r *http.Request, key string) (int64, error) {
	value := strings.TrimSpace(r.FormValue(key))
	if value == "" {
		return 0, fmt.Errorf("missing %s", key)
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s", key)
	}
	return parsed, nil
}

func parseInt64List(values []string) ([]int64, error) {
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid numeric value %q", value)
		}
		out = append(out, parsed)
	}
	return out, nil
}

func parseLeadingInt64(value string) (int64, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, false
	}
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return 0, false
	}
	parsed, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func (h *Handler) resolveManualAttendeeID(ctx context.Context, r *http.Request, meetingID int64) (int64, error) {
	if attendeeRaw := strings.TrimSpace(r.FormValue("attendee_id")); attendeeRaw != "" {
		attendeeID, err := strconv.ParseInt(attendeeRaw, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid attendee_id")
		}
		return attendeeID, nil
	}

	query := strings.TrimSpace(r.FormValue("attendee_query"))
	if query == "" {
		return 0, fmt.Errorf("attendee is required")
	}

	attendees, err := h.Repository.ListAttendeesForMeeting(ctx, meetingID)
	if err != nil {
		return 0, fmt.Errorf("list attendees: %w", err)
	}
	if len(attendees) == 0 {
		return 0, fmt.Errorf("no attendees available")
	}

	if number, ok := parseLeadingInt64(query); ok {
		for _, attendee := range attendees {
			if attendee.AttendeeNumber == number {
				return attendee.ID, nil
			}
		}
	}

	lowerQuery := strings.ToLower(query)

	// Prefer exact attendee name match.
	for _, attendee := range attendees {
		if strings.EqualFold(attendee.FullName, query) {
			return attendee.ID, nil
		}
	}

	// Fallback to unique substring match.
	matches := make([]int64, 0, 2)
	for _, attendee := range attendees {
		joined := strings.ToLower(fmt.Sprintf("%d %s", attendee.AttendeeNumber, attendee.FullName))
		if strings.Contains(joined, lowerQuery) {
			matches = append(matches, attendee.ID)
			if len(matches) > 1 {
				break
			}
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return 0, fmt.Errorf("attendee query is ambiguous")
	}

	return 0, fmt.Errorf("attendee not found")
}

func (h *Handler) loadModeratorVotesPanel(ctx context.Context, slug, meetingIDStr string, meetingID int64) (*templates.VoteModeratorPanelInput, error) {
	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("load meeting: %w", err)
	}

	input := &templates.VoteModeratorPanelInput{
		CommitteeSlug:   slug,
		MeetingIDStr:    meetingIDStr,
		HasActiveAgenda: meeting.CurrentAgendaPointID != nil,
		CreateURL:       fmt.Sprintf("/committee/%s/meeting/%s/votes/create", slug, meetingIDStr),
		RefreshURL:      fmt.Sprintf("/committee/%s/meeting/%s/votes/partial", slug, meetingIDStr),
	}

	if meeting.CurrentAgendaPointID == nil {
		return input, nil
	}

	agendaPoint, err := h.Repository.GetAgendaPointByID(ctx, *meeting.CurrentAgendaPointID)
	if err != nil {
		return nil, fmt.Errorf("load active agenda point: %w", err)
	}
	input.ActiveAPTitle = agendaPoint.Title

	attendees, err := h.Repository.ListAttendeesForMeeting(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("list attendees: %w", err)
	}
	input.Attendees = buildAttendeeItems(attendees)

	voteDefinitions, err := h.Repository.ListVoteDefinitionsForAgendaPoint(ctx, agendaPoint.ID)
	if err != nil {
		return nil, fmt.Errorf("list vote definitions: %w", err)
	}

	base := fmt.Sprintf("/committee/%s/meeting/%s/votes", slug, meetingIDStr)
	items := make([]templates.VoteModeratorItem, 0, len(voteDefinitions))
	for _, vote := range voteDefinitions {
		options, err := h.Repository.ListVoteOptions(ctx, vote.ID)
		if err != nil {
			return nil, fmt.Errorf("list vote options: %w", err)
		}
		optionViews := make([]templates.VoteOptionDisplay, 0, len(options))
		for _, opt := range options {
			optionViews = append(optionViews, templates.VoteOptionDisplay{
				ID:       opt.ID,
				IDString: strconv.FormatInt(opt.ID, 10),
				Label:    opt.Label,
				Position: opt.Position,
			})
		}

		item := templates.VoteModeratorItem{
			ID:                   vote.ID,
			IDString:             strconv.FormatInt(vote.ID, 10),
			Name:                 vote.Name,
			Visibility:           vote.Visibility,
			State:                vote.State,
			MinSelections:        vote.MinSelections,
			MaxSelections:        vote.MaxSelections,
			Options:              optionViews,
			OptionsText:          joinVoteOptionsText(optionViews),
			CreateOrUpdateURL:    fmt.Sprintf("%s/%d/update-draft", base, vote.ID),
			OpenURL:              fmt.Sprintf("%s/%d/open", base, vote.ID),
			CloseURL:             fmt.Sprintf("%s/%d/close", base, vote.ID),
			ArchiveURL:           fmt.Sprintf("%s/%d/archive", base, vote.ID),
			RegisterCastURL:      fmt.Sprintf("%s/%d/cast/register", base, vote.ID),
			CountOpenBallotURL:   fmt.Sprintf("%s/%d/ballot/open", base, vote.ID),
			CountSecretBallotURL: fmt.Sprintf("%s/%d/ballot/secret", base, vote.ID),
		}

		stats, statsErr := h.Repository.GetVoteSubmissionStatsLive(ctx, vote.ID)
		if statsErr == nil {
			item.EligibleCount = stats.EligibleCount
			item.CastCount = stats.CastCount
			item.BallotCount = stats.BallotCount
			item.OpenBallotCount = stats.OpenBallotCount
			item.SecretBallotCount = stats.SecretBallotCount
			item.OutstandingCount = stats.CastCount - stats.SecretBallotCount
			if item.OutstandingCount < 0 {
				item.OutstandingCount = 0
			}
		}

		switch vote.State {
		case model.VoteStateCounting:
			item.ResultsBlockedCounting = true
		case model.VoteStateClosed, model.VoteStateArchived:
			tallies, tallyErr := h.Repository.GetVoteTallies(ctx, vote.ID)
			if tallyErr == nil {
				item.HasResults = true
				item.Tallies = make([]templates.VoteOptionDisplay, 0, len(tallies))
				for _, row := range tallies {
					item.Tallies = append(item.Tallies, templates.VoteOptionDisplay{
						ID:       row.OptionID,
						IDString: strconv.FormatInt(row.OptionID, 10),
						Label:    row.Label,
						Count:    row.Count,
					})
				}
			}
		}

		items = append(items, item)
	}

	input.Votes = items
	return input, nil
}

func (h *Handler) loadLiveVotesPanel(ctx context.Context, slug, meetingIDStr string, meetingID, attendeeID int64) (*templates.VoteLivePanelInput, error) {
	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("load meeting: %w", err)
	}
	input := &templates.VoteLivePanelInput{
		CommitteeSlug:   slug,
		MeetingIDStr:    meetingIDStr,
		AttendeeID:      attendeeID,
		HasActiveAgenda: meeting.CurrentAgendaPointID != nil,
		RefreshURL:      fmt.Sprintf("/committee/%s/meeting/%s/votes/live/partial", slug, meetingIDStr),
	}
	if meeting.CurrentAgendaPointID == nil {
		return input, nil
	}

	votes, err := h.Repository.ListVoteDefinitionsForAgendaPoint(ctx, *meeting.CurrentAgendaPointID)
	if err != nil {
		return nil, fmt.Errorf("list vote definitions: %w", err)
	}

	items := make([]templates.VoteLiveItem, 0, len(votes))
	now := time.Now()
	for _, vote := range votes {
		options, err := h.Repository.ListVoteOptions(ctx, vote.ID)
		if err != nil {
			return nil, fmt.Errorf("list vote options: %w", err)
		}
		optionViews := make([]templates.VoteOptionDisplay, 0, len(options))
		for _, opt := range options {
			optionViews = append(optionViews, templates.VoteOptionDisplay{
				ID:       opt.ID,
				IDString: strconv.FormatInt(opt.ID, 10),
				Label:    opt.Label,
				Position: opt.Position,
			})
		}

		eligibleRows, err := h.Repository.ListEligibleVoters(ctx, vote.ID)
		if err != nil {
			return nil, fmt.Errorf("list eligible voters: %w", err)
		}
		eligible := false
		for _, row := range eligibleRows {
			if row.AttendeeID == attendeeID {
				eligible = true
				break
			}
		}

		item := templates.VoteLiveItem{
			ID:              vote.ID,
			IDString:        strconv.FormatInt(vote.ID, 10),
			Name:            vote.Name,
			Visibility:      vote.Visibility,
			MinSelections:   vote.MinSelections,
			MaxSelections:   vote.MaxSelections,
			State:           vote.State,
			IsEligible:      eligible,
			Options:         optionViews,
			SubmitOpenURL:   fmt.Sprintf("/committee/%s/meeting/%s/votes/%d/submit/open", slug, meetingIDStr, vote.ID),
			SubmitSecretURL: fmt.Sprintf("/committee/%s/meeting/%s/votes/%d/submit/secret", slug, meetingIDStr, vote.ID),
		}

		stats, statsErr := h.Repository.GetVoteSubmissionStatsLive(ctx, vote.ID)
		if statsErr == nil {
			item.EligibleCount = stats.EligibleCount
			item.CastCount = stats.CastCount
			item.BallotCount = stats.BallotCount
		}

		include := false
		switch vote.State {
		case model.VoteStateOpen, model.VoteStateCounting:
			include = true
		case model.VoteStateClosed:
			if vote.ClosedAt != nil {
				until := vote.ClosedAt.Add(30 * time.Second)
				if now.Before(until) {
					tallies, tallyErr := h.Repository.GetVoteTallies(ctx, vote.ID)
					if tallyErr == nil {
						item.HasTimedResults = true
						item.ResultsUntilUnix = until.Unix()
						item.ResultsRemaining = int64(until.Sub(now).Seconds()) + 1
						item.TimedResults = make([]templates.VoteOptionDisplay, 0, len(tallies))
						for _, row := range tallies {
							item.TimedResults = append(item.TimedResults, templates.VoteOptionDisplay{
								ID:       row.OptionID,
								IDString: strconv.FormatInt(row.OptionID, 10),
								Label:    row.Label,
								Count:    row.Count,
							})
						}
						include = true
					}
				}
			}
		}
		if include {
			items = append(items, item)
		}
	}

	input.Votes = items
	return input, nil
}

func (h *Handler) ModerateVotesPartial(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.VoteModeratorPanelInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	input, err := h.loadModeratorVotesPanel(ctx, params.Slug, params.MeetingId, meetingID)
	return input, nil, err
}

func (h *Handler) ModerateVoteCreate(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.VoteModeratorPanelInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	if err := r.ParseForm(); err != nil {
		return nil, nil, fmt.Errorf("parse form: %w", err)
	}

	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, nil, fmt.Errorf("load meeting: %w", err)
	}
	if meeting.CurrentAgendaPointID == nil {
		input, loadErr := h.loadModeratorVotesPanel(ctx, params.Slug, params.MeetingId, meetingID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		input.Error = "No active agenda point."
		return input, nil, nil
	}

	name := strings.TrimSpace(r.FormValue("name"))
	visibility := strings.TrimSpace(r.FormValue("visibility"))
	if name == "" || (visibility != model.VoteVisibilityOpen && visibility != model.VoteVisibilitySecret) {
		input, loadErr := h.loadModeratorVotesPanel(ctx, params.Slug, params.MeetingId, meetingID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		input.Error = "Name and visibility are required."
		return input, nil, nil
	}

	minSelections, minErr := parseInt64FormValue(r, "min_selections")
	maxSelections, maxErr := parseInt64FormValue(r, "max_selections")
	options := parseVoteOptionsFromForm(r)
	if minErr != nil || maxErr != nil || len(options) < 2 {
		input, loadErr := h.loadModeratorVotesPanel(ctx, params.Slug, params.MeetingId, meetingID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		input.Error = "Provide valid bounds and at least two non-empty options."
		return input, nil, nil
	}

	vote, err := h.Repository.CreateVoteDefinition(ctx, meetingID, *meeting.CurrentAgendaPointID, name, visibility, minSelections, maxSelections)
	if err == nil {
		err = h.Repository.ReplaceVoteOptions(ctx, vote.ID, options)
	}
	if err != nil {
		input, loadErr := h.loadModeratorVotesPanel(ctx, params.Slug, params.MeetingId, meetingID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		input.Error = fmt.Sprintf("Failed to create vote: %v", err)
		return input, nil, nil
	}

	h.publishMeetingVotesChanged(meetingID)
	input, err := h.loadModeratorVotesPanel(ctx, params.Slug, params.MeetingId, meetingID)
	if err != nil {
		return nil, nil, err
	}
	input.Notice = "Draft vote created."
	return input, nil, nil
}

func (h *Handler) ModerateVoteUpdateDraft(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.VoteModeratorPanelInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	voteID, err := strconv.ParseInt(params.VoteId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid vote ID")
	}
	if err := r.ParseForm(); err != nil {
		return nil, nil, fmt.Errorf("parse form: %w", err)
	}

	vote, err := h.Repository.GetVoteDefinitionByID(ctx, voteID)
	if err != nil {
		return nil, nil, err
	}
	name := strings.TrimSpace(r.FormValue("name"))
	visibility := strings.TrimSpace(r.FormValue("visibility"))
	minSelections, minErr := parseInt64FormValue(r, "min_selections")
	maxSelections, maxErr := parseInt64FormValue(r, "max_selections")
	options := parseVoteOptionsFromForm(r)
	if name == "" || minErr != nil || maxErr != nil || len(options) < 2 {
		input, loadErr := h.loadModeratorVotesPanel(ctx, params.Slug, params.MeetingId, meetingID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		input.Error = "Provide valid draft fields and at least two non-empty options."
		return input, nil, nil
	}

	_, err = h.Repository.UpdateVoteDefinitionDraft(ctx, voteID, vote.MeetingID, vote.AgendaPointID, name, visibility, minSelections, maxSelections)
	if err == nil {
		err = h.Repository.ReplaceVoteOptions(ctx, voteID, options)
	}
	if err != nil {
		input, loadErr := h.loadModeratorVotesPanel(ctx, params.Slug, params.MeetingId, meetingID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		input.Error = fmt.Sprintf("Failed to update draft: %v", err)
		return input, nil, nil
	}

	h.publishMeetingVotesChanged(meetingID)
	input, err := h.loadModeratorVotesPanel(ctx, params.Slug, params.MeetingId, meetingID)
	if err != nil {
		return nil, nil, err
	}
	input.Notice = "Draft vote updated."
	return input, nil, nil
}

func (h *Handler) ModerateVoteOpen(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.VoteModeratorPanelInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	voteID, err := strconv.ParseInt(params.VoteId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid vote ID")
	}

	attendees, err := h.Repository.ListAttendeesForMeeting(ctx, meetingID)
	if err != nil {
		return nil, nil, err
	}
	eligible := make([]int64, 0, len(attendees))
	for _, attendee := range attendees {
		eligible = append(eligible, attendee.ID)
	}

	_, err = h.Repository.OpenVoteWithEligibleVoters(ctx, voteID, eligible)
	if err != nil {
		input, loadErr := h.loadModeratorVotesPanel(ctx, params.Slug, params.MeetingId, meetingID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		input.Error = fmt.Sprintf("Failed to open vote: %v", err)
		return input, nil, nil
	}

	h.publishMeetingVotesChanged(meetingID)
	input, err := h.loadModeratorVotesPanel(ctx, params.Slug, params.MeetingId, meetingID)
	if err != nil {
		return nil, nil, err
	}
	input.Notice = "Vote opened with all attendees as eligible voters."
	return input, nil, nil
}

func (h *Handler) ModerateVoteClose(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.VoteModeratorPanelInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	voteID, err := strconv.ParseInt(params.VoteId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid vote ID")
	}

	result, err := h.Repository.CloseVote(ctx, voteID)
	if err != nil {
		input, loadErr := h.loadModeratorVotesPanel(ctx, params.Slug, params.MeetingId, meetingID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		input.Error = fmt.Sprintf("Failed to close vote: %v", err)
		return input, nil, nil
	}

	h.publishMeetingVotesChanged(meetingID)
	input, err := h.loadModeratorVotesPanel(ctx, params.Slug, params.MeetingId, meetingID)
	if err != nil {
		return nil, nil, err
	}
	switch result.Outcome {
	case model.CloseVoteOutcomeEnteredCounting:
		input.Notice = "Vote entered counting phase."
	case model.CloseVoteOutcomeStillCounting:
		input.Notice = "Vote is still in counting phase."
	default:
		input.Notice = "Vote closed."
	}
	return input, nil, nil
}

func (h *Handler) ModerateVoteArchive(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.VoteModeratorPanelInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	voteID, err := strconv.ParseInt(params.VoteId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid vote ID")
	}

	if _, err := h.Repository.ArchiveVote(ctx, voteID); err != nil {
		input, loadErr := h.loadModeratorVotesPanel(ctx, params.Slug, params.MeetingId, meetingID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		input.Error = fmt.Sprintf("Failed to archive vote: %v", err)
		return input, nil, nil
	}

	h.publishMeetingVotesChanged(meetingID)
	input, err := h.loadModeratorVotesPanel(ctx, params.Slug, params.MeetingId, meetingID)
	if err != nil {
		return nil, nil, err
	}
	input.Notice = "Vote archived."
	return input, nil, nil
}

func (h *Handler) ModerateVoteRegisterCast(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.VoteModeratorPanelInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	voteID, err := strconv.ParseInt(params.VoteId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid vote ID")
	}
	if err := r.ParseForm(); err != nil {
		return nil, nil, fmt.Errorf("parse form: %w", err)
	}
	attendeeID, parseErr := h.resolveManualAttendeeID(ctx, r, meetingID)
	source := strings.TrimSpace(r.FormValue("source"))
	if source == "" {
		source = model.VoteCastSourceManualSubmission
	}
	if parseErr != nil {
		input, loadErr := h.loadModeratorVotesPanel(ctx, params.Slug, params.MeetingId, meetingID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		input.Error = parseErr.Error()
		return input, nil, nil
	}

	if _, err := h.Repository.RegisterVoteCast(ctx, voteID, meetingID, attendeeID, source); err != nil {
		input, loadErr := h.loadModeratorVotesPanel(ctx, params.Slug, params.MeetingId, meetingID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		input.Error = fmt.Sprintf("Failed to register cast: %v", err)
		return input, nil, nil
	}

	h.publishMeetingVotesChanged(meetingID)
	input, err := h.loadModeratorVotesPanel(ctx, params.Slug, params.MeetingId, meetingID)
	if err != nil {
		return nil, nil, err
	}
	input.Notice = "Cast registered."
	return input, nil, nil
}

func (h *Handler) ModerateVoteCountOpenBallot(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.VoteModeratorPanelInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	voteID, err := strconv.ParseInt(params.VoteId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid vote ID")
	}
	if err := r.ParseForm(); err != nil {
		return nil, nil, fmt.Errorf("parse form: %w", err)
	}

	attendeeID, attendeeErr := h.resolveManualAttendeeID(ctx, r, meetingID)
	optionIDs, optionErr := parseInt64List(r.Form["option_id"])
	receiptToken := strings.TrimSpace(r.FormValue("receipt_token"))
	if receiptToken == "" {
		receiptToken, _ = generateSecret()
	}
	if attendeeErr != nil || optionErr != nil {
		input, loadErr := h.loadModeratorVotesPanel(ctx, params.Slug, params.MeetingId, meetingID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		if attendeeErr != nil {
			input.Error = attendeeErr.Error()
		} else {
			input.Error = optionErr.Error()
		}
		return input, nil, nil
	}

	_, err = h.Repository.SubmitOpenBallot(ctx, repository.OpenBallotSubmission{
		VoteDefinitionID: voteID,
		MeetingID:        meetingID,
		AttendeeID:       attendeeID,
		Source:           model.VoteCastSourceManualSubmission,
		ReceiptToken:     receiptToken,
		OptionIDs:        optionIDs,
	})
	if err != nil {
		input, loadErr := h.loadModeratorVotesPanel(ctx, params.Slug, params.MeetingId, meetingID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		input.Error = fmt.Sprintf("Failed to submit manual open ballot: %v", err)
		return input, nil, nil
	}

	h.publishMeetingVotesChanged(meetingID)
	input, err := h.loadModeratorVotesPanel(ctx, params.Slug, params.MeetingId, meetingID)
	if err != nil {
		return nil, nil, err
	}
	input.Notice = "Manual open ballot counted."
	return input, nil, nil
}

func (h *Handler) ModerateVoteCountSecretBallot(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.VoteModeratorPanelInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	voteID, err := strconv.ParseInt(params.VoteId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid vote ID")
	}
	if err := r.ParseForm(); err != nil {
		return nil, nil, fmt.Errorf("parse form: %w", err)
	}
	optionIDs, parseErr := parseInt64List(r.Form["option_id"])
	if parseErr != nil {
		input, loadErr := h.loadModeratorVotesPanel(ctx, params.Slug, params.MeetingId, meetingID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		input.Error = parseErr.Error()
		return input, nil, nil
	}

	receiptToken := strings.TrimSpace(r.FormValue("receipt_token"))
	if receiptToken == "" {
		receiptToken, _ = generateSecret()
	}
	cipher := strings.TrimSpace(r.FormValue("commitment_cipher"))
	if cipher == "" {
		cipher = "xchacha20poly1305"
	}
	version := int64(1)
	if raw := strings.TrimSpace(r.FormValue("commitment_version")); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
			version = parsed
		}
	}

	payload := []byte{}
	if raw := strings.TrimSpace(r.FormValue("encrypted_commitment_b64")); raw != "" {
		decoded, err := base64.StdEncoding.DecodeString(raw)
		if err == nil {
			payload = decoded
		}
	}
	if len(payload) == 0 {
		payload = []byte(fmt.Sprintf("manual:%s:%d", receiptToken, time.Now().UnixNano()))
	}

	_, err = h.Repository.SubmitSecretBallot(ctx, repository.SecretBallotSubmission{
		VoteDefinitionID:    voteID,
		ReceiptToken:        receiptToken,
		EncryptedCommitment: payload,
		CommitmentCipher:    cipher,
		CommitmentVersion:   version,
		OptionIDs:           optionIDs,
	})
	if err != nil {
		input, loadErr := h.loadModeratorVotesPanel(ctx, params.Slug, params.MeetingId, meetingID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		input.Error = fmt.Sprintf("Failed to count secret ballot: %v", err)
		return input, nil, nil
	}

	h.publishMeetingVotesChanged(meetingID)
	input, err := h.loadModeratorVotesPanel(ctx, params.Slug, params.MeetingId, meetingID)
	if err != nil {
		return nil, nil, err
	}
	input.Notice = "Secret ballot counted."
	return input, nil, nil
}

func (h *Handler) LiveVotesPartial(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.VoteLivePanelInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	ca, ok := session.GetCurrentAttendee(ctx)
	if !ok || ca.MeetingID != meetingID {
		input := &templates.VoteLivePanelInput{CommitteeSlug: params.Slug, MeetingIDStr: params.MeetingId, Error: "Attendee session required."}
		return input, nil, nil
	}
	input, err := h.loadLiveVotesPanel(ctx, params.Slug, params.MeetingId, meetingID, ca.AttendeeID)
	return input, nil, err
}

func (h *Handler) LiveSubmitOpenBallot(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.VoteLivePanelInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	voteID, err := strconv.ParseInt(params.VoteId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid vote ID")
	}
	ca, ok := session.GetCurrentAttendee(ctx)
	if !ok || ca.MeetingID != meetingID {
		return nil, nil, fmt.Errorf("attendee session required")
	}
	if err := r.ParseForm(); err != nil {
		return nil, nil, fmt.Errorf("parse form: %w", err)
	}

	optionIDs, parseErr := parseInt64List(r.Form["option_id"])
	receiptToken := strings.TrimSpace(r.FormValue("receipt_token"))
	if receiptToken == "" {
		receiptToken, _ = generateSecret()
	}
	if parseErr != nil {
		input, loadErr := h.loadLiveVotesPanel(ctx, params.Slug, params.MeetingId, meetingID, ca.AttendeeID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		input.Error = parseErr.Error()
		return input, nil, nil
	}

	_, err = h.Repository.SubmitOpenBallot(ctx, repository.OpenBallotSubmission{
		VoteDefinitionID: voteID,
		MeetingID:        meetingID,
		AttendeeID:       ca.AttendeeID,
		Source:           model.VoteCastSourceSelfSubmission,
		ReceiptToken:     receiptToken,
		OptionIDs:        optionIDs,
	})
	if err != nil {
		input, loadErr := h.loadLiveVotesPanel(ctx, params.Slug, params.MeetingId, meetingID, ca.AttendeeID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		input.Error = fmt.Sprintf("Open ballot rejected: %v", err)
		return input, nil, nil
	}

	vote, _ := h.Repository.GetVoteDefinitionByID(ctx, voteID)
	h.publishMeetingVotesChanged(meetingID)
	input, err := h.loadLiveVotesPanel(ctx, params.Slug, params.MeetingId, meetingID, ca.AttendeeID)
	if err != nil {
		return nil, nil, err
	}
	input.Notice = "Open ballot submitted. Receipt stored locally."
	voteName := fmt.Sprintf("Vote %d", voteID)
	if vote != nil {
		voteName = vote.Name
	}
	input.LastReceipt = &templates.VoteReceiptPayload{
		ID:           fmt.Sprintf("open:%d:%s", voteID, receiptToken),
		Kind:         "open",
		VoteID:       voteID,
		VoteName:     voteName,
		ReceiptToken: receiptToken,
		Receipt:      fmt.Sprintf("%d:%s:%d", voteID, receiptToken, ca.AttendeeID),
		AttendeeID:   ca.AttendeeID,
	}
	return input, nil, nil
}

func (h *Handler) LiveSubmitSecretBallot(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.VoteLivePanelInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	voteID, err := strconv.ParseInt(params.VoteId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid vote ID")
	}
	ca, ok := session.GetCurrentAttendee(ctx)
	if !ok || ca.MeetingID != meetingID {
		return nil, nil, fmt.Errorf("attendee session required")
	}
	if err := r.ParseForm(); err != nil {
		return nil, nil, fmt.Errorf("parse form: %w", err)
	}

	optionIDs, parseErr := parseInt64List(r.Form["option_id"])
	receiptToken := strings.TrimSpace(r.FormValue("receipt_token"))
	nonce := strings.TrimSpace(r.FormValue("nonce"))
	if receiptToken == "" {
		receiptToken, _ = generateSecret()
	}
	if nonce == "" {
		nonce, _ = generateSecret()
	}
	if parseErr != nil {
		input, loadErr := h.loadLiveVotesPanel(ctx, params.Slug, params.MeetingId, meetingID, ca.AttendeeID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		input.Error = parseErr.Error()
		return input, nil, nil
	}

	payload := []byte{}
	if raw := strings.TrimSpace(r.FormValue("encrypted_commitment_b64")); raw != "" {
		decoded, err := base64.StdEncoding.DecodeString(raw)
		if err == nil {
			payload = decoded
		}
	}
	if len(payload) == 0 {
		payload = []byte(fmt.Sprintf("%d:%v:%s", ca.AttendeeID, optionIDs, nonce))
	}

	cipher := strings.TrimSpace(r.FormValue("commitment_cipher"))
	if cipher == "" {
		cipher = "xchacha20poly1305"
	}
	version := int64(1)
	if raw := strings.TrimSpace(r.FormValue("commitment_version")); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
			version = parsed
		}
	}

	if _, err := h.Repository.RegisterVoteCast(ctx, voteID, meetingID, ca.AttendeeID, model.VoteCastSourceSelfSubmission); err != nil {
		input, loadErr := h.loadLiveVotesPanel(ctx, params.Slug, params.MeetingId, meetingID, ca.AttendeeID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		input.Error = fmt.Sprintf("Secret ballot rejected: %v", err)
		return input, nil, nil
	}

	_, err = h.Repository.SubmitSecretBallot(ctx, repository.SecretBallotSubmission{
		VoteDefinitionID:    voteID,
		ReceiptToken:        receiptToken,
		EncryptedCommitment: payload,
		CommitmentCipher:    cipher,
		CommitmentVersion:   version,
		OptionIDs:           optionIDs,
	})
	if err != nil {
		input, loadErr := h.loadLiveVotesPanel(ctx, params.Slug, params.MeetingId, meetingID, ca.AttendeeID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		input.Error = fmt.Sprintf("Secret ballot rejected: %v", err)
		return input, nil, nil
	}

	vote, _ := h.Repository.GetVoteDefinitionByID(ctx, voteID)
	h.publishMeetingVotesChanged(meetingID)
	input, err := h.loadLiveVotesPanel(ctx, params.Slug, params.MeetingId, meetingID, ca.AttendeeID)
	if err != nil {
		return nil, nil, err
	}
	input.Notice = "Secret ballot submitted. Receipt stored locally."
	voteName := fmt.Sprintf("Vote %d", voteID)
	if vote != nil {
		voteName = vote.Name
	}
	input.LastReceipt = &templates.VoteReceiptPayload{
		ID:           fmt.Sprintf("secret:%d:%s", voteID, receiptToken),
		Kind:         "secret",
		VoteID:       voteID,
		VoteName:     voteName,
		ReceiptToken: receiptToken,
		Receipt:      fmt.Sprintf("%d:%s:%s", voteID, receiptToken, nonce),
		Nonce:        nonce,
	}
	return input, nil, nil
}

func (h *Handler) ReceiptsVaultPage(ctx context.Context, r *http.Request) (*templates.ReceiptsVaultInput, *routes.ResponseMeta, error) {
	return &templates.ReceiptsVaultInput{}, nil, nil
}

type verifyOpenVoteRequest struct {
	VoteID       int64  `json:"vote_id"`
	ReceiptToken string `json:"receipt_token"`
	AttendeeID   *int64 `json:"attendee_id,omitempty"`
}

type verifySecretVoteRequest struct {
	VoteID       int64  `json:"vote_id"`
	ReceiptToken string `json:"receipt_token"`
}

func writeVoteJSON(w http.ResponseWriter, status int, payload any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(payload)
}

func mapVoteVerifyHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "not found"):
		return http.StatusNotFound
	case strings.Contains(message, "counting"):
		return http.StatusConflict
	case strings.Contains(message, "invalid"):
		return http.StatusBadRequest
	default:
		return http.StatusBadRequest
	}
}

func (h *Handler) PublicVerifyOpenVoteReceipt(w http.ResponseWriter, r *http.Request) error {
	var req verifyOpenVoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return writeVoteJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
	}
	if req.VoteID <= 0 || strings.TrimSpace(req.ReceiptToken) == "" {
		return writeVoteJSON(w, http.StatusBadRequest, map[string]any{"error": "vote_id and receipt_token are required"})
	}

	verification, err := h.Repository.VerifyOpenBallotByReceipt(r.Context(), req.VoteID, strings.TrimSpace(req.ReceiptToken))
	if err != nil {
		return writeVoteJSON(w, mapVoteVerifyHTTPStatus(err), map[string]any{"error": err.Error()})
	}
	if req.AttendeeID != nil && verification.AttendeeID != *req.AttendeeID {
		return writeVoteJSON(w, http.StatusNotFound, map[string]any{"error": "ballot not found for attendee"})
	}

	return writeVoteJSON(w, http.StatusOK, map[string]any{
		"vote_id":           verification.VoteDefinitionID,
		"vote_name":         verification.VoteName,
		"attendee_id":       verification.AttendeeID,
		"attendee_number":   verification.AttendeeNumber,
		"receipt_token":     verification.ReceiptToken,
		"choice_labels":     verification.ChoiceLabels,
		"choice_option_ids": verification.ChoiceOptionIDs,
	})
}

func (h *Handler) PublicVerifySecretVoteReceipt(w http.ResponseWriter, r *http.Request) error {
	var req verifySecretVoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return writeVoteJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
	}
	if req.VoteID <= 0 || strings.TrimSpace(req.ReceiptToken) == "" {
		return writeVoteJSON(w, http.StatusBadRequest, map[string]any{"error": "vote_id and receipt_token are required"})
	}

	verification, err := h.Repository.VerifySecretBallotByReceipt(r.Context(), req.VoteID, strings.TrimSpace(req.ReceiptToken))
	if err != nil {
		return writeVoteJSON(w, mapVoteVerifyHTTPStatus(err), map[string]any{"error": err.Error()})
	}

	return writeVoteJSON(w, http.StatusOK, map[string]any{
		"vote_id":                  verification.VoteDefinitionID,
		"vote_name":                verification.VoteName,
		"receipt_token":            verification.ReceiptToken,
		"encrypted_commitment_b64": base64.StdEncoding.EncodeToString(verification.EncryptedCommitment),
		"commitment_cipher":        verification.CommitmentCipher,
		"commitment_version":       verification.CommitmentVersion,
	})
}
