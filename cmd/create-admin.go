package cmd

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"

	"github.com/Y4shin/conference-tool/internal/config"
	"github.com/Y4shin/conference-tool/internal/repository/sqlite"
	"github.com/spf13/cobra"
)

var createAdminCmd = &cobra.Command{
	Use:   "create-admin",
	Short: "Create or promote an account to admin",
	Long: `Creates a new account with admin privileges, or promotes an existing account to admin.
If the account already exists, its password is not changed.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")

		if username == "" || password == "" {
			return fmt.Errorf("--username and --password are required")
		}

		cfg, err := config.LoadConfigSelective([]config.ConfigGroup{config.DatabaseGroup})
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		repo, err := sqlite.New(cfg.Database.Path)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer repo.Close()

		if err := repo.MigrateUp(); err != nil {
			return fmt.Errorf("run migrations: %w", err)
		}

		ctx := context.Background()

		// Check if account already exists
		account, err := repo.GetAccountByUsername(ctx, username)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("lookup account: %w", err)
			}

			// Account doesn't exist — create it
			hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				return fmt.Errorf("hash password: %w", err)
			}

			account, err = repo.CreateAccount(ctx, username, username, string(hash))
			if err != nil {
				return fmt.Errorf("create account: %w", err)
			}
			log.Printf("Created account %q", username)
		} else {
			log.Printf("Account %q already exists — password unchanged", username)
		}

		// Set admin flag
		if err := repo.SetAccountIsAdmin(ctx, account.ID, true); err != nil {
			return fmt.Errorf("set admin: %w", err)
		}

		log.Printf("Account %q is now an admin", username)
		return nil
	},
}

func init() {
	createAdminCmd.Flags().String("username", "", "Admin account username (required)")
	createAdminCmd.Flags().String("password", "", "Admin account password, only used when creating a new account (required)")
	rootCmd.AddCommand(createAdminCmd)
}
