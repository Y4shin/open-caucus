package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Y4shin/conference-tool/internal/oidcdev"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var oidcDevServeCmd = &cobra.Command{
	Use: "serve",
	Aliases: []string{
		"serve-oidc-dev",
	},
	Short: "Start a local OAuth/OIDC development provider",
	Long: `Starts an interactive local OIDC provider using the same OAUTH_* variables as the app.

Required shared env vars:
  OAUTH_ISSUER_URL
  OAUTH_CLIENT_ID
  OAUTH_CLIENT_SECRET
  OAUTH_REDIRECT_URL

OIDC dev server env vars:
  OIDC_DEV_USERS_FILE   (default: dev/oidc-users.yaml)
  OIDC_DEV_GROUPS_CLAIM (default: groups)
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := os.Stat(".env"); err == nil {
			if loadErr := godotenv.Load(".env"); loadErr != nil {
				return fmt.Errorf("load .env: %w", loadErr)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat .env: %w", err)
		}

		cfg, err := oidcdev.LoadConfigFromEnv()
		if err != nil {
			return err
		}

		if usersFile, _ := cmd.Flags().GetString("users-file"); usersFile != "" {
			cfg.UsersFile = usersFile
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		log.Printf("Local OIDC server ready. Stop with Ctrl+C.")
		return oidcdev.Run(ctx, cfg)
	},
}

func init() {
	oidcDevServeCmd.Flags().String("users-file", "", "Path to OIDC users YAML file (overrides OIDC_DEV_USERS_FILE)")
	oidcDevCmd.AddCommand(oidcDevServeCmd)
}
