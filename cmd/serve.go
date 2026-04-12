package cmd

import (
	"context"
	"net/http"
	"strings"

	connect "connectrpc.com/connect"
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
	"github.com/Y4shin/open-caucus/internal/email"
	"github.com/Y4shin/open-caucus/internal/webhooks"
	adminservice "github.com/Y4shin/open-caucus/internal/services/admin"
	agendaservice "github.com/Y4shin/open-caucus/internal/services/agenda"
	attendeeservice "github.com/Y4shin/open-caucus/internal/services/attendees"
	committeeservice "github.com/Y4shin/open-caucus/internal/services/committees"
	memberservice "github.com/Y4shin/open-caucus/internal/services/members"
	meetingservice "github.com/Y4shin/open-caucus/internal/services/meetings"
	moderationservice "github.com/Y4shin/open-caucus/internal/services/moderation"
	sessionservice "github.com/Y4shin/open-caucus/internal/services/session"
	speakerservice "github.com/Y4shin/open-caucus/internal/services/speakers"
	voteservice "github.com/Y4shin/open-caucus/internal/services/votes"
	"github.com/Y4shin/open-caucus/internal/session"
	webassets "github.com/Y4shin/open-caucus/internal/web"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:          "serve",
	Short:        "Start the SPA/API server",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		rt, err := loadServeRuntime()
		if err != nil {
			return err
		}
		defer rt.Close()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if len(rt.cfg.Webhook.URLs) > 0 {
			d := webhooks.New(rt.cfg.Webhook)
			d.Start(ctx, rt.broker)
		}

		return runHTTPServer(rt.cfg, newSPAServer(rt))
	},
}

func newSPAServer(rt *serveRuntime) http.Handler {
	spaHandler := webassets.NewSPAHandler()
	apiMux := newAPIMux(rt)
	oauthH := &apihttp.OAuthHandler{
		OAuthService:   rt.oauthService,
		Repository:     rt.repo,
		SessionManager: rt.sessionManager,
		AuthConfig:     rt.cfg.Auth,
	}

	return rt.middleware.Get("session")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/"):
			http.StripPrefix("/api", apiMux).ServeHTTP(w, r)
			return
		case r.Method == http.MethodGet && (r.URL.Path == "/admin" || strings.HasPrefix(r.URL.Path, "/admin/")) && r.URL.Path != "/admin/login":
			sd, ok := session.GetSession(r.Context())
			if !ok || sd == nil || sd.AccountID == nil || !sd.IsAdmin || sd.IsExpired() {
				http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
				return
			}
			spaHandler.ServeHTTP(w, r)
			return
		case r.URL.Path == "/oauth/start" && r.Method == http.MethodGet:
			apihttp.NewOAuthStartHandler(oauthH).ServeHTTP(w, r)
			return
		case r.URL.Path == "/oauth/callback" && r.Method == http.MethodGet:
			apihttp.NewOAuthCallbackHandler(oauthH).ServeHTTP(w, r)
			return
		case r.URL.Path == "/locale" && r.Method == http.MethodPost:
			handleLocaleSwitch(w, r)
			return
		case r.URL.Path == "/docs/assets" || strings.HasPrefix(r.URL.Path, "/docs/assets/"):
			apihttp.NewDocsAssetHandler(rt.docsService).ServeHTTP(w, r)
			return
		case r.URL.Path == "/blobs" || strings.HasPrefix(r.URL.Path, "/blobs/"):
			apihttp.NewBlobDownloadHandler(rt.repo, rt.store).ServeHTTP(w, r)
			return
		case r.Method == http.MethodGet || r.Method == http.MethodHead:
			spaHandler.ServeHTTP(w, r)
			return
		default:
			http.NotFound(w, r)
			return
		}
	}))
}

func newAPIMux(rt *serveRuntime) *http.ServeMux {
	apiMux := http.NewServeMux()

	sessionAPIPath, sessionAPIHandler := sessionv1connect.NewSessionServiceHandler(
		apiconnect.NewSessionHandler(sessionservice.New(rt.repo, rt.sessionManager, rt.cfg.Auth.PasswordEnabled, rt.cfg.Auth.OAuthEnabled)),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(sessionAPIPath, rt.middleware.Get("session")(sessionAPIHandler))

	committeeAPIPath, committeeAPIHandler := committeesv1connect.NewCommitteeServiceHandler(
		apiconnect.NewCommitteeHandler(committeeservice.New(rt.repo, email.NewSender(rt.cfg.Email).Enabled()), memberservice.New(rt.repo, email.NewSender(rt.cfg.Email))),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(committeeAPIPath, rt.middleware.Get("session")(committeeAPIHandler))

	meetingAPIPath, meetingAPIHandler := meetingsv1connect.NewMeetingServiceHandler(
		apiconnect.NewMeetingHandler(meetingservice.New(rt.repo), rt.broker),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(meetingAPIPath, rt.middleware.Get("session")(meetingAPIHandler))

	moderationAPIPath, moderationAPIHandler := moderationv1connect.NewModerationServiceHandler(
		apiconnect.NewModerationHandler(moderationservice.New(rt.repo, rt.broker)),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(moderationAPIPath, rt.middleware.Get("session")(moderationAPIHandler))

	attendeeAPIPath, attendeeAPIHandler := attendeesv1connect.NewAttendeeServiceHandler(
		apiconnect.NewAttendeeHandler(attendeeservice.New(rt.repo, rt.sessionManager, rt.broker)),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(attendeeAPIPath, rt.middleware.Get("session")(attendeeAPIHandler))

	agendaAPIPath, agendaAPIHandler := agendav1connect.NewAgendaServiceHandler(
		apiconnect.NewAgendaHandler(agendaservice.New(rt.repo, rt.broker, rt.store)),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(agendaAPIPath, rt.middleware.Get("session")(agendaAPIHandler))

	speakerAPIPath, speakerAPIHandler := speakersv1connect.NewSpeakerServiceHandler(
		apiconnect.NewSpeakerHandler(speakerservice.New(rt.repo, rt.broker)),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(speakerAPIPath, rt.middleware.Get("session")(speakerAPIHandler))

	voteAPIPath, voteAPIHandler := votesv1connect.NewVoteServiceHandler(
		apiconnect.NewVoteHandler(voteservice.New(rt.repo, rt.broker)),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(voteAPIPath, rt.middleware.Get("session")(voteAPIHandler))

	adminAPIPath, adminAPIHandler := adminv1connect.NewAdminServiceHandler(
		apiconnect.NewAdminHandler(adminservice.New(rt.repo, webhooks.NewCommitteeDispatcher(rt.cfg.Webhook), rt.cfg.Auth.OAuthCommitteeGroupPrefix)),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(adminAPIPath, rt.middleware.Get("session")(adminAPIHandler))

	docsAPIPath, docsAPIHandler := docsv1connect.NewDocsServiceHandler(
		apiconnect.NewDocsHandler(rt.docsService),
		connect.WithInterceptors(apiconnect.ErrorInterceptor()),
	)
	apiMux.Handle(docsAPIPath, docsAPIHandler)

	apiMux.Handle("POST /committee/{slug}/meeting/{meetingId}/agenda-point/{agendaPointId}/attachments",
		rt.middleware.Get("session")(apihttp.NewAttachmentUploadHandler(rt.repo, rt.store)),
	)
	apiMux.Handle("GET /docs/assets/{assetPath...}", apihttp.NewDocsAssetHandler(rt.docsService))

	return apiMux
}

func handleLocaleSwitch(w http.ResponseWriter, r *http.Request) {
	lang := r.FormValue("lang")
	supported := map[string]bool{"en": true, "de": true}
	if !supported[lang] {
		http.Error(w, "unsupported locale", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "locale",
		Value:    lang,
		Path:     "/",
		MaxAge:   365 * 24 * 60 * 60,
		SameSite: http.SameSiteLaxMode,
	})
	ref := r.Header.Get("Referer")
	if ref == "" {
		ref = "/"
	}
	http.Redirect(w, r, ref, http.StatusSeeOther)
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
