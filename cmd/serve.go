package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	connect "connectrpc.com/connect"
	docembed "github.com/Y4shin/conference-tool/doc"
	adminv1connect "github.com/Y4shin/conference-tool/gen/go/conference/admin/v1/adminv1connect"
	agendav1connect "github.com/Y4shin/conference-tool/gen/go/conference/agenda/v1/agendav1connect"
	attendeesv1connect "github.com/Y4shin/conference-tool/gen/go/conference/attendees/v1/attendeesv1connect"
	committeesv1connect "github.com/Y4shin/conference-tool/gen/go/conference/committees/v1/committeesv1connect"
	meetingsv1connect "github.com/Y4shin/conference-tool/gen/go/conference/meetings/v1/meetingsv1connect"
	moderationv1connect "github.com/Y4shin/conference-tool/gen/go/conference/moderation/v1/moderationv1connect"
	sessionv1connect "github.com/Y4shin/conference-tool/gen/go/conference/session/v1/sessionv1connect"
	speakersv1connect "github.com/Y4shin/conference-tool/gen/go/conference/speakers/v1/speakersv1connect"
	votesv1connect "github.com/Y4shin/conference-tool/gen/go/conference/votes/v1/votesv1connect"
	apiconnect "github.com/Y4shin/conference-tool/internal/api/connect"
	apihttp "github.com/Y4shin/conference-tool/internal/api/http"
	"github.com/Y4shin/conference-tool/internal/broker"
	"github.com/Y4shin/conference-tool/internal/config"
	"github.com/Y4shin/conference-tool/internal/docs"
	"github.com/Y4shin/conference-tool/internal/handlers"
	"github.com/Y4shin/conference-tool/internal/locale"
	"github.com/Y4shin/conference-tool/internal/middleware"
	"github.com/Y4shin/conference-tool/internal/oauth"
	"github.com/Y4shin/conference-tool/internal/repository/sqlite"
	"github.com/Y4shin/conference-tool/internal/routes"
	adminservice "github.com/Y4shin/conference-tool/internal/services/admin"
	agendaservice "github.com/Y4shin/conference-tool/internal/services/agenda"
	attendeeservice "github.com/Y4shin/conference-tool/internal/services/attendees"
	committeeservice "github.com/Y4shin/conference-tool/internal/services/committees"
	meetingservice "github.com/Y4shin/conference-tool/internal/services/meetings"
	moderationservice "github.com/Y4shin/conference-tool/internal/services/moderation"
	sessionservice "github.com/Y4shin/conference-tool/internal/services/session"
	speakerservice "github.com/Y4shin/conference-tool/internal/services/speakers"
	voteservice "github.com/Y4shin/conference-tool/internal/services/votes"
	"github.com/Y4shin/conference-tool/internal/session"
	"github.com/Y4shin/conference-tool/internal/storage"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:          "serve",
	Short:        "Start the HTTP server",
	SilenceUsage: true,
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

		apiMux := http.NewServeMux()

		sessionAPIPath, sessionAPIHandler := sessionv1connect.NewSessionServiceHandler(
			apiconnect.NewSessionHandler(sessionservice.New(repo, sessionManager, cfg.Auth.PasswordEnabled)),
			connect.WithInterceptors(apiconnect.ErrorInterceptor()),
		)
		apiMux.Handle(sessionAPIPath, mw.Get("session")(sessionAPIHandler))

		committeeAPIPath, committeeAPIHandler := committeesv1connect.NewCommitteeServiceHandler(
			apiconnect.NewCommitteeHandler(committeeservice.New(repo)),
			connect.WithInterceptors(apiconnect.ErrorInterceptor()),
		)
		apiMux.Handle(committeeAPIPath, mw.Get("session")(committeeAPIHandler))

		meetingAPIPath, meetingAPIHandler := meetingsv1connect.NewMeetingServiceHandler(
			apiconnect.NewMeetingHandler(meetingservice.New(repo)),
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
			apiconnect.NewAgendaHandler(agendaservice.New(repo, b)),
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

		apiMux.Handle("GET /realtime/meetings/{meetingId}/events",
			apihttp.NewMeetingEventsHandler(b),
		)

		appMux.Handle("/api/", http.StripPrefix("/api", apiMux))

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
		sigCh := make(chan os.Signal, 2)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(sigCh)

		server := &http.Server{
			Addr:    addr,
			Handler: handlerWithLocale,
		}

		serverErr := make(chan error, 1)
		go func() {
			err := server.ListenAndServe()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				serverErr <- err
				return
			}
			serverErr <- nil
		}()

		select {
		case err := <-serverErr:
			return err
		case sig := <-sigCh:
			slog.Info("shutdown signal received", "signal", sig.String())
		}

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		shutdownErrCh := make(chan error, 1)
		go func() {
			shutdownErrCh <- server.Shutdown(shutdownCtx)
		}()

		forceClosed := false
		select {
		case err := <-shutdownErrCh:
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					slog.Warn("graceful shutdown timeout reached; forcing immediate close")
					if closeErr := server.Close(); closeErr != nil && !errors.Is(closeErr, http.ErrServerClosed) {
						return fmt.Errorf("force close after graceful shutdown timeout: %w", closeErr)
					}
				} else {
					return fmt.Errorf("graceful shutdown failed: %w", err)
				}
			}
		case sig := <-sigCh:
			slog.Warn("second shutdown signal received; forcing immediate close", "signal", sig.String())
			forceClosed = true
			shutdownCancel()
			if err := server.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return fmt.Errorf("force close failed: %w", err)
			}
		case err := <-serverErr:
			return err
		}

		if err := <-serverErr; err != nil {
			return err
		}
		if forceClosed {
			slog.Info("server shutdown complete (forced)")
		} else {
			slog.Info("server shutdown complete")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
