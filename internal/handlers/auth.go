package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/Y4shin/conference-tool/internal/pagination"
	"github.com/Y4shin/conference-tool/internal/routes"
	"github.com/Y4shin/conference-tool/internal/session"
	"github.com/Y4shin/conference-tool/internal/templates"
)

// LoginPage displays the login form
func (h *Handler) LoginPage(ctx context.Context, r *http.Request) (*templates.LoginPageInput, *routes.ResponseMeta, error) {
	// Already authenticated — redirect to home
	if sd, ok := session.GetSession(ctx); ok && !sd.IsExpired() {
		meta := routes.NewResponseMeta().WithRedirect(http.StatusSeeOther, "/home")
		return nil, meta, nil
	}

	return &templates.LoginPageInput{
		PasswordEnabled: h.passwordAuthEnabled(),
		OAuthEnabled:    h.oauthAuthEnabled(),
	}, nil, nil
}

// LoginSubmit processes login credentials
func (h *Handler) LoginSubmit(ctx context.Context, r *http.Request) (*templates.LoginPageInput, *routes.ResponseMeta, error) {
	// Parse form
	if err := r.ParseForm(); err != nil {
		return &templates.LoginPageInput{
			Error:           "Invalid form submission",
			PasswordEnabled: h.passwordAuthEnabled(),
			OAuthEnabled:    h.oauthAuthEnabled(),
		}, nil, nil
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	// Validate inputs
	if username == "" || password == "" {
		return &templates.LoginPageInput{
			Error:           "All fields are required",
			Username:        username,
			PasswordEnabled: h.passwordAuthEnabled(),
			OAuthEnabled:    h.oauthAuthEnabled(),
		}, nil, nil
	}

	// Look up account and verify password
	account, err := h.Repository.GetAccountByUsername(ctx, username)
	if err != nil {
		slog.Warn("login failed: account not found", "username", username)
		return &templates.LoginPageInput{
			Error:           "Invalid credentials",
			Username:        username,
			PasswordEnabled: h.passwordAuthEnabled(),
			OAuthEnabled:    h.oauthAuthEnabled(),
		}, nil, nil
	}

	cred, err := h.Repository.GetPasswordCredential(ctx, account.ID)
	if err != nil {
		slog.Warn("login failed: no password credential", "username", username)
		return &templates.LoginPageInput{
			Error:           "Invalid credentials",
			Username:        username,
			PasswordEnabled: h.passwordAuthEnabled(),
			OAuthEnabled:    h.oauthAuthEnabled(),
		}, nil, nil
	}

	if err := bcrypt.CompareHashAndPassword([]byte(cred.PasswordHash), []byte(password)); err != nil {
		slog.Warn("login failed: wrong password", "username", username)
		return &templates.LoginPageInput{
			Error:           "Invalid credentials",
			Username:        username,
			PasswordEnabled: h.passwordAuthEnabled(),
			OAuthEnabled:    h.oauthAuthEnabled(),
		}, nil, nil
	}

	// Create lean account session
	sessionData := &session.SessionData{
		SessionType: session.SessionTypeAccount,
		AccountID:   &account.ID,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}

	sessionID, err := h.SessionManager.CreateSession(ctx, sessionData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create session: %w", err)
	}
	cookie := h.SessionManager.CreateCookie(sessionID)

	slog.Info("user logged in", "username", username)
	meta := routes.NewResponseMeta().
		WithCookie(cookie).
		WithRedirect(http.StatusSeeOther, "/home")

	return nil, meta, nil
}

// Home shows the user's committees
func (h *Handler) Home(ctx context.Context, r *http.Request) (*templates.HomeInput, *routes.ResponseMeta, error) {
	sd, ok := session.GetSession(ctx)
	if !ok || sd.AccountID == nil {
		return nil, routes.NewResponseMeta().WithRedirect(http.StatusSeeOther, "/"), nil
	}

	committees, err := h.Repository.ListCommitteesByAccountID(ctx, *sd.AccountID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list committees: %w", err)
	}

	items := make([]templates.HomeCommitteeItem, len(committees))
	for i, c := range committees {
		items[i] = templates.HomeCommitteeItem{Slug: c.Slug, Name: c.Name}
	}

	return &templates.HomeInput{
		Committees: items,
		IsAdmin:    sd.IsAdmin,
	}, nil, nil
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

	slog.Info("meeting created", "name", name, "slug", slug)
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

	role := ""
	if cu, ok := session.GetCurrentUser(ctx); ok {
		role = cu.Role
	}

	input := &templates.CommitteePageInput{
		CommitteeName: committee.Name,
		CommitteeSlug: committee.Slug,
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
	} else if role == "member" && committee.CurrentMeetingID != nil {
		meeting, err := h.Repository.GetMeetingByID(ctx, *committee.CurrentMeetingID)
		if err != nil {
			return nil, fmt.Errorf("failed to load active meeting: %w", err)
		}
		input.ActiveMeeting = &templates.MeetingItem{
			ID:          meeting.ID,
			IDString:    strconv.FormatInt(meeting.ID, 10),
			Name:        meeting.Name,
			Description: meeting.Description,
			SignupOpen:  meeting.SignupOpen,
			IsActive:    true,
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

// CommitteeDeleteMeeting deletes a meeting and returns the updated meeting list partial.
func (h *Handler) CommitteeDeleteMeeting(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.MeetingListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}

	if err := h.Repository.DeleteMeeting(ctx, meetingID); err != nil {
		return nil, nil, fmt.Errorf("failed to delete meeting: %w", err)
	}

	slog.Info("meeting deleted", "meeting_id", meetingID, "slug", params.Slug)
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

	committee, err := h.Repository.GetCommitteeBySlug(ctx, params.Slug)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load committee: %w", err)
	}

	var nextActiveMeetingID *int64
	if committee.CurrentMeetingID == nil || *committee.CurrentMeetingID != meetingID {
		nextActiveMeetingID = &meetingID
	}

	if err := h.Repository.SetActiveMeeting(ctx, params.Slug, nextActiveMeetingID); err != nil {
		return nil, nil, fmt.Errorf("failed to set active meeting: %w", err)
	}

	input, err := h.buildMeetingListPartialInput(ctx, params.Slug, "")
	if err != nil {
		return nil, nil, err
	}
	return input, nil, nil
}

// CommitteeToggleMeetingSignupOpen flips signup_open for a meeting and returns the updated meeting list partial.
func (h *Handler) CommitteeToggleMeetingSignupOpen(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.MeetingListPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}

	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load meeting: %w", err)
	}

	if err := h.Repository.SetMeetingSignupOpen(ctx, meetingID, !meeting.SignupOpen); err != nil {
		return nil, nil, fmt.Errorf("failed to update signup_open: %w", err)
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
	slog.Info("user logged out")

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
