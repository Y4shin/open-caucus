package attendeeservice

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"

	attendeesv1 "github.com/Y4shin/open-caucus/gen/go/conference/attendees/v1"
	commonv1 "github.com/Y4shin/open-caucus/gen/go/conference/common/v1"
	apierrors "github.com/Y4shin/open-caucus/internal/api/errors"
	"github.com/Y4shin/open-caucus/internal/broker"
	"github.com/Y4shin/open-caucus/internal/repository"
	"github.com/Y4shin/open-caucus/internal/repository/model"
	serviceauthz "github.com/Y4shin/open-caucus/internal/services/authz"
	"github.com/Y4shin/open-caucus/internal/session"
)

// MeetingAttendeesChangedEvent is the SSE event type published when attendee
// state changes on a meeting.
const MeetingAttendeesChangedEvent = "attendees.updated"

type Service struct {
	repo           repository.Repository
	sessionManager *session.Manager
	broker         broker.Broker
}

func New(repo repository.Repository, sessionManager *session.Manager, b broker.Broker) *Service {
	return &Service{repo: repo, sessionManager: sessionManager, broker: b}
}

func (s *Service) ListAttendees(ctx context.Context, committeeSlug, meetingIDStr string) (*attendeesv1.ListAttendeesResponse, error) {
	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}
	if err := serviceauthz.RequireModerationAccess(ctx, s.repo, committeeSlug, meetingID); err != nil {
		return nil, err
	}

	attendees, err := s.repo.ListAttendeesForMeeting(ctx, meetingID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to list attendees", err)
	}

	records := make([]*attendeesv1.AttendeeRecord, 0, len(attendees))
	for _, attendee := range attendees {
		records = append(records, toAttendeeRecord(attendee, 0))
	}

	return &attendeesv1.ListAttendeesResponse{Attendees: records}, nil
}

// SelfSignup registers the calling account-session user as an attendee.
// Idempotent: returns success if the attendee row already exists.
func (s *Service) SelfSignup(ctx context.Context, committeeSlug, meetingIDStr string) (*attendeesv1.SelfSignupResponse, error) {
	sd, ok := session.GetSession(ctx)
	if !ok || sd == nil || sd.IsExpired() || !sd.IsAccountSession() || sd.AccountID == nil {
		return nil, apierrors.New(apierrors.KindUnauthenticated, "account session required")
	}

	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}

	if _, err := s.repo.GetMeetingByID(ctx, meetingID); err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "meeting not found")
	}

	membership, err := s.repo.GetUserMembershipByAccountIDAndSlug(ctx, *sd.AccountID, committeeSlug)
	if err != nil {
		return nil, apierrors.New(apierrors.KindPermissionDenied, "not a member of this committee")
	}

	// Idempotent: check for existing attendee record.
	if existing, err := s.repo.GetAttendeeByUserIDAndMeetingID(ctx, membership.ID, meetingID); err == nil {
		return &attendeesv1.SelfSignupResponse{
			Attendee:       toAttendeeRecord(existing, membership.ID),
			AlreadyExisted: true,
		}, nil
	}

	// signupOpen only gates guest joins; committee members may always self-signup.

	user, err := s.repo.GetUserByID(ctx, membership.ID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to load user", err)
	}

	secret, err := generateSecret()
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to generate attendee secret", err)
	}

	attendee, err := s.repo.CreateAttendee(ctx, meetingID, &membership.ID, user.FullName, secret, user.Quoted)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to create attendee", err)
	}

	s.publishAttendeesChanged(meetingID)

	return &attendeesv1.SelfSignupResponse{
		Attendee:       toAttendeeRecord(attendee, membership.ID),
		AlreadyExisted: false,
	}, nil
}

// GuestJoin creates a new guest attendee and returns the attendee access secret.
func (s *Service) GuestJoin(ctx context.Context, committeeSlug, meetingIDStr, fullName, meetingSecret string, genderQuoted bool) (*attendeesv1.GuestJoinResponse, error) {
	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}

	meeting, err := s.repo.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "meeting not found")
	}

	if !meeting.SignupOpen {
		return nil, apierrors.New(apierrors.KindPermissionDenied, "guest signup is currently closed")
	}

	if fullName == "" {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "full name is required")
	}

	if meetingSecret == "" || meetingSecret != meeting.Secret {
		return nil, apierrors.New(apierrors.KindPermissionDenied, "invalid meeting secret")
	}

	// Suppress unused committeeSlug lint warning — kept for future committee
	// membership checks (e.g. dedup within committee).
	_ = committeeSlug

	attendeeSecret, err := generateSecret()
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to generate attendee secret", err)
	}

	attendee, err := s.repo.CreateAttendee(ctx, meetingID, nil, fullName, attendeeSecret, genderQuoted)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to create attendee", err)
	}

	s.publishAttendeesChanged(meetingID)

	return &attendeesv1.GuestJoinResponse{
		Attendee:       toAttendeeRecord(attendee, 0),
		AttendeeSecret: attendeeSecret,
	}, nil
}

// AttendeeLogin exchanges an attendee secret for a session cookie.
func (s *Service) AttendeeLogin(ctx context.Context, meetingIDStr, attendeeSecret string) (*attendeesv1.AttendeeLoginResponse, *http.Cookie, error) {
	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}

	if attendeeSecret == "" {
		return nil, nil, apierrors.New(apierrors.KindInvalidArgument, "attendee secret is required")
	}

	attendee, err := s.repo.GetAttendeeByMeetingIDAndSecret(ctx, meetingID, attendeeSecret)
	if err != nil {
		return nil, nil, apierrors.New(apierrors.KindUnauthenticated, "Invalid access code")
	}

	sd := &session.SessionData{
		SessionType: session.SessionTypeGuest,
		AttendeeID:  &attendee.ID,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}

	signedID, err := s.sessionManager.CreateSession(ctx, sd)
	if err != nil {
		return nil, nil, apierrors.Wrap(apierrors.KindInternal, "failed to create session", err)
	}

	cookie := s.sessionManager.CreateCookie(signedID)

	resp := &attendeesv1.AttendeeLoginResponse{
		Attendee: toAttendeeRecord(attendee, 0),
		Actor: &commonv1.ActorSummary{
			ActorKind:   "guest",
			AttendeeId:  strconv.FormatInt(attendee.ID, 10),
			DisplayName: attendee.FullName,
		},
	}
	return resp, cookie, nil
}

func (s *Service) publishAttendeesChanged(meetingID int64) {
	mid := meetingID
	s.broker.Publish(broker.SSEEvent{
		Event:     MeetingAttendeesChangedEvent,
		Data:      []byte(`{"type":"attendees.updated"}`),
		MeetingID: &mid,
	})
}

func toAttendeeRecord(a *model.Attendee, callerUserID int64) *attendeesv1.AttendeeRecord {
	mine := callerUserID != 0 && a.UserID != nil && *a.UserID == callerUserID
	return &attendeesv1.AttendeeRecord{
		AttendeeId:     strconv.FormatInt(a.ID, 10),
		FullName:       a.FullName,
		AttendeeNumber: a.AttendeeNumber,
		IsChair:        a.IsChair,
		IsGuest:        a.UserID == nil,
		Quoted:         a.Quoted,
		Mine:           mine,
	}
}

func generateSecret() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CreateAttendee creates a guest attendee on behalf of a moderator/chairperson.
func (s *Service) CreateAttendee(ctx context.Context, committeeSlug, meetingIDStr, fullName string, genderQuoted bool) (*attendeesv1.CreateAttendeeResponse, error) {
	if err := serviceauthz.RequireChairperson(ctx, s.repo, committeeSlug); err != nil {
		return nil, err
	}
	if fullName == "" {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "full name is required")
	}
	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}
	secret, err := generateSecret()
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to generate attendee secret", err)
	}
	attendee, err := s.repo.CreateAttendee(ctx, meetingID, nil, fullName, secret, genderQuoted)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to create attendee", err)
	}
	s.publishAttendeesChanged(meetingID)
	return &attendeesv1.CreateAttendeeResponse{
		Attendee:       toAttendeeRecord(attendee, 0),
		AttendeeSecret: secret,
	}, nil
}

// DeleteAttendee removes an attendee from a meeting.
func (s *Service) DeleteAttendee(ctx context.Context, committeeSlug, meetingIDStr, attendeeIDStr string) (*attendeesv1.DeleteAttendeeResponse, error) {
	if err := serviceauthz.RequireChairperson(ctx, s.repo, committeeSlug); err != nil {
		return nil, err
	}
	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}
	attendeeID, err := strconv.ParseInt(attendeeIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid attendee id")
	}
	if err := s.repo.DeleteAttendee(ctx, attendeeID); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to delete attendee", err)
	}
	s.publishAttendeesChanged(meetingID)
	return &attendeesv1.DeleteAttendeeResponse{}, nil
}

// SetChairperson updates the chairperson flag for an attendee.
func (s *Service) SetChairperson(ctx context.Context, committeeSlug, meetingIDStr, attendeeIDStr string, isChair bool) (*attendeesv1.SetChairpersonResponse, error) {
	if err := serviceauthz.RequireChairperson(ctx, s.repo, committeeSlug); err != nil {
		return nil, err
	}
	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}
	attendeeID, err := strconv.ParseInt(attendeeIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid attendee id")
	}
	if err := s.repo.SetAttendeeIsChair(ctx, attendeeID, isChair); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to update chairperson status", err)
	}
	attendee, err := s.repo.GetAttendeeByID(ctx, attendeeID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to reload attendee", err)
	}
	s.publishAttendeesChanged(meetingID)
	return &attendeesv1.SetChairpersonResponse{Attendee: toAttendeeRecord(attendee, 0)}, nil
}

// SetQuoted updates the gender-quotation flag for an attendee.
func (s *Service) SetQuoted(ctx context.Context, committeeSlug, meetingIDStr, attendeeIDStr string, quoted bool) (*attendeesv1.SetQuotedResponse, error) {
	if err := serviceauthz.RequireChairperson(ctx, s.repo, committeeSlug); err != nil {
		return nil, err
	}
	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}
	attendeeID, err := strconv.ParseInt(attendeeIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid attendee id")
	}
	if err := s.repo.SetAttendeeQuoted(ctx, attendeeID, quoted); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to update quoted status", err)
	}
	attendee, err := s.repo.GetAttendeeByID(ctx, attendeeID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to reload attendee", err)
	}
	s.publishAttendeesChanged(meetingID)
	return &attendeesv1.SetQuotedResponse{Attendee: toAttendeeRecord(attendee, 0)}, nil
}

func (s *Service) GetAttendeeRecovery(ctx context.Context, committeeSlug, meetingIDStr, attendeeIDStr, baseURL string) (*attendeesv1.GetAttendeeRecoveryResponse, error) {
	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}
	attendeeID, err := strconv.ParseInt(attendeeIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid attendee id")
	}
	if err := serviceauthz.RequireModerationAccess(ctx, s.repo, committeeSlug, meetingID); err != nil {
		return nil, err
	}

	committee, err := s.repo.GetCommitteeBySlug(ctx, committeeSlug)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "committee not found")
	}
	meeting, err := s.repo.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "meeting not found")
	}
	attendee, err := s.repo.GetAttendeeByID(ctx, attendeeID)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "attendee not found")
	}
	if attendee.MeetingID != meetingID {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "attendee does not belong to meeting")
	}
	if attendee.UserID != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "recovery link is only available for guests")
	}

	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "base url is required")
	}

	loginURL, err := url.Parse(baseURL + fmt.Sprintf("/committee/%s/meeting/%s/attendee-login", committeeSlug, meetingIDStr))
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInvalidArgument, "invalid base url", err)
	}
	query := loginURL.Query()
	query.Set("secret", attendee.Secret)
	loginURL.RawQuery = query.Encode()
	loginURLStr := loginURL.String()

	png, err := qrcode.Encode(loginURLStr, qrcode.Medium, 320)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to generate attendee recovery qr code", err)
	}

	return &attendeesv1.GetAttendeeRecoveryResponse{
		View: &attendeesv1.AttendeeRecoveryView{
			CommitteeSlug: committeeSlug,
			MeetingId:     meetingIDStr,
			AttendeeId:    attendeeIDStr,
			MeetingName:   meeting.Name,
			CommitteeName: committee.Name,
			AttendeeName:  attendee.FullName,
			LoginUrl:      loginURLStr,
			QrCodeDataUrl: "data:image/png;base64," + base64.StdEncoding.EncodeToString(png),
		},
	}, nil
}

func (s *Service) requireChairperson(ctx context.Context, committeeSlug string) error {
	return serviceauthz.RequireChairperson(ctx, s.repo, committeeSlug)
}
