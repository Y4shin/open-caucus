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

// ManageSpeakerTogglePriority flips the priority flag on a WAITING speaker entry.
func (h *Handler) ManageSpeakerTogglePriority(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.SpeakersListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	speakerID, err := strconv.ParseInt(params.SpeakerId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid speaker ID")
	}
	entry, err := h.Repository.GetSpeakerEntryByID(ctx, speakerID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load speaker entry: %w", err)
	}
	if err := h.Repository.SetSpeakerPriority(ctx, speakerID, !entry.Priority); err != nil {
		return nil, nil, fmt.Errorf("failed to set speaker priority: %w", err)
	}
	if err := h.Repository.RecomputeSpeakerOrder(ctx, entry.AgendaPointID); err != nil {
		return nil, nil, fmt.Errorf("failed to recompute speaker order: %w", err)
	}
	h.publishSpeakersUpdated()
	partial, err := h.loadSpeakersListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

// ManageMeetingSetQuotation updates the meeting-level gender/first-speaker quotation settings.
func (h *Handler) ManageMeetingSetQuotation(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.MeetingSettingsPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	if err := r.ParseForm(); err != nil {
		return nil, nil, fmt.Errorf("failed to parse form: %w", err)
	}

	genderEnabled := r.FormValue("gender_quotation_enabled") == "true"
	firstSpeakerEnabled := r.FormValue("first_speaker_quotation_enabled") == "true"

	if err := h.Repository.SetMeetingGenderQuotation(ctx, meetingID, genderEnabled); err != nil {
		return nil, nil, fmt.Errorf("failed to set gender quotation: %w", err)
	}
	if err := h.Repository.SetMeetingFirstSpeakerQuotation(ctx, meetingID, firstSpeakerEnabled); err != nil {
		return nil, nil, fmt.Errorf("failed to set first-speaker quotation: %w", err)
	}

	// Recompute order for the active agenda point since settings affect ordering.
	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load meeting: %w", err)
	}
	if meeting.CurrentAgendaPointID != nil {
		if err := h.Repository.RecomputeSpeakerOrder(ctx, *meeting.CurrentAgendaPointID); err != nil {
			return nil, nil, fmt.Errorf("failed to recompute speaker order: %w", err)
		}
		h.publishSpeakersUpdated()
	}

	partial, err := h.loadMeetingSettingsPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

// ManageMeetingSetModerator assigns or clears the meeting-level moderator.
func (h *Handler) ManageMeetingSetModerator(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.MeetingSettingsPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	if err := r.ParseForm(); err != nil {
		return nil, nil, fmt.Errorf("failed to parse form: %w", err)
	}

	var attendeeID *int64
	if raw := strings.TrimSpace(r.FormValue("attendee_id")); raw != "" {
		aid, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid attendee ID")
		}
		attendeeID = &aid
	}

	if err := h.Repository.SetMeetingModerator(ctx, meetingID, attendeeID); err != nil {
		return nil, nil, fmt.Errorf("failed to set meeting moderator: %w", err)
	}

	partial, err := h.loadMeetingSettingsPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

// ManageAgendaPointSetQuotation sets per-agenda-point quotation overrides (nil = inherit from meeting).
func (h *Handler) ManageAgendaPointSetQuotation(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.SpeakersListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	apID, err := strconv.ParseInt(params.AgendaPointId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid agenda point ID")
	}
	if err := r.ParseForm(); err != nil {
		return nil, nil, fmt.Errorf("failed to parse form: %w", err)
	}

	// Parse nullable bool: "" = nil (inherit), "true" = true, "false" = false.
	parseNullBool := func(v string) *bool {
		switch v {
		case "true":
			b := true
			return &b
		case "false":
			b := false
			return &b
		default:
			return nil
		}
	}

	genderVal := parseNullBool(r.FormValue("gender_quotation_enabled"))
	firstSpeakerVal := parseNullBool(r.FormValue("first_speaker_quotation_enabled"))

	if err := h.Repository.SetAgendaPointGenderQuotation(ctx, apID, genderVal); err != nil {
		return nil, nil, fmt.Errorf("failed to set gender quotation: %w", err)
	}
	if err := h.Repository.SetAgendaPointFirstSpeakerQuotation(ctx, apID, firstSpeakerVal); err != nil {
		return nil, nil, fmt.Errorf("failed to set first-speaker quotation: %w", err)
	}
	if err := h.Repository.RecomputeSpeakerOrder(ctx, apID); err != nil {
		return nil, nil, fmt.Errorf("failed to recompute speaker order: %w", err)
	}
	h.publishSpeakersUpdated()

	partial, err := h.loadSpeakersListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

// ManageAgendaPointSetModerator assigns or clears the per-agenda-point moderator.
func (h *Handler) ManageAgendaPointSetModerator(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.SpeakersListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	apID, err := strconv.ParseInt(params.AgendaPointId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid agenda point ID")
	}
	if err := r.ParseForm(); err != nil {
		return nil, nil, fmt.Errorf("failed to parse form: %w", err)
	}

	var attendeeID *int64
	if raw := strings.TrimSpace(r.FormValue("attendee_id")); raw != "" {
		aid, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid attendee ID")
		}
		attendeeID = &aid
	}

	if err := h.Repository.SetAgendaPointModerator(ctx, apID, attendeeID); err != nil {
		return nil, nil, fmt.Errorf("failed to set agenda point moderator: %w", err)
	}

	partial, err := h.loadSpeakersListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}
