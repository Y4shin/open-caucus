//go:build e2e

package e2e_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	connect "connectrpc.com/connect"
	playwright "github.com/playwright-community/playwright-go"

	docembed "github.com/Y4shin/open-caucus/doc"
	adminv1connect "github.com/Y4shin/open-caucus/gen/go/conference/admin/v1/adminv1connect"
	agendav1connect "github.com/Y4shin/open-caucus/gen/go/conference/agenda/v1/agendav1connect"
	attendeesv1connect "github.com/Y4shin/open-caucus/gen/go/conference/attendees/v1/attendeesv1connect"
	committeesv1connect "github.com/Y4shin/open-caucus/gen/go/conference/committees/v1/committeesv1connect"
	docsv1connect "github.com/Y4shin/open-caucus/gen/go/conference/docs/v1/docsv1connect"
	meetingsv1connect "github.com/Y4shin/open-caucus/gen/go/conference/meetings/v1/meetingsv1connect"
	moderationv1connect "github.com/Y4shin/open-caucus/gen/go/conference/moderation/v1/moderationv1connect"
	sessionv1connect "github.com/Y4shin/open-caucus/gen/go/conference/session/v1/sessionv1connect"
	speakersv1connect "github.com/Y4shin/open-caucus/gen/go/conference/speakers/v1/speakersv1connect"
	votesv1connect "github.com/Y4shin/open-caucus/gen/go/conference/votes/v1/votesv1connect"
	apiconnect "github.com/Y4shin/open-caucus/internal/api/connect"
	apihttp "github.com/Y4shin/open-caucus/internal/api/http"
	"github.com/Y4shin/open-caucus/internal/broker"
	"github.com/Y4shin/open-caucus/internal/config"
	"github.com/Y4shin/open-caucus/internal/docs"
	"github.com/Y4shin/open-caucus/internal/locale"
	"github.com/Y4shin/open-caucus/internal/middleware"
	"github.com/Y4shin/open-caucus/internal/oauth"
	"github.com/Y4shin/open-caucus/internal/repository/sqlite"
	adminservice "github.com/Y4shin/open-caucus/internal/services/admin"
	agendaservice "github.com/Y4shin/open-caucus/internal/services/agenda"
	attendeeservice "github.com/Y4shin/open-caucus/internal/services/attendees"
	committeeservice "github.com/Y4shin/open-caucus/internal/services/committees"
	meetingservice "github.com/Y4shin/open-caucus/internal/services/meetings"
	moderationservice "github.com/Y4shin/open-caucus/internal/services/moderation"
	sessionservice "github.com/Y4shin/open-caucus/internal/services/session"
	speakerservice "github.com/Y4shin/open-caucus/internal/services/speakers"
	voteservice "github.com/Y4shin/open-caucus/internal/services/votes"
	"github.com/Y4shin/open-caucus/internal/session"
	"github.com/Y4shin/open-caucus/internal/storage"
	oidctest "github.com/Y4shin/open-caucus/internal/testsupport/oidc"
	webassets "github.com/Y4shin/open-caucus/internal/web"
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

	oauthH := &apihttp.OAuthHandler{
		OAuthService:   oauthSvc,
		Repository:     repo,
		SessionManager: sessionMgr,
		AuthConfig:     authCfg,
	}

	apiMux := http.NewServeMux()

	sessionAPIPath, sessionAPIHandler := sessionv1connect.NewSessionServiceHandler(
		apiconnect.NewSessionHandler(sessionservice.New(repo, sessionMgr, authCfg.PasswordEnabled, authCfg.OAuthEnabled)),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(sessionAPIPath, mw.Get("session")(sessionAPIHandler))

	committeeAPIPath, committeeAPIHandler := committeesv1connect.NewCommitteeServiceHandler(
		apiconnect.NewCommitteeHandler(committeeservice.New(repo)),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(committeeAPIPath, mw.Get("session")(committeeAPIHandler))

	meetingAPIPath, meetingAPIHandler := meetingsv1connect.NewMeetingServiceHandler(
		apiconnect.NewMeetingHandler(meetingservice.New(repo), b),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(meetingAPIPath, mw.Get("session")(meetingAPIHandler))

	moderationAPIPath, moderationAPIHandler := moderationv1connect.NewModerationServiceHandler(
		apiconnect.NewModerationHandler(moderationservice.New(repo, b)),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(moderationAPIPath, mw.Get("session")(moderationAPIHandler))

	attendeeAPIPath, attendeeAPIHandler := attendeesv1connect.NewAttendeeServiceHandler(
		apiconnect.NewAttendeeHandler(attendeeservice.New(repo, sessionMgr, b)),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(attendeeAPIPath, mw.Get("session")(attendeeAPIHandler))

	agendaAPIPath, agendaAPIHandler := agendav1connect.NewAgendaServiceHandler(
		apiconnect.NewAgendaHandler(agendaservice.New(repo, b, store)),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(agendaAPIPath, mw.Get("session")(agendaAPIHandler))

	speakerAPIPath, speakerAPIHandler := speakersv1connect.NewSpeakerServiceHandler(
		apiconnect.NewSpeakerHandler(speakerservice.New(repo, b)),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(speakerAPIPath, mw.Get("session")(speakerAPIHandler))

	voteAPIPath, voteAPIHandler := votesv1connect.NewVoteServiceHandler(
		apiconnect.NewVoteHandler(voteservice.New(repo, b)),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(voteAPIPath, mw.Get("session")(voteAPIHandler))

	adminAPIPath, adminAPIHandler := adminv1connect.NewAdminServiceHandler(
		apiconnect.NewAdminHandler(adminservice.New(repo)),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(adminAPIPath, mw.Get("session")(adminAPIHandler))

	docsAPIPath, docsAPIHandler := docsv1connect.NewDocsServiceHandler(
		apiconnect.NewDocsHandler(docsService),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(docsAPIPath, docsAPIHandler)

	apiMux.Handle("POST /committee/{slug}/meeting/{meetingId}/agenda-point/{agendaPointId}/attachments",
		mw.Get("session")(apihttp.NewAttachmentUploadHandler(repo, store)),
	)
	apiMux.Handle("GET /docs/assets/{assetPath...}", apihttp.NewDocsAssetHandler(docsService))

	spaHandler := webassets.NewSPAHandler()

	appHandler := mw.Get("session")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/"):
			http.StripPrefix("/api", apiMux).ServeHTTP(w, r)
		case r.Method == http.MethodGet && (r.URL.Path == "/admin" || strings.HasPrefix(r.URL.Path, "/admin/")) && r.URL.Path != "/admin/login":
			sd, ok := session.GetSession(r.Context())
			if !ok || sd == nil || sd.AccountID == nil || !sd.IsAdmin || sd.IsExpired() {
				http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
				return
			}
			spaHandler.ServeHTTP(w, r)
		case r.URL.Path == "/oauth/start" && r.Method == http.MethodGet:
			apihttp.NewOAuthStartHandler(oauthH).ServeHTTP(w, r)
		case r.URL.Path == "/oauth/callback" && r.Method == http.MethodGet:
			apihttp.NewOAuthCallbackHandler(oauthH).ServeHTTP(w, r)
		case r.URL.Path == "/docs/assets" || strings.HasPrefix(r.URL.Path, "/docs/assets/"):
			apihttp.NewDocsAssetHandler(docsService).ServeHTTP(w, r)
		case r.URL.Path == "/blobs" || strings.HasPrefix(r.URL.Path, "/blobs/"):
			apihttp.NewBlobDownloadHandler(repo, store).ServeHTTP(w, r)
		case r.Method == http.MethodGet || r.Method == http.MethodHead:
			spaHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	}))

	handler := locale.NewMiddleware(appHandler, locale.Config{
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
	} else {
		loginPage = baseURL + "/login"
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
