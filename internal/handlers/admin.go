package handlers

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/Y4shin/conference-tool/internal/routes"
	"github.com/Y4shin/conference-tool/internal/session"
	"github.com/Y4shin/conference-tool/internal/templates"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// AdminLogin shows the admin login page
func (h *Handler) AdminLogin(ctx context.Context, r *http.Request) (*templates.AdminLoginInput, *routes.ResponseMeta, error) {
	// Check if already logged in
	if session.IsAdminAuthenticated(ctx) {
		meta := routes.NewResponseMeta().WithRedirect(http.StatusSeeOther, "/admin")
		return nil, meta, nil
	}

	return &templates.AdminLoginInput{}, nil, nil
}

// AdminLoginSubmit processes admin login
func (h *Handler) AdminLoginSubmit(ctx context.Context, r *http.Request) (*templates.AdminLoginInput, *routes.ResponseMeta, error) {
	// Parse form
	if err := r.ParseForm(); err != nil {
		return &templates.AdminLoginInput{
			Error: "Invalid form submission",
		}, nil, nil
	}

	adminKey := r.FormValue("admin_key")

	// Verify admin key
	if adminKey != h.AdminKey {
		return &templates.AdminLoginInput{
			Error: "Invalid admin key",
		}, nil, nil
	}

	// Create admin session
	sessionToken := h.AdminSessionManager.CreateAdminSession()
	cookie := h.AdminSessionManager.CreateAdminCookie(sessionToken)

	// Redirect to admin dashboard with cookie
	meta := routes.NewResponseMeta().
		WithCookie(cookie).
		WithRedirect(http.StatusSeeOther, "/admin")

	return nil, meta, nil
}

// AdminLogout logs out the admin
func (h *Handler) AdminLogout(ctx context.Context, r *http.Request) (*templates.AdminLoginInput, *routes.ResponseMeta, error) {
	// Clear admin cookie
	clearCookie := h.AdminSessionManager.ClearAdminCookie()

	// Redirect to admin login
	meta := routes.NewResponseMeta().
		WithCookie(clearCookie).
		WithRedirect(http.StatusSeeOther, "/admin/login")

	return nil, meta, nil
}

// AdminDashboard shows the admin dashboard with committee list
func (h *Handler) AdminDashboard(ctx context.Context, r *http.Request) (*templates.AdminDashboardInput, *routes.ResponseMeta, error) {
	// List all committees
	committees, err := h.Repository.ListAllCommittees(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list committees: %w", err)
	}

	// Convert to template items
	items := make([]templates.CommitteeItem, len(committees))
	for i, c := range committees {
		items[i] = templates.CommitteeItem{
			Slug: c.Slug,
			Name: c.Name,
		}
	}

	return &templates.AdminDashboardInput{
		Committees: items,
	}, nil, nil
}

// AdminCreateCommittee creates a new committee
func (h *Handler) AdminCreateCommittee(ctx context.Context, r *http.Request) (*templates.CommitteeListPartialInput, *routes.ResponseMeta, error) {
	// Parse form
	if err := r.ParseForm(); err != nil {
		committees, _ := h.Repository.ListAllCommittees(ctx)
		items := make([]templates.CommitteeItem, len(committees))
		for i, c := range committees {
			items[i] = templates.CommitteeItem{Slug: c.Slug, Name: c.Name}
		}
		return &templates.CommitteeListPartialInput{
			Committees: items,
			Error:      "Invalid form submission",
		}, nil, nil
	}

	name := strings.TrimSpace(r.FormValue("name"))
	slug := strings.TrimSpace(strings.ToLower(r.FormValue("slug")))

	// Validate inputs
	if name == "" || slug == "" {
		committees, _ := h.Repository.ListAllCommittees(ctx)
		items := make([]templates.CommitteeItem, len(committees))
		for i, c := range committees {
			items[i] = templates.CommitteeItem{Slug: c.Slug, Name: c.Name}
		}
		return &templates.CommitteeListPartialInput{
			Committees: items,
			Error:      "Name and slug are required",
		}, nil, nil
	}

	// Validate slug format (lowercase letters, numbers, hyphens only)
	if !slugRegex.MatchString(slug) {
		committees, _ := h.Repository.ListAllCommittees(ctx)
		items := make([]templates.CommitteeItem, len(committees))
		for i, c := range committees {
			items[i] = templates.CommitteeItem{Slug: c.Slug, Name: c.Name}
		}
		return &templates.CommitteeListPartialInput{
			Committees: items,
			Error:      "Slug must contain only lowercase letters, numbers, and hyphens (no spaces or special characters)",
		}, nil, nil
	}

	// Create committee
	if err := h.Repository.CreateCommitteeWithSlug(ctx, name, slug); err != nil {
		committees, _ := h.Repository.ListAllCommittees(ctx)
		items := make([]templates.CommitteeItem, len(committees))
		for i, c := range committees {
			items[i] = templates.CommitteeItem{Slug: c.Slug, Name: c.Name}
		}
		return &templates.CommitteeListPartialInput{
			Committees: items,
			Error:      fmt.Sprintf("Failed to create committee: %v", err),
		}, nil, nil
	}

	// Success - return updated list with no error
	committees, _ := h.Repository.ListAllCommittees(ctx)
	items := make([]templates.CommitteeItem, len(committees))
	for i, c := range committees {
		items[i] = templates.CommitteeItem{Slug: c.Slug, Name: c.Name}
	}
	return &templates.CommitteeListPartialInput{
		Committees: items,
		Error:      "",
	}, nil, nil
}

// AdminDeleteCommittee deletes a committee
func (h *Handler) AdminDeleteCommittee(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.AdminDashboardInput, *routes.ResponseMeta, error) {
	slug := params.Slug

	// Delete committee
	if err := h.Repository.DeleteCommitteeBySlug(ctx, slug); err != nil {
		return nil, nil, fmt.Errorf("failed to delete committee: %w", err)
	}

	// Redirect back to dashboard
	meta := routes.NewResponseMeta().WithRedirect(http.StatusSeeOther, "/admin")
	return nil, meta, nil
}

// AdminCommitteeUsers shows users in a committee
func (h *Handler) AdminCommitteeUsers(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.AdminCommitteeUsersInput, *routes.ResponseMeta, error) {
	slug := params.Slug

	// Get committee
	committee, err := h.Repository.GetCommitteeBySlug(ctx, slug)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get committee: %w", err)
	}

	// List users in committee
	users, err := h.Repository.ListUsersInCommittee(ctx, slug)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list users: %w", err)
	}

	// Convert to template items
	items := make([]templates.UserItem, len(users))
	for i, u := range users {
		items[i] = templates.UserItem{
			ID:       u.ID,
			IDString: fmt.Sprintf("%d", u.ID),
			Username: u.Username,
			FullName: u.FullName,
			Role:     u.Role,
			Quoted:   u.Quoted,
		}
	}

	return &templates.AdminCommitteeUsersInput{
		CommitteeName: committee.Name,
		CommitteeSlug: committee.Slug,
		Users:         items,
	}, nil, nil
}

// AdminCreateUser creates a new user in a committee
func (h *Handler) AdminCreateUser(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.AdminCommitteeUsersInput, *routes.ResponseMeta, error) {
	slug := params.Slug

	// Parse form
	if err := r.ParseForm(); err != nil {
		return h.AdminCommitteeUsers(ctx, r, params)
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	fullName := strings.TrimSpace(r.FormValue("full_name"))
	role := r.FormValue("role")
	quoted := r.FormValue("quoted") == "true"

	// Validate inputs
	if username == "" || password == "" || fullName == "" || role == "" {
		committee, _ := h.Repository.GetCommitteeBySlug(ctx, slug)
		users, _ := h.Repository.ListUsersInCommittee(ctx, slug)
		items := make([]templates.UserItem, len(users))
		for i, u := range users {
			items[i] = templates.UserItem{ID: u.ID, IDString: fmt.Sprintf("%d", u.ID), Username: u.Username, FullName: u.FullName, Role: u.Role, Quoted: u.Quoted}
		}
		return &templates.AdminCommitteeUsersInput{
			CommitteeName: committee.Name,
			CommitteeSlug: committee.Slug,
			Users:         items,
			Error:         "All fields are required",
		}, nil, nil
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Get committee ID
	committeeID, err := h.Repository.GetCommitteeIDBySlug(ctx, slug)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get committee ID: %w", err)
	}

	// Create user
	if err := h.Repository.CreateUser(ctx, committeeID, username, string(hashedPassword), fullName, quoted, role); err != nil {
		committee, _ := h.Repository.GetCommitteeBySlug(ctx, slug)
		users, _ := h.Repository.ListUsersInCommittee(ctx, slug)
		items := make([]templates.UserItem, len(users))
		for i, u := range users {
			items[i] = templates.UserItem{ID: u.ID, IDString: fmt.Sprintf("%d", u.ID), Username: u.Username, FullName: u.FullName, Role: u.Role, Quoted: u.Quoted}
		}
		return &templates.AdminCommitteeUsersInput{
			CommitteeName: committee.Name,
			CommitteeSlug: committee.Slug,
			Users:         items,
			Error:         fmt.Sprintf("Failed to create user: %v", err),
		}, nil, nil
	}

	// Redirect back to committee users page
	meta := routes.NewResponseMeta().WithRedirect(http.StatusSeeOther, fmt.Sprintf("/admin/committee/%s", slug))
	return nil, meta, nil
}

// AdminDeleteUser deletes a user
func (h *Handler) AdminDeleteUser(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.AdminCommitteeUsersInput, *routes.ResponseMeta, error) {
	slug := params.Slug
	userIDStr := params.UserId

	// Parse user ID
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Delete user
	if err := h.Repository.DeleteUserByID(ctx, userID); err != nil {
		return nil, nil, fmt.Errorf("failed to delete user: %w", err)
	}

	// Redirect back to committee users page
	meta := routes.NewResponseMeta().WithRedirect(http.StatusSeeOther, fmt.Sprintf("/admin/committee/%s", slug))
	return nil, meta, nil
}
