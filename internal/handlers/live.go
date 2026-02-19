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
func (h *Handler) loadAttendeeSpeakersPartial(ctx context.Context, meetingID, attendeeID int64) (*templates.AttendeeSpeakersListPartialInput, error) {
	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("failed to load meeting: %w", err)
	}

	var speakers []*model.SpeakerEntry
	agendaTitle := ""
	hasActiveAP := false

	if meeting.CurrentAgendaPointID != nil {
		hasActiveAP = true
		ap, err := h.Repository.GetAgendaPointByID(ctx, *meeting.CurrentAgendaPointID)
		if err != nil {
			return nil, fmt.Errorf("failed to load agenda point: %w", err)
		}
		agendaTitle = ap.Title
		speakers, err = h.Repository.ListSpeakersForAgendaPoint(ctx, *meeting.CurrentAgendaPointID)
		if err != nil {
			return nil, fmt.Errorf("failed to load speakers: %w", err)
		}
	}

	return &templates.AttendeeSpeakersListPartialInput{
		Speakers:          buildSpeakerItems(speakers),
		CurrentAttendeeID: attendeeID,
		AgendaTitle:       agendaTitle,
		HasActiveAP:       hasActiveAP,
	}, nil
}

// publishSpeakersUpdated broadcasts a speakers-updated SSE event to all connected clients.
func (h *Handler) publishSpeakersUpdated() {
	h.Broker.Publish(broker.SSEEvent{Event: "speakers-updated", Data: []byte("{}")})
}

// AttendeeSpeakersStream streams live speaker list updates to the attendee via SSE.
func (h *Handler) AttendeeSpeakersStream(ctx context.Context, r *http.Request, params routes.RouteParams) (<-chan routes.AttendeeSpeakersStreamEvent, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid meeting ID")
	}

	sd, _ := session.GetSession(ctx)
	var attendeeID int64
	if sd != nil && sd.AttendeeID != nil {
		attendeeID = *sd.AttendeeID
	}

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
				if evt.Event != "speakers-updated" {
					continue
				}
				partial, err := h.loadAttendeeSpeakersPartial(ctx, meetingID, attendeeID)
				if err != nil {
					continue
				}
				select {
				case eventCh <- routes.SpeakersUpdatedEvent{Data: *partial}:
				default:
					// drop if consumer is slow
				}
			}
		}
	}()

	return eventCh, nil
}
