package attendeeservice

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	attendeesv1 "github.com/Y4shin/conference-tool/gen/go/conference/attendees/v1"
	commonv1 "github.com/Y4shin/conference-tool/gen/go/conference/common/v1"
	apierrors "github.com/Y4shin/conference-tool/internal/api/errors"
	"github.com/Y4shin/conference-tool/internal/broker"
	"github.com/Y4shin/conference-tool/internal/repository"
	"github.com/Y4shin/conference-tool/internal/repository/model"
	"github.com/Y4shin/conference-tool/internal/session"
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

	meeting, err := s.repo.GetMeetingByID(ctx, meetingID)
	if err != nil {
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

	if !meeting.SignupOpen {
		return nil, apierrors.New(apierrors.KindPermissionDenied, "meeting signup is currently closed")
	}

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
		return nil, nil, apierrors.New(apierrors.KindUnauthenticated, "invalid attendee secret")
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
