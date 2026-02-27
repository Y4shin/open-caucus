package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Y4shin/conference-tool/internal/oidcdev"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

const oidcDevUsersExampleYAML = `users:
  - id: "id1"
    username: "alice"
    password: "alice"
    first_name: "Alice"
    last_name: "Example"
    email: "alice@example.local"
    email_verified: true
    preferred_language: "en"
    is_admin: false
    groups:
      - "committee-a"
      - "committee-a-chair"
      - "ca-admin"

  - id: "id2"
    username: "bob"
    password: "bob"
    first_name: "Bob"
    last_name: "Example"
    email: "bob@example.local"
    email_verified: true
    preferred_language: "en"
    is_admin: false
    groups:
      - "committee-b"
`

var oidcDevGenerateUsersCmd = &cobra.Command{
	Use:   "generate-users",
	Short: "Generate an example OIDC users YAML file for local development",
	RunE: func(cmd *cobra.Command, args []string) error {
		usersFile, _ := cmd.Flags().GetString("users-file")
		force, _ := cmd.Flags().GetBool("force")

		if _, err := os.Stat(".env"); err == nil {
			if loadErr := godotenv.Load(".env"); loadErr != nil {
				return fmt.Errorf("load .env: %w", loadErr)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat .env: %w", err)
		}

		if strings.TrimSpace(usersFile) == "" {
			usersFile = strings.TrimSpace(os.Getenv("OIDC_DEV_USERS_FILE"))
		}
		if strings.TrimSpace(usersFile) == "" {
			usersFile = oidcdev.DefaultUsersFilePath
		}

		if _, err := os.Stat(usersFile); err == nil && !force {
			return fmt.Errorf("%s already exists (use --force to overwrite)", usersFile)
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat %s: %w", usersFile, err)
		}

		dir := filepath.Dir(usersFile)
		if dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("create directory %s: %w", dir, err)
			}
		}

		if err := os.WriteFile(usersFile, []byte(oidcDevUsersExampleYAML), 0o600); err != nil {
			return fmt.Errorf("write %s: %w", usersFile, err)
		}
		fmt.Printf("Wrote example OIDC users file: %s\n", usersFile)
		return nil
	},
}

func init() {
	oidcDevGenerateUsersCmd.Flags().String("users-file", "", "Path to users YAML (defaults to OIDC_DEV_USERS_FILE or dev/users.yaml)")
	oidcDevGenerateUsersCmd.Flags().Bool("force", false, "Overwrite file if it already exists")
	oidcDevCmd.AddCommand(oidcDevGenerateUsersCmd)
}
