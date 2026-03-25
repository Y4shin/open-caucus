package agendaservice

import (
	"context"
	"strconv"

	agendav1 "github.com/Y4shin/conference-tool/gen/go/conference/agenda/v1"
	commonv1 "github.com/Y4shin/conference-tool/gen/go/conference/common/v1"
	apierrors "github.com/Y4shin/conference-tool/internal/api/errors"
	"github.com/Y4shin/conference-tool/internal/broker"
	"github.com/Y4shin/conference-tool/internal/repository"
	"github.com/Y4shin/conference-tool/internal/repository/model"
	"github.com/Y4shin/conference-tool/internal/session"
)

type Service struct {
	repo   repository.Repository
	broker broker.Broker
}

func New(repo repository.Repository, b broker.Broker) *Service {
	return &Service{repo: repo, broker: b}
}

// ListAgendaPoints returns the full agenda tree for a meeting. Requires
// committee membership or chairperson role.
func (s *Service) ListAgendaPoints(ctx context.Context, committeeSlug, meetingIDStr string) (*agendav1.ListAgendaPointsResponse, error) {
	if err := s.requireMembership(ctx, committeeSlug); err != nil {
		return nil, err
	}

	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}

	meeting, err := s.repo.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "meeting not found")
	}

	topLevel, err := s.repo.ListAgendaPointsForMeeting(ctx, meetingID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to list agenda points", err)
	}

	subPoints, err := s.repo.ListSubAgendaPointsForMeeting(ctx, meetingID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to list sub-agenda points", err)
	}

	subByParent := make(map[int64][]*model.AgendaPoint)
	for _, sub := range subPoints {
		if sub.ParentID != nil {
			subByParent[*sub.ParentID] = append(subByParent[*sub.ParentID], sub)
		}
	}

	records := make([]*agendav1.AgendaPointRecord, 0, len(topLevel))
	for i, ap := range topLevel {
		record := toAgendaPointRecord(ap, strconv.Itoa(i+1), meeting.CurrentAgendaPointID)
		for j, sub := range subByParent[ap.ID] {
			subRecord := toAgendaPointRecord(sub, strconv.Itoa(i+1)+"."+strconv.Itoa(j+1), meeting.CurrentAgendaPointID)
			record.SubPoints = append(record.SubPoints, subRecord)
		}
		records = append(records, record)
	}

	activeID := ""
	if meeting.CurrentAgendaPointID != nil {
		activeID = strconv.FormatInt(*meeting.CurrentAgendaPointID, 10)
	}

	return &agendav1.ListAgendaPointsResponse{
		AgendaPoints:        records,
		ActiveAgendaPointId: activeID,
	}, nil
}

// CreateAgendaPoint creates a new agenda point. Requires chairperson role.
func (s *Service) CreateAgendaPoint(ctx context.Context, committeeSlug, meetingIDStr, title, parentIDStr string) (*agendav1.CreateAgendaPointResponse, error) {
	if err := s.requireChairperson(ctx, committeeSlug); err != nil {
		return nil, err
	}

	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}

	if title == "" {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "title is required")
	}

	var ap *model.AgendaPoint
	if parentIDStr != "" {
		parentID, err := strconv.ParseInt(parentIDStr, 10, 64)
		if err != nil {
			return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid parent agenda point id")
		}
		ap, err = s.repo.CreateSubAgendaPoint(ctx, meetingID, parentID, title)
		if err != nil {
			return nil, apierrors.Wrap(apierrors.KindInternal, "failed to create sub-agenda point", err)
		}
	} else {
		ap, err = s.repo.CreateAgendaPoint(ctx, meetingID, title)
		if err != nil {
			return nil, apierrors.Wrap(apierrors.KindInternal, "failed to create agenda point", err)
		}
	}

	s.publishAgendaUpdated(meetingID)

	return &agendav1.CreateAgendaPointResponse{
		AgendaPoint:      toAgendaPointRecord(ap, "", nil),
		InvalidatedViews: []string{"moderation", "live"},
	}, nil
}

// DeleteAgendaPoint removes an agenda point. Requires chairperson role.
func (s *Service) DeleteAgendaPoint(ctx context.Context, committeeSlug, meetingIDStr, agendaPointIDStr string) (*agendav1.DeleteAgendaPointResponse, error) {
	if err := s.requireChairperson(ctx, committeeSlug); err != nil {
		return nil, err
	}

	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}

	agendaPointID, err := strconv.ParseInt(agendaPointIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid agenda point id")
	}

	if err := s.repo.DeleteAgendaPoint(ctx, agendaPointID); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to delete agenda point", err)
	}

	s.publishAgendaUpdated(meetingID)

	return &agendav1.DeleteAgendaPointResponse{
		InvalidatedViews: []string{"moderation", "live"},
	}, nil
}

// MoveAgendaPoint changes the ordering of an agenda point. Requires chairperson role.
func (s *Service) MoveAgendaPoint(ctx context.Context, committeeSlug, meetingIDStr, agendaPointIDStr, direction string) (*agendav1.MoveAgendaPointResponse, error) {
	if err := s.requireChairperson(ctx, committeeSlug); err != nil {
		return nil, err
	}

	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}

	agendaPointID, err := strconv.ParseInt(agendaPointIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid agenda point id")
	}

	switch direction {
	case "up":
		if err := s.repo.MoveAgendaPointUp(ctx, meetingID, agendaPointID); err != nil {
			return nil, apierrors.Wrap(apierrors.KindInternal, "failed to move agenda point up", err)
		}
	case "down":
		if err := s.repo.MoveAgendaPointDown(ctx, meetingID, agendaPointID); err != nil {
			return nil, apierrors.Wrap(apierrors.KindInternal, "failed to move agenda point down", err)
		}
	default:
		return nil, apierrors.New(apierrors.KindInvalidArgument, "direction must be 'up' or 'down'")
	}

	// Re-fetch the updated list.
	topLevel, err := s.repo.ListAgendaPointsForMeeting(ctx, meetingID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to reload agenda points", err)
	}

	meeting, err := s.repo.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to reload meeting", err)
	}

	records := make([]*agendav1.AgendaPointRecord, len(topLevel))
	for i, ap := range topLevel {
		records[i] = toAgendaPointRecord(ap, strconv.Itoa(i+1), meeting.CurrentAgendaPointID)
	}

	s.publishAgendaUpdated(meetingID)

	return &agendav1.MoveAgendaPointResponse{
		AgendaPoints:     records,
		InvalidatedViews: []string{"moderation", "live"},
	}, nil
}

// ActivateAgendaPoint sets the active agenda point. Requires chairperson role.
func (s *Service) ActivateAgendaPoint(ctx context.Context, committeeSlug, meetingIDStr, agendaPointIDStr string) (*agendav1.ActivateAgendaPointResponse, error) {
	if err := s.requireChairperson(ctx, committeeSlug); err != nil {
		return nil, err
	}

	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}

	var apID *int64
	if agendaPointIDStr != "" {
		id, err := strconv.ParseInt(agendaPointIDStr, 10, 64)
		if err != nil {
			return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid agenda point id")
		}
		apID = &id
	}

	if err := s.repo.SetCurrentAgendaPoint(ctx, meetingID, apID); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to activate agenda point", err)
	}

	resp := &agendav1.ActivateAgendaPointResponse{
		ActiveAgendaPointId: agendaPointIDStr,
		InvalidatedViews:    []string{"moderation", "live"},
	}

	if apID != nil {
		ap, err := s.repo.GetAgendaPointByID(ctx, *apID)
		if err == nil {
			resp.ActiveAgendaPoint = &commonv1.AgendaPointSummary{
				AgendaPointId: agendaPointIDStr,
				Title:         ap.Title,
				IsActive:      true,
			}
		}
	}

	s.publishAgendaUpdated(meetingID)

	return resp, nil
}

func (s *Service) publishAgendaUpdated(meetingID int64) {
	mid := meetingID
	s.broker.Publish(broker.SSEEvent{
		Event:     "agenda.updated",
		Data:      []byte(`{"type":"agenda.updated"}`),
		MeetingID: &mid,
	})
}

func (s *Service) requireMembership(ctx context.Context, committeeSlug string) error {
	sd, ok := session.GetSession(ctx)
	if !ok || sd == nil || sd.IsExpired() || !sd.IsAccountSession() || sd.AccountID == nil {
		return apierrors.New(apierrors.KindUnauthenticated, "account session required")
	}
	if _, err := s.repo.GetUserMembershipByAccountIDAndSlug(ctx, *sd.AccountID, committeeSlug); err != nil {
		account, aErr := s.repo.GetAccountByID(ctx, *sd.AccountID)
		if aErr != nil || !account.IsAdmin {
			return apierrors.New(apierrors.KindPermissionDenied, "committee membership required")
		}
	}
	return nil
}

func (s *Service) requireChairperson(ctx context.Context, committeeSlug string) error {
	sd, ok := session.GetSession(ctx)
	if !ok || sd == nil || sd.IsExpired() || !sd.IsAccountSession() || sd.AccountID == nil {
		return apierrors.New(apierrors.KindUnauthenticated, "account session required")
	}
	account, err := s.repo.GetAccountByID(ctx, *sd.AccountID)
	if err != nil {
		return apierrors.New(apierrors.KindUnauthenticated, "account not found")
	}
	if account.IsAdmin {
		return nil
	}
	membership, err := s.repo.GetUserMembershipByAccountIDAndSlug(ctx, *sd.AccountID, committeeSlug)
	if err != nil || membership.Role != "chairperson" {
		return apierrors.New(apierrors.KindPermissionDenied, "chairperson role required")
	}
	return nil
}

func toAgendaPointRecord(ap *model.AgendaPoint, displayNumber string, currentID *int64) *agendav1.AgendaPointRecord {
	isActive := currentID != nil && ap.ID == *currentID
	parentID := ""
	if ap.ParentID != nil {
		parentID = strconv.FormatInt(*ap.ParentID, 10)
	}
	genderQ := false
	if ap.GenderQuotationEnabled != nil {
		genderQ = *ap.GenderQuotationEnabled
	}
	firstSpeakerQ := false
	if ap.FirstSpeakerQuotationEnabled != nil {
		firstSpeakerQ = *ap.FirstSpeakerQuotationEnabled
	}
	return &agendav1.AgendaPointRecord{
		AgendaPointId:         strconv.FormatInt(ap.ID, 10),
		DisplayNumber:         displayNumber,
		Title:                 ap.Title,
		IsActive:              isActive,
		Position:              ap.Position,
		ParentId:              parentID,
		GenderQuotation:       genderQ,
		FirstSpeakerQuotation: firstSpeakerQ,
	}
}
