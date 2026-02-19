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
	isLoggedIn := hasSession && sessionData.IsUserSession() && !sessionData.IsExpired()

	alreadySignedUp := false
	if isLoggedIn && sessionData.UserID != nil {
		_, err := h.Repository.GetAttendeeByUserIDAndMeetingID(ctx, *sessionData.UserID, meetingID)
		if err == nil {
			alreadySignedUp = true
		}
	}

	return &templates.MeetingJoinInput{
		CommitteeName:   committee.Name,
		CommitteeSlug:   committee.Slug,
		MeetingName:     meeting.Name,
		MeetingID:       meeting.ID,
		IDString:        params.MeetingId,
		SignupOpen:      meeting.SignupOpen,
		IsLoggedIn:      isLoggedIn,
		AlreadySignedUp: alreadySignedUp,
	}, nil, nil
}

// MeetingJoinSubmit handles signup for a registered committee member.
// Creates an attendee row and an attendee session, then redirects to /live.
// Idempotent: if already signed up, creates a new attendee session and redirects.
func (h *Handler) MeetingJoinSubmit(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.MeetingJoinInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}

	sessionData, _ := session.GetSession(ctx)
	userID := sessionData.UserID

	// Check for existing attendee row (idempotent signup)
	attendee, err := h.Repository.GetAttendeeByUserIDAndMeetingID(ctx, *userID, meetingID)
	if err != nil {
		// No existing row — create one
		user, err := h.Repository.GetUserByID(ctx, *userID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load user: %w", err)
		}

		secret, err := generateSecret()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate attendee secret: %w", err)
		}

		attendee, err = h.Repository.CreateAttendee(ctx, meetingID, userID, user.FullName, secret)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create attendee: %w", err)
		}
	}

	meta, err := h.createAttendeeSession(ctx, attendee, params.Slug, params.MeetingId)
	if err != nil {
		return nil, nil, err
	}
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
	if fullName == "" {
		return &templates.MeetingGuestSuccessInput{
			CommitteeName: committee.Name,
			CommitteeSlug: committee.Slug,
			MeetingName:   meeting.Name,
			IDString:      params.MeetingId,
			Error:         "Name is required.",
		}, nil, nil
	}

	secret, err := generateSecret()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate attendee secret: %w", err)
	}

	if _, err := h.Repository.CreateAttendee(ctx, meetingID, nil, fullName, secret); err != nil {
		return nil, nil, fmt.Errorf("failed to create attendee: %w", err)
	}

	return &templates.MeetingGuestSuccessInput{
		CommitteeName:  committee.Name,
		CommitteeSlug:  committee.Slug,
		MeetingName:    meeting.Name,
		IDString:       params.MeetingId,
		AttendeeSecret: secret,
	}, nil, nil
}

// AttendeeLoginPage renders the access-code entry form for guests.
func (h *Handler) AttendeeLoginPage(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.AttendeeLoginInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}

	// Redirect to /live if already holding a valid attendee session for this meeting
	if sd, ok := session.GetSession(ctx); ok && sd.IsAttendeeSession() && !sd.IsExpired() {
		if sd.MeetingID != nil && *sd.MeetingID == meetingID {
			meta := routes.NewResponseMeta().WithRedirect(http.StatusSeeOther,
				fmt.Sprintf("/committee/%s/meeting/%s/live", params.Slug, params.MeetingId))
			return nil, meta, nil
		}
	}

	committee, err := h.Repository.GetCommitteeBySlug(ctx, params.Slug)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load committee: %w", err)
	}

	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load meeting: %w", err)
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
func (h *Handler) MeetingLivePage(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.MeetingLiveInput, *routes.ResponseMeta, error) {
	sd, _ := session.GetSession(ctx)

	committee, err := h.Repository.GetCommitteeBySlug(ctx, params.Slug)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load committee: %w", err)
	}

	meeting, err := h.Repository.GetMeetingByID(ctx, *sd.MeetingID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load meeting: %w", err)
	}

	fullName := ""
	if sd.FullName != nil {
		fullName = *sd.FullName
	}
	isChair := sd.IsChair != nil && *sd.IsChair

	return &templates.MeetingLiveInput{
		CommitteeName: committee.Name,
		CommitteeSlug: committee.Slug,
		MeetingName:   meeting.Name,
		IDString:      params.MeetingId,
		FullName:      fullName,
		IsChair:       isChair,
	}, nil, nil
}

// createAttendeeSession creates a new attendee session and returns a ResponseMeta
// that sets the session cookie and redirects to /live.
func (h *Handler) createAttendeeSession(ctx context.Context, attendee *model.Attendee, slug, meetingIDStr string) (*routes.ResponseMeta, error) {
	sd := &session.SessionData{
		SessionType: session.SessionTypeAttendee,
		AttendeeID:  &attendee.ID,
		MeetingID:   &attendee.MeetingID,
		FullName:    &attendee.FullName,
		IsChair:     &attendee.IsChair,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}

	sessionID, err := h.SessionManager.CreateSession(ctx, sd)
	if err != nil {
		return nil, fmt.Errorf("failed to create attendee session: %w", err)
	}

	cookie := h.SessionManager.CreateCookie(sessionID)
	meta := routes.NewResponseMeta().
		WithCookie(cookie).
		WithRedirect(http.StatusSeeOther, fmt.Sprintf("/committee/%s/meeting/%s/live", slug, meetingIDStr))
	return meta, nil
}
