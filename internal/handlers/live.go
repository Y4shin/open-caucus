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
	moderatorName := ""
	var effectiveModeratorID *int64

	if meeting.CurrentAgendaPointID != nil {
		hasActiveAP = true
		ap, err := h.Repository.GetAgendaPointByID(ctx, *meeting.CurrentAgendaPointID)
		if err != nil {
			return nil, fmt.Errorf("failed to load agenda point: %w", err)
		}
		agendaTitle = ap.Title
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
		CommitteeSlug:     "",
		IDString:          strconv.FormatInt(meetingID, 10),
		Speakers:          buildSpeakerItems(speakers),
		CurrentAttendeeID: attendeeID,
		AgendaTitle:       agendaTitle,
		HasActiveAP:       hasActiveAP,
		ModeratorName:     moderatorName,
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

	sd, ok := session.GetSession(ctx)
	if !ok || sd.IsExpired() {
		return nil, fmt.Errorf("forbidden")
	}

	var attendeeID int64
	if sd.IsUserSession() {
		if sd.UserID == nil {
			return nil, fmt.Errorf("forbidden")
		}
		attendee, err := h.Repository.GetAttendeeByUserIDAndMeetingID(ctx, *sd.UserID, meetingID)
		if err != nil {
			return nil, fmt.Errorf("forbidden")
		}
		attendeeID = attendee.ID
	} else {
		if !sd.IsAttendeeSession() || sd.MeetingID == nil || *sd.MeetingID != meetingID || sd.AttendeeID == nil {
			return nil, fmt.Errorf("forbidden")
		}
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
				partial.CommitteeSlug = params.Slug
				partial.IDString = params.MeetingId
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
		partial, loadErr := h.loadAttendeeSpeakersPartial(ctx, meetingID, 0)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		partial.CommitteeSlug = params.Slug
		partial.IDString = params.MeetingId
		partial.Error = "Invalid speaker type."
		return partial, nil, nil
	}

	sd, ok := session.GetSession(ctx)
	if !ok || sd.IsExpired() {
		return nil, routes.NewResponseMeta().WithRedirect(http.StatusSeeOther, "/"), nil
	}

	var attendeeID int64
	if sd.IsUserSession() {
		if sd.UserID == nil {
			return nil, routes.NewResponseMeta().WithRedirect(http.StatusSeeOther, "/"), nil
		}
		attendee, err := h.Repository.GetAttendeeByUserIDAndMeetingID(ctx, *sd.UserID, meetingID)
		if err != nil {
			return nil, nil, fmt.Errorf("forbidden")
		}
		attendeeID = attendee.ID
	} else {
		if !sd.IsAttendeeSession() || sd.MeetingID == nil || *sd.MeetingID != meetingID || sd.AttendeeID == nil {
			return nil, nil, fmt.Errorf("forbidden")
		}
		attendeeID = *sd.AttendeeID
	}

	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load meeting: %w", err)
	}
	if meeting.CurrentAgendaPointID == nil {
		partial, loadErr := h.loadAttendeeSpeakersPartial(ctx, meetingID, attendeeID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		partial.CommitteeSlug = params.Slug
		partial.IDString = params.MeetingId
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
			partial, loadErr := h.loadAttendeeSpeakersPartial(ctx, meetingID, attendeeID)
			if loadErr != nil {
				return nil, nil, loadErr
			}
			partial.CommitteeSlug = params.Slug
			partial.IDString = params.MeetingId
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
	firstSpeaker := !hasSpoken

	if _, err := h.Repository.AddSpeaker(ctx, apID, attendeeID, speakerType, genderQuoted, firstSpeaker); err != nil {
		return nil, nil, fmt.Errorf("failed to add speaker: %w", err)
	}
	if err := h.Repository.RecomputeSpeakerOrder(ctx, apID); err != nil {
		return nil, nil, fmt.Errorf("failed to recompute speaker order: %w", err)
	}
	h.publishSpeakersUpdated()

	partial, err := h.loadAttendeeSpeakersPartial(ctx, meetingID, attendeeID)
	if err != nil {
		return nil, nil, err
	}
	partial.CommitteeSlug = params.Slug
	partial.IDString = params.MeetingId
	return partial, nil, nil
}
