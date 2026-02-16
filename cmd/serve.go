package cmd

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Y4shin/conference-tool/internal/broker"
	"github.com/Y4shin/conference-tool/internal/config"
	"github.com/Y4shin/conference-tool/internal/handlers"
	"github.com/Y4shin/conference-tool/internal/middleware"
	"github.com/Y4shin/conference-tool/internal/routes"
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

		b := broker.NewMemoryBroker()
		defer b.Shutdown()

		handler := handlers.NewHandler(b)
		mw := middleware.NewRegistry()

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
