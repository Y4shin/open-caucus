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

	docembed "github.com/Y4shin/conference-tool/doc"
	"github.com/Y4shin/conference-tool/internal/broker"
	"github.com/Y4shin/conference-tool/internal/config"
	"github.com/Y4shin/conference-tool/internal/docs"
	"github.com/Y4shin/conference-tool/internal/handlers"
	"github.com/Y4shin/conference-tool/internal/locale"
	"github.com/Y4shin/conference-tool/internal/middleware"
	"github.com/Y4shin/conference-tool/internal/oauth"
	"github.com/Y4shin/conference-tool/internal/repository"
	"github.com/Y4shin/conference-tool/internal/repository/sqlite"
	"github.com/Y4shin/conference-tool/internal/routes"
	"github.com/Y4shin/conference-tool/internal/session"
	"github.com/Y4shin/conference-tool/internal/storage"
	oidctest "github.com/Y4shin/conference-tool/internal/testsupport/oidc"
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
	h := &handlers.Handler{
		Broker:         b,
		Repository:     repo,
		Storage:        store,
		SessionManager: sessionManager,
		AuthConfig:     authCfg,
		OAuthService:   oauthSvc,
		DocsService:    docsService,
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

	mux := routes.NewRouter(h, mw).RegisterRoutes()
	appMux := http.NewServeMux()
	appMux.Handle("/", mux)
	handler := locale.NewMiddleware(appMux, locale.Config{
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
