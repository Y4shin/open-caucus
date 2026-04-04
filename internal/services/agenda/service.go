package agendaservice

import (
	"context"
	"fmt"
	"strconv"
	"time"

	agendav1 "github.com/Y4shin/open-caucus/gen/go/conference/agenda/v1"
	commonv1 "github.com/Y4shin/open-caucus/gen/go/conference/common/v1"
	apierrors "github.com/Y4shin/open-caucus/internal/api/errors"
	"github.com/Y4shin/open-caucus/internal/broker"
	"github.com/Y4shin/open-caucus/internal/repository"
	"github.com/Y4shin/open-caucus/internal/repository/model"
	serviceauthz "github.com/Y4shin/open-caucus/internal/services/authz"
	"github.com/Y4shin/open-caucus/internal/session"
	"github.com/Y4shin/open-caucus/internal/storage"
)

type Service struct {
	repo   repository.Repository
	broker broker.Broker
	store  storage.Service
}

func New(repo repository.Repository, b broker.Broker, store storage.Service) *Service {
	return &Service{repo: repo, broker: b, store: store}
}

// ListAgendaPoints returns the full agenda tree for a meeting. Requires
// committee membership or chairperson role.
func (s *Service) ListAgendaPoints(ctx context.Context, committeeSlug, meetingIDStr string) (*agendav1.ListAgendaPointsResponse, error) {
	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}
	if err := s.requireAgendaViewer(ctx, committeeSlug, meetingID); err != nil {
		return nil, err
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

// GetAgendaPointTools returns the attachment/current-document tools state for one agenda point.
func (s *Service) GetAgendaPointTools(ctx context.Context, committeeSlug, meetingIDStr, agendaPointIDStr string) (*agendav1.GetAgendaPointToolsResponse, error) {
	if err := s.requireChairperson(ctx, committeeSlug); err != nil {
		return nil, err
	}

	view, err := s.loadAgendaPointToolsView(ctx, committeeSlug, meetingIDStr, agendaPointIDStr)
	if err != nil {
		return nil, err
	}

	return &agendav1.GetAgendaPointToolsResponse{View: view}, nil
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

	// Record timestamps: mark the old point as left and the new one as entered.
	now := time.Now().UTC().Format(time.RFC3339)
	meeting, err := s.repo.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to load meeting", err)
	}
	if meeting.CurrentAgendaPointID != nil {
		_ = s.repo.SetAgendaPointLeftAt(ctx, *meeting.CurrentAgendaPointID, now)
	}
	if apID != nil {
		_ = s.repo.SetAgendaPointEnteredAt(ctx, *apID, now)
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

// SetCurrentAttachment marks one attachment as the published live document.
func (s *Service) SetCurrentAttachment(ctx context.Context, committeeSlug, meetingIDStr, agendaPointIDStr, attachmentIDStr string) (*agendav1.SetCurrentAttachmentResponse, error) {
	if err := s.requireChairperson(ctx, committeeSlug); err != nil {
		return nil, err
	}

	meetingID, agendaPointID, attachmentID, err := parseAgendaToolIDs(meetingIDStr, agendaPointIDStr, attachmentIDStr)
	if err != nil {
		return nil, err
	}

	ap, err := s.loadAgendaPointForMeeting(ctx, meetingID, agendaPointID)
	if err != nil {
		return nil, err
	}

	attachment, err := s.repo.GetAttachmentByID(ctx, attachmentID)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "attachment not found")
	}
	if attachment.AgendaPointID != ap.ID {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "attachment does not belong to agenda point")
	}

	if err := s.repo.SetCurrentAttachment(ctx, ap.ID, attachmentID); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to set current attachment", err)
	}

	s.publishAgendaUpdated(meetingID)

	view, err := s.loadAgendaPointToolsView(ctx, committeeSlug, meetingIDStr, agendaPointIDStr)
	if err != nil {
		return nil, err
	}

	return &agendav1.SetCurrentAttachmentResponse{
		View:             view,
		InvalidatedViews: []string{"live", "moderation"},
	}, nil
}

// ClearCurrentDocument removes the published live document for an agenda point.
func (s *Service) ClearCurrentDocument(ctx context.Context, committeeSlug, meetingIDStr, agendaPointIDStr string) (*agendav1.ClearCurrentDocumentResponse, error) {
	if err := s.requireChairperson(ctx, committeeSlug); err != nil {
		return nil, err
	}

	meetingID, agendaPointID, err := parseMeetingAndAgendaPointIDs(meetingIDStr, agendaPointIDStr)
	if err != nil {
		return nil, err
	}

	ap, err := s.loadAgendaPointForMeeting(ctx, meetingID, agendaPointID)
	if err != nil {
		return nil, err
	}

	if err := s.repo.ClearCurrentDocument(ctx, ap.ID); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to clear current document", err)
	}

	s.publishAgendaUpdated(meetingID)

	view, err := s.loadAgendaPointToolsView(ctx, committeeSlug, meetingIDStr, agendaPointIDStr)
	if err != nil {
		return nil, err
	}

	return &agendav1.ClearCurrentDocumentResponse{
		View:             view,
		InvalidatedViews: []string{"live", "moderation"},
	}, nil
}

// DeleteAttachment removes an attachment and its backing blob.
func (s *Service) DeleteAttachment(ctx context.Context, committeeSlug, meetingIDStr, agendaPointIDStr, attachmentIDStr string) (*agendav1.DeleteAttachmentResponse, error) {
	if err := s.requireChairperson(ctx, committeeSlug); err != nil {
		return nil, err
	}
	if s.store == nil {
		return nil, apierrors.New(apierrors.KindInternal, "storage service unavailable")
	}

	meetingID, agendaPointID, attachmentID, err := parseAgendaToolIDs(meetingIDStr, agendaPointIDStr, attachmentIDStr)
	if err != nil {
		return nil, err
	}

	ap, err := s.loadAgendaPointForMeeting(ctx, meetingID, agendaPointID)
	if err != nil {
		return nil, err
	}

	attachment, err := s.repo.GetAttachmentByID(ctx, attachmentID)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "attachment not found")
	}
	if attachment.AgendaPointID != ap.ID {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "attachment does not belong to agenda point")
	}

	blob, err := s.repo.GetBlobByID(ctx, attachment.BlobID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to load attachment blob", err)
	}

	if err := s.repo.DeleteAttachment(ctx, attachmentID); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to delete attachment", err)
	}
	if err := s.repo.DeleteBlob(ctx, blob.ID); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to delete blob record", err)
	}
	_ = s.store.Delete(blob.StoragePath)

	s.publishAgendaUpdated(meetingID)

	view, err := s.loadAgendaPointToolsView(ctx, committeeSlug, meetingIDStr, agendaPointIDStr)
	if err != nil {
		return nil, err
	}

	return &agendav1.DeleteAttachmentResponse{
		View:             view,
		InvalidatedViews: []string{"live", "moderation"},
	}, nil
}

func (s *Service) UpdateAgendaPoint(ctx context.Context, committeeSlug, meetingIDStr, agendaPointIDStr, title string) (*agendav1.UpdateAgendaPointResponse, error) {
	if err := s.requireChairperson(ctx, committeeSlug); err != nil {
		return nil, err
	}
	if title == "" {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "title is required")
	}
	meetingID, agendaPointID, err := parseMeetingAndAgendaPointIDs(meetingIDStr, agendaPointIDStr)
	if err != nil {
		return nil, err
	}
	if _, err := s.loadAgendaPointForMeeting(ctx, meetingID, agendaPointID); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateAgendaPointTitle(ctx, agendaPointID, title); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to update agenda point", err)
	}
	ap, err := s.repo.GetAgendaPointByID(ctx, agendaPointID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to reload agenda point", err)
	}
	s.publishAgendaUpdated(meetingID)
	return &agendav1.UpdateAgendaPointResponse{
		AgendaPoint:      toAgendaPointRecord(ap, "", nil),
		InvalidatedViews: []string{"live", "moderation"},
	}, nil
}

func (s *Service) publishAgendaUpdated(meetingID int64) {
	mid := meetingID
	s.broker.Publish(broker.SSEEvent{
		Event:     "agenda.updated",
		Data:      []byte(`{"type":"agenda.updated"}`),
		MeetingID: &mid,
	})
}

func (s *Service) loadAgendaPointForMeeting(ctx context.Context, meetingID, agendaPointID int64) (*model.AgendaPoint, error) {
	ap, err := s.repo.GetAgendaPointByID(ctx, agendaPointID)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "agenda point not found")
	}
	if ap.MeetingID != meetingID {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "agenda point does not belong to meeting")
	}
	return ap, nil
}

func (s *Service) loadAgendaPointToolsView(ctx context.Context, committeeSlug, meetingIDStr, agendaPointIDStr string) (*agendav1.AgendaPointToolsView, error) {
	meetingID, agendaPointID, err := parseMeetingAndAgendaPointIDs(meetingIDStr, agendaPointIDStr)
	if err != nil {
		return nil, err
	}

	ap, err := s.loadAgendaPointForMeeting(ctx, meetingID, agendaPointID)
	if err != nil {
		return nil, err
	}

	attachments, err := s.repo.ListAttachmentsForAgendaPoint(ctx, ap.ID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to list attachments", err)
	}

	view := &agendav1.AgendaPointToolsView{
		CommitteeSlug:    committeeSlug,
		MeetingId:        meetingIDStr,
		AgendaPointId:    agendaPointIDStr,
		AgendaPointTitle: ap.Title,
	}
	if ap.CurrentAttachmentID != nil {
		view.CurrentAttachmentId = strconv.FormatInt(*ap.CurrentAttachmentID, 10)
	}

	for _, attachment := range attachments {
		blob, err := s.repo.GetBlobByID(ctx, attachment.BlobID)
		if err != nil {
			return nil, apierrors.Wrap(apierrors.KindInternal, "failed to load attachment blob", err)
		}

		label := ""
		if attachment.Label != nil {
			label = *attachment.Label
		}

		view.Attachments = append(view.Attachments, &agendav1.AttachmentRecord{
			AttachmentId: strconv.FormatInt(attachment.ID, 10),
			BlobId:       strconv.FormatInt(blob.ID, 10),
			Filename:     blob.Filename,
			Label:        label,
			DownloadUrl:  fmt.Sprintf("/blobs/%d/download", blob.ID),
			IsCurrent:    ap.CurrentAttachmentID != nil && *ap.CurrentAttachmentID == attachment.ID,
		})
	}

	return view, nil
}

func parseMeetingAndAgendaPointIDs(meetingIDStr, agendaPointIDStr string) (int64, int64, error) {
	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return 0, 0, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}
	agendaPointID, err := strconv.ParseInt(agendaPointIDStr, 10, 64)
	if err != nil {
		return 0, 0, apierrors.New(apierrors.KindInvalidArgument, "invalid agenda point id")
	}
	return meetingID, agendaPointID, nil
}

func parseAgendaToolIDs(meetingIDStr, agendaPointIDStr, attachmentIDStr string) (int64, int64, int64, error) {
	meetingID, agendaPointID, err := parseMeetingAndAgendaPointIDs(meetingIDStr, agendaPointIDStr)
	if err != nil {
		return 0, 0, 0, err
	}
	attachmentID, err := strconv.ParseInt(attachmentIDStr, 10, 64)
	if err != nil {
		return 0, 0, 0, apierrors.New(apierrors.KindInvalidArgument, "invalid attachment id")
	}
	return meetingID, agendaPointID, attachmentID, nil
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
	return serviceauthz.RequireChairperson(ctx, s.repo, committeeSlug)
}

func (s *Service) requireAgendaViewer(ctx context.Context, committeeSlug string, meetingID int64) error {
	if err := s.requireMembership(ctx, committeeSlug); err == nil {
		return nil
	}
	sd, ok := session.GetSession(ctx)
	if ok && sd != nil && !sd.IsExpired() && sd.IsGuestSession() && sd.AttendeeID != nil {
		attendee, err := s.repo.GetAttendeeByID(ctx, *sd.AttendeeID)
		if err == nil && attendee.MeetingID == meetingID {
			return nil
		}
	}
	return serviceauthz.RequireModerationAccess(ctx, s.repo, committeeSlug, meetingID)
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
	enteredAt := ""
	if ap.EnteredAt != nil {
		enteredAt = *ap.EnteredAt
	}
	leftAt := ""
	if ap.LeftAt != nil {
		leftAt = *ap.LeftAt
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
		EnteredAt:             enteredAt,
		LeftAt:                leftAt,
	}
}
