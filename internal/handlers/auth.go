package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

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
	input, err := h.buildCommitteePageInput(ctx, params.Slug, "")
	if err != nil {
		return nil, nil, err
	}
	return input, nil, nil
}

// CommitteeCreateMeeting handles meeting creation for chairpersons
func (h *Handler) CommitteeCreateMeeting(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.CommitteePageInput, *routes.ResponseMeta, error) {
	slug := params.Slug

	if err := r.ParseForm(); err != nil {
		input, loadErr := h.buildCommitteePageInput(ctx, slug, "Invalid form submission")
		if loadErr != nil {
			return nil, nil, loadErr
		}
		return input, nil, nil
	}

	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))
	signupOpen := r.FormValue("signup_open") == "true"

	if name == "" {
		input, loadErr := h.buildCommitteePageInput(ctx, slug, "Meeting name is required")
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

	meta := routes.NewResponseMeta().WithRedirect(http.StatusSeeOther, fmt.Sprintf("/committee/%s", slug))
	return nil, meta, nil
}

// buildCommitteePageInput loads all data needed to render the committee dashboard
func (h *Handler) buildCommitteePageInput(ctx context.Context, slug, formError string) (*templates.CommitteePageInput, error) {
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
		meetings, err := h.Repository.ListMeetingsForCommittee(ctx, slug)
		if err != nil {
			return nil, fmt.Errorf("failed to load meetings: %w", err)
		}
		for _, m := range meetings {
			input.Meetings = append(input.Meetings, templates.MeetingItem{
				ID:          m.ID,
				Name:        m.Name,
				Description: m.Description,
				SignupOpen:  m.SignupOpen,
			})
		}
	}

	return input, nil
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
