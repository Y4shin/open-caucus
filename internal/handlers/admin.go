package handlers

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
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

var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// AdminLogin shows the admin login page
func (h *Handler) AdminLogin(ctx context.Context, r *http.Request) (*templates.AdminLoginInput, *routes.ResponseMeta, error) {
	// Redirect if already logged in as admin
	sd, ok := session.GetSession(ctx)
	if ok && !sd.IsExpired() && sd.IsAdmin {
		meta := routes.NewResponseMeta().WithRedirect(http.StatusSeeOther, "/admin")
		return nil, meta, nil
	}

	return &templates.AdminLoginInput{
		PasswordEnabled: h.passwordAuthEnabled(),
		OAuthEnabled:    h.oauthAuthEnabled(),
	}, nil, nil
}

// AdminLoginSubmit processes admin login
func (h *Handler) AdminLoginSubmit(ctx context.Context, r *http.Request) (*templates.AdminLoginInput, *routes.ResponseMeta, error) {
	// Parse form
	if err := r.ParseForm(); err != nil {
		return &templates.AdminLoginInput{
			Error:           "Invalid form submission",
			PasswordEnabled: h.passwordAuthEnabled(),
			OAuthEnabled:    h.oauthAuthEnabled(),
		}, nil, nil
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	// Lookup account
	account, err := h.Repository.GetAccountByUsername(ctx, username)
	if err != nil {
		return &templates.AdminLoginInput{Error: "Invalid username or password", PasswordEnabled: h.passwordAuthEnabled(), OAuthEnabled: h.oauthAuthEnabled()}, nil, nil
	}

	// Verify password
	cred, err := h.Repository.GetPasswordCredential(ctx, account.ID)
	if err != nil {
		return &templates.AdminLoginInput{Error: "Invalid username or password", PasswordEnabled: h.passwordAuthEnabled(), OAuthEnabled: h.oauthAuthEnabled()}, nil, nil
	}
	if err := bcrypt.CompareHashAndPassword([]byte(cred.PasswordHash), []byte(password)); err != nil {
		return &templates.AdminLoginInput{Error: "Invalid username or password", PasswordEnabled: h.passwordAuthEnabled(), OAuthEnabled: h.oauthAuthEnabled()}, nil, nil
	}

	// Check admin flag
	if !account.IsAdmin {
		return &templates.AdminLoginInput{Error: "Invalid username or password", PasswordEnabled: h.passwordAuthEnabled(), OAuthEnabled: h.oauthAuthEnabled()}, nil, nil
	}

	// Create account session (admin status comes from DB, not session type)
	sessionData := &session.SessionData{
		SessionType: session.SessionTypeAccount,
		AccountID:   &account.ID,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}
	signedID, err := h.SessionManager.CreateSession(ctx, sessionData)
	if err != nil {
		return nil, nil, fmt.Errorf("create admin session: %w", err)
	}

	cookie := h.SessionManager.CreateCookie(signedID)
	meta := routes.NewResponseMeta().
		WithCookie(cookie).
		WithRedirect(http.StatusSeeOther, "/admin")

	return nil, meta, nil
}

// AdminLogout logs out the admin
func (h *Handler) AdminLogout(ctx context.Context, r *http.Request) (*templates.AdminLoginInput, *routes.ResponseMeta, error) {
	// Destroy session from store
	if cookie, err := r.Cookie("session_id"); err == nil {
		_ = h.SessionManager.DestroySession(ctx, cookie.Value)
	}

	// Clear the session cookie
	clearCookie := &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	}

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

// AdminAccounts shows the account management page.
func (h *Handler) AdminAccounts(ctx context.Context, r *http.Request) (*templates.AdminAccountsInput, *routes.ResponseMeta, error) {
	page, pageSize := parsePaginationParams(r)

	total, err := h.Repository.CountAllAccounts(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to count accounts: %w", err)
	}

	p := pagination.New(page, pageSize, total)

	accounts, err := h.Repository.ListAllAccounts(ctx, p.PageSize, p.Offset)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	items := make([]templates.AccountItem, len(accounts))
	for i, a := range accounts {
		items[i] = templates.AccountItem{
			ID:       a.ID,
			Username: a.Username,
			FullName: a.FullName,
			IsAdmin:  a.IsAdmin,
		}
	}

	return &templates.AdminAccountsInput{
		Accounts:        items,
		Pagination:      p,
		PasswordEnabled: h.passwordAuthEnabled(),
		OAuthEnabled:    h.oauthAuthEnabled(),
	}, nil, nil
}

// buildAccountListPartialInput builds the account list partial after mutations.
func (h *Handler) buildAccountListPartialInput(ctx context.Context, formError string) (*templates.AccountListPartialInput, error) {
	total, err := h.Repository.CountAllAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count accounts: %w", err)
	}

	p := pagination.New(1, pagination.DefaultPageSize, total)

	accounts, err := h.Repository.ListAllAccounts(ctx, p.PageSize, p.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	items := make([]templates.AccountItem, len(accounts))
	for i, a := range accounts {
		items[i] = templates.AccountItem{
			ID:       a.ID,
			Username: a.Username,
			FullName: a.FullName,
			IsAdmin:  a.IsAdmin,
		}
	}

	return &templates.AccountListPartialInput{
		Accounts:        items,
		Pagination:      p,
		Error:           formError,
		PasswordEnabled: h.passwordAuthEnabled(),
		OAuthEnabled:    h.oauthAuthEnabled(),
	}, nil
}

// AdminCreateAccount creates a new account and returns updated account list partial.
func (h *Handler) AdminCreateAccount(ctx context.Context, r *http.Request) (*templates.AccountListPartialInput, *routes.ResponseMeta, error) {
	if err := r.ParseForm(); err != nil {
		input, err := h.buildAccountListPartialInput(ctx, "Invalid form submission")
		if err != nil {
			return nil, nil, err
		}
		return input, nil, nil
	}

	username := strings.TrimSpace(r.FormValue("username"))
	fullName := strings.TrimSpace(r.FormValue("full_name"))
	password := r.FormValue("password")
	if username == "" || fullName == "" {
		input, err := h.buildAccountListPartialInput(ctx, "All fields are required")
		if err != nil {
			return nil, nil, err
		}
		return input, nil, nil
	}

	if h.passwordAuthEnabled() {
		if strings.TrimSpace(password) == "" {
			input, err := h.buildAccountListPartialInput(ctx, "All fields are required")
			if err != nil {
				return nil, nil, err
			}
			return input, nil, nil
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to hash password: %w", err)
		}
		if _, err := h.Repository.CreateAccount(ctx, username, fullName, string(hashedPassword)); err != nil {
			input, loadErr := h.buildAccountListPartialInput(ctx, fmt.Sprintf("Failed to create account: %v", err))
			if loadErr != nil {
				return nil, nil, loadErr
			}
			return input, nil, nil
		}
	} else if h.oauthAuthEnabled() {
		if _, err := h.Repository.CreateOAuthAccount(ctx, username, fullName); err != nil {
			input, loadErr := h.buildAccountListPartialInput(ctx, fmt.Sprintf("Failed to create account: %v", err))
			if loadErr != nil {
				return nil, nil, loadErr
			}
			return input, nil, nil
		}
	} else {
		input, loadErr := h.buildAccountListPartialInput(ctx, "No authentication provider is enabled")
		if loadErr != nil {
			return nil, nil, loadErr
		}
		return input, nil, nil
	}

	input, err := h.buildAccountListPartialInput(ctx, "")
	if err != nil {
		return nil, nil, err
	}
	return input, nil, nil
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

func mapOAuthGroupRulesForTemplate(rules []*model.OAuthCommitteeGroupRule) []templates.OAuthCommitteeGroupRuleItem {
	items := make([]templates.OAuthCommitteeGroupRuleItem, len(rules))
	for i, rule := range rules {
		items[i] = templates.OAuthCommitteeGroupRuleItem{
			ID:       rule.ID,
			IDString: strconv.FormatInt(rule.ID, 10),
			Group:    rule.GroupName,
			Role:     rule.Role,
		}
	}
	return items
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
			ID:           u.ID,
			IDString:     fmt.Sprintf("%d", u.ID),
			Username:     u.Username,
			FullName:     u.FullName,
			Role:         u.Role,
			Quoted:       u.Quoted,
			OAuthManaged: u.OAuthManaged,
		}
	}

	assignable, err := h.Repository.ListUnassignedAccountsForCommittee(ctx, committee.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list assignable accounts: %w", err)
	}
	assignableItems := make([]templates.AccountOption, len(assignable))
	for i, a := range assignable {
		assignableItems[i] = templates.AccountOption{
			ID:       a.ID,
			IDString: fmt.Sprintf("%d", a.ID),
			Username: a.Username,
			FullName: a.FullName,
		}
	}
	ruleItems := []templates.OAuthCommitteeGroupRuleItem{}
	if h.oauthAuthEnabled() {
		rules, err := h.Repository.ListOAuthCommitteeGroupRulesByCommitteeSlug(ctx, slug)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to list oauth group rules: %w", err)
		}
		ruleItems = mapOAuthGroupRulesForTemplate(rules)
	}

	return &templates.AdminCommitteeUsersInput{
		CommitteeName:      committee.Name,
		CommitteeSlug:      committee.Slug,
		Users:              items,
		AssignableAccounts: assignableItems,
		OAuthGroupRules:    ruleItems,
		OAuthEnabled:       h.oauthAuthEnabled(),
		Pagination:         p,
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
		items[i] = templates.UserItem{
			ID:           u.ID,
			IDString:     fmt.Sprintf("%d", u.ID),
			Username:     u.Username,
			FullName:     u.FullName,
			Role:         u.Role,
			Quoted:       u.Quoted,
			OAuthManaged: u.OAuthManaged,
		}
	}

	committeeID, err := h.Repository.GetCommitteeIDBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to get committee ID: %w", err)
	}
	assignable, err := h.Repository.ListUnassignedAccountsForCommittee(ctx, committeeID)
	if err != nil {
		return nil, fmt.Errorf("failed to list assignable accounts: %w", err)
	}
	assignableItems := make([]templates.AccountOption, len(assignable))
	for i, a := range assignable {
		assignableItems[i] = templates.AccountOption{
			ID:       a.ID,
			IDString: fmt.Sprintf("%d", a.ID),
			Username: a.Username,
			FullName: a.FullName,
		}
	}
	ruleItems := []templates.OAuthCommitteeGroupRuleItem{}
	if h.oauthAuthEnabled() {
		rules, err := h.Repository.ListOAuthCommitteeGroupRulesByCommitteeSlug(ctx, slug)
		if err != nil {
			return nil, fmt.Errorf("failed to list oauth group rules: %w", err)
		}
		ruleItems = mapOAuthGroupRulesForTemplate(rules)
	}

	return &templates.UserListPartialInput{
		CommitteeSlug:      slug,
		Users:              items,
		AssignableAccounts: assignableItems,
		OAuthGroupRules:    ruleItems,
		OAuthEnabled:       h.oauthAuthEnabled(),
		Pagination:         p,
		Error:              formError,
	}, nil
}

// AdminAssignAccount assigns an existing account to a committee.
func (h *Handler) AdminAssignAccount(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.UserListPartialInput, *routes.ResponseMeta, error) {
	slug := params.Slug

	if err := r.ParseForm(); err != nil {
		input, err := h.buildUserListPartialInput(ctx, slug, "Invalid form submission")
		if err != nil {
			return nil, nil, err
		}
		return input, nil, nil
	}

	accountIDStr := strings.TrimSpace(r.FormValue("account_id"))
	role := r.FormValue("role")
	quoted := r.FormValue("quoted") == "true"

	if accountIDStr == "" || role == "" {
		input, err := h.buildUserListPartialInput(ctx, slug, "All fields are required")
		if err != nil {
			return nil, nil, err
		}
		return input, nil, nil
	}

	accountID, err := strconv.ParseInt(accountIDStr, 10, 64)
	if err != nil {
		input, loadErr := h.buildUserListPartialInput(ctx, slug, "Invalid account selection")
		if loadErr != nil {
			return nil, nil, loadErr
		}
		return input, nil, nil
	}

	committeeID, err := h.Repository.GetCommitteeIDBySlug(ctx, slug)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get committee ID: %w", err)
	}

	if err := h.Repository.AssignAccountToCommittee(ctx, committeeID, accountID, quoted, role); err != nil {
		input, loadErr := h.buildUserListPartialInput(ctx, slug, fmt.Sprintf("Failed to assign account: %v", err))
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

// AdminUpdateUserMembership updates quoted and role for a committee membership.
// Role updates are blocked for OAuth-managed memberships.
func (h *Handler) AdminUpdateUserMembership(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.UserListPartialInput, *routes.ResponseMeta, error) {
	slug := params.Slug
	userID, err := strconv.ParseInt(params.UserId, 10, 64)
	if err != nil {
		input, loadErr := h.buildUserListPartialInput(ctx, slug, "Invalid user ID")
		if loadErr != nil {
			return nil, nil, loadErr
		}
		return input, nil, nil
	}

	if err := r.ParseForm(); err != nil {
		input, loadErr := h.buildUserListPartialInput(ctx, slug, "Invalid form submission")
		if loadErr != nil {
			return nil, nil, loadErr
		}
		return input, nil, nil
	}

	role := strings.TrimSpace(r.FormValue("role"))
	quoted := false
	for _, v := range r.Form["quoted"] {
		if v == "true" {
			quoted = true
			break
		}
	}
	if role != "member" && role != "chairperson" {
		input, loadErr := h.buildUserListPartialInput(ctx, slug, "Invalid role")
		if loadErr != nil {
			return nil, nil, loadErr
		}
		return input, nil, nil
	}

	user, err := h.Repository.GetUserByID(ctx, userID)
	if err != nil {
		input, loadErr := h.buildUserListPartialInput(ctx, slug, "Membership not found")
		if loadErr != nil {
			return nil, nil, loadErr
		}
		return input, nil, nil
	}

	committeeID, err := h.Repository.GetCommitteeIDBySlug(ctx, slug)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get committee ID: %w", err)
	}
	if user.CommitteeID != committeeID {
		input, loadErr := h.buildUserListPartialInput(ctx, slug, "Membership does not belong to this committee")
		if loadErr != nil {
			return nil, nil, loadErr
		}
		return input, nil, nil
	}

	oauthManaged, err := h.Repository.IsOAuthManagedMembership(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check oauth-managed membership: %w", err)
	}
	if oauthManaged {
		role = user.Role
	}

	if err := h.Repository.UpdateUserMembership(ctx, userID, quoted, role); err != nil {
		input, loadErr := h.buildUserListPartialInput(ctx, slug, fmt.Sprintf("Failed to update membership: %v", err))
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

// AdminCreateOAuthCommitteeGroupRule creates a committee OAuth group-to-role rule.
func (h *Handler) AdminCreateOAuthCommitteeGroupRule(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.UserListPartialInput, *routes.ResponseMeta, error) {
	slug := params.Slug
	if !h.oauthAuthEnabled() {
		input, err := h.buildUserListPartialInput(ctx, slug, "OAuth provider is disabled")
		if err != nil {
			return nil, nil, err
		}
		return input, nil, nil
	}
	if err := r.ParseForm(); err != nil {
		input, err := h.buildUserListPartialInput(ctx, slug, "Invalid form submission")
		if err != nil {
			return nil, nil, err
		}
		return input, nil, nil
	}
	groupName := strings.TrimSpace(r.FormValue("group_name"))
	role := strings.TrimSpace(r.FormValue("role"))
	if groupName == "" || (role != "member" && role != "chairperson") {
		input, err := h.buildUserListPartialInput(ctx, slug, "Group and role are required")
		if err != nil {
			return nil, nil, err
		}
		return input, nil, nil
	}
	if _, err := h.Repository.CreateOAuthCommitteeGroupRuleByCommitteeSlug(ctx, slug, groupName, role); err != nil {
		input, loadErr := h.buildUserListPartialInput(ctx, slug, fmt.Sprintf("Failed to create OAuth group rule: %v", err))
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

// AdminDeleteOAuthCommitteeGroupRule deletes a committee OAuth group-to-role rule.
func (h *Handler) AdminDeleteOAuthCommitteeGroupRule(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.UserListPartialInput, *routes.ResponseMeta, error) {
	slug := params.Slug
	if !h.oauthAuthEnabled() {
		input, err := h.buildUserListPartialInput(ctx, slug, "OAuth provider is disabled")
		if err != nil {
			return nil, nil, err
		}
		return input, nil, nil
	}
	ruleID, err := strconv.ParseInt(params.RuleId, 10, 64)
	if err != nil {
		input, loadErr := h.buildUserListPartialInput(ctx, slug, "Invalid OAuth group rule ID")
		if loadErr != nil {
			return nil, nil, loadErr
		}
		return input, nil, nil
	}
	if err := h.Repository.DeleteOAuthCommitteeGroupRuleByIDAndCommitteeSlug(ctx, ruleID, slug); err != nil {
		input, loadErr := h.buildUserListPartialInput(ctx, slug, fmt.Sprintf("Failed to delete OAuth group rule: %v", err))
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
