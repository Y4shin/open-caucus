package cmd

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Y4shin/conference-tool/internal/broker"
	"github.com/Y4shin/conference-tool/internal/config"
	"github.com/Y4shin/conference-tool/internal/handlers"
	"github.com/Y4shin/conference-tool/internal/locale"
	"github.com/Y4shin/conference-tool/internal/middleware"
	"github.com/Y4shin/conference-tool/internal/repository/sqlite"
	"github.com/Y4shin/conference-tool/internal/routes"
	"github.com/Y4shin/conference-tool/internal/session"
	"github.com/Y4shin/conference-tool/internal/storage"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	RunE: func(cmd *cobra.Command, args []string) error {
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

		// Run migrations
		if err := repo.MigrateUp(); err != nil {
			return fmt.Errorf("failed to run migrations: %w", err)
		}

		// Initialize file storage
		store, err := storage.NewDirStorage(cfg.Application.StorageDir)
		if err != nil {
			return fmt.Errorf("failed to create storage: %w", err)
		}

		// Initialize broker
		b := broker.NewMemoryBroker()
		defer b.Shutdown()

		// Initialize session manager with repository as store
		sessionManager := session.NewManager(repo, []byte(cfg.Application.SessionSecret))

		// Initialize middleware registry with session manager
		mw := middleware.NewRegistry(sessionManager, repo)

		// Initialize handlers with repository, broker, storage, and session manager
		handler := &handlers.Handler{
			Broker:         b,
			Repository:     repo,
			Storage:        store,
			SessionManager: sessionManager,
		}

		// Create router
		router := routes.NewRouter(handler, mw)
		mux := router.RegisterRoutes()

		// Load translations (must happen before any request is served).
		if err := locale.LoadTranslations(); err != nil {
			return fmt.Errorf("failed to load translations: %w", err)
		}

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
		log.Printf("Starting server on %s", addr)

		return http.ListenAndServe(addr, handlerWithLocale)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
