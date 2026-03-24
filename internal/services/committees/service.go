package committeeservice

import (
	"context"
	"strconv"

	commonv1 "github.com/Y4shin/conference-tool/gen/go/conference/common/v1"
	committeesv1 "github.com/Y4shin/conference-tool/gen/go/conference/committees/v1"
	apierrors "github.com/Y4shin/conference-tool/internal/api/errors"
	"github.com/Y4shin/conference-tool/internal/repository"
	"github.com/Y4shin/conference-tool/internal/session"
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
			Committee:       ref,
			MeetingCount:    int32(count),
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
