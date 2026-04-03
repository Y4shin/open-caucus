package apiconnect

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"

	connect "connectrpc.com/connect"
	"golang.org/x/crypto/bcrypt"

	sessionv1 "github.com/Y4shin/open-caucus/gen/go/conference/session/v1"
	sessionv1connect "github.com/Y4shin/open-caucus/gen/go/conference/session/v1/sessionv1connect"
	"github.com/Y4shin/open-caucus/internal/locale"
	"github.com/Y4shin/open-caucus/internal/middleware"
	"github.com/Y4shin/open-caucus/internal/repository"
	"github.com/Y4shin/open-caucus/internal/repository/sqlite"
	sessionservice "github.com/Y4shin/open-caucus/internal/services/session"
	"github.com/Y4shin/open-caucus/internal/session"
)

const testSecret = "test-secret-32-bytes-exactly!!!!"

type apiTestServer struct {
	server *httptest.Server
	repo   repository.Repository
}

func newAPITestServer(t *testing.T) *apiTestServer {
	t.Helper()

	repo, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}
	if err := repo.MigrateUp(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := locale.LoadTranslations(); err != nil {
		t.Fatalf("load translations: %v", err)
	}

	sessionManager := session.NewManager(repo, []byte(testSecret))
	mw := middleware.NewRegistry(sessionManager, repo, true)
	service := sessionservice.New(repo, sessionManager, true, false)
	handler := NewSessionHandler(service)

	path, connectHandler := sessionv1connect.NewSessionServiceHandler(
		handler,
		connect.WithInterceptors(ErrorInterceptor()),
	)

	mux := http.NewServeMux()
	mux.Handle("/api"+path, mw.Get("session")(http.StripPrefix("/api", connectHandler)))

	server := httptest.NewServer(locale.NewMiddleware(mux, locale.Config{
		Default:   "en",
		Supported: []string{"en", "de"},
	}))

	t.Cleanup(func() {
		server.Close()
		repo.Close()
	})

	return &apiTestServer{
		server: server,
		repo:   repo,
	}
}

func (s *apiTestServer) client(t *testing.T) sessionv1connect.SessionServiceClient {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create cookie jar: %v", err)
	}
	httpClient := &http.Client{Jar: jar}
	return sessionv1connect.NewSessionServiceClient(httpClient, s.server.URL+"/api")
}

func (s *apiTestServer) seedCommittee(t *testing.T, name, slug string) {
	t.Helper()
	if err := s.repo.CreateCommitteeWithSlug(context.Background(), name, slug); err != nil {
		t.Fatalf("seed committee: %v", err)
	}
}

func (s *apiTestServer) seedUser(t *testing.T, slug, username, password, fullName, role string) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	committeeID, err := s.repo.GetCommitteeIDBySlug(context.Background(), slug)
	if err != nil {
		t.Fatalf("get committee id: %v", err)
	}
	if err := s.repo.CreateUser(context.Background(), committeeID, username, string(hash), fullName, false, role); err != nil {
		t.Fatalf("create user: %v", err)
	}
}

func TestSessionServiceGetSessionAnonymous(t *testing.T) {
	ts := newAPITestServer(t)
	client := ts.client(t)

	resp, err := client.GetSession(context.Background(), connect.NewRequest(&sessionv1.GetSessionRequest{}))
	if err != nil {
		t.Fatalf("get session: %v", err)
	}

	if resp.Msg.GetSession().GetAuthenticated() {
		t.Fatalf("expected anonymous session bootstrap")
	}
	if got := resp.Msg.GetSession().GetLocale(); got != "en" {
		t.Fatalf("unexpected locale: got %q want %q", got, "en")
	}
	if !resp.Msg.GetSession().GetPasswordEnabled() {
		t.Fatalf("expected password auth to be enabled in bootstrap")
	}
	if resp.Msg.GetSession().GetOauthEnabled() {
		t.Fatalf("expected oauth auth to be disabled in bootstrap")
	}
}

func TestSessionServiceLoginAndGetSession(t *testing.T) {
	ts := newAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	client := ts.client(t)

	loginResp, err := client.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass123",
	}))
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if !loginResp.Msg.GetSession().GetAuthenticated() {
		t.Fatalf("expected authenticated login bootstrap")
	}
	if got := loginResp.Msg.GetSession().GetRedirectTo(); got != "/home" {
		t.Fatalf("unexpected login redirect: got %q want %q", got, "/home")
	}
	if !loginResp.Msg.GetSession().GetPasswordEnabled() {
		t.Fatalf("expected password auth to be enabled after login")
	}

	resp, err := client.GetSession(context.Background(), connect.NewRequest(&sessionv1.GetSessionRequest{}))
	if err != nil {
		t.Fatalf("get session after login: %v", err)
	}

	sessionMsg := resp.Msg.GetSession()
	if !sessionMsg.GetAuthenticated() {
		t.Fatalf("expected authenticated session")
	}
	if sessionMsg.GetActor().GetActorKind() != "account" {
		t.Fatalf("unexpected actor kind: %q", sessionMsg.GetActor().GetActorKind())
	}
	if sessionMsg.GetActor().GetDisplayName() != "Chair Person" {
		t.Fatalf("unexpected actor display name: %q", sessionMsg.GetActor().GetDisplayName())
	}
	if len(sessionMsg.GetAvailableCommittees()) != 1 {
		t.Fatalf("unexpected committee count: got %d want 1", len(sessionMsg.GetAvailableCommittees()))
	}
	if got := sessionMsg.GetAvailableCommittees()[0].GetSlug(); got != "test-committee" {
		t.Fatalf("unexpected committee slug: %q", got)
	}
	if !sessionMsg.GetAvailableCommittees()[0].GetIsChairperson() {
		t.Fatalf("expected chairperson membership flag")
	}
}

func TestSessionServiceLogoutClearsSession(t *testing.T) {
	ts := newAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Regular Member", "member")
	client := ts.client(t)

	if _, err := client.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "member1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	logoutResp, err := client.Logout(context.Background(), connect.NewRequest(&sessionv1.LogoutRequest{}))
	if err != nil {
		t.Fatalf("logout: %v", err)
	}
	if !logoutResp.Msg.GetCleared() {
		t.Fatalf("expected cleared logout response")
	}

	resp, err := client.GetSession(context.Background(), connect.NewRequest(&sessionv1.GetSessionRequest{}))
	if err != nil {
		t.Fatalf("get session after logout: %v", err)
	}
	if resp.Msg.GetSession().GetAuthenticated() {
		t.Fatalf("expected anonymous session after logout")
	}
}
