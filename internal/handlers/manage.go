package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Y4shin/conference-tool/internal/broker"
	"github.com/Y4shin/conference-tool/internal/repository/model"
	"github.com/Y4shin/conference-tool/internal/routes"
	"github.com/Y4shin/conference-tool/internal/session"
	"github.com/Y4shin/conference-tool/internal/templates"
)

// buildAttendeeItems converts model attendees to template items.
func buildAttendeeItems(attendees []*model.Attendee) []templates.AttendeeItem {
	items := make([]templates.AttendeeItem, len(attendees))
	for i, a := range attendees {
		items[i] = templates.AttendeeItem{
			ID:             a.ID,
			IDString:       strconv.FormatInt(a.ID, 10),
			AttendeeNumber: a.AttendeeNumber,
			FullName:       a.FullName,
			IsChair:        a.IsChair,
			IsGuest:        a.UserID == nil,
			Quoted:         a.Quoted,
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
			ParentID: ap.ParentID,
			Position: ap.Position,
			Title:    ap.Title,
			IsActive: isActive,
		}
	}
	return items
}

func flattenAgendaPoints(topLevel, children []*model.AgendaPoint) []*model.AgendaPoint {
	childrenByParent := make(map[int64][]*model.AgendaPoint, len(topLevel))
	for _, child := range children {
		if child.ParentID == nil {
			continue
		}
		parentID := *child.ParentID
		childrenByParent[parentID] = append(childrenByParent[parentID], child)
	}

	flattened := make([]*model.AgendaPoint, 0, len(topLevel)+len(children))
	for _, top := range topLevel {
		flattened = append(flattened, top)
		flattened = append(flattened, childrenByParent[top.ID]...)
	}
	return flattened
}

// buildSpeakerItems converts model speaker entries to template items.
func buildSpeakerItems(entries []*model.SpeakerEntry) []templates.SpeakerItem {
	items := make([]templates.SpeakerItem, len(entries))
	for i, e := range entries {
		var speakingSinceUnix int64
		if e.StartOfSpeech != nil {
			speakingSinceUnix = e.StartOfSpeech.Unix()
		}
		doneDurationLabel := ""
		if e.Status == "DONE" && e.DurationSeconds > 0 {
			doneDurationLabel = formatElapsedLabel(e.DurationSeconds)
		}
		items[i] = templates.SpeakerItem{
			ID:                e.ID,
			IDString:          strconv.FormatInt(e.ID, 10),
			AttendeeID:        e.AttendeeID,
			AttendeeName:      e.AttendeeName,
			Type:              e.Type,
			Status:            e.Status,
			IsWaiting:         e.Status == "WAITING",
			IsSpeaking:        e.Status == "SPEAKING",
			GenderQuoted:      e.GenderQuoted,
			FirstSpeaker:      e.FirstSpeaker,
			Priority:          e.Priority,
			OrderPosition:     e.OrderPosition,
			SpeakingSinceUnix: speakingSinceUnix,
			DoneDurationLabel: doneDurationLabel,
		}
	}
	return items
}

func formatElapsedLabel(totalSeconds int64) string {
	if totalSeconds < 0 {
		totalSeconds = 0
	}
	mins := totalSeconds / 60
	secs := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d", mins, secs)
}

// loadAttendeeListPartial loads the current attendee list for a meeting and
// returns an AttendeeListPartialInput ready for rendering.
func (h *Handler) loadAttendeeListPartial(ctx context.Context, slug, meetingIDStr string, meetingID int64) (*templates.AttendeeListPartialInput, error) {
	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("failed to load meeting: %w", err)
	}
	attendees, err := h.Repository.ListAttendeesForMeeting(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("failed to load attendees: %w", err)
	}

	// showSelfSignup: only for account-session users who don't yet have an attendee record.
	// meeting_access already checked and populated CurrentAttendee if one exists.
	showSelfSignup := false
	if sd, ok := session.GetSession(ctx); ok && !sd.IsExpired() && sd.IsAccountSession() {
		if _, hasAttendee := session.GetCurrentAttendee(ctx); !hasAttendee {
			showSelfSignup = true
		}
	}

	return &templates.AttendeeListPartialInput{
		CommitteeSlug:  slug,
		IDString:       meetingIDStr,
		SignupOpen:     meeting.SignupOpen,
		ShowSelfSignup: showSelfSignup,
		Attendees:      buildAttendeeItems(attendees),
	}, nil
}

func (h *Handler) loadManageAttendeeDependentPartial(ctx context.Context, slug, meetingIDStr string, meetingID int64) (*templates.ManageAttendeeDependentPartialInput, error) {
	settingsPartial, err := h.loadMeetingSettingsPartial(ctx, slug, meetingIDStr, meetingID)
	if err != nil {
		return nil, err
	}
	speakersPartial, err := h.loadSpeakersListPartial(ctx, slug, meetingIDStr, meetingID)
	if err != nil {
		return nil, err
	}
	attendeePartial, err := h.loadAttendeeListPartial(ctx, slug, meetingIDStr, meetingID)
	if err != nil {
		return nil, err
	}
	return &templates.ManageAttendeeDependentPartialInput{
		MeetingSettings: *settingsPartial,
		SpeakersList:    *speakersPartial,
		AttendeeList:    *attendeePartial,
	}, nil
}

func parseClientIDFromForm(r *http.Request) string {
	_ = r.ParseForm()
	return strings.TrimSpace(r.FormValue("client_id"))
}

func parseGenderQuotedFormValue(r *http.Request) bool {
	parseBool := func(raw string) bool {
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "true", "1", "on", "yes":
			return true
		default:
			return false
		}
	}
	if parseBool(r.FormValue("gender_quoted")) {
		return true
	}
	return parseBool(r.FormValue("quoted"))
}

func (h *Handler) publishMeetingAttendeesChanged(meetingID int64, originClientID string) {
	mid := meetingID
	h.Broker.Publish(broker.SSEEvent{
		Event:          "meeting-attendees-changed",
		Data:           []byte("{}"),
		MeetingID:      &mid,
		OriginClientID: originClientID,
	})
}

// ManageAttendeeCreate adds a guest attendee on behalf of the chairperson.
func (h *Handler) ManageAttendeeCreate(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.ManageAttendeeDependentPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}

	if err := r.ParseForm(); err != nil {
		return nil, nil, fmt.Errorf("failed to parse form: %w", err)
	}
	clientID := strings.TrimSpace(r.FormValue("client_id"))

	fullName := strings.TrimSpace(r.FormValue("full_name"))
	quoted := parseGenderQuotedFormValue(r)
	if fullName == "" {
		partial, loadErr := h.loadManageAttendeeDependentPartial(ctx, params.Slug, params.MeetingId, meetingID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		partial.AttendeeList.Error = "Name is required."
		return partial, nil, nil
	}

	secret, err := generateSecret()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate secret: %w", err)
	}

	if _, err := h.Repository.CreateAttendee(ctx, meetingID, nil, fullName, secret, quoted); err != nil {
		return nil, nil, fmt.Errorf("failed to create attendee: %w", err)
	}
	h.publishMeetingAttendeesChanged(meetingID, clientID)

	partial, err := h.loadManageAttendeeDependentPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

// ManageAttendeeSelfSignup signs up the currently logged-in user as attendee.
// Idempotent: if already signed up, returns the current attendee list unchanged.
func (h *Handler) ManageAttendeeSelfSignup(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.ManageAttendeeDependentPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}

	cu, ok := session.GetCurrentUser(ctx)
	if !ok {
		return nil, nil, fmt.Errorf("account session is required")
	}
	userID := cu.UserID
	clientID := parseClientIDFromForm(r)
	changed := false

	_, err = h.Repository.GetAttendeeByUserIDAndMeetingID(ctx, userID, meetingID)
	if err != nil {
		if !strings.Contains(err.Error(), "attendee not found") {
			return nil, nil, fmt.Errorf("failed to load attendee: %w", err)
		}

		user, err := h.Repository.GetUserByID(ctx, userID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load user: %w", err)
		}

		secret, err := generateSecret()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate attendee secret: %w", err)
		}

		if _, err := h.Repository.CreateAttendee(ctx, meetingID, &userID, user.FullName, secret, user.Quoted); err != nil {
			return nil, nil, fmt.Errorf("failed to create attendee: %w", err)
		}
		changed = true
	}
	if changed {
		h.publishMeetingAttendeesChanged(meetingID, clientID)
	}

	partial, err := h.loadManageAttendeeDependentPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

// ManageAttendeeDelete removes an attendee from the meeting.
func (h *Handler) ManageAttendeeDelete(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.ManageAttendeeDependentPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}

	attendeeID, err := strconv.ParseInt(params.AttendeeId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid attendee ID")
	}
	clientID := parseClientIDFromForm(r)

	if err := h.Repository.DeleteAttendee(ctx, attendeeID); err != nil {
		return nil, nil, fmt.Errorf("failed to delete attendee: %w", err)
	}
	h.publishMeetingAttendeesChanged(meetingID, clientID)

	partial, err := h.loadManageAttendeeDependentPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

// ManageAttendeeToggleChair flips the is_chair flag for an attendee.
func (h *Handler) ManageAttendeeToggleChair(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.ManageAttendeeDependentPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}

	attendeeID, err := strconv.ParseInt(params.AttendeeId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid attendee ID")
	}
	clientID := parseClientIDFromForm(r)

	attendee, err := h.Repository.GetAttendeeByID(ctx, attendeeID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load attendee: %w", err)
	}

	if err := h.Repository.SetAttendeeIsChair(ctx, attendeeID, !attendee.IsChair); err != nil {
		return nil, nil, fmt.Errorf("failed to update attendee: %w", err)
	}
	h.publishMeetingAttendeesChanged(meetingID, clientID)

	partial, err := h.loadManageAttendeeDependentPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

// ManageAttendeeToggleQuoted flips the quoted flag for a guest attendee.
func (h *Handler) ManageAttendeeToggleQuoted(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.ManageAttendeeDependentPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}

	attendeeID, err := strconv.ParseInt(params.AttendeeId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid attendee ID")
	}
	clientID := parseClientIDFromForm(r)

	attendee, err := h.Repository.GetAttendeeByID(ctx, attendeeID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load attendee: %w", err)
	}
	if attendee.MeetingID != meetingID {
		return nil, nil, fmt.Errorf("attendee does not belong to this meeting")
	}
	if attendee.UserID != nil {
		return nil, nil, fmt.Errorf("only guest attendees can change gender quoted status")
	}

	if err := h.Repository.SetAttendeeQuoted(ctx, attendeeID, !attendee.Quoted); err != nil {
		return nil, nil, fmt.Errorf("failed to update attendee: %w", err)
	}
	h.publishMeetingAttendeesChanged(meetingID, clientID)

	partial, err := h.loadManageAttendeeDependentPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

// loadAgendaPointListPartial loads the agenda point list for a meeting.
func (h *Handler) loadAgendaPointListPartial(ctx context.Context, slug, meetingIDStr string, meetingID int64) (*templates.AgendaPointListPartialInput, error) {
	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("failed to load meeting: %w", err)
	}
	topLevel, err := h.Repository.ListAgendaPointsForMeeting(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("failed to load agenda points: %w", err)
	}
	children, err := h.Repository.ListSubAgendaPointsForMeeting(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("failed to load sub-agenda points: %w", err)
	}
	aps := flattenAgendaPoints(topLevel, children)
	return &templates.AgendaPointListPartialInput{
		CommitteeSlug:        slug,
		IDString:             meetingIDStr,
		AgendaPoints:         buildAgendaPointItems(aps, meeting.CurrentAgendaPointID),
		ParentOptions:        buildAgendaPointItems(topLevel, nil),
		CurrentAgendaPointID: meeting.CurrentAgendaPointID,
	}, nil
}

// loadSpeakersListPartial loads the speakers list for the active agenda point.
func (h *Handler) loadSpeakersListPartial(ctx context.Context, slug, meetingIDStr string, meetingID int64) (*templates.SpeakersListPartialInput, error) {
	return h.loadSpeakersListPartialWithSearch(ctx, slug, meetingIDStr, meetingID, "")
}

func (h *Handler) loadSpeakersListPartialWithSearch(ctx context.Context, slug, meetingIDStr string, meetingID int64, searchQuery string) (*templates.SpeakersListPartialInput, error) {
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
		SearchQuery:                      searchQuery,
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

	parentIDRaw := strings.TrimSpace(r.FormValue("parent_id"))
	if parentIDRaw == "" {
		if _, err := h.Repository.CreateAgendaPoint(ctx, meetingID, title); err != nil {
			return nil, nil, fmt.Errorf("failed to create agenda point: %w", err)
		}
	} else {
		parentID, err := strconv.ParseInt(parentIDRaw, 10, 64)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid parent agenda point ID")
		}
		if _, err := h.Repository.CreateSubAgendaPoint(ctx, meetingID, parentID, title); err != nil {
			return nil, nil, fmt.Errorf("failed to create sub-agenda point: %w", err)
		}
	}
	partial, err := h.loadAgendaPointListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	h.publishSpeakersUpdated(meetingID)
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
	h.publishSpeakersUpdated(meetingID)
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

	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load meeting: %w", err)
	}

	// If switching away from an agenda point with an active speech, end it first.
	if meeting.CurrentAgendaPointID != nil && *meeting.CurrentAgendaPointID != apID {
		currentSpeakers, err := h.Repository.ListSpeakersForAgendaPoint(ctx, *meeting.CurrentAgendaPointID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load current agenda speakers: %w", err)
		}
		for _, s := range currentSpeakers {
			if s.Status != "SPEAKING" {
				continue
			}
			if err := h.Repository.SetSpeakerDone(ctx, s.ID); err != nil {
				return nil, nil, fmt.Errorf("failed to end ongoing speech: %w", err)
			}
		}
		if err := h.Repository.RecomputeSpeakerOrder(ctx, *meeting.CurrentAgendaPointID); err != nil {
			return nil, nil, fmt.Errorf("failed to recompute previous agenda speaker order: %w", err)
		}
	}

	if err := h.Repository.SetCurrentAgendaPoint(ctx, meetingID, &apID); err != nil {
		return nil, nil, fmt.Errorf("failed to activate agenda point: %w", err)
	}
	partial, err := h.loadAgendaPointListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	h.publishSpeakersUpdated(meetingID)
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

	// Enforce at most one non-DONE entry per attendee+type on the active agenda point.
	existingEntries, err := h.Repository.ListSpeakersForAgendaPoint(ctx, apID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load speakers: %w", err)
	}
	for _, e := range existingEntries {
		if e.AttendeeID == attendeeID && e.Type == speakerType && e.Status != "DONE" {
			partial, loadErr := h.loadSpeakersListPartial(ctx, params.Slug, params.MeetingId, meetingID)
			if loadErr != nil {
				return nil, nil, loadErr
			}
			partial.Error = fmt.Sprintf("Attendee already has a non-done %s entry.", speakerType)
			return partial, nil, nil
		}
	}

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
	firstSpeaker := speakerType == "regular" && !hasSpoken

	if _, err := h.Repository.AddSpeaker(ctx, apID, attendeeID, speakerType, genderQuoted, firstSpeaker); err != nil {
		return nil, nil, fmt.Errorf("failed to add speaker: %w", err)
	}
	if err := h.Repository.RecomputeSpeakerOrder(ctx, apID); err != nil {
		return nil, nil, fmt.Errorf("failed to recompute speaker order: %w", err)
	}
	h.publishSpeakersUpdated(meetingID)
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
	h.publishSpeakersUpdated(meetingID)
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
	h.publishSpeakersUpdated(meetingID)
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
	h.publishSpeakersUpdated(meetingID)
	partial, err := h.loadSpeakersListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

// ManageSpeakerWithdraw removes a speaker entry (legacy route name).
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
		return nil, nil, fmt.Errorf("failed to remove speaker: %w", err)
	}
	if err := h.Repository.RecomputeSpeakerOrder(ctx, entry.AgendaPointID); err != nil {
		return nil, nil, fmt.Errorf("failed to recompute speaker order: %w", err)
	}
	h.publishSpeakersUpdated(meetingID)
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
		GenderQuotationEnabled:       meeting.GenderQuotationEnabled,
		FirstSpeakerQuotationEnabled: meeting.FirstSpeakerQuotationEnabled,
		ModeratorID:                  meeting.ModeratorID,
		Attendees:                    buildAttendeeItems(attendees),
	}, nil
}

// ManageToggleSignupOpen flips the signup_open flag on the meeting.
func (h *Handler) ManageToggleSignupOpen(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.ManageAttendeeDependentPartialInput, *routes.ResponseMeta, error) {
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

	partial, err := h.loadManageAttendeeDependentPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

// ManageMeetingSettingsPartial refreshes the meeting settings partial.
func (h *Handler) ManageMeetingSettingsPartial(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.MeetingSettingsPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	partial, err := h.loadMeetingSettingsPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

// ManageSpeakersListPartial refreshes the speakers list partial.
func (h *Handler) ManageSpeakersListPartial(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.SpeakersListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	partial, err := h.loadSpeakersListPartial(ctx, params.Slug, params.MeetingId, meetingID)
	return partial, nil, err
}

// ManageSpeakerAddCandidates returns the filtered attendee candidates list for the add-speaker modal.
func (h *Handler) ManageSpeakerAddCandidates(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.SpeakersListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	searchQuery := strings.TrimSpace(r.URL.Query().Get("q"))
	partial, err := h.loadSpeakersListPartialWithSearch(ctx, params.Slug, params.MeetingId, meetingID, searchQuery)
	return partial, nil, err
}
