package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Y4shin/conference-tool/internal/repository/model"
	"github.com/Y4shin/conference-tool/internal/routes"
	"github.com/Y4shin/conference-tool/internal/templates"
)

// buildAttendeeItems converts model attendees to template items.
func buildAttendeeItems(attendees []*model.Attendee) []templates.AttendeeItem {
	items := make([]templates.AttendeeItem, len(attendees))
	for i, a := range attendees {
		items[i] = templates.AttendeeItem{
			ID:       a.ID,
			IDString: strconv.FormatInt(a.ID, 10),
			FullName: a.FullName,
			IsChair:  a.IsChair,
			IsGuest:  a.UserID == nil,
		}
	}
	return items
}

// buildAgendaPointItems converts model agenda points to template items.
func buildAgendaPointItems(aps []*model.AgendaPoint, currentID *int64) []templates.AgendaPointItem {
	items := make([]templates.AgendaPointItem, len(aps))
	for i, ap := range aps {
		isActive := currentID != nil && ap.ID == *currentID
		items[i] = templates.AgendaPointItem{
			ID:       ap.ID,
			IDString: strconv.FormatInt(ap.ID, 10),
			Position: ap.Position,
			Title:    ap.Title,
			IsActive: isActive,
		}
	}
	return items
}

// buildSpeakerItems converts model speaker entries to template items.
func buildSpeakerItems(entries []*model.SpeakerEntry) []templates.SpeakerItem {
	items := make([]templates.SpeakerItem, len(entries))
	for i, e := range entries {
		items[i] = templates.SpeakerItem{
			ID:           e.ID,
			IDString:     strconv.FormatInt(e.ID, 10),
			AttendeeName: e.AttendeeName,
			Type:         e.Type,
			Status:       e.Status,
			IsWaiting:    e.Status == "WAITING",
			IsSpeaking:   e.Status == "SPEAKING",
			GenderQuoted: e.GenderQuoted,
			FirstSpeaker: e.FirstSpeaker,
			Priority:     e.Priority,
		}
	}
	return items
}

// loadAttendeeListPartial loads the current attendee list for a meeting and
// returns an AttendeeListPartialInput ready for rendering.
func (h *Handler) loadAttendeeListPartial(ctx context.Context, slug, meetingIDStr string, meetingID int64) (*templates.AttendeeListPartialInput, error) {
	attendees, err := h.Repository.ListAttendeesForMeeting(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("failed to load attendees: %w", err)
	}
	return &templates.AttendeeListPartialInput{
		CommitteeSlug: slug,
		IDString:      meetingIDStr,
		Attendees:     buildAttendeeItems(attendees),
	}, nil
}

// ManageAttendeeCreate adds a guest attendee on behalf of the chairperson.
func (h *Handler) ManageAttendeeCreate(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.AttendeeListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}

	if err := r.ParseForm(); err != nil {
		return nil, nil, fmt.Errorf("failed to parse form: %w", err)
	}

	fullName := strings.TrimSpace(r.FormValue("full_name"))
	if fullName == "" {
		partial, loadErr := h.loadAttendeeListPartial(ctx, params.Slug, params.MeetingId, meetingID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		partial.Error = "Name is required."
		return partial, nil, nil
	}

	secret, err := generateSecret()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate secret: %w", err)
	}

	if _, err := h.Repository.CreateAttendee(ctx, meetingID, nil, fullName, secret); err != nil {
		return nil, nil, fmt.Errorf("failed to create attendee: %w", err)
	}

	partial, err := h.loadAttendeeListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

// ManageAttendeeDelete removes an attendee from the meeting.
func (h *Handler) ManageAttendeeDelete(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.AttendeeListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}

	attendeeID, err := strconv.ParseInt(params.AttendeeId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid attendee ID")
	}

	if err := h.Repository.DeleteAttendee(ctx, attendeeID); err != nil {
		return nil, nil, fmt.Errorf("failed to delete attendee: %w", err)
	}

	partial, err := h.loadAttendeeListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

// ManageAttendeeToggleChair flips the is_chair flag for an attendee.
func (h *Handler) ManageAttendeeToggleChair(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.AttendeeListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}

	attendeeID, err := strconv.ParseInt(params.AttendeeId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid attendee ID")
	}

	attendee, err := h.Repository.GetAttendeeByID(ctx, attendeeID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load attendee: %w", err)
	}

	if err := h.Repository.SetAttendeeIsChair(ctx, attendeeID, !attendee.IsChair); err != nil {
		return nil, nil, fmt.Errorf("failed to update attendee: %w", err)
	}

	partial, err := h.loadAttendeeListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

// loadAgendaPointListPartial loads the agenda point list for a meeting.
func (h *Handler) loadAgendaPointListPartial(ctx context.Context, slug, meetingIDStr string, meetingID int64) (*templates.AgendaPointListPartialInput, error) {
	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("failed to load meeting: %w", err)
	}
	aps, err := h.Repository.ListAgendaPointsForMeeting(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("failed to load agenda points: %w", err)
	}
	return &templates.AgendaPointListPartialInput{
		CommitteeSlug:        slug,
		IDString:             meetingIDStr,
		AgendaPoints:         buildAgendaPointItems(aps, meeting.CurrentAgendaPointID),
		CurrentAgendaPointID: meeting.CurrentAgendaPointID,
	}, nil
}

// loadSpeakersListPartial loads the speakers list for the active agenda point.
func (h *Handler) loadSpeakersListPartial(ctx context.Context, slug, meetingIDStr string, meetingID int64) (*templates.SpeakersListPartialInput, error) {
	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("failed to load meeting: %w", err)
	}
	attendees, err := h.Repository.ListAttendeesForMeeting(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("failed to load attendees: %w", err)
	}
	var speakers []*model.SpeakerEntry

	// Compute effective quotation settings; defaults to meeting level.
	effectiveGender := meeting.GenderQuotationEnabled
	effectiveFirstSpeaker := meeting.FirstSpeakerQuotationEnabled
	var apGenderQuotation *bool
	var apFirstSpeakerQuotation *bool
	var apModeratorID *int64
	apIDStr := ""

	if meeting.CurrentAgendaPointID != nil {
		apIDStr = strconv.FormatInt(*meeting.CurrentAgendaPointID, 10)
		speakers, err = h.Repository.ListSpeakersForAgendaPoint(ctx, *meeting.CurrentAgendaPointID)
		if err != nil {
			return nil, fmt.Errorf("failed to load speakers: %w", err)
		}
		ap, err := h.Repository.GetAgendaPointByID(ctx, *meeting.CurrentAgendaPointID)
		if err != nil {
			return nil, fmt.Errorf("failed to load agenda point: %w", err)
		}
		apGenderQuotation = ap.GenderQuotationEnabled
		apFirstSpeakerQuotation = ap.FirstSpeakerQuotationEnabled
		apModeratorID = ap.ModeratorID
		if ap.GenderQuotationEnabled != nil {
			effectiveGender = *ap.GenderQuotationEnabled
		}
		if ap.FirstSpeakerQuotationEnabled != nil {
			effectiveFirstSpeaker = *ap.FirstSpeakerQuotationEnabled
		}
		// Effective moderator: AP overrides meeting.
		if ap.ModeratorID == nil {
			apModeratorID = meeting.ModeratorID
		}
	}

	return &templates.SpeakersListPartialInput{
		CommitteeSlug:                    slug,
		IDString:                         meetingIDStr,
		CurrentAgendaPointID:             meeting.CurrentAgendaPointID,
		AgendaPointIDString:              apIDStr,
		GenderQuotationEnabled:           meeting.GenderQuotationEnabled,
		FirstSpeakerQuotationEnabled:     meeting.FirstSpeakerQuotationEnabled,
		AgendaPointGenderQuotation:       apGenderQuotation,
		AgendaPointFirstSpeakerQuotation: apFirstSpeakerQuotation,
		EffectiveGenderQuotation:         effectiveGender,
		EffectiveFirstSpeakerQuotation:   effectiveFirstSpeaker,
		ModeratorID:                      apModeratorID,
		Speakers:                         buildSpeakerItems(speakers),
		Attendees:                        buildAttendeeItems(attendees),
	}, nil
}

// ManageAgendaPointCreate adds a new agenda point to the meeting.
func (h *Handler) ManageAgendaPointCreate(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.AgendaPointListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	if err := r.ParseForm(); err != nil {
		return nil, nil, fmt.Errorf("failed to parse form: %w", err)
	}
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		partial, loadErr := h.loadAgendaPointListPartial(ctx, params.Slug, params.MeetingId, meetingID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		partial.Error = "Title is required."
		return partial, nil, nil
	}
	if _, err := h.Repository.CreateAgendaPoint(ctx, meetingID, title); err != nil {
		return nil, nil, fmt.Errorf("failed to create agenda point: %w", err)
	}
	partial, err := h.loadAgendaPointListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

// ManageAgendaPointDelete removes an agenda point.
func (h *Handler) ManageAgendaPointDelete(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.AgendaPointListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	apID, err := strconv.ParseInt(params.AgendaPointId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid agenda point ID")
	}
	if err := h.Repository.DeleteAgendaPoint(ctx, apID); err != nil {
		return nil, nil, fmt.Errorf("failed to delete agenda point: %w", err)
	}
	partial, err := h.loadAgendaPointListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

// ManageActivateAgendaPoint sets the meeting's active agenda point.
func (h *Handler) ManageActivateAgendaPoint(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.AgendaPointListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	apID, err := strconv.ParseInt(params.AgendaPointId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid agenda point ID")
	}
	if err := h.Repository.SetCurrentAgendaPoint(ctx, meetingID, &apID); err != nil {
		return nil, nil, fmt.Errorf("failed to activate agenda point: %w", err)
	}
	partial, err := h.loadAgendaPointListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

// ManageSpeakerAdd adds an attendee to the speakers list for the active agenda point.
func (h *Handler) ManageSpeakerAdd(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.SpeakersListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	if err := r.ParseForm(); err != nil {
		return nil, nil, fmt.Errorf("failed to parse form: %w", err)
	}
	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load meeting: %w", err)
	}
	if meeting.CurrentAgendaPointID == nil {
		partial, loadErr := h.loadSpeakersListPartial(ctx, params.Slug, params.MeetingId, meetingID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		partial.Error = "No active agenda point."
		return partial, nil, nil
	}
	attendeeID, err := strconv.ParseInt(r.FormValue("attendee_id"), 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid attendee ID")
	}
	speakerType := r.FormValue("type")
	if speakerType != "regular" && speakerType != "ropm" {
		return nil, nil, fmt.Errorf("invalid speaker type")
	}

	apID := *meeting.CurrentAgendaPointID

	// Resolve effective gender quotation setting: agenda point overrides meeting.
	ap, err := h.Repository.GetAgendaPointByID(ctx, apID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load agenda point: %w", err)
	}
	effectiveGenderQuotation := meeting.GenderQuotationEnabled
	if ap.GenderQuotationEnabled != nil {
		effectiveGenderQuotation = *ap.GenderQuotationEnabled
	}

	// Load the attendee to determine their quoted status.
	attendee, err := h.Repository.GetAttendeeByID(ctx, attendeeID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load attendee: %w", err)
	}
	genderQuoted := attendee.Quoted && effectiveGenderQuotation

	// Determine first-speaker status.
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
	partial, err := h.loadSpeakersListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

// ManageSpeakerRemove removes a speaker entry.
func (h *Handler) ManageSpeakerRemove(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.SpeakersListPartialInput, *routes.ResponseMeta, error) {
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
	if err := h.Repository.DeleteSpeaker(ctx, speakerID); err != nil {
		return nil, nil, fmt.Errorf("failed to remove speaker: %w", err)
	}
	if err := h.Repository.RecomputeSpeakerOrder(ctx, entry.AgendaPointID); err != nil {
		return nil, nil, fmt.Errorf("failed to recompute speaker order: %w", err)
	}
	partial, err := h.loadSpeakersListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

// ManageSpeakerStart moves a speaker to SPEAKING status.
func (h *Handler) ManageSpeakerStart(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.SpeakersListPartialInput, *routes.ResponseMeta, error) {
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
	if err := h.Repository.SetSpeakerSpeaking(ctx, speakerID, entry.AgendaPointID); err != nil {
		return nil, nil, fmt.Errorf("failed to start speech: %w", err)
	}
	if err := h.Repository.RecomputeSpeakerOrder(ctx, entry.AgendaPointID); err != nil {
		return nil, nil, fmt.Errorf("failed to recompute speaker order: %w", err)
	}
	partial, err := h.loadSpeakersListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

// ManageSpeakerEnd moves a speaker to DONE status.
func (h *Handler) ManageSpeakerEnd(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.SpeakersListPartialInput, *routes.ResponseMeta, error) {
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
	if err := h.Repository.SetSpeakerDone(ctx, speakerID); err != nil {
		return nil, nil, fmt.Errorf("failed to end speech: %w", err)
	}
	if err := h.Repository.RecomputeSpeakerOrder(ctx, entry.AgendaPointID); err != nil {
		return nil, nil, fmt.Errorf("failed to recompute speaker order: %w", err)
	}
	partial, err := h.loadSpeakersListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

// ManageSpeakerWithdraw moves a speaker to WITHDRAWN status.
func (h *Handler) ManageSpeakerWithdraw(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.SpeakersListPartialInput, *routes.ResponseMeta, error) {
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
	if err := h.Repository.SetSpeakerWithdrawn(ctx, speakerID); err != nil {
		return nil, nil, fmt.Errorf("failed to withdraw speaker: %w", err)
	}
	if err := h.Repository.RecomputeSpeakerOrder(ctx, entry.AgendaPointID); err != nil {
		return nil, nil, fmt.Errorf("failed to recompute speaker order: %w", err)
	}
	partial, err := h.loadSpeakersListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

// loadMeetingSettingsPartial loads settings partial data for a meeting.
func (h *Handler) loadMeetingSettingsPartial(ctx context.Context, slug, meetingIDStr string, meetingID int64) (*templates.MeetingSettingsPartialInput, error) {
	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("failed to load meeting: %w", err)
	}
	attendees, err := h.Repository.ListAttendeesForMeeting(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("failed to load attendees: %w", err)
	}
	return &templates.MeetingSettingsPartialInput{
		CommitteeSlug:                slug,
		IDString:                     meetingIDStr,
		SignupOpen:                   meeting.SignupOpen,
		ProtocolWriterID:             meeting.ProtocolWriterID,
		GenderQuotationEnabled:       meeting.GenderQuotationEnabled,
		FirstSpeakerQuotationEnabled: meeting.FirstSpeakerQuotationEnabled,
		ModeratorID:                  meeting.ModeratorID,
		Attendees:                    buildAttendeeItems(attendees),
	}, nil
}

// ManageToggleSignupOpen flips the signup_open flag on the meeting.
func (h *Handler) ManageToggleSignupOpen(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.MeetingSettingsPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}

	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load meeting: %w", err)
	}

	newValue := !meeting.SignupOpen
	if err := h.Repository.SetMeetingSignupOpen(ctx, meetingID, newValue); err != nil {
		return nil, nil, fmt.Errorf("failed to update signup_open: %w", err)
	}

	partial, err := h.loadMeetingSettingsPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

// ManageSetProtocolWriter assigns or clears the protocol writer for a meeting.
func (h *Handler) ManageSetProtocolWriter(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.MeetingSettingsPartialInput, *routes.ResponseMeta, error) {
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

	if err := h.Repository.SetProtocolWriter(ctx, meetingID, attendeeID); err != nil {
		return nil, nil, fmt.Errorf("failed to set protocol writer: %w", err)
	}

	partial, err := h.loadMeetingSettingsPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}
