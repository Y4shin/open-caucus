package cmd

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Y4shin/conference-tool/internal/broker"
	"github.com/Y4shin/conference-tool/internal/config"
	"github.com/Y4shin/conference-tool/internal/handlers"
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
		repo, err := sqlite.New("conference.db")
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

		// Initialize admin session manager
		adminSessionManager := session.NewAdminSessionManager([]byte(cfg.Application.SessionSecret))

		// Initialize middleware registry with session managers
		mw := middleware.NewRegistry(sessionManager, adminSessionManager)

		// Initialize handlers with repository, broker, storage, and session managers
		handler := &handlers.Handler{
			Broker:              b,
			Repository:          repo,
			Storage:             store,
			SessionManager:      sessionManager,
			AdminSessionManager: adminSessionManager,
			AdminKey:            cfg.Application.AdminKey,
		}

		// Create router
		router := routes.NewRouter(handler, mw)
		mux := router.RegisterRoutes()

		addr := fmt.Sprintf("%s:%d", cfg.Application.Host, cfg.Application.Port)
		log.Printf("Starting server on %s", addr)

		return http.ListenAndServe(addr, mux)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
