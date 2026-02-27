//go:build e2e

package e2e_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	playwright "github.com/playwright-community/playwright-go"

	docembed "github.com/Y4shin/conference-tool/doc"
	"github.com/Y4shin/conference-tool/internal/broker"
	"github.com/Y4shin/conference-tool/internal/config"
	"github.com/Y4shin/conference-tool/internal/docs"
	"github.com/Y4shin/conference-tool/internal/handlers"
	"github.com/Y4shin/conference-tool/internal/locale"
	"github.com/Y4shin/conference-tool/internal/middleware"
	"github.com/Y4shin/conference-tool/internal/oauth"
	"github.com/Y4shin/conference-tool/internal/repository/sqlite"
	"github.com/Y4shin/conference-tool/internal/routes"
	"github.com/Y4shin/conference-tool/internal/session"
	"github.com/Y4shin/conference-tool/internal/storage"
	oidctest "github.com/Y4shin/conference-tool/internal/testsupport/oidc"
)

type oauthServerOptions struct {
	PasswordEnabled  bool
	ProvisioningMode string
	RequiredGroups   []string
	AdminGroup       string
	GroupsClaim      string
	ProviderGroups   []string
}

type oauthTestServer struct {
	*testServer
	provider *oidctest.Provider
}

func newOAuthTestServer(t *testing.T, opts oauthServerOptions) *oauthTestServer {
	t.Helper()

	if opts.ProvisioningMode == "" {
		opts.ProvisioningMode = "preprovisioned"
	}

	repo, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}
	if err := repo.MigrateUp(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen test server: %v", err)
	}
	baseURL := "http://" + listener.Addr().String()
	redirectURL := baseURL + "/oauth/callback"

	provider, err := oidctest.Start(oidctest.Config{
		RedirectURL: redirectURL,
		Groups:      append([]string(nil), opts.ProviderGroups...),
	})
	if err != nil {
		t.Fatalf("start oidc provider: %v", err)
	}
	adminGroup := opts.AdminGroup
	if adminGroup == "$client_id" {
		adminGroup = provider.ClientID
	}

	secret := []byte(testSecret)
	sessionMgr := session.NewManager(repo, secret)
	groupsClaim := opts.GroupsClaim
	if groupsClaim == "" {
		groupsClaim = "aud"
	}
	authCfg := &config.AuthConfig{
		PasswordEnabled:       opts.PasswordEnabled,
		OAuthEnabled:          true,
		OAuthIssuerURL:        provider.Issuer,
		OAuthClientID:         provider.ClientID,
		OAuthClientSecret:     provider.ClientSecret,
		OAuthRedirectURL:      redirectURL,
		OAuthScopes:           []string{"openid", "profile", "email"},
		OAuthGroupsClaim:      groupsClaim,
		OAuthUsernameClaims:   []string{"preferred_username", "email", "sub"},
		OAuthFullNameClaims:   []string{"name", "preferred_username", "email"},
		OAuthProvisioningMode: opts.ProvisioningMode,
		OAuthRequiredGroups:   append([]string(nil), opts.RequiredGroups...),
		OAuthAdminGroup:       adminGroup,
		OAuthStateTTLSeconds:  300,
	}
	oauthSvc, err := oauth.New(context.Background(), oauth.Config{
		Enabled:        authCfg.OAuthEnabled,
		IssuerURL:      authCfg.OAuthIssuerURL,
		ClientID:       authCfg.OAuthClientID,
		ClientSecret:   authCfg.OAuthClientSecret,
		RedirectURL:    authCfg.OAuthRedirectURL,
		Scopes:         authCfg.OAuthScopes,
		GroupsClaim:    authCfg.OAuthGroupsClaim,
		UsernameClaims: authCfg.OAuthUsernameClaims,
		FullNameClaims: authCfg.OAuthFullNameClaims,
		StateTTL:       time.Duration(authCfg.OAuthStateTTLSeconds) * time.Second,
	}, secret)
	if err != nil {
		t.Fatalf("create oauth service: %v", err)
	}

	mw := middleware.NewRegistry(sessionMgr, repo, authCfg.PasswordEnabled)
	b := broker.NewMemoryBroker()
	store := storage.NewMemStorage()

	if err := locale.LoadTranslations(); err != nil {
		t.Fatalf("load translations: %v", err)
	}
	docsService, err := docs.Load(docembed.ContentFS(), docembed.AssetsFS())
	if err != nil {
		t.Fatalf("load embedded docs: %v", err)
	}

	h := &handlers.Handler{
		Broker:         b,
		Repository:     repo,
		Storage:        store,
		SessionManager: sessionMgr,
		AuthConfig:     authCfg,
		OAuthService:   oauthSvc,
		DocsService:    docsService,
	}

	mux := routes.NewRouter(h, mw).RegisterRoutes()
	appMux := http.NewServeMux()
	appMux.Handle("/", mux)
	handler := locale.NewMiddleware(appMux, locale.Config{
		Default:   "en",
		Supported: []string{"en"},
	})

	ts := httptest.NewUnstartedServer(handler)
	ts.Listener = listener
	ts.Start()

	t.Cleanup(func() {
		ts.Close()
		provider.Close()
		b.Shutdown()
		_ = docsService.Close()
		repo.Close()
	})

	result := &testServer{Server: ts, repo: repo, storage: store}
	result.seedAdminAccount(t, testAdminUsername, testAdminPassword)
	return &oauthTestServer{
		testServer: result,
		provider:   provider,
	}
}

func oauthLogin(t *testing.T, page playwright.Page, baseURL, target, username, password string) {
	t.Helper()

	loginPage := baseURL + "/"
	successURL := baseURL + "/home"
	if target == "admin" {
		loginPage = baseURL + "/admin/login"
		successURL = baseURL + "/admin"
	}

	if _, err := page.Goto(loginPage); err != nil {
		t.Fatalf("goto login page: %v", err)
	}

	link := page.Locator("a[href*='/oauth/start'][href*='target=" + target + "']").First()
	if err := link.WaitFor(); err != nil {
		t.Fatalf("wait oauth start link: %v", err)
	}
	if err := link.Click(); err != nil {
		t.Fatalf("click oauth start link: %v", err)
	}

	if err := page.WaitForURL("**/login/username**"); err != nil {
		t.Fatalf("wait provider login page: %v", err)
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
	if err := page.WaitForURL(successURL); err != nil {
		t.Fatalf("wait success URL %q: %v", successURL, err)
	}
}
