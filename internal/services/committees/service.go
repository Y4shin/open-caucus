package committeeservice

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strconv"

	committeesv1 "github.com/Y4shin/open-caucus/gen/go/conference/committees/v1"
	commonv1 "github.com/Y4shin/open-caucus/gen/go/conference/common/v1"
	apierrors "github.com/Y4shin/open-caucus/internal/api/errors"
	"github.com/Y4shin/open-caucus/internal/repository"
	"github.com/Y4shin/open-caucus/internal/repository/model"
	"github.com/Y4shin/open-caucus/internal/session"
)

type Service struct {
	repo repository.Repository
}

func New(repo repository.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListMyCommittees(ctx context.Context) (*committeesv1.ListMyCommitteesResponse, error) {
	sd, ok := session.GetSession(ctx)
	if !ok || sd == nil || sd.IsExpired() || !sd.IsAccountSession() || sd.AccountID == nil {
		return nil, apierrors.New(apierrors.KindUnauthenticated, "account session required")
	}
	accountID := *sd.AccountID

	committees, err := s.repo.ListCommitteesByAccountID(ctx, accountID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to list committees", err)
	}

	items := make([]*committeesv1.CommitteeListItem, 0, len(committees))
	for _, c := range committees {
		count, err := s.repo.CountMeetingsForCommittee(ctx, c.Slug)
		if err != nil {
			return nil, apierrors.Wrap(apierrors.KindInternal, "failed to count meetings", err)
		}

		ref := &commonv1.CommitteeReference{
			CommitteeId: strconv.FormatInt(c.ID, 10),
			Slug:        c.Slug,
			Name:        c.Name,
		}
		if membership, err := s.repo.GetUserMembershipByAccountIDAndSlug(ctx, accountID, c.Slug); err == nil {
			ref.IsChairperson = membership.Role == "chairperson"
			ref.IsMember = membership.Role == "member"
		}

		items = append(items, &committeesv1.CommitteeListItem{
			Committee:        ref,
			MeetingCount:     int32(count),
			HasActiveMeeting: c.CurrentMeetingID != nil,
		})
	}

	return &committeesv1.ListMyCommitteesResponse{Committees: items}, nil
}

func (s *Service) GetCommitteeOverview(ctx context.Context, slug string) (*committeesv1.GetCommitteeOverviewResponse, error) {
	sd, ok := session.GetSession(ctx)
	if !ok || sd == nil || sd.IsExpired() || !sd.IsAccountSession() || sd.AccountID == nil {
		return nil, apierrors.New(apierrors.KindUnauthenticated, "account session required")
	}
	accountID := *sd.AccountID

	committee, err := s.repo.GetCommitteeBySlug(ctx, slug)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "committee not found")
	}

	isChairperson := false
	isMember := false
	account, _ := s.repo.GetAccountByID(ctx, accountID)
	isAdmin := account != nil && account.IsAdmin
	if membership, err := s.repo.GetUserMembershipByAccountIDAndSlug(ctx, accountID, slug); err == nil {
		isChairperson = membership.Role == "chairperson"
		isMember = membership.Role == "member"
	}

	if !isAdmin && !isChairperson && !isMember {
		return nil, apierrors.New(apierrors.KindPermissionDenied, "not a member of this committee")
	}

	committeeRef := &commonv1.CommitteeReference{
		CommitteeId:   strconv.FormatInt(committee.ID, 10),
		Slug:          committee.Slug,
		Name:          committee.Name,
		IsAdmin:       isAdmin,
		IsChairperson: isChairperson,
		IsMember:      isMember,
	}

	meetings, err := s.repo.ListMeetingsForCommittee(ctx, slug, 50, 0)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to list meetings", err)
	}

	overviewMeetings := make([]*committeesv1.CommitteeOverviewMeeting, 0, len(meetings))
	for _, m := range meetings {
		isActive := committee.CurrentMeetingID != nil && *committee.CurrentMeetingID == m.ID
		meetingRef := &commonv1.MeetingReference{
			MeetingId:     strconv.FormatInt(m.ID, 10),
			CommitteeSlug: slug,
			Name:          m.Name,
			SignupOpen:    m.SignupOpen,
			Description:   m.Description,
		}
		overviewMeetings = append(overviewMeetings, &committeesv1.CommitteeOverviewMeeting{
			Meeting:     meetingRef,
			CanModerate: isChairperson || isAdmin,
			CanJoin:     m.SignupOpen,
			CanViewLive: isActive,
		})
	}

	caps := []*commonv1.Capability{
		{Key: "committee.view", Allowed: true},
		{Key: "committee.moderate", Allowed: isChairperson || isAdmin},
		{Key: "committee.manage", Allowed: isChairperson || isAdmin},
	}

	overview := &committeesv1.CommitteeOverview{
		Committee:    committeeRef,
		Meetings:     overviewMeetings,
		Capabilities: caps,
	}

	return &committeesv1.GetCommitteeOverviewResponse{Overview: overview}, nil
}

func (s *Service) CreateMeeting(ctx context.Context, slug, name, description string) (*committeesv1.CreateMeetingResponse, error) {
	committee, err := s.requireCommitteeManager(ctx, slug)
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "meeting name is required")
	}
	secret, err := generateMeetingSecret()
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to generate meeting secret", err)
	}
	if err := s.repo.CreateMeeting(ctx, committee.ID, name, description, secret, false); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to create meeting", err)
	}

	meetings, err := s.repo.ListMeetingsForCommittee(ctx, slug, 1, 0)
	if err != nil || len(meetings) == 0 {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to load created meeting", err)
	}

	return &committeesv1.CreateMeetingResponse{
		Meeting: &commonv1.MeetingReference{
			MeetingId:     strconv.FormatInt(meetings[0].ID, 10),
			CommitteeSlug: slug,
			Name:          meetings[0].Name,
			SignupOpen:    meetings[0].SignupOpen,
			Description:   meetings[0].Description,
		},
	}, nil
}

func (s *Service) DeleteMeeting(ctx context.Context, slug, meetingIDStr string) (*committeesv1.DeleteMeetingResponse, error) {
	if _, err := s.requireCommitteeManager(ctx, slug); err != nil {
		return nil, err
	}

	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}
	if err := s.repo.DeleteMeeting(ctx, meetingID); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to delete meeting", err)
	}

	return &committeesv1.DeleteMeetingResponse{
		MeetingId: meetingIDStr,
		Deleted:   true,
	}, nil
}

func (s *Service) ToggleMeetingActive(ctx context.Context, slug, meetingIDStr string) (*committeesv1.ToggleMeetingActiveResponse, error) {
	committee, err := s.requireCommitteeManager(ctx, slug)
	if err != nil {
		return nil, err
	}

	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}

	var newActiveMeetingID *int64
	if committee.CurrentMeetingID == nil || *committee.CurrentMeetingID != meetingID {
		newActiveMeetingID = &meetingID
	}

	if err := s.repo.SetActiveMeeting(ctx, slug, newActiveMeetingID); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to set active meeting", err)
	}

	return &committeesv1.ToggleMeetingActiveResponse{
		MeetingId: meetingIDStr,
		Active:    newActiveMeetingID != nil,
	}, nil
}

func (s *Service) requireCommitteeManager(ctx context.Context, slug string) (*model.Committee, error) {
	sd, ok := session.GetSession(ctx)
	if !ok || sd == nil || sd.IsExpired() || !sd.IsAccountSession() || sd.AccountID == nil {
		return nil, apierrors.New(apierrors.KindUnauthenticated, "account session required")
	}
	accountID := *sd.AccountID

	committee, err := s.repo.GetCommitteeBySlug(ctx, slug)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "committee not found")
	}

	account, _ := s.repo.GetAccountByID(ctx, accountID)
	if account != nil && account.IsAdmin {
		return committee, nil
	}

	membership, err := s.repo.GetUserMembershipByAccountIDAndSlug(ctx, accountID, slug)
	if err != nil {
		return nil, apierrors.New(apierrors.KindPermissionDenied, "not a member of this committee")
	}
	if membership.Role != "chairperson" {
		return nil, apierrors.New(apierrors.KindPermissionDenied, "chairperson role required")
	}

	return committee, nil
}

func generateMeetingSecret() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
