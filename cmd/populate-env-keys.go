package cmd

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var oidcDevPopulateEnvCmd = &cobra.Command{
	Use:     "populate-env",
	Aliases: []string{"populate-env-keys"},
	Short:   "Create/update a ready-to-use local .env for app + dev OIDC",
	Long: `Creates or updates an env file with generated values for:
  - SESSION_SECRET
  - OAUTH_CLIENT_ID
  - OAUTH_CLIENT_SECRET

If the env file does not exist, it is initialized from .env.example when available.
Then it writes local-development defaults so the app server and the dev OIDC server
work together out of the box.

Existing non-empty values are kept unless --force is provided.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		envFile, _ := cmd.Flags().GetString("env-file")
		force, _ := cmd.Flags().GetBool("force")

		content, err := ensureEnvFile(envFile)
		if err != nil {
			return err
		}

		sessionSecret, err := randomToken(48)
		if err != nil {
			return fmt.Errorf("generate SESSION_SECRET: %w", err)
		}
		clientIDSuffix, err := randomToken(10)
		if err != nil {
			return fmt.Errorf("generate OAUTH_CLIENT_ID suffix: %w", err)
		}
		clientSecret, err := randomToken(48)
		if err != nil {
			return fmt.Errorf("generate OAUTH_CLIENT_SECRET: %w", err)
		}

		updates := map[string]string{
			"HOST":                    "127.0.0.1",
			"PORT":                    "8080",
			"SESSION_SECRET":          sessionSecret,
			"AUTH_PASSWORD_ENABLED":   "true",
			"AUTH_OAUTH_ENABLED":      "true",
			"OAUTH_ISSUER_URL":        "http://127.0.0.1:9096",
			"OAUTH_CLIENT_ID":         "conference-tool-dev-" + clientIDSuffix,
			"OAUTH_CLIENT_SECRET":     clientSecret,
			"OAUTH_REDIRECT_URL":      "http://127.0.0.1:8080/oauth/callback",
			"OAUTH_GROUPS_CLAIM":      "groups",
			"OAUTH_ADMIN_GROUP":       "ca-admin",
			"OAUTH_PROVISIONING_MODE": "auto_create",
			"OIDC_DEV_USERS_FILE":     "dev/users.yaml",
			"OIDC_DEV_GROUPS_CLAIM":   "groups",
		}
		placeholderByKey := map[string]string{
			"HOST":                    "0.0.0.0",
			"PORT":                    "8080",
			"SESSION_SECRET":          "change-this-to-a-random-32-character-string",
			"AUTH_PASSWORD_ENABLED":   "true",
			"AUTH_OAUTH_ENABLED":      "false",
			"OAUTH_ISSUER_URL":        "",
			"OAUTH_CLIENT_ID":         "",
			"OAUTH_CLIENT_SECRET":     "",
			"OAUTH_REDIRECT_URL":      "",
			"OAUTH_GROUPS_CLAIM":      "groups",
			"OAUTH_ADMIN_GROUP":       "",
			"OAUTH_PROVISIONING_MODE": "preprovisioned",
			"OIDC_DEV_USERS_FILE":     "dev/users.yaml",
			"OIDC_DEV_GROUPS_CLAIM":   "groups",
		}

		newContent, changedKeys, err := applyEnvUpdates(content, updates, placeholderByKey, force)
		if err != nil {
			return err
		}

		if err := os.WriteFile(envFile, []byte(newContent), 0o600); err != nil {
			return fmt.Errorf("write %s: %w", envFile, err)
		}

		if len(changedKeys) == 0 {
			fmt.Printf("No keys changed in %s (use --force to overwrite existing values)\n", envFile)
			return nil
		}
		fmt.Printf("Updated %s:\n", envFile)
		for _, key := range changedKeys {
			fmt.Printf("  - %s\n", key)
		}
		return nil
	},
}

func init() {
	oidcDevPopulateEnvCmd.Flags().String("env-file", ".env", "Path to env file to populate")
	oidcDevPopulateEnvCmd.Flags().Bool("force", false, "Overwrite existing non-empty values")
	oidcDevCmd.AddCommand(oidcDevPopulateEnvCmd)
}

func ensureEnvFile(envFile string) (string, error) {
	raw, err := os.ReadFile(envFile)
	if err == nil {
		return string(raw), nil
	}
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("read %s: %w", envFile, err)
	}

	examplePath := filepath.Join(filepath.Dir(envFile), ".env.example")
	exampleRaw, exampleErr := os.ReadFile(examplePath)
	if exampleErr == nil {
		if writeErr := os.WriteFile(envFile, exampleRaw, 0o600); writeErr != nil {
			return "", fmt.Errorf("create %s from %s: %w", envFile, examplePath, writeErr)
		}
		return string(exampleRaw), nil
	}
	if !os.IsNotExist(exampleErr) {
		return "", fmt.Errorf("read %s: %w", examplePath, exampleErr)
	}

	if writeErr := os.WriteFile(envFile, []byte(""), 0o600); writeErr != nil {
		return "", fmt.Errorf("create %s: %w", envFile, writeErr)
	}
	return "", nil
}

func applyEnvUpdates(content string, updates map[string]string, placeholders map[string]string, force bool) (string, []string, error) {
	lines := strings.Split(content, "\n")

	type keyEntry struct {
		lineIdx int
		value   string
	}
	entries := make(map[string]keyEntry, len(updates))

	scanner := bufio.NewScanner(strings.NewReader(content))
	lineIdx := 0
	for scanner.Scan() {
		line := scanner.Text()
		key, value, ok := parseEnvAssignment(line)
		if ok {
			if _, tracked := updates[key]; tracked {
				entries[key] = keyEntry{lineIdx: lineIdx, value: value}
			}
		}
		lineIdx++
	}
	if err := scanner.Err(); err != nil {
		return "", nil, fmt.Errorf("scan env content: %w", err)
	}

	changed := make([]string, 0, len(updates))
	for key, nextValue := range updates {
		entry, exists := entries[key]
		shouldWrite := true
		if exists {
			current := strings.TrimSpace(entry.value)
			placeholder := strings.TrimSpace(placeholders[key])
			if !force && current != "" && (placeholder == "" || current != placeholder) {
				shouldWrite = false
			}
		}
		if !shouldWrite {
			continue
		}

		assignment := key + "=" + nextValue
		if exists {
			lines[entry.lineIdx] = assignment
		} else {
			lines = append(lines, assignment)
		}
		changed = append(changed, key)
	}

	return strings.Join(lines, "\n"), changed, nil
}

func parseEnvAssignment(line string) (key string, value string, ok bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", "", false
	}
	if strings.HasPrefix(trimmed, "export ") {
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "export "))
	}
	k, v, found := strings.Cut(trimmed, "=")
	if !found {
		return "", "", false
	}
	key = strings.TrimSpace(k)
	if key == "" {
		return "", "", false
	}
	return key, strings.TrimSpace(v), true
}

func randomToken(numBytes int) (string, error) {
	buf := make([]byte, numBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
