package handlers

import (
	"context"
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
func (h *Handler) LoginPage(ctx context.Context, w http.ResponseWriter, r *http.Request) (*templates.LoginPageInput, *routes.ResponseMeta, error) {
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
func (h *Handler) LoginSubmit(ctx context.Context, w http.ResponseWriter, r *http.Request) (*templates.LoginPageInput, *routes.ResponseMeta, error) {
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
func (h *Handler) CommitteePage(ctx context.Context, w http.ResponseWriter, r *http.Request, params routes.RouteParams) (*templates.CommitteePageInput, *routes.ResponseMeta, error) {
	// Note: Auth middleware guarantees session exists
	// Note: committee_access middleware guarantees slug matches session

	slug := params.Slug

	// Load committee data
	committee, err := h.Repository.GetCommitteeBySlug(ctx, slug)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load committee: %w", err)
	}

	// Get session for user info
	sessionData, _ := session.GetSession(ctx)

	username := ""
	role := ""
	if sessionData.Username != nil {
		username = *sessionData.Username
	}
	if sessionData.Role != nil {
		role = *sessionData.Role
	}

	return &templates.CommitteePageInput{
		CommitteeName: committee.Name,
		CommitteeSlug: committee.Slug,
		Username:      username,
		Role:          role,
	}, nil, nil
}

// LogoutSubmit clears session and redirects to login
func (h *Handler) LogoutSubmit(ctx context.Context, w http.ResponseWriter, r *http.Request) (*templates.LoginPageInput, *routes.ResponseMeta, error) {
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
