package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

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
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := os.Stat(".env"); err == nil {
			if loadErr := godotenv.Load(".env"); loadErr != nil {
				return fmt.Errorf("load .env: %w", loadErr)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat .env: %w", err)
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Initialize database repository
		repo, err := sqlite.New(cfg.Database.Path)
		if err != nil {
			return fmt.Errorf("failed to create repository: %w", err)
		}
		defer repo.Close()
		slog.Info("database opened", "path", cfg.Database.Path)

		// Run migrations
		if err := repo.MigrateUp(); err != nil {
			return fmt.Errorf("failed to run migrations: %w", err)
		}
		slog.Info("database migrations applied")

		// Initialize file storage
		store, err := storage.NewDirStorage(cfg.Application.StorageDir)
		if err != nil {
			return fmt.Errorf("failed to create storage: %w", err)
		}
		slog.Info("file storage initialized", "dir", cfg.Application.StorageDir)

		// Initialize broker
		b := broker.NewMemoryBroker()
		defer b.Shutdown()

		// Initialize session manager with repository as store
		sessionManager := session.NewManager(repo, []byte(cfg.Application.SessionSecret))

		oauthService, err := oauth.New(context.Background(), oauth.Config{
			Enabled:        cfg.Auth.OAuthEnabled,
			IssuerURL:      cfg.Auth.OAuthIssuerURL,
			ClientID:       cfg.Auth.OAuthClientID,
			ClientSecret:   cfg.Auth.OAuthClientSecret,
			RedirectURL:    cfg.Auth.OAuthRedirectURL,
			Scopes:         cfg.Auth.OAuthScopes,
			GroupsClaim:    cfg.Auth.OAuthGroupsClaim,
			UsernameClaims: cfg.Auth.OAuthUsernameClaims,
			FullNameClaims: cfg.Auth.OAuthFullNameClaims,
			StateTTL:       time.Duration(cfg.Auth.OAuthStateTTLSeconds) * time.Second,
		}, []byte(cfg.Application.SessionSecret))
		if err != nil {
			return fmt.Errorf("failed to initialize oauth service: %w", err)
		}
		slog.Info("auth configured", "password_enabled", cfg.Auth.PasswordEnabled, "oauth_enabled", cfg.Auth.OAuthEnabled)

		// Initialize middleware registry with session manager
		mw := middleware.NewRegistry(sessionManager, repo, cfg.Auth.PasswordEnabled)

		// Load translations (must happen before any request is served).
		if err := locale.LoadTranslations(); err != nil {
			return fmt.Errorf("failed to load translations: %w", err)
		}
		docsService, err := docs.Load(docembed.ContentFS(), docembed.AssetsFS())
		if err != nil {
			return fmt.Errorf("failed to load embedded docs: %w", err)
		}
		defer docsService.Close()

		// Initialize handlers with repository, broker, storage, and session manager
		handler := &handlers.Handler{
			Broker:         b,
			Repository:     repo,
			Storage:        store,
			SessionManager: sessionManager,
			AuthConfig:     cfg.Auth,
			OAuthService:   oauthService,
			DocsService:    docsService,
		}

		// Create router
		router := routes.NewRouter(handler, mw)
		mux := router.RegisterRoutes()

		// Compose app routes and locale switcher in a top-level mux.
		appMux := http.NewServeMux()
		appMux.Handle("/", mux)

		// Locale switcher: POST /locale sets the "locale" cookie and redirects.
		appMux.HandleFunc("POST /locale", func(w http.ResponseWriter, r *http.Request) {
			lang := r.FormValue("lang")
			supported := map[string]bool{"en": true, "de": true}
			if !supported[lang] {
				http.Error(w, "unsupported locale", http.StatusBadRequest)
				return
			}
			http.SetCookie(w, &http.Cookie{
				Name:     "locale",
				Value:    lang,
				Path:     "/",
				MaxAge:   365 * 24 * 60 * 60,
				SameSite: http.SameSiteLaxMode,
			})
			ref := r.Header.Get("Referer")
			if ref == "" {
				ref = "/"
			}
			http.Redirect(w, r, ref, http.StatusSeeOther)
		})

		handlerWithLocale := locale.NewMiddleware(appMux, locale.Config{
			Default:   "en",
			Supported: []string{"en", "de"},
		})

		addr := fmt.Sprintf("%s:%d", cfg.Application.Host, cfg.Application.Port)
		slog.Info("starting server", "addr", addr, "env", cfg.Application.Environment)

		return http.ListenAndServe(addr, handlerWithLocale)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
