package oidcdev

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	exampleop "github.com/zitadel/oidc/v4/example/server/exampleop"
	examplestorage "github.com/zitadel/oidc/v4/example/server/storage"
	"github.com/zitadel/oidc/v4/pkg/oidc"
	"github.com/zitadel/oidc/v4/pkg/op"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

const (
	DefaultUsersFilePath = "dev/users.yaml"
	DefaultGroupsClaim   = "groups"
)

// Config controls the local OIDC development server.
type Config struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	UsersFile    string
	GroupsClaim  string
}

// LoadConfigFromEnv reads OIDC server configuration from environment variables.
//
// Shared auth variables with app server:
//   - OAUTH_ISSUER_URL
//   - OAUTH_CLIENT_ID
//   - OAUTH_CLIENT_SECRET
//   - OAUTH_REDIRECT_URL
//
// Local OIDC dev variables:
//   - OIDC_DEV_USERS_FILE (default: dev/users.yaml)
//   - OIDC_DEV_GROUPS_CLAIM (default: groups)
func LoadConfigFromEnv() (Config, error) {
	cfg := Config{
		IssuerURL:    strings.TrimSpace(os.Getenv("OAUTH_ISSUER_URL")),
		ClientID:     strings.TrimSpace(os.Getenv("OAUTH_CLIENT_ID")),
		ClientSecret: strings.TrimSpace(os.Getenv("OAUTH_CLIENT_SECRET")),
		RedirectURL:  strings.TrimSpace(os.Getenv("OAUTH_REDIRECT_URL")),
		UsersFile:    strings.TrimSpace(os.Getenv("OIDC_DEV_USERS_FILE")),
		GroupsClaim:  strings.TrimSpace(os.Getenv("OIDC_DEV_GROUPS_CLAIM")),
	}
	if cfg.UsersFile == "" {
		cfg.UsersFile = DefaultUsersFilePath
	}
	if cfg.GroupsClaim == "" {
		cfg.GroupsClaim = DefaultGroupsClaim
	}
	if cfg.IssuerURL == "" {
		return Config{}, fmt.Errorf("OAUTH_ISSUER_URL is required")
	}
	if cfg.ClientID == "" {
		return Config{}, fmt.Errorf("OAUTH_CLIENT_ID is required")
	}
	if cfg.ClientSecret == "" {
		return Config{}, fmt.Errorf("OAUTH_CLIENT_SECRET is required")
	}
	if cfg.RedirectURL == "" {
		return Config{}, fmt.Errorf("OAUTH_REDIRECT_URL is required")
	}
	if err := validateIssuerURL(cfg.IssuerURL); err != nil {
		return Config{}, err
	}
	if err := validateAbsoluteURL(cfg.RedirectURL, "OAUTH_REDIRECT_URL"); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run starts the local OIDC development server and blocks until context cancellation.
func Run(ctx context.Context, cfg Config) error {
	userStore, groupsByUserID, err := loadUsersFromYAML(cfg.UsersFile, cfg.ClientID)
	if err != nil {
		return err
	}

	clients := map[string]*examplestorage.Client{
		cfg.ClientID: examplestorage.WebClient(cfg.ClientID, cfg.ClientSecret, cfg.RedirectURL),
	}
	baseStorage := examplestorage.NewStorageWithClients(userStore, clients)
	storage := &groupClaimsStorage{
		Storage:        baseStorage,
		groupsByUserID: groupsByUserID,
		groupsClaim:    cfg.GroupsClaim,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	router := exampleop.SetupServer(cfg.IssuerURL, storage, logger, true)

	addr, err := listenAddrFromIssuer(cfg.IssuerURL)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	logger.Info("starting local oidc server",
		"issuer", cfg.IssuerURL,
		"addr", addr,
		"client_id", cfg.ClientID,
		"redirect_url", cfg.RedirectURL,
		"users_file", cfg.UsersFile,
		"groups_claim", cfg.GroupsClaim,
	)

	errCh := make(chan error, 1)
	go func() {
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case serveErr := <-errCh:
		return serveErr
	}
}

type yamlUsersFile struct {
	Users []yamlUser `yaml:"users"`
}

type yamlUser struct {
	ID                string   `yaml:"id"`
	Username          string   `yaml:"username"`
	Password          string   `yaml:"password"`
	FirstName         string   `yaml:"first_name"`
	LastName          string   `yaml:"last_name"`
	Email             string   `yaml:"email"`
	EmailVerified     *bool    `yaml:"email_verified"`
	Phone             string   `yaml:"phone"`
	PhoneVerified     *bool    `yaml:"phone_verified"`
	PreferredLanguage string   `yaml:"preferred_language"`
	IsAdmin           bool     `yaml:"is_admin"`
	Groups            []string `yaml:"groups"`
}

type yamlUserStore struct {
	clientID       string
	byID           map[string]*examplestorage.User
	byUsername     map[string]*examplestorage.User
	groupsByUserID map[string][]string
}

func loadUsersFromYAML(path, clientID string) (*yamlUserStore, map[string][]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read oidc users file %q: %w", path, err)
	}
	var parsed yamlUsersFile
	if err := yaml.Unmarshal(raw, &parsed); err != nil {
		return nil, nil, fmt.Errorf("parse oidc users file %q: %w", path, err)
	}
	if len(parsed.Users) == 0 {
		return nil, nil, fmt.Errorf("oidc users file %q has no users", path)
	}

	store := &yamlUserStore{
		clientID:       clientID,
		byID:           make(map[string]*examplestorage.User, len(parsed.Users)),
		byUsername:     make(map[string]*examplestorage.User, len(parsed.Users)),
		groupsByUserID: make(map[string][]string, len(parsed.Users)),
	}

	for idx, item := range parsed.Users {
		lineInfo := fmt.Sprintf("users[%d]", idx)
		if strings.TrimSpace(item.ID) == "" {
			return nil, nil, fmt.Errorf("%s.id is required", lineInfo)
		}
		if strings.TrimSpace(item.Username) == "" {
			return nil, nil, fmt.Errorf("%s.username is required", lineInfo)
		}
		if strings.TrimSpace(item.Password) == "" {
			return nil, nil, fmt.Errorf("%s.password is required", lineInfo)
		}
		if _, exists := store.byID[item.ID]; exists {
			return nil, nil, fmt.Errorf("duplicate user id %q in %s", item.ID, lineInfo)
		}
		if _, exists := store.byUsername[item.Username]; exists {
			return nil, nil, fmt.Errorf("duplicate username %q in %s", item.Username, lineInfo)
		}

		langTag := language.English
		if strings.TrimSpace(item.PreferredLanguage) != "" {
			parsedLang, err := language.Parse(item.PreferredLanguage)
			if err != nil {
				return nil, nil, fmt.Errorf("%s.preferred_language invalid: %w", lineInfo, err)
			}
			langTag = parsedLang
		}

		emailVerified := false
		if item.EmailVerified != nil {
			emailVerified = *item.EmailVerified
		}
		phoneVerified := false
		if item.PhoneVerified != nil {
			phoneVerified = *item.PhoneVerified
		}

		user := &examplestorage.User{
			ID:                item.ID,
			Username:          item.Username,
			Password:          item.Password,
			FirstName:         item.FirstName,
			LastName:          item.LastName,
			Email:             item.Email,
			EmailVerified:     emailVerified,
			Phone:             item.Phone,
			PhoneVerified:     phoneVerified,
			PreferredLanguage: langTag,
			IsAdmin:           item.IsAdmin,
		}

		store.byID[user.ID] = user
		store.byUsername[user.Username] = user
		store.groupsByUserID[user.ID] = compactStrings(item.Groups)
	}

	return store, store.groupsByUserID, nil
}

func (s *yamlUserStore) GetUserByID(id string) *examplestorage.User {
	return s.byID[id]
}

func (s *yamlUserStore) GetUserByUsername(username string) *examplestorage.User {
	return s.byUsername[username]
}

func (s *yamlUserStore) ExampleClientID() string {
	return s.clientID
}

type groupClaimsStorage struct {
	*examplestorage.Storage
	groupsByUserID map[string][]string
	groupsClaim    string
}

func (s *groupClaimsStorage) GetPrivateClaimsFromScopes(ctx context.Context, userID, clientID string, scopes []string) (map[string]any, error) {
	claims, err := s.Storage.GetPrivateClaimsFromScopes(ctx, userID, clientID, scopes)
	if err != nil {
		return nil, err
	}
	groups := s.groupsByUserID[userID]
	if len(groups) == 0 {
		return claims, nil
	}
	if claims == nil {
		claims = make(map[string]any)
	}
	claims[s.groupsClaim] = groups
	return claims, nil
}

func (s *groupClaimsStorage) SetUserinfoFromRequest(ctx context.Context, userinfo *oidc.UserInfo, token op.IDTokenRequest, scopes []string) error {
	if err := s.Storage.SetUserinfoFromRequest(ctx, userinfo, token, scopes); err != nil {
		return err
	}
	s.appendGroupsClaim(userinfo, token.GetSubject())
	return nil
}

func (s *groupClaimsStorage) SetUserinfoFromToken(ctx context.Context, userinfo *oidc.UserInfo, tokenID, subject, origin string) error {
	if err := s.Storage.SetUserinfoFromToken(ctx, userinfo, tokenID, subject, origin); err != nil {
		return err
	}
	s.appendGroupsClaim(userinfo, subject)
	return nil
}

func (s *groupClaimsStorage) appendGroupsClaim(userinfo *oidc.UserInfo, userID string) {
	groups := s.groupsByUserID[userID]
	if len(groups) == 0 {
		return
	}
	userinfo.AppendClaims(s.groupsClaim, groups)
}

func validateIssuerURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("OAUTH_ISSUER_URL invalid: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("OAUTH_ISSUER_URL must be an absolute URL")
	}
	if u.Path != "" && u.Path != "/" {
		return fmt.Errorf("OAUTH_ISSUER_URL path must be empty or '/': got %q", u.Path)
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("OAUTH_ISSUER_URL must not include query or fragment")
	}
	return nil
}

func validateAbsoluteURL(raw, envName string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%s invalid: %w", envName, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("%s must be an absolute URL", envName)
	}
	return nil
}

func listenAddrFromIssuer(issuer string) (string, error) {
	u, err := url.Parse(issuer)
	if err != nil {
		return "", fmt.Errorf("parse issuer URL: %w", err)
	}
	host := u.Hostname()
	port := u.Port()
	if host == "" {
		return "", fmt.Errorf("issuer host is empty")
	}
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	return host + ":" + port, nil
}

func compactStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
