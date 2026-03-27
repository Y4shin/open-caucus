package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Y4shin/conference-tool/internal/broker"
	"github.com/Y4shin/conference-tool/internal/repository/model"
	"github.com/Y4shin/conference-tool/internal/routes"
	"github.com/Y4shin/conference-tool/internal/session"
	"github.com/Y4shin/conference-tool/internal/templates"
)

// loadAttendeeSpeakersPartial loads the speakers list for the meeting's active agenda
// point and returns an AttendeeSpeakersListPartialInput ready for rendering.
func (h *Handler) loadAttendeeSpeakersPartial(ctx context.Context, slug, meetingIDStr string, meetingID, attendeeID int64) (*templates.AttendeeSpeakersListPartialInput, error) {
	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("failed to load meeting: %w", err)
	}

	topLevelAgendaPoints, err := h.Repository.ListAgendaPointsForMeeting(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("failed to load agenda points: %w", err)
	}
	subAgendaPoints, err := h.Repository.ListSubAgendaPointsForMeeting(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("failed to load sub-agenda points: %w", err)
	}
	agendaPoints := flattenAgendaPoints(topLevelAgendaPoints, subAgendaPoints)

	var speakers []*model.SpeakerEntry
	agendaTitle := ""
	hasActiveAP := false
	moderatorName := ""
	var effectiveModeratorID *int64
	var currentDoc *templates.LiveCurrentDocInfo
	votesPanel, err := h.loadLiveVotesPanel(ctx, slug, meetingIDStr, meetingID, attendeeID)
	if err != nil {
		return nil, fmt.Errorf("failed to load live votes panel: %w", err)
	}

	if meeting.CurrentAgendaPointID != nil {
		hasActiveAP = true
		ap, err := h.Repository.GetAgendaPointByID(ctx, *meeting.CurrentAgendaPointID)
		if err != nil {
			return nil, fmt.Errorf("failed to load agenda point: %w", err)
		}
		agendaTitle = ap.Title
		currentDoc, err = h.loadCurrentDocInfoForAgendaPoint(ctx, ap)
		if err != nil {
			return nil, fmt.Errorf("failed to load current document: %w", err)
		}
		effectiveModeratorID = ap.ModeratorID
		if effectiveModeratorID == nil {
			effectiveModeratorID = meeting.ModeratorID
		}
		speakers, err = h.Repository.ListSpeakersForAgendaPoint(ctx, *meeting.CurrentAgendaPointID)
		if err != nil {
			return nil, fmt.Errorf("failed to load speakers: %w", err)
		}
	}

	if effectiveModeratorID != nil {
		attendee, err := h.Repository.GetAttendeeByID(ctx, *effectiveModeratorID)
		if err == nil {
			moderatorName = attendee.FullName
		}
	}

	return &templates.AttendeeSpeakersListPartialInput{
		CommitteeSlug:     slug,
		IDString:          meetingIDStr,
		Speakers:          buildSpeakerItems(speakers),
		AgendaPoints:      buildAgendaPointItems(agendaPoints, meeting.CurrentAgendaPointID),
		CurrentAttendeeID: attendeeID,
		AgendaTitle:       agendaTitle,
		HasActiveAP:       hasActiveAP,
		ModeratorName:     moderatorName,
		CurrentDoc:        currentDoc,
		Votes:             *votesPanel,
	}, nil
}

// publishSpeakersUpdated broadcasts a speakers-updated SSE event scoped to a meeting.
func (h *Handler) publishSpeakersUpdated(meetingID int64) {
	mid := meetingID
	h.Broker.Publish(broker.SSEEvent{Event: "speakers-updated", Data: []byte("{}"), MeetingID: &mid})
}

// publishCurrentDocumentChanged broadcasts a speakers-updated event so live view OOB updates refresh.
func (h *Handler) publishCurrentDocumentChanged(meetingID int64) {
	mid := meetingID
	h.Broker.Publish(broker.SSEEvent{Event: "speakers-updated", Data: []byte("{}"), MeetingID: &mid})
}

// AttendeeSpeakersStream streams live speaker list updates to the attendee via SSE.
func (h *Handler) AttendeeSpeakersStream(ctx context.Context, r *http.Request, params routes.RouteParams) (<-chan routes.AttendeeSpeakersStreamEvent, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid meeting ID")
	}

	// meeting_access middleware has already validated access and populated CurrentAttendee
	ca, ok := session.GetCurrentAttendee(ctx)
	if !ok || ca.MeetingID != meetingID {
		return nil, fmt.Errorf("forbidden")
	}
	attendeeID := ca.AttendeeID

	brokerCh := h.Broker.Subscribe(ctx)
	eventCh := make(chan routes.AttendeeSpeakersStreamEvent, 4)

	go func() {
		defer close(eventCh)
		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-brokerCh:
				if !ok {
					return
				}
				if evt.MeetingID != nil && *evt.MeetingID != meetingID {
					continue
				}

				switch evt.Event {
				case "speakers-updated", "speakers.updated", "agenda.updated":
					partial, err := h.loadAttendeeSpeakersPartial(ctx, params.Slug, params.MeetingId, meetingID, attendeeID)
					if err != nil {
						continue
					}
					partial.CommitteeSlug = params.Slug
					partial.IDString = params.MeetingId
					partial.Votes.CommitteeSlug = params.Slug
					partial.Votes.MeetingIDStr = params.MeetingId
					partial.Votes.RefreshURL = fmt.Sprintf("/committee/%s/meeting/%s/votes/live/partial", params.Slug, params.MeetingId)
					select {
					case eventCh <- routes.SpeakersUpdatedEvent{Data: *partial}:
					default:
						// drop if consumer is slow
					}
				case meetingVotesChangedEvent, "votes.updated":
					votes, err := h.loadLiveVotesPanel(ctx, params.Slug, params.MeetingId, meetingID, attendeeID)
					if err != nil {
						continue
					}
					votes.CommitteeSlug = params.Slug
					votes.MeetingIDStr = params.MeetingId
					votes.RefreshURL = fmt.Sprintf("/committee/%s/meeting/%s/votes/live/partial", params.Slug, params.MeetingId)
					select {
					case eventCh <- routes.VotesUpdatedEvent{Data: *votes}:
					default:
						// drop if consumer is slow
					}
				}
			}
		}
	}()

	return eventCh, nil
}

// AttendeeSpeakerSelfAdd lets the current attendee add themselves as a speaker
// from the live page (regular or ropm).
func (h *Handler) AttendeeSpeakerSelfAdd(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.AttendeeSpeakersListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	if err := r.ParseForm(); err != nil {
		return nil, nil, fmt.Errorf("failed to parse form: %w", err)
	}

	speakerType := r.FormValue("type")
	if speakerType != "regular" && speakerType != "ropm" {
		partial, loadErr := h.loadAttendeeSpeakersPartial(ctx, params.Slug, params.MeetingId, meetingID, 0)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		partial.CommitteeSlug = params.Slug
		partial.IDString = params.MeetingId
		partial.Votes.CommitteeSlug = params.Slug
		partial.Votes.MeetingIDStr = params.MeetingId
		partial.Votes.RefreshURL = fmt.Sprintf("/committee/%s/meeting/%s/votes/live/partial", params.Slug, params.MeetingId)
		partial.Error = "Invalid speaker type."
		return partial, nil, nil
	}

	// meeting_access middleware has already validated access and populated CurrentAttendee
	ca, ok := session.GetCurrentAttendee(ctx)
	if !ok || ca.MeetingID != meetingID {
		return nil, routes.NewResponseMeta().WithRedirect(http.StatusSeeOther, "/"), nil
	}
	attendeeID := ca.AttendeeID

	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load meeting: %w", err)
	}
	if meeting.CurrentAgendaPointID == nil {
		partial, loadErr := h.loadAttendeeSpeakersPartial(ctx, params.Slug, params.MeetingId, meetingID, attendeeID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		partial.CommitteeSlug = params.Slug
		partial.IDString = params.MeetingId
		partial.Votes.CommitteeSlug = params.Slug
		partial.Votes.MeetingIDStr = params.MeetingId
		partial.Votes.RefreshURL = fmt.Sprintf("/committee/%s/meeting/%s/votes/live/partial", params.Slug, params.MeetingId)
		partial.Error = "No active agenda point."
		return partial, nil, nil
	}
	apID := *meeting.CurrentAgendaPointID

	entries, err := h.Repository.ListSpeakersForAgendaPoint(ctx, apID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load speakers: %w", err)
	}
	for _, e := range entries {
		if e.AttendeeID == attendeeID && e.Type == speakerType && e.Status != "DONE" {
			partial, loadErr := h.loadAttendeeSpeakersPartial(ctx, params.Slug, params.MeetingId, meetingID, attendeeID)
			if loadErr != nil {
				return nil, nil, loadErr
			}
			partial.CommitteeSlug = params.Slug
			partial.IDString = params.MeetingId
			partial.Votes.CommitteeSlug = params.Slug
			partial.Votes.MeetingIDStr = params.MeetingId
			partial.Votes.RefreshURL = fmt.Sprintf("/committee/%s/meeting/%s/votes/live/partial", params.Slug, params.MeetingId)
			partial.Error = fmt.Sprintf("You already have a non-done %s entry.", speakerType)
			return partial, nil, nil
		}
	}

	ap, err := h.Repository.GetAgendaPointByID(ctx, apID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load agenda point: %w", err)
	}
	effectiveGenderQuotation := meeting.GenderQuotationEnabled
	if ap.GenderQuotationEnabled != nil {
		effectiveGenderQuotation = *ap.GenderQuotationEnabled
	}

	attendee, err := h.Repository.GetAttendeeByID(ctx, attendeeID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load attendee: %w", err)
	}
	genderQuoted := attendee.Quoted && effectiveGenderQuotation
	hasSpoken, err := h.Repository.HasAttendeeSpokenOnAgendaPoint(ctx, apID, attendeeID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check first speaker: %w", err)
	}
	firstSpeaker := speakerType == "regular" && !hasSpoken

	if _, err := h.Repository.AddSpeaker(ctx, apID, attendeeID, speakerType, genderQuoted, firstSpeaker); err != nil {
		return nil, nil, fmt.Errorf("failed to add speaker: %w", err)
	}
	if err := h.Repository.RecomputeSpeakerOrder(ctx, apID); err != nil {
		return nil, nil, fmt.Errorf("failed to recompute speaker order: %w", err)
	}
	h.publishSpeakersUpdated(meetingID)

	partial, err := h.loadAttendeeSpeakersPartial(ctx, params.Slug, params.MeetingId, meetingID, attendeeID)
	if err != nil {
		return nil, nil, err
	}
	partial.CommitteeSlug = params.Slug
	partial.IDString = params.MeetingId
	partial.Votes.CommitteeSlug = params.Slug
	partial.Votes.MeetingIDStr = params.MeetingId
	partial.Votes.RefreshURL = fmt.Sprintf("/committee/%s/meeting/%s/votes/live/partial", params.Slug, params.MeetingId)
	return partial, nil, nil
}

// AttendeeSpeakerSelfYield lets the currently speaking attendee end their own speech.
func (h *Handler) AttendeeSpeakerSelfYield(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.AttendeeSpeakersListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}

	// meeting_access middleware has already validated access and populated CurrentAttendee
	ca, ok := session.GetCurrentAttendee(ctx)
	if !ok || ca.MeetingID != meetingID {
		return nil, routes.NewResponseMeta().WithRedirect(http.StatusSeeOther, "/"), nil
	}
	attendeeID := ca.AttendeeID

	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load meeting: %w", err)
	}
	if meeting.CurrentAgendaPointID == nil {
		partial, loadErr := h.loadAttendeeSpeakersPartial(ctx, params.Slug, params.MeetingId, meetingID, attendeeID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		partial.CommitteeSlug = params.Slug
		partial.IDString = params.MeetingId
		partial.Votes.CommitteeSlug = params.Slug
		partial.Votes.MeetingIDStr = params.MeetingId
		partial.Votes.RefreshURL = fmt.Sprintf("/committee/%s/meeting/%s/votes/live/partial", params.Slug, params.MeetingId)
		partial.Error = "No active agenda point."
		return partial, nil, nil
	}
	apID := *meeting.CurrentAgendaPointID

	entries, err := h.Repository.ListSpeakersForAgendaPoint(ctx, apID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load speakers: %w", err)
	}

	var speakingEntryID int64
	for _, e := range entries {
		if e.AttendeeID == attendeeID && e.Status == "SPEAKING" {
			speakingEntryID = e.ID
			break
		}
	}
	if speakingEntryID == 0 {
		partial, loadErr := h.loadAttendeeSpeakersPartial(ctx, params.Slug, params.MeetingId, meetingID, attendeeID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		partial.CommitteeSlug = params.Slug
		partial.IDString = params.MeetingId
		partial.Votes.CommitteeSlug = params.Slug
		partial.Votes.MeetingIDStr = params.MeetingId
		partial.Votes.RefreshURL = fmt.Sprintf("/committee/%s/meeting/%s/votes/live/partial", params.Slug, params.MeetingId)
		partial.Error = "You are not currently speaking."
		return partial, nil, nil
	}

	if err := h.Repository.SetSpeakerDone(ctx, speakingEntryID); err != nil {
		return nil, nil, fmt.Errorf("failed to yield speech: %w", err)
	}
	if err := h.Repository.RecomputeSpeakerOrder(ctx, apID); err != nil {
		return nil, nil, fmt.Errorf("failed to recompute speaker order: %w", err)
	}
	h.publishSpeakersUpdated(meetingID)

	partial, err := h.loadAttendeeSpeakersPartial(ctx, params.Slug, params.MeetingId, meetingID, attendeeID)
	if err != nil {
		return nil, nil, err
	}
	partial.CommitteeSlug = params.Slug
	partial.IDString = params.MeetingId
	partial.Votes.CommitteeSlug = params.Slug
	partial.Votes.MeetingIDStr = params.MeetingId
	partial.Votes.RefreshURL = fmt.Sprintf("/committee/%s/meeting/%s/votes/live/partial", params.Slug, params.MeetingId)
	return partial, nil, nil
}
