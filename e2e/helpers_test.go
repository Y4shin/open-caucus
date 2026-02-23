//go:build e2e

package e2e_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"golang.org/x/crypto/bcrypt"

	"github.com/Y4shin/conference-tool/internal/broker"
	"github.com/Y4shin/conference-tool/internal/handlers"
	"github.com/Y4shin/conference-tool/internal/locale"
	"github.com/Y4shin/conference-tool/internal/middleware"
	"github.com/Y4shin/conference-tool/internal/repository"
	"github.com/Y4shin/conference-tool/internal/repository/sqlite"
	"github.com/Y4shin/conference-tool/internal/routes"
	"github.com/Y4shin/conference-tool/internal/session"
	"github.com/Y4shin/conference-tool/internal/storage"
	"github.com/Y4shin/conference-tool/internal/templates"
)

const (
	testAdminUsername = "admin"
	testAdminPassword = "admin-password"
	testSecret        = "test-secret-32-bytes-exactly!!!!"
)

type testServer struct {
	*httptest.Server
	repo    repository.Repository
	storage storage.Service
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
	mw := middleware.NewRegistry(sessionMgr, repo)
	b := broker.NewMemoryBroker()
	store := storage.NewMemStorage()

	h := &handlers.Handler{
		Broker:         b,
		Repository:     repo,
		Storage:        store,
		SessionManager: sessionMgr,
	}

	if err := locale.LoadTranslations(); err != nil {
		t.Fatalf("load translations: %v", err)
	}

	mux := routes.NewRouter(h, mw).RegisterRoutes()
	appMux := http.NewServeMux()
	appMux.Handle("/", mux)
	handler := locale.NewMiddleware(appMux, locale.Config{
		Default:   "en",
		Supported: []string{"en"},
	})
	handler = templ.NewCSSMiddleware(handler, templates.GlobalCSSClasses()...)
	ts := httptest.NewServer(handler)

	t.Cleanup(func() {
		ts.Close()
		b.Shutdown()
		repo.Close()
	})

	result := &testServer{Server: ts, repo: repo, storage: store}

	// Seed the default admin account so adminLogin() always works
	result.seedAdminAccount(t, testAdminUsername, testAdminPassword)

	return result
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

// seedAdminAccount creates an account with admin privileges.
func (ts *testServer) seedAdminAccount(t *testing.T, username, password string) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password for admin: %v", err)
	}
	account, err := ts.repo.CreateAccount(context.Background(), username, string(hash))
	if err != nil {
		t.Fatalf("create admin account %q: %v", username, err)
	}
	if err := ts.repo.SetAccountIsAdmin(context.Background(), account.ID, true); err != nil {
		t.Fatalf("set admin flag for %q: %v", username, err)
	}
}

// seedMeeting creates a meeting in the given committee with signup closed.
func (ts *testServer) seedMeeting(t *testing.T, slug, name, description string) {
	t.Helper()
	ts.seedMeetingFull(t, slug, name, description, false)
}

// seedMeetingOpen creates a meeting in the given committee with signup open.
func (ts *testServer) seedMeetingOpen(t *testing.T, slug, name, description string) {
	t.Helper()
	ts.seedMeetingFull(t, slug, name, description, true)
}

// seedMeetingFull creates a meeting with an explicit signup_open value.
func (ts *testServer) seedMeetingFull(t *testing.T, slug, name, description string, signupOpen bool) {
	t.Helper()
	committeeID, err := ts.repo.GetCommitteeIDBySlug(context.Background(), slug)
	if err != nil {
		t.Fatalf("get committee ID for %q: %v", slug, err)
	}
	if err := ts.repo.CreateMeeting(context.Background(), committeeID, name, description, "test-meeting-secret", signupOpen); err != nil {
		t.Fatalf("seed meeting %q: %v", name, err)
	}
}

// seedAgendaPoint creates an agenda point for the given meeting and returns its string ID.
func (ts *testServer) seedAgendaPoint(t *testing.T, slug, meetingName, title string) string {
	t.Helper()
	meetingIDStr := ts.getMeetingID(t, slug, meetingName)
	var mid int64
	fmt.Sscanf(meetingIDStr, "%d", &mid)
	ap, err := ts.repo.CreateAgendaPoint(context.Background(), mid, title)
	if err != nil {
		t.Fatalf("seed agenda point %q: %v", title, err)
	}
	return strconv.FormatInt(ap.ID, 10)
}

// seedMotion creates a motion (with a dummy PDF blob) under the given agenda point and returns its string ID.
func (ts *testServer) seedMotion(t *testing.T, agendaPointIDStr, motionTitle string) string {
	t.Helper()
	apID, err := strconv.ParseInt(agendaPointIDStr, 10, 64)
	if err != nil {
		t.Fatalf("parse agenda point ID %q: %v", agendaPointIDStr, err)
	}
	storagePath, sizeBytes, err := ts.storage.Store("document.pdf", "application/pdf", strings.NewReader("dummy pdf content"))
	if err != nil {
		t.Fatalf("store blob for motion %q: %v", motionTitle, err)
	}
	blob, err := ts.repo.CreateBlob(context.Background(), "document.pdf", "application/pdf", sizeBytes, storagePath)
	if err != nil {
		t.Fatalf("create blob for motion %q: %v", motionTitle, err)
	}
	motion, err := ts.repo.CreateMotion(context.Background(), apID, blob.ID, motionTitle)
	if err != nil {
		t.Fatalf("seed motion %q: %v", motionTitle, err)
	}
	return strconv.FormatInt(motion.ID, 10)
}

// seedAttachment creates an attachment (with a dummy blob) under the given agenda point and returns its string ID.
func (ts *testServer) seedAttachment(t *testing.T, agendaPointIDStr string, label *string) string {
	t.Helper()
	apID, err := strconv.ParseInt(agendaPointIDStr, 10, 64)
	if err != nil {
		t.Fatalf("parse agenda point ID %q: %v", agendaPointIDStr, err)
	}
	storagePath, sizeBytes, err := ts.storage.Store("document.pdf", "application/pdf", strings.NewReader("dummy pdf content"))
	if err != nil {
		t.Fatalf("store blob for attachment: %v", err)
	}
	blob, err := ts.repo.CreateBlob(context.Background(), "document.pdf", "application/pdf", sizeBytes, storagePath)
	if err != nil {
		t.Fatalf("create blob for attachment: %v", err)
	}
	attachment, err := ts.repo.CreateAttachment(context.Background(), apID, blob.ID, label)
	if err != nil {
		t.Fatalf("seed attachment: %v", err)
	}
	return strconv.FormatInt(attachment.ID, 10)
}

// getMeetingID returns the string ID of the first meeting with the given name in the committee.
func (ts *testServer) getMeetingID(t *testing.T, slug, name string) string {
	t.Helper()
	meetings, err := ts.repo.ListMeetingsForCommittee(context.Background(), slug, 100, 0)
	if err != nil {
		t.Fatalf("list meetings for %q: %v", slug, err)
	}
	for _, m := range meetings {
		if m.Name == name {
			return strconv.FormatInt(m.ID, 10)
		}
	}
	t.Fatalf("meeting %q not found in committee %q", name, slug)
	return ""
}

// setAttendeeChair updates the attendee's chair flag.
func (ts *testServer) setAttendeeChair(t *testing.T, attendeeID int64, isChair bool) {
	t.Helper()
	if err := ts.repo.SetAttendeeIsChair(context.Background(), attendeeID, isChair); err != nil {
		t.Fatalf("set attendee %d chair=%v: %v", attendeeID, isChair, err)
	}
}
