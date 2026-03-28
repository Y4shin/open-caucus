package cmd

import (
	"net/http"

	"github.com/Y4shin/conference-tool/internal/handlers"
	"github.com/Y4shin/conference-tool/internal/routes"
	"github.com/spf13/cobra"
)

var serveLegacyCmd = &cobra.Command{
	Use:          "serve-legacy",
	Short:        "Start the legacy HTMX/Templ server",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		rt, err := loadServeRuntime()
		if err != nil {
			return err
		}
		defer rt.Close()

		return runHTTPServer(rt.cfg, newLegacyServer(rt))
	},
}

func newLegacyServer(rt *serveRuntime) http.Handler {
	h := &handlers.Handler{
		Broker:         rt.broker,
		Repository:     rt.repo,
		Storage:        rt.store,
		SessionManager: rt.sessionManager,
		AuthConfig:     rt.cfg.Auth,
		OAuthService:   rt.oauthService,
		DocsService:    rt.docsService,
	}
	return routes.NewRouter(h, rt.middleware).RegisterRoutes()
}

func init() {
	rootCmd.AddCommand(serveLegacyCmd)
}
