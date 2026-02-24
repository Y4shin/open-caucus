package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Y4shin/conference-tool/internal/repository/model"
	"github.com/Y4shin/conference-tool/internal/routes"
	"github.com/Y4shin/conference-tool/internal/session"
	"github.com/Y4shin/conference-tool/internal/templates"
)

// MeetingJoinPage renders the join page for a meeting.
// Registered users (user session) see a one-click sign-up button.
// Guests (no session) see a name-entry form when signup_open is true, or a closed message.
func (h *Handler) MeetingJoinPage(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.MeetingJoinInput, *routes.ResponseMeta, error) {
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

	sessionData, hasSession := session.GetSession(ctx)
	isLoggedIn := hasSession && sessionData.IsAccountSession() && !sessionData.IsExpired()

	alreadySignedUp := false
	if isLoggedIn && sessionData.AccountID != nil {
		membership, err := h.Repository.GetUserMembershipByAccountIDAndSlug(ctx, *sessionData.AccountID, params.Slug)
		if err == nil {
			_, err := h.Repository.GetAttendeeByUserIDAndMeetingID(ctx, membership.ID, meetingID)
			if err == nil {
				alreadySignedUp = true
			}
		}
	}

	prefilledMeetingSecret := strings.TrimSpace(r.URL.Query().Get("meeting_secret"))

	return &templates.MeetingJoinInput{
		CommitteeName:          committee.Name,
		CommitteeSlug:          committee.Slug,
		MeetingName:            meeting.Name,
		MeetingID:              meeting.ID,
		IDString:               params.MeetingId,
		SignupOpen:             meeting.SignupOpen,
		IsLoggedIn:             isLoggedIn,
		AlreadySignedUp:        alreadySignedUp,
		PrefilledMeetingSecret: prefilledMeetingSecret,
	}, nil, nil
}

// MeetingJoinSubmit handles signup for a registered committee member.
// Creates an attendee row and redirects to the meeting live view.
// The account session persists; meeting_access middleware finds the attendee on the next request.
// Idempotent: if already signed up, redirects to live view directly.
func (h *Handler) MeetingJoinSubmit(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.MeetingJoinInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}

	cu, ok := session.GetCurrentUser(ctx)
	if !ok {
		return nil, routes.NewResponseMeta().WithRedirect(http.StatusSeeOther, "/"), nil
	}

	// Check for existing attendee row (idempotent signup)
	_, err = h.Repository.GetAttendeeByUserIDAndMeetingID(ctx, cu.UserID, meetingID)
	if err != nil {
		// No existing row — create one
		user, err := h.Repository.GetUserByID(ctx, cu.UserID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load user: %w", err)
		}

		secret, err := generateSecret()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate attendee secret: %w", err)
		}

		if _, err = h.Repository.CreateAttendee(ctx, meetingID, &cu.UserID, user.FullName, secret, user.Quoted); err != nil {
			return nil, nil, fmt.Errorf("failed to create attendee: %w", err)
		}
	}

	// Redirect to the meeting live page; meeting_access will find the new attendee record
	meta := routes.NewResponseMeta().WithRedirect(
		http.StatusSeeOther,
		fmt.Sprintf("/committee/%s/meeting/%s", params.Slug, params.MeetingId),
	)
	return nil, meta, nil
}

// MeetingGuestSignup handles self-registration for guests.
// Only accepted when signup_open is true on the meeting.
func (h *Handler) MeetingGuestSignup(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.MeetingGuestSuccessInput, *routes.ResponseMeta, error) {
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

	if !meeting.SignupOpen {
		return &templates.MeetingGuestSuccessInput{
			CommitteeName: committee.Name,
			CommitteeSlug: committee.Slug,
			MeetingName:   meeting.Name,
			IDString:      params.MeetingId,
			Error:         "Guest signup is currently closed.",
		}, nil, nil
	}

	if err := r.ParseForm(); err != nil {
		return nil, nil, fmt.Errorf("failed to parse form: %w", err)
	}

	fullName := strings.TrimSpace(r.FormValue("full_name"))
	quoted := parseGenderQuotedFormValue(r)
	if fullName == "" {
		return &templates.MeetingGuestSuccessInput{
			CommitteeName: committee.Name,
			CommitteeSlug: committee.Slug,
			MeetingName:   meeting.Name,
			IDString:      params.MeetingId,
			Error:         "Name is required.",
		}, nil, nil
	}

	meetingSecret := strings.TrimSpace(r.FormValue("meeting_secret"))
	if meetingSecret == "" {
		return &templates.MeetingGuestSuccessInput{
			CommitteeName: committee.Name,
			CommitteeSlug: committee.Slug,
			MeetingName:   meeting.Name,
			IDString:      params.MeetingId,
			Error:         "Meeting secret is required.",
		}, nil, nil
	}
	if meetingSecret != meeting.Secret {
		return &templates.MeetingGuestSuccessInput{
			CommitteeName: committee.Name,
			CommitteeSlug: committee.Slug,
			MeetingName:   meeting.Name,
			IDString:      params.MeetingId,
			Error:         "Invalid meeting secret.",
		}, nil, nil
	}

	secret, err := generateSecret()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate attendee secret: %w", err)
	}

	attendee, err := h.Repository.CreateAttendee(ctx, meetingID, nil, fullName, secret, quoted)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create attendee: %w", err)
	}
	h.publishMeetingAttendeesChanged(meetingID, "")

	meta, err := h.createAttendeeSession(ctx, attendee, params.Slug, params.MeetingId)
	if err != nil {
		return nil, nil, err
	}
	return nil, meta, nil
}

// AttendeeLoginPage renders the access-code entry form for guests.
func (h *Handler) AttendeeLoginPage(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.AttendeeLoginInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}

	// Redirect to the meeting live view if already holding a valid attendee session for this meeting
	// (sessionMiddleware populates CurrentAttendee for guest sessions)
	if ca, ok := session.GetCurrentAttendee(ctx); ok && ca.MeetingID == meetingID {
		meta := routes.NewResponseMeta().WithRedirect(http.StatusSeeOther,
			fmt.Sprintf("/committee/%s/meeting/%s", params.Slug, params.MeetingId))
		return nil, meta, nil
	}

	committee, err := h.Repository.GetCommitteeBySlug(ctx, params.Slug)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load committee: %w", err)
	}

	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load meeting: %w", err)
	}

	// Recovery links may prefill a secret in query params. If valid, log the
	// attendee in directly and skip the manual secret form.
	prefilledSecret := strings.TrimSpace(r.URL.Query().Get("secret"))
	if prefilledSecret != "" {
		attendee, err := h.Repository.GetAttendeeByMeetingIDAndSecret(ctx, meetingID, prefilledSecret)
		if err == nil {
			meta, err := h.createAttendeeSession(ctx, attendee, params.Slug, params.MeetingId)
			if err != nil {
				return nil, nil, err
			}
			return nil, meta, nil
		}
	}

	return &templates.AttendeeLoginInput{
		CommitteeName: committee.Name,
		CommitteeSlug: committee.Slug,
		MeetingName:   meeting.Name,
		IDString:      params.MeetingId,
	}, nil, nil
}

// AttendeeLoginSubmit validates the access code and creates an attendee session.
func (h *Handler) AttendeeLoginSubmit(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.AttendeeLoginInput, *routes.ResponseMeta, error) {
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

	if err := r.ParseForm(); err != nil {
		return nil, nil, fmt.Errorf("failed to parse form: %w", err)
	}

	secret := strings.TrimSpace(r.FormValue("secret"))
	if secret == "" {
		return &templates.AttendeeLoginInput{
			CommitteeName: committee.Name,
			CommitteeSlug: committee.Slug,
			MeetingName:   meeting.Name,
			IDString:      params.MeetingId,
			Error:         "Access code is required.",
		}, nil, nil
	}

	attendee, err := h.Repository.GetAttendeeByMeetingIDAndSecret(ctx, meetingID, secret)
	if err != nil {
		return &templates.AttendeeLoginInput{
			CommitteeName: committee.Name,
			CommitteeSlug: committee.Slug,
			MeetingName:   meeting.Name,
			IDString:      params.MeetingId,
			Error:         "Invalid access code.",
		}, nil, nil
	}

	meta, err := h.createAttendeeSession(ctx, attendee, params.Slug, params.MeetingId)
	if err != nil {
		return nil, nil, err
	}
	return nil, meta, nil
}

// MeetingLivePage renders the attendee live view of a meeting.
// Access is validated by meeting_access middleware which also populates CurrentAttendee.
func (h *Handler) MeetingLivePage(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.MeetingLiveInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}

	// meeting_access middleware populates CurrentAttendee when an attendee record exists.
	// For account sessions without a record, redirect to the join page.
	ca, ok := session.GetCurrentAttendee(ctx)
	if !ok {
		return nil, routes.NewResponseMeta().WithRedirect(
			http.StatusSeeOther,
			fmt.Sprintf("/committee/%s/meeting/%s/join", params.Slug, params.MeetingId),
		), nil
	}

	attendeeID := ca.AttendeeID
	isChair := ca.IsChair
	canManage := isChair
	canModerate := isChair
	if cu, cuOK := session.GetCurrentUser(ctx); cuOK && cu.Role == "chairperson" {
		canManage = true
		canModerate = true
	}

	committee, err := h.Repository.GetCommitteeBySlug(ctx, params.Slug)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load committee: %w", err)
	}

	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load meeting: %w", err)
	}
	if meeting.ModeratorID != nil && *meeting.ModeratorID == attendeeID {
		canModerate = true
	}

	topLevelAgendaPoints, err := h.Repository.ListAgendaPointsForMeeting(ctx, meetingID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load agenda points: %w", err)
	}
	subAgendaPoints, err := h.Repository.ListSubAgendaPointsForMeeting(ctx, meetingID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load sub-agenda points: %w", err)
	}
	agendaPoints := flattenAgendaPoints(topLevelAgendaPoints, subAgendaPoints)

	speakersInput, err := h.loadAttendeeSpeakersPartial(ctx, meeting.ID, attendeeID)
	if err != nil {
		return nil, nil, err
	}
	speakersInput.CommitteeSlug = params.Slug
	speakersInput.IDString = params.MeetingId

	return &templates.MeetingLiveInput{
		CommitteeName: committee.Name,
		CommitteeSlug: committee.Slug,
		MeetingName:   meeting.Name,
		IDString:      params.MeetingId,
		IsChair:       isChair,
		CanManage:     canManage,
		CanModerate:   canModerate,
		AgendaPoints:  buildAgendaPointItems(agendaPoints, meeting.CurrentAgendaPointID),
		Speakers:      *speakersInput,
		CurrentDoc:    speakersInput.CurrentDoc,
	}, nil, nil
}

// MeetingLiveLegacyRedirect keeps /live working as a compatibility alias.
func (h *Handler) MeetingLiveLegacyRedirect(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.MeetingLiveInput, *routes.ResponseMeta, error) {
	meta := routes.NewResponseMeta().WithRedirect(
		http.StatusSeeOther,
		fmt.Sprintf("/committee/%s/meeting/%s", params.Slug, params.MeetingId),
	)
	return nil, meta, nil
}

// createAttendeeSession creates a new guest session and returns a ResponseMeta
// that sets the session cookie and redirects to the meeting live view.
func (h *Handler) createAttendeeSession(ctx context.Context, attendee *model.Attendee, slug, meetingIDStr string) (*routes.ResponseMeta, error) {
	sd := &session.SessionData{
		SessionType: session.SessionTypeGuest,
		AttendeeID:  &attendee.ID,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}

	sessionID, err := h.SessionManager.CreateSession(ctx, sd)
	if err != nil {
		return nil, fmt.Errorf("failed to create guest session: %w", err)
	}

	cookie := h.SessionManager.CreateCookie(sessionID)
	meta := routes.NewResponseMeta().
		WithCookie(cookie).
		WithRedirect(http.StatusSeeOther, fmt.Sprintf("/committee/%s/meeting/%s", slug, meetingIDStr))
	return meta, nil
}
