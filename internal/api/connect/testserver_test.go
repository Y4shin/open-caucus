package apiconnect

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"

	connect "connectrpc.com/connect"
	"golang.org/x/crypto/bcrypt"

	committeesv1connect "github.com/Y4shin/conference-tool/gen/go/conference/committees/v1/committeesv1connect"
	meetingsv1connect "github.com/Y4shin/conference-tool/gen/go/conference/meetings/v1/meetingsv1connect"
	moderationv1connect "github.com/Y4shin/conference-tool/gen/go/conference/moderation/v1/moderationv1connect"
	sessionv1connect "github.com/Y4shin/conference-tool/gen/go/conference/session/v1/sessionv1connect"
	apihttp "github.com/Y4shin/conference-tool/internal/api/http"
	"github.com/Y4shin/conference-tool/internal/broker"
	"github.com/Y4shin/conference-tool/internal/locale"
	"github.com/Y4shin/conference-tool/internal/middleware"
	"github.com/Y4shin/conference-tool/internal/repository"
	"github.com/Y4shin/conference-tool/internal/repository/sqlite"
	committeeservice "github.com/Y4shin/conference-tool/internal/services/committees"
	meetingservice "github.com/Y4shin/conference-tool/internal/services/meetings"
	moderationservice "github.com/Y4shin/conference-tool/internal/services/moderation"
	sessionservice "github.com/Y4shin/conference-tool/internal/services/session"
	"github.com/Y4shin/conference-tool/internal/session"
)

type combinedTestServer struct {
	server *httptest.Server
	repo   repository.Repository
	broker broker.Broker
}

type combinedTestClient struct {
	session    sessionv1connect.SessionServiceClient
	committees committeesv1connect.CommitteeServiceClient
	meetings   meetingsv1connect.MeetingServiceClient
	moderation moderationv1connect.ModerationServiceClient
}

func newCombinedAPITestServer(t *testing.T) *combinedTestServer {
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
	b := broker.NewMemoryBroker()

	mux := http.NewServeMux()

	// Session service
	sessionSvc := sessionservice.New(repo, sessionManager, true)
	sessionPath, sessionHandler := sessionv1connect.NewSessionServiceHandler(
		NewSessionHandler(sessionSvc),
		connect.WithInterceptors(ErrorInterceptor()),
	)
	mux.Handle("/api"+sessionPath, mw.Get("session")(http.StripPrefix("/api", sessionHandler)))

	// Committee service
	committeePath, committeeHandler := committeesv1connect.NewCommitteeServiceHandler(
		NewCommitteeHandler(committeeservice.New(repo)),
		connect.WithInterceptors(ErrorInterceptor()),
	)
	mux.Handle("/api"+committeePath, mw.Get("session")(http.StripPrefix("/api", committeeHandler)))

	// Meeting service
	meetingPath, meetingHandler := meetingsv1connect.NewMeetingServiceHandler(
		NewMeetingHandler(meetingservice.New(repo)),
		connect.WithInterceptors(ErrorInterceptor()),
	)
	mux.Handle("/api"+meetingPath, mw.Get("session")(http.StripPrefix("/api", meetingHandler)))

	// Moderation service
	moderationPath, moderationHandler := moderationv1connect.NewModerationServiceHandler(
		NewModerationHandler(moderationservice.New(repo, b)),
		connect.WithInterceptors(ErrorInterceptor()),
	)
	mux.Handle("/api"+moderationPath, mw.Get("session")(http.StripPrefix("/api", moderationHandler)))

	// Realtime SSE endpoint
	mux.Handle("GET /api/realtime/meetings/{meetingId}/events",
		apihttp.NewMeetingEventsHandler(b),
	)

	server := httptest.NewServer(locale.NewMiddleware(mux, locale.Config{
		Default:   "en",
		Supported: []string{"en", "de"},
	}))

	t.Cleanup(func() {
		server.Close()
		b.Shutdown()
		repo.Close()
	})

	return &combinedTestServer{server: server, repo: repo, broker: b}
}

func newCombinedTestClient(t *testing.T, ts *combinedTestServer) *combinedTestClient {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create cookie jar: %v", err)
	}
	httpClient := &http.Client{Jar: jar}
	base := ts.server.URL + "/api"
	return &combinedTestClient{
		session:    sessionv1connect.NewSessionServiceClient(httpClient, base),
		committees: committeesv1connect.NewCommitteeServiceClient(httpClient, base),
		meetings:   meetingsv1connect.NewMeetingServiceClient(httpClient, base),
		moderation: moderationv1connect.NewModerationServiceClient(httpClient, base),
	}
}

func (ts *combinedTestServer) seedCommittee(t *testing.T, name, slug string) {
	t.Helper()
	if err := ts.repo.CreateCommitteeWithSlug(context.Background(), name, slug); err != nil {
		t.Fatalf("seed committee %q: %v", slug, err)
	}
}

func (ts *combinedTestServer) seedUser(t *testing.T, slug, username, password, fullName, role string) {
	t.Helper()
	hash := hashPassword(t, password)
	committeeID, err := ts.repo.GetCommitteeIDBySlug(context.Background(), slug)
	if err != nil {
		t.Fatalf("get committee id for %q: %v", slug, err)
	}
	if err := ts.repo.CreateUser(context.Background(), committeeID, username, hash, fullName, false, role); err != nil {
		t.Fatalf("seed user %q: %v", username, err)
	}
}

func (ts *combinedTestServer) seedMeeting(t *testing.T, slug, name string, signupOpen bool) int64 {
	t.Helper()
	committeeID, err := ts.repo.GetCommitteeIDBySlug(context.Background(), slug)
	if err != nil {
		t.Fatalf("get committee id for %q: %v", slug, err)
	}
	if err := ts.repo.CreateMeeting(context.Background(), committeeID, name, "", "secret", signupOpen); err != nil {
		t.Fatalf("seed meeting %q: %v", name, err)
	}
	meetings, err := ts.repo.ListMeetingsForCommittee(context.Background(), slug, 1, 0)
	if err != nil || len(meetings) == 0 {
		t.Fatalf("get seeded meeting: %v", err)
	}
	return meetings[0].ID
}

func hashPassword(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	return string(hash)
}
