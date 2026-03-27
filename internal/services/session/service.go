package sessionservice

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	commonv1 "github.com/Y4shin/conference-tool/gen/go/conference/common/v1"
	sessionv1 "github.com/Y4shin/conference-tool/gen/go/conference/session/v1"
	apierrors "github.com/Y4shin/conference-tool/internal/api/errors"
	"github.com/Y4shin/conference-tool/internal/locale"
	"github.com/Y4shin/conference-tool/internal/repository"
	"github.com/Y4shin/conference-tool/internal/session"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo            repository.Repository
	sessionManager  *session.Manager
	passwordEnabled bool
	oauthEnabled    bool
}

func New(repo repository.Repository, sessionManager *session.Manager, passwordEnabled, oauthEnabled bool) *Service {
	return &Service{
		repo:            repo,
		sessionManager:  sessionManager,
		passwordEnabled: passwordEnabled,
		oauthEnabled:    oauthEnabled,
	}
}

func (s *Service) GetSessionBootstrap(ctx context.Context) (*sessionv1.SessionBootstrap, error) {
	sd, _ := session.GetSession(ctx)
	return s.buildBootstrap(ctx, sd)
}

func (s *Service) Login(ctx context.Context, username, password string) (*sessionv1.SessionBootstrap, *http.Cookie, error) {
	if !s.passwordEnabled {
		return nil, nil, apierrors.New(apierrors.KindUnimplemented, "password login is disabled")
	}

	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, nil, apierrors.New(apierrors.KindInvalidArgument, "username and password are required")
	}

	account, err := s.repo.GetAccountByUsername(ctx, username)
	if err != nil {
		return nil, nil, apierrors.New(apierrors.KindUnauthenticated, "invalid credentials")
	}

	cred, err := s.repo.GetPasswordCredential(ctx, account.ID)
	if err != nil {
		return nil, nil, apierrors.New(apierrors.KindUnauthenticated, "invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(cred.PasswordHash), []byte(password)); err != nil {
		return nil, nil, apierrors.New(apierrors.KindUnauthenticated, "invalid credentials")
	}

	accountID := account.ID
	sessionData := &session.SessionData{
		SessionType: session.SessionTypeAccount,
		AccountID:   &accountID,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}

	signedID, err := s.sessionManager.CreateSession(ctx, sessionData)
	if err != nil {
		return nil, nil, apierrors.Wrap(apierrors.KindInternal, "failed to create session", err)
	}

	bootstrap, err := s.buildBootstrap(ctx, sessionData)
	if err != nil {
		return nil, nil, err
	}
	bootstrap.RedirectTo = "/home"

	return bootstrap, s.sessionManager.CreateCookie(signedID), nil
}

func (s *Service) Logout(ctx context.Context, signedSessionID string) (*sessionv1.LogoutResponse, *http.Cookie, error) {
	if strings.TrimSpace(signedSessionID) != "" {
		_ = s.sessionManager.DestroySession(ctx, signedSessionID)
	}

	clearCookie := &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	return &sessionv1.LogoutResponse{
		Cleared:    true,
		RedirectTo: "/",
	}, clearCookie, nil
}

func (s *Service) buildBootstrap(ctx context.Context, sd *session.SessionData) (*sessionv1.SessionBootstrap, error) {
	bootstrap := &sessionv1.SessionBootstrap{
		Authenticated:   false,
		Locale:          resolvedLocale(ctx),
		Capabilities:    []*commonv1.Capability{},
		PasswordEnabled: s.passwordEnabled,
		OauthEnabled:    s.oauthEnabled,
	}

	if sd == nil || sd.IsExpired() {
		return bootstrap, nil
	}

	bootstrap.Authenticated = true

	switch {
	case sd.IsAccountSession() && sd.AccountID != nil:
		account, err := s.repo.GetAccountByID(ctx, *sd.AccountID)
		if err != nil {
			return &sessionv1.SessionBootstrap{
				Authenticated:   false,
				Locale:          bootstrap.Locale,
				Capabilities:    []*commonv1.Capability{},
				PasswordEnabled: s.passwordEnabled,
				OauthEnabled:    s.oauthEnabled,
			}, nil
		}

		bootstrap.Actor = &commonv1.ActorSummary{
			ActorKind:   "account",
			AccountId:   strconv.FormatInt(account.ID, 10),
			DisplayName: displayName(account.FullName, account.Username),
			Username:    account.Username,
		}
		bootstrap.IsAdmin = account.IsAdmin
		bootstrap.Capabilities = append(bootstrap.Capabilities,
			&commonv1.Capability{Key: "session.authenticated", Allowed: true},
			&commonv1.Capability{Key: "session.account", Allowed: true},
			&commonv1.Capability{Key: "nav.home", Allowed: true},
		)
		if account.IsAdmin {
			bootstrap.Capabilities = append(bootstrap.Capabilities, &commonv1.Capability{Key: "admin.access", Allowed: true})
		}

		committees, err := s.repo.ListCommitteesByAccountID(ctx, account.ID)
		if err != nil {
			return nil, apierrors.Wrap(apierrors.KindInternal, "failed to list committees", err)
		}

		bootstrap.AvailableCommittees = make([]*commonv1.CommitteeReference, 0, len(committees))
		for _, committee := range committees {
			ref := &commonv1.CommitteeReference{
				CommitteeId: strconv.FormatInt(committee.ID, 10),
				Slug:        committee.Slug,
				Name:        committee.Name,
				IsAdmin:     account.IsAdmin,
			}
			if membership, err := s.repo.GetUserMembershipByAccountIDAndSlug(ctx, account.ID, committee.Slug); err == nil {
				ref.IsChairperson = membership.Role == "chairperson"
				ref.IsMember = membership.Role == "member"
			}
			bootstrap.AvailableCommittees = append(bootstrap.AvailableCommittees, ref)
		}

	case sd.IsGuestSession() && sd.AttendeeID != nil:
		attendee, err := s.repo.GetAttendeeByID(ctx, *sd.AttendeeID)
		if err != nil {
			return &sessionv1.SessionBootstrap{
				Authenticated:   false,
				Locale:          bootstrap.Locale,
				Capabilities:    []*commonv1.Capability{},
				PasswordEnabled: s.passwordEnabled,
				OauthEnabled:    s.oauthEnabled,
			}, nil
		}

		bootstrap.Actor = &commonv1.ActorSummary{
			ActorKind:   "guest",
			AttendeeId:  strconv.FormatInt(attendee.ID, 10),
			DisplayName: attendee.FullName,
		}
		bootstrap.Capabilities = append(bootstrap.Capabilities,
			&commonv1.Capability{Key: "session.authenticated", Allowed: true},
			&commonv1.Capability{Key: "session.guest", Allowed: true},
		)

	default:
		bootstrap.Authenticated = false
	}

	return bootstrap, nil
}

func resolvedLocale(ctx context.Context) string {
	if l, ok := locale.GetLocale(ctx); ok && strings.TrimSpace(l) != "" {
		return l
	}
	return "en"
}

func displayName(fullName, fallback string) string {
	fullName = strings.TrimSpace(fullName)
	if fullName != "" {
		return fullName
	}
	return strings.TrimSpace(fallback)
}
