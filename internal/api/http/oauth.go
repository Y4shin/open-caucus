package apihttp

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Y4shin/open-caucus/internal/config"
	"github.com/Y4shin/open-caucus/internal/oauth"
	"github.com/Y4shin/open-caucus/internal/repository"
	"github.com/Y4shin/open-caucus/internal/repository/model"
	"github.com/Y4shin/open-caucus/internal/session"
)

// OAuthHandler handles OAuth/OIDC start and callback flows for the SPA server.
type OAuthHandler struct {
	OAuthService   *oauth.Service
	Repository     repository.Repository
	SessionManager *session.Manager
	AuthConfig     *config.AuthConfig
}

// NewOAuthStartHandler returns an http.Handler that initiates the OAuth/OIDC login flow.
func NewOAuthStartHandler(h *OAuthHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.OAuthService == nil || !h.oauthEnabled() {
			http.NotFound(w, r)
			return
		}
		target := strings.TrimSpace(r.URL.Query().Get("target"))
		authURL, stateCookie, err := h.OAuthService.Start(target)
		if err != nil {
			http.Error(w, fmt.Sprintf("oauth start: %v", err), http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, stateCookie)
		http.Redirect(w, r, authURL, http.StatusSeeOther)
	})
}

// NewOAuthCallbackHandler returns an http.Handler that processes the OAuth/OIDC callback.
func NewOAuthCallbackHandler(h *OAuthHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.OAuthService == nil || !h.oauthEnabled() {
			http.NotFound(w, r)
			return
		}
		callbackResult, err := h.OAuthService.HandleCallback(r.Context(), r)
		http.SetCookie(w, h.OAuthService.ClearStateCookie())
		if err != nil {
			slog.Warn("oauth callback failed", "err", err)
			redirect := "/"
			if callbackTarget(callbackResult) == "admin" {
				redirect = "/admin/login"
			}
			http.Redirect(w, r, redirect, http.StatusSeeOther)
			return
		}

		account, err := h.resolveAccount(r.Context(), callbackResult.Principal)
		if err != nil {
			slog.Warn("oauth account resolution failed", "subject", callbackResult.Principal.Subject, "issuer", callbackResult.Principal.Issuer, "username", callbackResult.Principal.Username, "groups", callbackResult.Principal.Groups, "err", err)
			redirect := "/"
			if callbackResult.Target == "admin" {
				redirect = "/admin/login"
			}
			http.Redirect(w, r, redirect, http.StatusSeeOther)
			return
		}

		slog.Info("oauth groups detected", "username", account.Username, "groups", callbackResult.Principal.Groups, "groups_count", len(callbackResult.Principal.Groups))

		if err := h.syncAdmin(r.Context(), account.ID, callbackResult.Principal.Groups); err != nil {
			http.Error(w, fmt.Sprintf("oauth callback admin sync: %v", err), http.StatusInternalServerError)
			return
		}
		if err := h.syncCommittees(r.Context(), account.ID, callbackResult.Principal.Groups); err != nil {
			http.Error(w, fmt.Sprintf("oauth callback committee sync: %v", err), http.StatusInternalServerError)
			return
		}

		refreshed, err := h.Repository.GetAccountByID(r.Context(), account.ID)
		if err != nil {
			http.Error(w, fmt.Sprintf("oauth callback reload account: %v", err), http.StatusInternalServerError)
			return
		}
		sd := &session.SessionData{
			SessionType: session.SessionTypeAccount,
			AccountID:   &refreshed.ID,
			ExpiresAt:   time.Now().Add(24 * time.Hour),
		}
		signedID, err := h.SessionManager.CreateSession(r.Context(), sd)
		if err != nil {
			http.Error(w, fmt.Sprintf("oauth callback create session: %v", err), http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, h.SessionManager.CreateCookie(signedID))

		redirect := "/home"
		if callbackResult.Target == "admin" {
			if !refreshed.IsAdmin {
				slog.Warn("oauth admin login denied: account is not admin", "username", refreshed.Username)
				http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
				return
			}
			redirect = "/admin"
		}
		slog.Info("oauth login successful", "username", refreshed.Username, "target", callbackResult.Target)
		http.Redirect(w, r, redirect, http.StatusSeeOther)
	})
}

func (h *OAuthHandler) oauthEnabled() bool {
	if h.AuthConfig == nil {
		return false
	}
	return h.AuthConfig.OAuthEnabled
}

func callbackTarget(result *oauth.CallbackResult) string {
	if result == nil {
		return ""
	}
	return result.Target
}

func (h *OAuthHandler) resolveAccount(ctx context.Context, principal oauth.Principal) (*model.Account, error) {
	if err := h.validateRequiredGroups(principal.Groups); err != nil {
		return nil, err
	}

	identity, err := h.Repository.GetOAuthIdentityByIssuerSubject(ctx, principal.Issuer, principal.Subject)
	if err == nil {
		account, err := h.Repository.GetAccountByID(ctx, identity.AccountID)
		if err != nil {
			return nil, err
		}
		if err := h.upsertIdentity(ctx, principal, account.ID); err != nil {
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
		if strings.EqualFold(h.provisioningMode(), "auto_create") {
			account, err = h.Repository.CreateOAuthAccount(ctx, username, fullName, strings.TrimSpace(principal.Email))
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
	if err := h.upsertIdentity(ctx, principal, account.ID); err != nil {
		return nil, err
	}
	// Update display name and email from OIDC claims on every login.
	if fullName != "" || strings.TrimSpace(principal.Email) != "" {
		_ = h.Repository.UpdateAccountProfile(ctx, account.ID, fullName, strings.TrimSpace(principal.Email))
	}
	return account, nil
}

func (h *OAuthHandler) provisioningMode() string {
	if h.AuthConfig == nil {
		return ""
	}
	return h.AuthConfig.OAuthProvisioningMode
}

func (h *OAuthHandler) validateRequiredGroups(groups []string) error {
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
	return fmt.Errorf("missing required oauth group: user has groups %v, but needs at least one of %v", groups, h.AuthConfig.OAuthRequiredGroups)
}

func (h *OAuthHandler) syncAdmin(ctx context.Context, accountID int64, groups []string) error {
	if h.AuthConfig == nil {
		return nil
	}
	adminGroup := strings.TrimSpace(h.AuthConfig.OAuthAdminGroup)
	if adminGroup == "" {
		return nil
	}
	return h.Repository.SetAccountIsAdmin(ctx, accountID, oauthGroupContains(groups, adminGroup))
}

func (h *OAuthHandler) syncCommittees(ctx context.Context, accountID int64, groups []string) error {
	rules, err := h.Repository.ListAllOAuthCommitteeGroupRules(ctx)
	if err != nil {
		return err
	}
	slog.Debug("oauth committee sync: evaluating rules", "account_id", accountID, "groups", groups, "rule_count", len(rules))
	desiredByCommittee := map[int64]string{}
	for _, rule := range rules {
		if !oauthGroupContains(groups, rule.GroupName) {
			continue
		}
		slog.Debug("oauth committee sync: group matched rule", "group", rule.GroupName, "committee_id", rule.CommitteeID, "role", rule.Role)
		currentRole, exists := desiredByCommittee[rule.CommitteeID]
		if !exists {
			desiredByCommittee[rule.CommitteeID] = rule.Role
			continue
		}
		if oauthRoleRank(rule.Role) > oauthRoleRank(currentRole) {
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
	slog.Debug("oauth committee sync: desired memberships", "account_id", accountID, "desired_count", len(desired), "desired", desired)
	return h.Repository.SyncOAuthCommitteeMemberships(ctx, accountID, desired)
}

func (h *OAuthHandler) upsertIdentity(ctx context.Context, principal oauth.Principal, accountID int64) error {
	var (
		username   = oauthStrPtr(strings.TrimSpace(principal.Username))
		fullName   = oauthStrPtr(strings.TrimSpace(principal.FullName))
		email      = oauthStrPtr(strings.TrimSpace(principal.Email))
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

func oauthRoleRank(role string) int {
	switch role {
	case "chairperson":
		return 2
	case "member":
		return 1
	default:
		return 0
	}
}

func oauthStrPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
