package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Y4shin/conference-tool/internal/oauth"
	"github.com/Y4shin/conference-tool/internal/repository/model"
	"github.com/Y4shin/conference-tool/internal/session"
)

// OAuthStart initiates OAuth/OIDC login.
func (h *Handler) OAuthStart(w http.ResponseWriter, r *http.Request) error {
	if h.OAuthService == nil || !h.oauthAuthEnabled() {
		http.NotFound(w, r)
		return nil
	}
	target := strings.TrimSpace(r.URL.Query().Get("target"))
	authURL, stateCookie, err := h.OAuthService.Start(target)
	if err != nil {
		return fmt.Errorf("oauth start: %w", err)
	}
	http.SetCookie(w, stateCookie)
	http.Redirect(w, r, authURL, http.StatusSeeOther)
	return nil
}

// OAuthCallback handles the OAuth/OIDC callback and creates a local session.
func (h *Handler) OAuthCallback(w http.ResponseWriter, r *http.Request) error {
	if h.OAuthService == nil || !h.oauthAuthEnabled() {
		http.NotFound(w, r)
		return nil
	}
	callbackResult, err := h.OAuthService.HandleCallback(r.Context(), r)
	http.SetCookie(w, h.OAuthService.ClearStateCookie())
	if err != nil {
		redirect := "/"
		if strings.EqualFold(callbackResultTarget(callbackResult), "admin") {
			redirect = "/admin/login"
		}
		http.Redirect(w, r, redirect, http.StatusSeeOther)
		return nil
	}

	account, err := h.resolveOAuthAccount(r.Context(), callbackResult.Principal)
	if err != nil {
		redirect := "/"
		if callbackResult.Target == "admin" {
			redirect = "/admin/login"
		}
		http.Redirect(w, r, redirect, http.StatusSeeOther)
		return nil
	}

	if err := h.syncOAuthAdmin(r.Context(), account.ID, callbackResult.Principal.Groups); err != nil {
		return fmt.Errorf("oauth callback admin sync: %w", err)
	}
	if err := h.syncOAuthCommittees(r.Context(), account.ID, callbackResult.Principal.Groups); err != nil {
		return fmt.Errorf("oauth callback committee sync: %w", err)
	}

	refreshedAccount, err := h.Repository.GetAccountByID(r.Context(), account.ID)
	if err != nil {
		return fmt.Errorf("oauth callback reload account: %w", err)
	}
	sessionData := &session.SessionData{
		SessionType: session.SessionTypeAccount,
		AccountID:   &refreshedAccount.ID,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}
	signedID, err := h.SessionManager.CreateSession(r.Context(), sessionData)
	if err != nil {
		return fmt.Errorf("oauth callback create session: %w", err)
	}
	http.SetCookie(w, h.SessionManager.CreateCookie(signedID))

	redirect := "/home"
	if callbackResult.Target == "admin" {
		if !refreshedAccount.IsAdmin {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return nil
		}
		redirect = "/admin"
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
	return nil
}

func callbackResultTarget(result *oauth.CallbackResult) string {
	if result == nil {
		return ""
	}
	return result.Target
}

func (h *Handler) resolveOAuthAccount(ctx context.Context, principal oauth.Principal) (*model.Account, error) {
	if err := h.validateRequiredOAuthGroups(principal.Groups); err != nil {
		return nil, err
	}

	identity, err := h.Repository.GetOAuthIdentityByIssuerSubject(ctx, principal.Issuer, principal.Subject)
	if err == nil {
		account, err := h.Repository.GetAccountByID(ctx, identity.AccountID)
		if err != nil {
			return nil, err
		}
		if err := h.upsertOAuthIdentity(ctx, principal, account.ID); err != nil {
			return nil, err
		}
		return account, nil
	}

	username := strings.TrimSpace(principal.Username)
	if username == "" {
		username = strings.TrimSpace(principal.Subject)
	}
	username = strings.ToLower(username)
	fullName := strings.TrimSpace(principal.FullName)
	if fullName == "" {
		fullName = username
	}

	var account *model.Account
	account, err = h.Repository.GetAccountByUsername(ctx, username)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if account == nil {
		if strings.EqualFold(h.AuthConfig.OAuthProvisioningMode, "auto_create") {
			account, err = h.Repository.CreateOAuthAccount(ctx, username, fullName)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("oauth account is not preprovisioned")
		}
	}
	if account.AuthMethod != "oauth" {
		return nil, fmt.Errorf("account uses a different auth method")
	}
	if err := h.upsertOAuthIdentity(ctx, principal, account.ID); err != nil {
		return nil, err
	}
	return account, nil
}

func (h *Handler) validateRequiredOAuthGroups(groups []string) error {
	if h.AuthConfig == nil {
		return nil
	}
	if len(h.AuthConfig.OAuthRequiredGroups) == 0 {
		return nil
	}
	for _, required := range h.AuthConfig.OAuthRequiredGroups {
		if oauthGroupContains(groups, required) {
			return nil
		}
	}
	return fmt.Errorf("missing required oauth group")
}

func oauthGroupContains(groups []string, group string) bool {
	needle := strings.TrimSpace(group)
	if needle == "" {
		return false
	}
	for _, g := range groups {
		if strings.TrimSpace(g) == needle {
			return true
		}
	}
	return false
}

func (h *Handler) upsertOAuthIdentity(ctx context.Context, principal oauth.Principal, accountID int64) error {
	var (
		username   = stringPtrOrNil(strings.TrimSpace(principal.Username))
		fullName   = stringPtrOrNil(strings.TrimSpace(principal.FullName))
		email      = stringPtrOrNil(strings.TrimSpace(principal.Email))
		groupsJSON *string
	)
	if payload, err := json.Marshal(principal.Groups); err == nil {
		text := string(payload)
		groupsJSON = &text
	}
	_, err := h.Repository.UpsertOAuthIdentity(
		ctx,
		principal.Issuer,
		principal.Subject,
		accountID,
		username,
		fullName,
		email,
		groupsJSON,
	)
	return err
}

func stringPtrOrNil(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func (h *Handler) syncOAuthAdmin(ctx context.Context, accountID int64, groups []string) error {
	if h.AuthConfig == nil {
		return nil
	}
	adminGroup := strings.TrimSpace(h.AuthConfig.OAuthAdminGroup)
	if adminGroup == "" {
		return nil
	}
	return h.Repository.SetAccountIsAdmin(ctx, accountID, oauthGroupContains(groups, adminGroup))
}

func (h *Handler) syncOAuthCommittees(ctx context.Context, accountID int64, groups []string) error {
	rules, err := h.Repository.ListAllOAuthCommitteeGroupRules(ctx)
	if err != nil {
		return err
	}
	desiredByCommittee := map[int64]string{}
	for _, rule := range rules {
		if !oauthGroupContains(groups, rule.GroupName) {
			continue
		}
		currentRole, exists := desiredByCommittee[rule.CommitteeID]
		if !exists {
			desiredByCommittee[rule.CommitteeID] = rule.Role
			continue
		}
		if roleRank(rule.Role) > roleRank(currentRole) {
			desiredByCommittee[rule.CommitteeID] = rule.Role
		}
	}
	desired := make([]model.OAuthDesiredMembership, 0, len(desiredByCommittee))
	for committeeID, role := range desiredByCommittee {
		desired = append(desired, model.OAuthDesiredMembership{
			CommitteeID: committeeID,
			Role:        role,
		})
	}
	return h.Repository.SyncOAuthCommitteeMemberships(ctx, accountID, desired)
}

func roleRank(role string) int {
	switch role {
	case "chairperson":
		return 2
	case "member":
		return 1
	default:
		return 0
	}
}
