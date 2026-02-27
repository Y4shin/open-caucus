package oidc

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http/httptest"
	"strings"
	"sync"

	exampleop "github.com/zitadel/oidc/v4/example/server/exampleop"
	examplestorage "github.com/zitadel/oidc/v4/example/server/storage"
	"github.com/zitadel/oidc/v4/pkg/oidc"
	"github.com/zitadel/oidc/v4/pkg/op"
)

const (
	defaultClientID     = "conference-tool-test-client"
	defaultClientSecret = "conference-tool-test-secret"
	defaultPassword     = "verysecure"
)

// Config configures an in-process OIDC provider for tests.
type Config struct {
	RedirectURL  string
	ClientID     string
	ClientSecret string
	Groups       []string
}

// Provider is a running in-process OIDC provider.
type Provider struct {
	server  *httptest.Server
	storage *groupStorage

	Issuer       string
	ClientID     string
	ClientSecret string
	Username     string
	Password     string
}

// Start boots an in-process OIDC provider using zitadel/oidc example OP storage.
func Start(cfg Config) (*Provider, error) {
	if strings.TrimSpace(cfg.RedirectURL) == "" {
		return nil, fmt.Errorf("redirect URL is required")
	}

	clientID := strings.TrimSpace(cfg.ClientID)
	if clientID == "" {
		clientID = defaultClientID
	}
	clientSecret := strings.TrimSpace(cfg.ClientSecret)
	if clientSecret == "" {
		clientSecret = defaultClientSecret
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen oidc test server: %w", err)
	}
	issuer := "http://" + listener.Addr().String()

	examplestorage.RegisterClients(
		examplestorage.WebClient(clientID, clientSecret, cfg.RedirectURL),
	)
	userStore := examplestorage.NewUserStore(issuer)
	storage := &groupStorage{
		Storage: examplestorage.NewStorage(userStore),
	}
	storage.SetGroups(cfg.Groups)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := exampleop.SetupServer(issuer, storage, logger, true)

	server := httptest.NewUnstartedServer(router)
	server.Listener = listener
	server.Start()

	usernameHost := strings.Split(strings.Split(issuer, "://")[1], ":")[0]
	return &Provider{
		server:       server,
		storage:      storage,
		Issuer:       issuer,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Username:     "test-user@" + usernameHost,
		Password:     defaultPassword,
	}, nil
}

// Close shuts down the provider server.
func (p *Provider) Close() {
	if p == nil || p.server == nil {
		return
	}
	p.server.Close()
}

// SetGroups updates the groups claim returned by the provider on new logins.
func (p *Provider) SetGroups(groups []string) {
	if p == nil || p.storage == nil {
		return
	}
	p.storage.SetGroups(groups)
}

type groupStorage struct {
	*examplestorage.Storage
	mu     sync.RWMutex
	groups []string
}

func (s *groupStorage) SetGroups(groups []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(groups) == 0 {
		s.groups = nil
		return
	}
	s.groups = append([]string(nil), groups...)
}

func (s *groupStorage) GetPrivateClaimsFromScopes(ctx context.Context, userID, clientID string, scopes []string) (map[string]any, error) {
	claims, err := s.Storage.GetPrivateClaimsFromScopes(ctx, userID, clientID, scopes)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	groups := append([]string(nil), s.groups...)
	s.mu.RUnlock()
	if len(groups) == 0 {
		return claims, nil
	}
	if claims == nil {
		claims = make(map[string]any)
	}
	claims["groups"] = groups
	return claims, nil
}

func (s *groupStorage) SetUserinfoFromRequest(ctx context.Context, userinfo *oidc.UserInfo, token op.IDTokenRequest, scopes []string) error {
	if err := s.Storage.SetUserinfoFromRequest(ctx, userinfo, token, scopes); err != nil {
		return err
	}
	s.appendGroups(userinfo)
	return nil
}

func (s *groupStorage) SetUserinfoFromToken(ctx context.Context, userinfo *oidc.UserInfo, tokenID, subject, origin string) error {
	if err := s.Storage.SetUserinfoFromToken(ctx, userinfo, tokenID, subject, origin); err != nil {
		return err
	}
	s.appendGroups(userinfo)
	return nil
}

func (s *groupStorage) appendGroups(userinfo *oidc.UserInfo) {
	s.mu.RLock()
	groups := append([]string(nil), s.groups...)
	s.mu.RUnlock()
	if len(groups) == 0 {
		return
	}
	userinfo.AppendClaims("groups", groups)
}
