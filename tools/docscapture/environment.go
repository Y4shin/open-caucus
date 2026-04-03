package docscapture

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	connect "connectrpc.com/connect"
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
	"github.com/Y4shin/open-caucus/internal/repository"
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

const captureSessionSecret = "docs-capture-session-secret-at-least-32-bytes"

type EnvironmentOptions struct {
	EnablePassword bool
	EnableOAuth    bool

	OAuthProvisioningMode string
	OAuthRequiredGroups   []string
	OAuthAdminGroup       string
	OAuthGroupsClaim      string
	OAuthProviderGroups   []string
}

func (o EnvironmentOptions) withDefaults() EnvironmentOptions {
	if !o.EnablePassword && !o.EnableOAuth {
		o.EnablePassword = true
	}
	if strings.TrimSpace(o.OAuthProvisioningMode) == "" {
		o.OAuthProvisioningMode = "preprovisioned"
	}
	if strings.TrimSpace(o.OAuthGroupsClaim) == "" {
		o.OAuthGroupsClaim = "groups"
	}
	return o
}

// Environment is a self-contained app runtime used by docs-capture scripts.
type Environment struct {
	server       *httptest.Server
	repo         repository.Repository
	broker       broker.Broker
	storage      storage.Service
	docsService  *docs.Service
	seeder       *Seeder
	authConfig   *config.AuthConfig
	oidcProvider *oidctest.Provider
}

// NewEnvironment starts an in-memory app server with migrated schema and optional
// in-process OIDC provider.
func NewEnvironment(rawOpts EnvironmentOptions) (*Environment, error) {
	opts := rawOpts.withDefaults()

	repo, err := sqlite.New(":memory:")
	if err != nil {
		return nil, fmt.Errorf("create in-memory repository: %w", err)
	}
	if err := repo.MigrateUp(); err != nil {
		repo.Close()
		return nil, fmt.Errorf("migrate in-memory repository: %w", err)
	}

	secret := []byte(captureSessionSecret)
	sessionManager := session.NewManager(repo, secret)
	b := broker.NewMemoryBroker()
	store := storage.NewMemStorage()

	authCfg := &config.AuthConfig{
		PasswordEnabled:       opts.EnablePassword,
		OAuthEnabled:          opts.EnableOAuth,
		OAuthProvisioningMode: opts.OAuthProvisioningMode,
		OAuthRequiredGroups:   append([]string(nil), opts.OAuthRequiredGroups...),
		OAuthAdminGroup:       strings.TrimSpace(opts.OAuthAdminGroup),
		OAuthGroupsClaim:      opts.OAuthGroupsClaim,
		OAuthUsernameClaims:   []string{"preferred_username", "email", "sub"},
		OAuthFullNameClaims:   []string{"name", "preferred_username", "email"},
		OAuthStateTTLSeconds:  300,
	}

	var (
		oauthSvc     *oauth.Service
		oidcProvider *oidctest.Provider
		listener     net.Listener
		baseURL      string
	)

	if opts.EnableOAuth {
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			b.Shutdown()
			repo.Close()
			return nil, fmt.Errorf("listen self-hosted app server: %w", err)
		}
		baseURL = "http://" + listener.Addr().String()
		redirectURL := baseURL + "/oauth/callback"

		oidcProvider, err = oidctest.Start(oidctest.Config{
			RedirectURL: redirectURL,
			Groups:      append([]string(nil), opts.OAuthProviderGroups...),
		})
		if err != nil {
			listener.Close()
			b.Shutdown()
			repo.Close()
			return nil, fmt.Errorf("start in-process oidc provider: %w", err)
		}

		authCfg.OAuthIssuerURL = oidcProvider.Issuer
		authCfg.OAuthClientID = oidcProvider.ClientID
		authCfg.OAuthClientSecret = oidcProvider.ClientSecret
		authCfg.OAuthRedirectURL = redirectURL
		authCfg.OAuthScopes = []string{"openid", "profile", "email"}
		if authCfg.OAuthGroupsClaim == "" {
			authCfg.OAuthGroupsClaim = "groups"
		}
		if authCfg.OAuthAdminGroup == "$client_id" {
			authCfg.OAuthAdminGroup = oidcProvider.ClientID
		}
	}

	oauthSvc, err = oauth.New(context.Background(), oauth.Config{
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
		if listener != nil {
			listener.Close()
		}
		if oidcProvider != nil {
			oidcProvider.Close()
		}
		b.Shutdown()
		repo.Close()
		return nil, fmt.Errorf("create oauth service: %w", err)
	}

	mw := middleware.NewRegistry(sessionManager, repo, authCfg.PasswordEnabled)

	docsService, err := docs.Load(docembed.ContentFS(), docembed.AssetsFS())
	if err != nil {
		if listener != nil {
			listener.Close()
		}
		if oidcProvider != nil {
			oidcProvider.Close()
		}
		b.Shutdown()
		repo.Close()
		return nil, fmt.Errorf("load embedded docs: %w", err)
	}
	if err := locale.LoadTranslations(); err != nil {
		if listener != nil {
			listener.Close()
		}
		if oidcProvider != nil {
			oidcProvider.Close()
		}
		b.Shutdown()
		repo.Close()
		return nil, fmt.Errorf("load translations: %w", err)
	}

	oauthHandler := &apihttp.OAuthHandler{
		OAuthService:   oauthSvc,
		Repository:     repo,
		SessionManager: sessionManager,
		AuthConfig:     authCfg,
	}

	apiMux := http.NewServeMux()

	sessionAPIPath, sessionAPIHandler := sessionv1connect.NewSessionServiceHandler(
		apiconnect.NewSessionHandler(sessionservice.New(repo, sessionManager, authCfg.PasswordEnabled, authCfg.OAuthEnabled)),
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
		apiconnect.NewAttendeeHandler(attendeeservice.New(repo, sessionManager, b)),
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
			apihttp.NewOAuthStartHandler(oauthHandler).ServeHTTP(w, r)
		case r.URL.Path == "/oauth/callback" && r.Method == http.MethodGet:
			apihttp.NewOAuthCallbackHandler(oauthHandler).ServeHTTP(w, r)
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
		Supported: []string{"en", "de"},
	})

	var ts *httptest.Server
	if listener != nil {
		ts = httptest.NewUnstartedServer(handler)
		ts.Listener = listener
		ts.Start()
	} else {
		ts = httptest.NewServer(handler)
	}

	env := &Environment{
		server:       ts,
		repo:         repo,
		broker:       b,
		storage:      store,
		docsService:  docsService,
		authConfig:   authCfg,
		oidcProvider: oidcProvider,
	}
	env.seeder = NewSeeder(repo, store)
	return env, nil
}

func (e *Environment) BaseURL() string {
	if e == nil || e.server == nil {
		return ""
	}
	return e.server.URL
}

func (e *Environment) Seeder() *Seeder {
	if e == nil {
		return nil
	}
	return e.seeder
}

func (e *Environment) AuthConfig() *config.AuthConfig {
	if e == nil {
		return nil
	}
	return e.authConfig
}

func (e *Environment) OIDCProvider() *oidctest.Provider {
	if e == nil {
		return nil
	}
	return e.oidcProvider
}

func (e *Environment) OAuthCredentials() (username, password string, ok bool) {
	if e == nil || e.oidcProvider == nil {
		return "", "", false
	}
	return e.oidcProvider.Username, e.oidcProvider.Password, true
}

func (e *Environment) Close() error {
	if e == nil {
		return nil
	}

	var errs []error
	if e.server != nil {
		e.server.Close()
		e.server = nil
	}
	if e.oidcProvider != nil {
		e.oidcProvider.Close()
		e.oidcProvider = nil
	}
	if e.broker != nil {
		e.broker.Shutdown()
		e.broker = nil
	}
	if e.docsService != nil {
		if err := e.docsService.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close docs service: %w", err))
		}
		e.docsService = nil
	}
	if e.repo != nil {
		if err := e.repo.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close repository: %w", err))
		}
		e.repo = nil
	}
	return errors.Join(errs...)
}
