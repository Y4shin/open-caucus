//go:build e2e

package e2e_test

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"

	connect "connectrpc.com/connect"
	playwright "github.com/playwright-community/playwright-go"

	adminv1 "github.com/Y4shin/open-caucus/gen/go/conference/admin/v1"
	adminv1connect "github.com/Y4shin/open-caucus/gen/go/conference/admin/v1/adminv1connect"
	sessionv1 "github.com/Y4shin/open-caucus/gen/go/conference/session/v1"
	sessionv1connect "github.com/Y4shin/open-caucus/gen/go/conference/session/v1/sessionv1connect"
)

const oauthTestSubject = "id1"

func TestOAuthLogin_AutoCreateAccount(t *testing.T) {
	ts := newOAuthTestServer(t, oauthServerOptions{
		PasswordEnabled:  false,
		ProvisioningMode: "auto_create",
	})
	page := newPage(t)

	oauthLogin(t, page, ts.URL, "user", ts.provider.Username, ts.provider.Password)

	identity, err := ts.repo.GetOAuthIdentityByIssuerSubject(context.Background(), ts.provider.Issuer, oauthTestSubject)
	if err != nil {
		t.Fatalf("expected oauth identity to be created: %v", err)
	}
	account, err := ts.repo.GetAccountByID(context.Background(), identity.AccountID)
	if err != nil {
		t.Fatalf("expected oauth account to be created: %v", err)
	}
	if account.AuthMethod != "oauth" {
		t.Fatalf("expected auth_method oauth, got=%q", account.AuthMethod)
	}
}

func TestOAuthLogin_PreprovisionedVsAutoCreate(t *testing.T) {
	ts := newOAuthTestServer(t, oauthServerOptions{
		PasswordEnabled:  false,
		ProvisioningMode: "preprovisioned",
	})
	page := newPage(t)

	oauthLoginExpectFailureToRoot(t, page, ts.URL, ts.provider.Username, ts.provider.Password)

	candidates := []string{ts.provider.Username, "test-user@zitadel.ch", oauthTestSubject}
	for _, candidate := range candidates {
		if _, err := ts.repo.CreateOAuthAccount(context.Background(), candidate, candidate, ""); err != nil && !strings.Contains(err.Error(), "UNIQUE constraint failed: accounts.username") {
			t.Fatalf("preprovision oauth account %q: %v", candidate, err)
		}
	}

	oauthLogin(t, page, ts.URL, "user", ts.provider.Username, ts.provider.Password)
}

func TestOAuthAdminLogin_UsesAdminGroupSync(t *testing.T) {
	ts := newOAuthTestServer(t, oauthServerOptions{
		PasswordEnabled:  false,
		ProvisioningMode: "auto_create",
		AdminGroup:       "$client_id",
	})
	page := newPage(t)

	oauthLogin(t, page, ts.URL, "admin", ts.provider.Username, ts.provider.Password)

	identity, err := ts.repo.GetOAuthIdentityByIssuerSubject(context.Background(), ts.provider.Issuer, oauthTestSubject)
	if err != nil {
		t.Fatalf("get oauth identity after admin login: %v", err)
	}
	account, err := ts.repo.GetAccountByID(context.Background(), identity.AccountID)
	if err != nil {
		t.Fatalf("get oauth account after admin login: %v", err)
	}
	if !account.IsAdmin {
		t.Fatalf("expected oauth user to be admin after admin-group sync")
	}
}

func TestOAuthAdminLogin_UsesGroupsClaimAdminGroup(t *testing.T) {
	ts := newOAuthTestServer(t, oauthServerOptions{
		PasswordEnabled:  false,
		ProvisioningMode: "auto_create",
		GroupsClaim:      "groups",
		ProviderGroups:   []string{"ca-admin", "committee-a"},
		AdminGroup:       "ca-admin",
	})
	page := newPage(t)

	oauthLogin(t, page, ts.URL, "admin", ts.provider.Username, ts.provider.Password)

	identity, err := ts.repo.GetOAuthIdentityByIssuerSubject(context.Background(), ts.provider.Issuer, oauthTestSubject)
	if err != nil {
		t.Fatalf("get oauth identity after admin login via groups claim: %v", err)
	}
	account, err := ts.repo.GetAccountByID(context.Background(), identity.AccountID)
	if err != nil {
		t.Fatalf("get oauth account after admin login via groups claim: %v", err)
	}
	if !account.IsAdmin {
		t.Fatalf("expected oauth user to be admin when groups claim contains ca-admin")
	}
}

func TestPasswordDisabled_LoginUIAndSubmitPaths(t *testing.T) {
	ts := newOAuthTestServer(t, oauthServerOptions{
		PasswordEnabled:  false,
		ProvisioningMode: "auto_create",
	})
	page := newPage(t)

	if _, err := page.Goto(ts.URL + "/login"); err != nil {
		t.Fatalf("goto user login: %v", err)
	}
	if count, err := page.Locator("input[name=password]").Count(); err != nil {
		t.Fatalf("count user password inputs: %v", err)
	} else if count != 0 {
		t.Fatalf("expected no password field on user login when disabled, got=%d", count)
	}

	if _, err := page.Goto(ts.URL + "/admin/login"); err != nil {
		t.Fatalf("goto admin login: %v", err)
	}
	if count, err := page.Locator("input[name=password]").Count(); err != nil {
		t.Fatalf("count admin password inputs: %v", err)
	} else if count != 0 {
		t.Fatalf("expected no password field on admin login when disabled, got=%d", count)
	}

	resp, err := http.PostForm(ts.URL+"/login", url.Values{
		"username": {"x"},
		"password": {"x"},
	})
	if err != nil {
		t.Fatalf("post /login: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected /login submit status 404, got=%d", resp.StatusCode)
	}

	resp, err = http.PostForm(ts.URL+"/admin/login", url.Values{
		"username": {"x"},
		"password": {"x"},
	})
	if err != nil {
		t.Fatalf("post /admin/login: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected /admin/login submit status 404, got=%d", resp.StatusCode)
	}
}

func TestOAuthCommitteeSync_AddsAndRemovesManagedMemberships(t *testing.T) {
	ts := newOAuthTestServer(t, oauthServerOptions{
		PasswordEnabled:  false,
		ProvisioningMode: "auto_create",
	})
	ctx := context.Background()
	const (
		committeeSlug = "oauth-sync-committee"
		committeeName = "OAuth Sync Committee"
	)
	ts.seedCommittee(t, committeeName, committeeSlug)

	if _, err := ts.repo.CreateOAuthCommitteeGroupRuleByCommitteeSlug(ctx, committeeSlug, ts.provider.ClientID, "member"); err != nil {
		t.Fatalf("create oauth committee group rule: %v", err)
	}

	page := newPage(t)
	oauthLogin(t, page, ts.URL, "user", ts.provider.Username, ts.provider.Password)

	identity, err := ts.repo.GetOAuthIdentityByIssuerSubject(ctx, ts.provider.Issuer, oauthTestSubject)
	if err != nil {
		t.Fatalf("get oauth identity after login: %v", err)
	}
	if _, err := ts.repo.GetUserMembershipByAccountIDAndSlug(ctx, identity.AccountID, committeeSlug); err != nil {
		t.Fatalf("expected membership to be created after oauth login: %v", err)
	}

	rules, err := ts.repo.ListOAuthCommitteeGroupRulesByCommitteeSlug(ctx, committeeSlug)
	if err != nil {
		t.Fatalf("list oauth committee group rules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected one oauth rule, got=%d", len(rules))
	}
	if err := ts.repo.DeleteOAuthCommitteeGroupRuleByIDAndCommitteeSlug(ctx, rules[0].ID, committeeSlug); err != nil {
		t.Fatalf("delete oauth rule: %v", err)
	}

	page2 := newPage(t)
	oauthLogin(t, page2, ts.URL, "user", ts.provider.Username, ts.provider.Password)

	if _, err := ts.repo.GetUserMembershipByAccountIDAndSlug(ctx, identity.AccountID, committeeSlug); err == nil {
		t.Fatalf("expected oauth-managed committee membership to be removed after rule deletion and re-login")
	}
}

func TestOAuthCommitteeSync_PicksHighestRoleAcrossMatchingGroups(t *testing.T) {
	ts := newOAuthTestServer(t, oauthServerOptions{
		PasswordEnabled:  false,
		ProvisioningMode: "auto_create",
		GroupsClaim:      "groups",
		ProviderGroups:   []string{"committee-a", "committee-a-chair"},
	})
	ctx := context.Background()
	const (
		committeeSlug = "oauth-role-precedence"
		committeeName = "OAuth Role Precedence"
	)
	ts.seedCommittee(t, committeeName, committeeSlug)

	if _, err := ts.repo.CreateOAuthCommitteeGroupRuleByCommitteeSlug(ctx, committeeSlug, "committee-a", "member"); err != nil {
		t.Fatalf("create member oauth committee group rule: %v", err)
	}
	if _, err := ts.repo.CreateOAuthCommitteeGroupRuleByCommitteeSlug(ctx, committeeSlug, "committee-a-chair", "chairperson"); err != nil {
		t.Fatalf("create chair oauth committee group rule: %v", err)
	}

	page := newPage(t)
	oauthLogin(t, page, ts.URL, "user", ts.provider.Username, ts.provider.Password)

	identity, err := ts.repo.GetOAuthIdentityByIssuerSubject(ctx, ts.provider.Issuer, oauthTestSubject)
	if err != nil {
		t.Fatalf("get oauth identity after login: %v", err)
	}
	membership, err := ts.repo.GetUserMembershipByAccountIDAndSlug(ctx, identity.AccountID, committeeSlug)
	if err != nil {
		t.Fatalf("get committee membership after login: %v", err)
	}
	if membership.Role != "chairperson" {
		t.Fatalf("expected highest matched role chairperson, got=%q", membership.Role)
	}
}

func TestOAuthCommitteeGroupPrefix_EnforcedViaAdminAPI(t *testing.T) {
	ts := newOAuthTestServer(t, oauthServerOptions{
		PasswordEnabled:      true,
		ProvisioningMode:     "auto_create",
		CommitteeGroupPrefix: "conference-",
	})
	ctx := context.Background()

	ts.seedCommittee(t, "Prefix Committee", "prefix-committee")

	// Create a Connect admin client with a cookie jar to hold the session.
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create cookie jar: %v", err)
	}
	httpClient := &http.Client{Jar: jar}

	sessionClient := sessionv1connect.NewSessionServiceClient(httpClient, ts.URL+"/api")
	adminClient := adminv1connect.NewAdminServiceClient(httpClient, ts.URL+"/api")

	// Log in as admin via the session API.
	if _, err := sessionClient.Login(ctx, connect.NewRequest(&sessionv1.LoginRequest{
		Username: testAdminUsername,
		Password: testAdminPassword,
	})); err != nil {
		t.Fatalf("admin login: %v", err)
	}

	// Creating a rule with a non-matching prefix must fail.
	_, err = adminClient.CreateOAuthRule(ctx, connect.NewRequest(&adminv1.CreateOAuthRuleRequest{
		Slug:      "prefix-committee",
		GroupName: "wrong-prefix-group",
		Role:      "member",
	}))
	if err == nil {
		t.Fatal("expected error when creating rule with group name that does not match the configured prefix")
	}

	// Creating a rule with the correct prefix must succeed.
	resp, err := adminClient.CreateOAuthRule(ctx, connect.NewRequest(&adminv1.CreateOAuthRuleRequest{
		Slug:      "prefix-committee",
		GroupName: "conference-members",
		Role:      "member",
	}))
	if err != nil {
		t.Fatalf("expected success for group matching prefix: %v", err)
	}
	if resp.Msg.GetRule().GetGroupName() != "conference-members" {
		t.Fatalf("unexpected group name: %q", resp.Msg.GetRule().GetGroupName())
	}

	// GetCommitteeAdmin must expose the prefix.
	adminResp, err := adminClient.GetCommitteeAdmin(ctx, connect.NewRequest(&adminv1.GetCommitteeAdminRequest{
		Slug: "prefix-committee",
	}))
	if err != nil {
		t.Fatalf("get committee admin: %v", err)
	}
	if got := adminResp.Msg.GetOauthGroupPrefix(); got != "conference-" {
		t.Fatalf("expected oauth_group_prefix=%q, got=%q", "conference-", got)
	}
}

func oauthLoginExpectFailureToRoot(t *testing.T, page playwright.Page, baseURL, username, password string) {
	t.Helper()
	if _, err := page.Goto(baseURL + "/login"); err != nil {
		t.Fatalf("goto /login: %v", err)
	}
	link := page.Locator("a[href*='/oauth/start'][href*='target=user']").First()
	if err := link.Click(); err != nil {
		t.Fatalf("click oauth start: %v", err)
	}
	if err := page.WaitForURL("**/login/username**"); err != nil {
		t.Fatalf("wait provider login: %v", err)
	}
	if err := page.Locator("#username").Fill(username); err != nil {
		t.Fatalf("fill provider username: %v", err)
	}
	if err := page.Locator("#password").Fill(password); err != nil {
		t.Fatalf("fill provider password: %v", err)
	}
	if err := page.Locator("#password").Press("Enter"); err != nil {
		t.Fatalf("submit provider login: %v", err)
	}
	if err := page.WaitForURL(baseURL + "/login"); err != nil {
		t.Fatalf("wait redirect back to /login for failed preprovisioned login: %v", err)
	}
}
