package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/Y4shin/conference-tool/internal/pagination"
	"github.com/Y4shin/conference-tool/internal/repository/model"
	"github.com/Y4shin/conference-tool/internal/routes"
	"github.com/Y4shin/conference-tool/internal/session"
	"github.com/Y4shin/conference-tool/internal/templates"
)

// LoginPage displays the login form
func (h *Handler) LoginPage(ctx context.Context, r *http.Request) (*templates.LoginPageInput, *routes.ResponseMeta, error) {
	// Check if already logged in
	if sessionData, ok := session.GetSession(ctx); ok && !sessionData.IsExpired() {
		// Already authenticated - redirect to their committee page
		if sessionData.CommitteeSlug != nil {
			meta := routes.NewResponseMeta().WithRedirect(http.StatusSeeOther,
				fmt.Sprintf("/committee/%s", *sessionData.CommitteeSlug))
			return nil, meta, nil
		}
	}

	return &templates.LoginPageInput{}, nil, nil
}

// LoginSubmit processes login credentials
func (h *Handler) LoginSubmit(ctx context.Context, r *http.Request) (*templates.LoginPageInput, *routes.ResponseMeta, error) {
	// Parse form
	if err := r.ParseForm(); err != nil {
		return &templates.LoginPageInput{
			Error: "Invalid form submission",
		}, nil, nil
	}

	committeeSlug := strings.TrimSpace(r.FormValue("committee"))
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	// Validate inputs
	if committeeSlug == "" || username == "" || password == "" {
		return &templates.LoginPageInput{
			Error:     "All fields are required",
			Committee: committeeSlug,
			Username:  username,
		}, nil, nil
	}

	// Look up user
	user, err := h.Repository.GetUserByCommitteeAndUsername(ctx, committeeSlug, username)
	if err != nil {
		// Generic error message for security (don't reveal if user exists)
		return &templates.LoginPageInput{
			Error:     "Invalid credentials",
			Committee: committeeSlug,
			Username:  username,
		}, nil, nil
	}

	// Verify password using bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return &templates.LoginPageInput{
			Error:     "Invalid credentials",
			Committee: committeeSlug,
			Username:  username,
		}, nil, nil
	}

	// Create session
	sessionData := &session.SessionData{
		SessionType:   session.SessionTypeUser,
		UserID:        &user.ID,
		CommitteeSlug: &committeeSlug,
		Username:      &user.Username,
		Role:          &user.Role,
		Quoted:        &user.Quoted,
		ExpiresAt:     time.Now().Add(24 * time.Hour),
	}

	// Store session and create cookie
	sessionID, err := h.SessionManager.CreateSession(ctx, sessionData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create session: %w", err)
	}
	cookie := h.SessionManager.CreateCookie(sessionID)

	// Return redirect with cookie
	meta := routes.NewResponseMeta().
		WithCookie(cookie).
		WithRedirect(http.StatusSeeOther, fmt.Sprintf("/committee/%s", committeeSlug))

	return nil, meta, nil
}

// CommitteePage shows the committee dashboard
func (h *Handler) CommitteePage(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.CommitteePageInput, *routes.ResponseMeta, error) {
	page, pageSize := parsePaginationParams(r)
	input, err := h.buildCommitteePageInput(ctx, params.Slug, "", page, pageSize)
	if err != nil {
		return nil, nil, err
	}
	return input, nil, nil
}

// buildMeetingListPartialInput builds the MeetingListPartialInput for a committee.
// It always uses page 1 so the updated list is immediately visible after a mutation.
func (h *Handler) buildMeetingListPartialInput(ctx context.Context, slug, formError string) (*templates.MeetingListPartialInput, error) {
	committee, err := h.Repository.GetCommitteeBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to load committee: %w", err)
	}

	total, err := h.Repository.CountMeetingsForCommittee(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to count meetings: %w", err)
	}

	p := pagination.New(1, pagination.DefaultPageSize, total)

	meetings, err := h.Repository.ListMeetingsForCommittee(ctx, slug, p.PageSize, p.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to load meetings: %w", err)
	}

	items := make([]templates.MeetingItem, len(meetings))
	for i, m := range meetings {
		isActive := committee.CurrentMeetingID != nil && *committee.CurrentMeetingID == m.ID
		items[i] = templates.MeetingItem{
			ID:          m.ID,
			IDString:    strconv.FormatInt(m.ID, 10),
			Name:        m.Name,
			Description: m.Description,
			SignupOpen:  m.SignupOpen,
			IsActive:    isActive,
		}
	}

	return &templates.MeetingListPartialInput{
		CommitteeSlug: slug,
		Meetings:      items,
		Pagination:    p,
		FormError:     formError,
	}, nil
}

// CommitteeCreateMeeting handles meeting creation for chairpersons and returns the updated meeting list partial.
func (h *Handler) CommitteeCreateMeeting(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.MeetingListPartialInput, *routes.ResponseMeta, error) {
	slug := params.Slug

	if err := r.ParseForm(); err != nil {
		input, loadErr := h.buildMeetingListPartialInput(ctx, slug, "Invalid form submission")
		if loadErr != nil {
			return nil, nil, loadErr
		}
		return input, nil, nil
	}

	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))
	signupOpen := r.FormValue("signup_open") == "true"

	if name == "" {
		input, loadErr := h.buildMeetingListPartialInput(ctx, slug, "Meeting name is required")
		if loadErr != nil {
			return nil, nil, loadErr
		}
		return input, nil, nil
	}

	committeeID, err := h.Repository.GetCommitteeIDBySlug(ctx, slug)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load committee: %w", err)
	}

	secret, err := generateSecret()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate meeting secret: %w", err)
	}

	if err := h.Repository.CreateMeeting(ctx, committeeID, name, description, secret, signupOpen); err != nil {
		return nil, nil, fmt.Errorf("failed to create meeting: %w", err)
	}

	input, err := h.buildMeetingListPartialInput(ctx, slug, "")
	if err != nil {
		return nil, nil, err
	}
	return input, nil, nil
}

// buildCommitteePageInput loads all data needed to render the committee dashboard
func (h *Handler) buildCommitteePageInput(ctx context.Context, slug, formError string, page, pageSize int) (*templates.CommitteePageInput, error) {
	committee, err := h.Repository.GetCommitteeBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to load committee: %w", err)
	}

	sessionData, _ := session.GetSession(ctx)

	username := ""
	role := ""
	if sessionData.Username != nil {
		username = *sessionData.Username
	}
	if sessionData.Role != nil {
		role = *sessionData.Role
	}

	input := &templates.CommitteePageInput{
		CommitteeName: committee.Name,
		CommitteeSlug: committee.Slug,
		Username:      username,
		Role:          role,
		FormError:     formError,
	}

	if role == "chairperson" {
		total, err := h.Repository.CountMeetingsForCommittee(ctx, slug)
		if err != nil {
			return nil, fmt.Errorf("failed to count meetings: %w", err)
		}

		p := pagination.New(page, pageSize, total)
		input.Pagination = p

		meetings, err := h.Repository.ListMeetingsForCommittee(ctx, slug, p.PageSize, p.Offset)
		if err != nil {
			return nil, fmt.Errorf("failed to load meetings: %w", err)
		}
		for _, m := range meetings {
			isActive := committee.CurrentMeetingID != nil && *committee.CurrentMeetingID == m.ID
			input.Meetings = append(input.Meetings, templates.MeetingItem{
				ID:          m.ID,
				IDString:    strconv.FormatInt(m.ID, 10),
				Name:        m.Name,
				Description: m.Description,
				SignupOpen:  m.SignupOpen,
				IsActive:    isActive,
			})
		}
	}

	return input, nil
}

// CommitteeMeetingView shows the meeting view page
func (h *Handler) CommitteeMeetingView(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.MeetingViewInput, *routes.ResponseMeta, error) {
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

	return &templates.MeetingViewInput{
		CommitteeName: committee.Name,
		CommitteeSlug: committee.Slug,
		MeetingName:   meeting.Name,
		MeetingID:     meeting.ID,
		IDString:      params.MeetingId,
	}, nil, nil
}

// CommitteeMeetingManage shows the meeting management page
func (h *Handler) CommitteeMeetingManage(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.MeetingManageInput, *routes.ResponseMeta, error) {
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

	attendees, err := h.Repository.ListAttendeesForMeeting(ctx, meetingID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load attendees: %w", err)
	}

	agendaPoints, err := h.Repository.ListAgendaPointsForMeeting(ctx, meetingID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load agenda points: %w", err)
	}

	var speakers []*model.SpeakerEntry

	// Compute effective quotation settings for the active agenda point.
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
			return nil, nil, fmt.Errorf("failed to load speakers: %w", err)
		}
		activeAP, err := h.Repository.GetAgendaPointByID(ctx, *meeting.CurrentAgendaPointID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load active agenda point: %w", err)
		}
		apGenderQuotation = activeAP.GenderQuotationEnabled
		apFirstSpeakerQuotation = activeAP.FirstSpeakerQuotationEnabled
		apModeratorID = activeAP.ModeratorID
		if activeAP.GenderQuotationEnabled != nil {
			effectiveGender = *activeAP.GenderQuotationEnabled
		}
		if activeAP.FirstSpeakerQuotationEnabled != nil {
			effectiveFirstSpeaker = *activeAP.FirstSpeakerQuotationEnabled
		}
		if apModeratorID == nil {
			apModeratorID = meeting.ModeratorID
		}
	}

	agendaPointMotions := make([]templates.MotionListPartialInput, 0, len(agendaPoints))
	agendaPointAttachments := make([]templates.AttachmentListPartialInput, 0, len(agendaPoints))
	for _, ap := range agendaPoints {
		motionPartial, err := h.loadMotionListPartial(ctx, params.Slug, params.MeetingId, ap)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load motions: %w", err)
		}
		agendaPointMotions = append(agendaPointMotions, *motionPartial)

		attachmentPartial, err := h.loadAttachmentListPartial(ctx, params.Slug, params.MeetingId, ap)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load attachments: %w", err)
		}
		agendaPointAttachments = append(agendaPointAttachments, *attachmentPartial)
	}

	return &templates.MeetingManageInput{
		CommitteeName:                   committee.Name,
		CommitteeSlug:                   committee.Slug,
		MeetingName:                     meeting.Name,
		MeetingID:                       meeting.ID,
		IDString:                        params.MeetingId,
		Attendees:                       buildAttendeeItems(attendees),
		SignupOpen:                      meeting.SignupOpen,
		ProtocolWriterID:                meeting.ProtocolWriterID,
		GenderQuotationEnabled:          meeting.GenderQuotationEnabled,
		FirstSpeakerQuotationEnabled:    meeting.FirstSpeakerQuotationEnabled,
		ModeratorID:                     meeting.ModeratorID,
		AgendaPoints:                    buildAgendaPointItems(agendaPoints, meeting.CurrentAgendaPointID),
		CurrentAgendaPointID:            meeting.CurrentAgendaPointID,
		AgendaPointIDString:             apIDStr,
		AgendaPointGenderQuotation:      apGenderQuotation,
		AgendaPointFirstSpeakerQuotation: apFirstSpeakerQuotation,
		AgendaPointModeratorID:          apModeratorID,
		EffectiveGenderQuotation:        effectiveGender,
		EffectiveFirstSpeakerQuotation:  effectiveFirstSpeaker,
		Speakers:                        buildSpeakerItems(speakers),
		AgendaPointMotions:              agendaPointMotions,
		AgendaPointAttachments:          agendaPointAttachments,
	}, nil, nil
}

// CommitteeDeleteMeeting deletes a meeting and returns the updated meeting list partial.
func (h *Handler) CommitteeDeleteMeeting(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.MeetingListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}

	if err := h.Repository.DeleteMeeting(ctx, meetingID); err != nil {
		return nil, nil, fmt.Errorf("failed to delete meeting: %w", err)
	}

	input, err := h.buildMeetingListPartialInput(ctx, params.Slug, "")
	if err != nil {
		return nil, nil, err
	}
	return input, nil, nil
}

// CommitteeActivateMeeting sets a meeting as active and returns the updated meeting list partial.
func (h *Handler) CommitteeActivateMeeting(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.MeetingListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}

	if err := h.Repository.SetActiveMeeting(ctx, params.Slug, meetingID); err != nil {
		return nil, nil, fmt.Errorf("failed to set active meeting: %w", err)
	}

	input, err := h.buildMeetingListPartialInput(ctx, params.Slug, "")
	if err != nil {
		return nil, nil, err
	}
	return input, nil, nil
}

func generateSecret() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// LogoutSubmit clears session and redirects to login
func (h *Handler) LogoutSubmit(ctx context.Context, r *http.Request) (*templates.LoginPageInput, *routes.ResponseMeta, error) {
	// Get session ID from cookie and destroy it
	if cookie, err := r.Cookie("session_id"); err == nil {
		_ = h.SessionManager.DestroySession(ctx, cookie.Value)
	}

	// Create cookie that clears the session
	clearCookie := &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	}

	// Return redirect with cleared cookie
	meta := routes.NewResponseMeta().
		WithCookie(clearCookie).
		WithRedirect(http.StatusSeeOther, "/")

	return nil, meta, nil
}
