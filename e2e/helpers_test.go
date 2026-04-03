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

	connect "connectrpc.com/connect"
	docembed "github.com/Y4shin/open-caucus/doc"
	adminv1connect "github.com/Y4shin/open-caucus/gen/go/conference/admin/v1/adminv1connect"
	agendav1connect "github.com/Y4shin/open-caucus/gen/go/conference/agenda/v1/agendav1connect"
	attendeesv1connect "github.com/Y4shin/open-caucus/gen/go/conference/attendees/v1/attendeesv1connect"
	committeesv1connect "github.com/Y4shin/open-caucus/gen/go/conference/committees/v1/committeesv1connect"
	docsv1connect "github.com/Y4shin/open-caucus/gen/go/conference/docs/v1/docsv1connect"
	meetingsv1connect "github.com/Y4shin/open-caucus/gen/go/conference/meetings/v1/meetingsv1connect"
	moderationv1connect "github.com/Y4shin/open-caucus/gen/go/conference/moderation/v1/moderationv1connect"
	sessionv1connect "github.com/Y4shin/open-caucus/gen/go/conference/session/v1/sessionv1connect"
	speakersv1connect "github.com/Y4shin/open-caucus/gen/go/conference/speakers/v1/speakersv1connect"
	votesv1connect "github.com/Y4shin/open-caucus/gen/go/conference/votes/v1/votesv1connect"
	apiconnect "github.com/Y4shin/open-caucus/internal/api/connect"
	apihttp "github.com/Y4shin/open-caucus/internal/api/http"
	"golang.org/x/crypto/bcrypt"

	"github.com/Y4shin/open-caucus/internal/broker"
	"github.com/Y4shin/open-caucus/internal/config"
	"github.com/Y4shin/open-caucus/internal/docs"
	"github.com/Y4shin/open-caucus/internal/locale"
	"github.com/Y4shin/open-caucus/internal/middleware"
	"github.com/Y4shin/open-caucus/internal/oauth"
	"github.com/Y4shin/open-caucus/internal/repository"
	"github.com/Y4shin/open-caucus/internal/repository/sqlite"
	adminservice "github.com/Y4shin/open-caucus/internal/services/admin"
	agendaservice "github.com/Y4shin/open-caucus/internal/services/agenda"
	attendeeservice "github.com/Y4shin/open-caucus/internal/services/attendees"
	committeeservice "github.com/Y4shin/open-caucus/internal/services/committees"
	meetingservice "github.com/Y4shin/open-caucus/internal/services/meetings"
	moderationservice "github.com/Y4shin/open-caucus/internal/services/moderation"
	sessionservice "github.com/Y4shin/open-caucus/internal/services/session"
	speakerservice "github.com/Y4shin/open-caucus/internal/services/speakers"
	voteservice "github.com/Y4shin/open-caucus/internal/services/votes"
	"github.com/Y4shin/open-caucus/internal/session"
	"github.com/Y4shin/open-caucus/internal/storage"
	webassets "github.com/Y4shin/open-caucus/internal/web"
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
	authCfg := &config.AuthConfig{
		PasswordEnabled:       true,
		OAuthEnabled:          false,
		OAuthProvisioningMode: "preprovisioned",
		OAuthGroupsClaim:      "groups",
		OAuthUsernameClaims:   []string{"preferred_username", "email", "sub"},
		OAuthFullNameClaims:   []string{"name", "preferred_username", "email"},
		OAuthStateTTLSeconds:  300,
	}
	oauthSvc, err := oauth.New(context.Background(), oauth.Config{Enabled: false}, secret)
	if err != nil {
		t.Fatalf("create oauth service: %v", err)
	}
	mw := middleware.NewRegistry(sessionMgr, repo, authCfg.PasswordEnabled)
	b := broker.NewMemoryBroker()
	store := storage.NewMemStorage()

	if err := locale.LoadTranslations(); err != nil {
		t.Fatalf("load translations: %v", err)
	}
	docsService, err := docs.Load(docembed.ContentFS(), docembed.AssetsFS())
	if err != nil {
		t.Fatalf("load embedded docs: %v", err)
	}

	oauthH := &apihttp.OAuthHandler{
		OAuthService:   oauthSvc,
		Repository:     repo,
		SessionManager: sessionMgr,
		AuthConfig:     authCfg,
	}

	apiMux := http.NewServeMux()

	sessionAPIPath, sessionAPIHandler := sessionv1connect.NewSessionServiceHandler(
		apiconnect.NewSessionHandler(sessionservice.New(repo, sessionMgr, authCfg.PasswordEnabled, authCfg.OAuthEnabled)),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(sessionAPIPath, mw.Get("session")(sessionAPIHandler))

	committeeAPIPath, committeeAPIHandler := committeesv1connect.NewCommitteeServiceHandler(
		apiconnect.NewCommitteeHandler(committeeservice.New(repo)),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(committeeAPIPath, mw.Get("session")(committeeAPIHandler))

	meetingAPIPath, meetingAPIHandler := meetingsv1connect.NewMeetingServiceHandler(
		apiconnect.NewMeetingHandler(meetingservice.New(repo), b),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(meetingAPIPath, mw.Get("session")(meetingAPIHandler))

	moderationAPIPath, moderationAPIHandler := moderationv1connect.NewModerationServiceHandler(
		apiconnect.NewModerationHandler(moderationservice.New(repo, b)),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(moderationAPIPath, mw.Get("session")(moderationAPIHandler))

	attendeeAPIPath, attendeeAPIHandler := attendeesv1connect.NewAttendeeServiceHandler(
		apiconnect.NewAttendeeHandler(attendeeservice.New(repo, sessionMgr, b)),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(attendeeAPIPath, mw.Get("session")(attendeeAPIHandler))

	agendaAPIPath, agendaAPIHandler := agendav1connect.NewAgendaServiceHandler(
		apiconnect.NewAgendaHandler(agendaservice.New(repo, b, store)),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(agendaAPIPath, mw.Get("session")(agendaAPIHandler))

	speakerAPIPath, speakerAPIHandler := speakersv1connect.NewSpeakerServiceHandler(
		apiconnect.NewSpeakerHandler(speakerservice.New(repo, b)),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(speakerAPIPath, mw.Get("session")(speakerAPIHandler))

	voteAPIPath, voteAPIHandler := votesv1connect.NewVoteServiceHandler(
		apiconnect.NewVoteHandler(voteservice.New(repo, b)),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(voteAPIPath, mw.Get("session")(voteAPIHandler))

	adminAPIPath, adminAPIHandler := adminv1connect.NewAdminServiceHandler(
		apiconnect.NewAdminHandler(adminservice.New(repo)),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(adminAPIPath, mw.Get("session")(adminAPIHandler))

	docsAPIPath, docsAPIHandler := docsv1connect.NewDocsServiceHandler(
		apiconnect.NewDocsHandler(docsService),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(docsAPIPath, docsAPIHandler)

	apiMux.Handle("POST /committee/{slug}/meeting/{meetingId}/agenda-point/{agendaPointId}/attachments",
		mw.Get("session")(apihttp.NewAttachmentUploadHandler(repo, store)),
	)
	apiMux.Handle("GET /docs/assets/{assetPath...}", apihttp.NewDocsAssetHandler(docsService))

	spaHandler := webassets.NewSPAHandler()

	appHandler := mw.Get("session")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/"):
			http.StripPrefix("/api", apiMux).ServeHTTP(w, r)
		case r.Method == http.MethodGet && (r.URL.Path == "/admin" || strings.HasPrefix(r.URL.Path, "/admin/")) && r.URL.Path != "/admin/login":
			sd, ok := session.GetSession(r.Context())
			if !ok || sd == nil || sd.AccountID == nil || !sd.IsAdmin || sd.IsExpired() {
				http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
				return
			}
			spaHandler.ServeHTTP(w, r)
		case r.URL.Path == "/oauth/start" && r.Method == http.MethodGet:
			apihttp.NewOAuthStartHandler(oauthH).ServeHTTP(w, r)
		case r.URL.Path == "/oauth/callback" && r.Method == http.MethodGet:
			apihttp.NewOAuthCallbackHandler(oauthH).ServeHTTP(w, r)
		case r.URL.Path == "/docs/assets" || strings.HasPrefix(r.URL.Path, "/docs/assets/"):
			apihttp.NewDocsAssetHandler(docsService).ServeHTTP(w, r)
		case r.URL.Path == "/blobs" || strings.HasPrefix(r.URL.Path, "/blobs/"):
			apihttp.NewBlobDownloadHandler(repo, store).ServeHTTP(w, r)
		case r.Method == http.MethodGet || r.Method == http.MethodHead:
			spaHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	}))

	handler := locale.NewMiddleware(appHandler, locale.Config{
		Default:   "en",
		Supported: []string{"en"},
	})
	ts := httptest.NewServer(handler)

	t.Cleanup(func() {
		ts.Close()
		b.Shutdown()
		_ = docsService.Close()
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
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
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
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password for admin: %v", err)
	}
	account, err := ts.repo.CreateAccount(context.Background(), username, username, string(hash))
	if err != nil {
		t.Fatalf("create admin account %q: %v", username, err)
	}
	if err := ts.repo.SetAccountIsAdmin(context.Background(), account.ID, true); err != nil {
		t.Fatalf("set admin flag for %q: %v", username, err)
	}
}

// seedAccount creates a non-admin account.
func (ts *testServer) seedAccount(t *testing.T, username, password, fullName string) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password for account: %v", err)
	}
	if _, err := ts.repo.CreateAccount(context.Background(), username, fullName, string(hash)); err != nil {
		t.Fatalf("create account %q: %v", username, err)
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

// setMeetingModerator assigns or clears the designated moderator for a meeting.
func (ts *testServer) setMeetingModerator(t *testing.T, slug, meetingName string, moderatorID *int64) {
	t.Helper()
	meetingIDStr := ts.getMeetingID(t, slug, meetingName)
	var mid int64
	fmt.Sscanf(meetingIDStr, "%d", &mid)
	if err := ts.repo.SetMeetingModerator(context.Background(), mid, moderatorID); err != nil {
		t.Fatalf("set meeting moderator: %v", err)
	}
}
