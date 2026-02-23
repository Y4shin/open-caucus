package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Y4shin/conference-tool/internal/routes"
	"github.com/Y4shin/conference-tool/internal/templates"
)

// MeetingModerate serves the condensed moderator view for a meeting.
func (h *Handler) MeetingModerate(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.ModeratePageInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}

	committee, err := h.Repository.GetCommitteeBySlug(ctx, params.Slug)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load committee: %w", err)
	}

	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load meeting: %w", err)
	}

	agendaPartial, err := h.loadAgendaPointListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	if err != nil {
		return nil, nil, err
	}

	speakersPartial, err := h.loadSpeakersListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	if err != nil {
		return nil, nil, err
	}

	attendeePartial, err := h.loadAttendeeListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	if err != nil {
		return nil, nil, err
	}

	input := &templates.ModeratePageInput{
		CommitteeName: committee.Name,
		CommitteeSlug: committee.Slug,
		MeetingName:   meeting.Name,
		IDString:      params.MeetingId,
		AgendaPoints:  *agendaPartial,
		Speakers:      *speakersPartial,
		Attendees:     *attendeePartial,
	}

	// Load tools data if there is an active agenda point.
	if meeting.CurrentAgendaPointID != nil {
		activeAP, err := h.Repository.GetAgendaPointByID(ctx, *meeting.CurrentAgendaPointID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load active agenda point: %w", err)
		}

		attachmentsPartial, err := h.loadAttachmentListPartial(ctx, params.Slug, params.MeetingId, activeAP)
		if err != nil {
			return nil, nil, err
		}

		motionsPartial, err := h.loadMotionListPartial(ctx, params.Slug, params.MeetingId, activeAP)
		if err != nil {
			return nil, nil, err
		}

		input.HasActiveAP = true
		input.ActiveAPTitle = activeAP.Title
		input.Attachments = *attachmentsPartial
		input.Motions = *motionsPartial
	}

	return input, nil, nil
}

// ModerateAgendaPartial refreshes the compact agenda list on the moderate page.
func (h *Handler) ModerateAgendaPartial(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.AgendaPointListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	partial, err := h.loadAgendaPointListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

// ModerateStream streams real-time updates for the moderate page right column.
func (h *Handler) ModerateStream(ctx context.Context, r *http.Request, params routes.RouteParams) (<-chan routes.ModerateStreamEvent, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid meeting ID")
	}
	clientID := strings.TrimSpace(r.URL.Query().Get("client_id"))

	brokerCh := h.Broker.Subscribe(ctx)
	eventCh := make(chan routes.ModerateStreamEvent, 8)

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
				if evt.Event != meetingAttendeesChangedEvent && evt.Event != speakersUpdatedEvent {
					continue
				}
				if evt.MeetingID == nil || *evt.MeetingID != meetingID {
					continue
				}
				if clientID != "" && evt.OriginClientID == clientID {
					continue
				}

				speakersPartial, err := h.loadSpeakersListPartial(ctx, params.Slug, params.MeetingId, meetingID)
				if err != nil {
					continue
				}
				attendeePartial, err := h.loadAttendeeListPartial(ctx, params.Slug, params.MeetingId, meetingID)
				if err != nil {
					continue
				}

				dependentInput := templates.ModerateDependentPartialInput{
					CommitteeSlug: params.Slug,
					IDString:      params.MeetingId,
					Speakers:      *speakersPartial,
					Attendees:     *attendeePartial,
				}

				select {
				case eventCh <- routes.ModerateUpdatedEvent{Data: dependentInput}:
				default:
				}
			}
		}
	}()

	return eventCh, nil
}
