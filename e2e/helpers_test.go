//go:build e2e

package e2e_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/Y4shin/conference-tool/internal/broker"
	"github.com/Y4shin/conference-tool/internal/handlers"
	"github.com/Y4shin/conference-tool/internal/middleware"
	"github.com/Y4shin/conference-tool/internal/repository"
	"github.com/Y4shin/conference-tool/internal/routes"
	"github.com/Y4shin/conference-tool/internal/session"
	"github.com/Y4shin/conference-tool/internal/repository/sqlite"
)

const (
	testAdminKey = "test-admin-key"
	testSecret   = "test-secret-32-bytes-exactly!!!!"
)

type testServer struct {
	*httptest.Server
	repo repository.Repository
}

// newTestServer boots a full HTTP server with an in-memory SQLite database,
// mirroring the wiring in cmd/serve.go.
func newTestServer(t *testing.T) *testServer {
	t.Helper()

	repo, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}
	if err := repo.MigrateUp(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	secret := []byte(testSecret)
	sessionMgr := session.NewManager(repo, secret)
	adminMgr := session.NewAdminSessionManager(secret)
	mw := middleware.NewRegistry(sessionMgr, adminMgr)
	b := broker.NewMemoryBroker()

	h := &handlers.Handler{
		Broker:              b,
		Repository:          repo,
		SessionManager:      sessionMgr,
		AdminSessionManager: adminMgr,
		AdminKey:            testAdminKey,
	}

	mux := routes.NewRouter(h, mw).RegisterRoutes()
	ts := httptest.NewServer(mux)

	t.Cleanup(func() {
		ts.Close()
		b.Shutdown()
		repo.Close()
	})

	return &testServer{Server: ts, repo: repo}
}

// seedCommittee creates a committee with the given name and slug.
func (ts *testServer) seedCommittee(t *testing.T, name, slug string) {
	t.Helper()
	if err := ts.repo.CreateCommitteeWithSlug(context.Background(), name, slug); err != nil {
		t.Fatalf("seed committee %q: %v", slug, err)
	}
}

// seedUser creates a user with the given credentials in the given committee.
func (ts *testServer) seedUser(t *testing.T, slug, username, password, fullName, role string) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	committeeID, err := ts.repo.GetCommitteeIDBySlug(context.Background(), slug)
	if err != nil {
		t.Fatalf("get committee ID for %q: %v", slug, err)
	}
	if err := ts.repo.CreateUser(context.Background(), committeeID, username, string(hash), fullName, false, role); err != nil {
		t.Fatalf("seed user %q: %v", username, err)
	}
}

// seedMeeting creates a meeting in the given committee.
func (ts *testServer) seedMeeting(t *testing.T, slug, name, description string) {
	t.Helper()
	committeeID, err := ts.repo.GetCommitteeIDBySlug(context.Background(), slug)
	if err != nil {
		t.Fatalf("get committee ID for %q: %v", slug, err)
	}
	if err := ts.repo.CreateMeeting(context.Background(), committeeID, name, description, "test-meeting-secret", false); err != nil {
		t.Fatalf("seed meeting %q: %v", name, err)
	}
}
