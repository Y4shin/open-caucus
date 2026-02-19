package handlers

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/Y4shin/conference-tool/internal/pagination"
	"github.com/Y4shin/conference-tool/internal/routes"
	"github.com/Y4shin/conference-tool/internal/session"
	"github.com/Y4shin/conference-tool/internal/templates"

	playwright "github.com/playwright-community/playwright-go"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// AdminLogin shows the admin login page
func (h *Handler) AdminLogin(ctx context.Context, r *http.Request) (*templates.AdminLoginInput, *routes.ResponseMeta, error) {
	playwright.Run()
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
	page, pageSize := parsePaginationParams(r)

	total, err := h.Repository.CountAllCommittees(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to count committees: %w", err)
	}

	p := pagination.New(page, pageSize, total)

	committees, err := h.Repository.ListAllCommittees(ctx, p.PageSize, p.Offset)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list committees: %w", err)
	}

	items := make([]templates.CommitteeItem, len(committees))
	for i, c := range committees {
		items[i] = templates.CommitteeItem{Slug: c.Slug, Name: c.Name}
	}

	return &templates.AdminDashboardInput{
		Committees: items,
		Pagination: p,
	}, nil, nil
}

// listAllCommitteesForPartial fetches all committees for the HTMX partial (no pagination).
func (h *Handler) listAllCommitteesForPartial(ctx context.Context) ([]templates.CommitteeItem, error) {
	// Use a high limit — the partial doesn't support pagination.
	committees, err := h.Repository.ListAllCommittees(ctx, 1000, 0)
	if err != nil {
		return nil, err
	}
	items := make([]templates.CommitteeItem, len(committees))
	for i, c := range committees {
		items[i] = templates.CommitteeItem{Slug: c.Slug, Name: c.Name}
	}
	return items, nil
}

// AdminCreateCommittee creates a new committee
func (h *Handler) AdminCreateCommittee(ctx context.Context, r *http.Request) (*templates.CommitteeListPartialInput, *routes.ResponseMeta, error) {
	// Parse form
	if err := r.ParseForm(); err != nil {
		items, _ := h.listAllCommitteesForPartial(ctx)
		return &templates.CommitteeListPartialInput{Committees: items, Error: "Invalid form submission"}, nil, nil
	}

	name := strings.TrimSpace(r.FormValue("name"))
	slug := strings.TrimSpace(strings.ToLower(r.FormValue("slug")))

	if name == "" || slug == "" {
		items, _ := h.listAllCommitteesForPartial(ctx)
		return &templates.CommitteeListPartialInput{Committees: items, Error: "Name and slug are required"}, nil, nil
	}

	if !slugRegex.MatchString(slug) {
		items, _ := h.listAllCommitteesForPartial(ctx)
		return &templates.CommitteeListPartialInput{
			Committees: items,
			Error:      "Slug must contain only lowercase letters, numbers, and hyphens (no spaces or special characters)",
		}, nil, nil
	}

	if err := h.Repository.CreateCommitteeWithSlug(ctx, name, slug); err != nil {
		items, _ := h.listAllCommitteesForPartial(ctx)
		return &templates.CommitteeListPartialInput{
			Committees: items,
			Error:      fmt.Sprintf("Failed to create committee: %v", err),
		}, nil, nil
	}

	items, _ := h.listAllCommitteesForPartial(ctx)
	return &templates.CommitteeListPartialInput{Committees: items}, nil, nil
}

// AdminDeleteCommittee deletes a committee and returns the updated committee list partial.
func (h *Handler) AdminDeleteCommittee(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.CommitteeListPartialInput, *routes.ResponseMeta, error) {
	slug := params.Slug

	if err := h.Repository.DeleteCommitteeBySlug(ctx, slug); err != nil {
		items, _ := h.listAllCommitteesForPartial(ctx)
		return &templates.CommitteeListPartialInput{
			Committees: items,
			Error:      fmt.Sprintf("Failed to delete committee: %v", err),
		}, nil, nil
	}

	items, err := h.listAllCommitteesForPartial(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list committees: %w", err)
	}
	return &templates.CommitteeListPartialInput{Committees: items}, nil, nil
}

// AdminCommitteeUsers shows users in a committee
func (h *Handler) AdminCommitteeUsers(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.AdminCommitteeUsersInput, *routes.ResponseMeta, error) {
	slug := params.Slug
	page, pageSize := parsePaginationParams(r)

	committee, err := h.Repository.GetCommitteeBySlug(ctx, slug)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get committee: %w", err)
	}

	total, err := h.Repository.CountUsersInCommittee(ctx, slug)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to count users: %w", err)
	}

	p := pagination.New(page, pageSize, total)

	users, err := h.Repository.ListUsersInCommittee(ctx, slug, p.PageSize, p.Offset)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list users: %w", err)
	}

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
		Pagination:    p,
	}, nil, nil
}

// buildUserListPartialInput builds the UserListPartialInput for a committee.
// It always uses page 1 so the updated list is immediately visible after a mutation.
func (h *Handler) buildUserListPartialInput(ctx context.Context, slug, formError string) (*templates.UserListPartialInput, error) {
	total, err := h.Repository.CountUsersInCommittee(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	p := pagination.New(1, pagination.DefaultPageSize, total)

	users, err := h.Repository.ListUsersInCommittee(ctx, slug, p.PageSize, p.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	items := make([]templates.UserItem, len(users))
	for i, u := range users {
		items[i] = templates.UserItem{ID: u.ID, IDString: fmt.Sprintf("%d", u.ID), Username: u.Username, FullName: u.FullName, Role: u.Role, Quoted: u.Quoted}
	}

	return &templates.UserListPartialInput{
		CommitteeSlug: slug,
		Users:         items,
		Pagination:    p,
		Error:         formError,
	}, nil
}

// AdminCreateUser creates a new user in a committee and returns the updated user list partial.
func (h *Handler) AdminCreateUser(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.UserListPartialInput, *routes.ResponseMeta, error) {
	slug := params.Slug

	if err := r.ParseForm(); err != nil {
		input, err := h.buildUserListPartialInput(ctx, slug, "Invalid form submission")
		if err != nil {
			return nil, nil, err
		}
		return input, nil, nil
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	fullName := strings.TrimSpace(r.FormValue("full_name"))
	role := r.FormValue("role")
	quoted := r.FormValue("quoted") == "true"

	if username == "" || password == "" || fullName == "" || role == "" {
		input, err := h.buildUserListPartialInput(ctx, slug, "All fields are required")
		if err != nil {
			return nil, nil, err
		}
		return input, nil, nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hash password: %w", err)
	}

	committeeID, err := h.Repository.GetCommitteeIDBySlug(ctx, slug)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get committee ID: %w", err)
	}

	if err := h.Repository.CreateUser(ctx, committeeID, username, string(hashedPassword), fullName, quoted, role); err != nil {
		input, loadErr := h.buildUserListPartialInput(ctx, slug, fmt.Sprintf("Failed to create user: %v", err))
		if loadErr != nil {
			return nil, nil, loadErr
		}
		return input, nil, nil
	}

	input, err := h.buildUserListPartialInput(ctx, slug, "")
	if err != nil {
		return nil, nil, err
	}
	return input, nil, nil
}

// AdminDeleteUser deletes a user and returns the updated user list partial.
func (h *Handler) AdminDeleteUser(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.UserListPartialInput, *routes.ResponseMeta, error) {
	slug := params.Slug
	userIDStr := params.UserId

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid user ID: %w", err)
	}

	if err := h.Repository.DeleteUserByID(ctx, userID); err != nil {
		return nil, nil, fmt.Errorf("failed to delete user: %w", err)
	}

	input, err := h.buildUserListPartialInput(ctx, slug, "")
	if err != nil {
		return nil, nil, err
	}
	return input, nil, nil
}
