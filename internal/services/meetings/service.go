package meetingservice

import (
	"context"
	"fmt"
	"strconv"

	commonv1 "github.com/Y4shin/conference-tool/gen/go/conference/common/v1"
	meetingsv1 "github.com/Y4shin/conference-tool/gen/go/conference/meetings/v1"
	apierrors "github.com/Y4shin/conference-tool/internal/api/errors"
	"github.com/Y4shin/conference-tool/internal/repository"
	"github.com/Y4shin/conference-tool/internal/repository/model"
	"github.com/Y4shin/conference-tool/internal/session"
)

type Service struct {
	repo repository.Repository
}

func New(repo repository.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetLiveMeeting(ctx context.Context, committeeSlug, meetingIDStr string) (*meetingsv1.GetLiveMeetingResponse, error) {
	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}

	committee, err := s.repo.GetCommitteeBySlug(ctx, committeeSlug)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "committee not found")
	}

	meeting, err := s.repo.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "meeting not found")
	}

	// Resolve actor context for capability decisions
	attendeeID, isAttendee := s.resolveAttendeeID(ctx, meetingID)

	// Active agenda point + speakers
	var activeAP *commonv1.AgendaPointSummary
	var speakers []*commonv1.SpeakerSummary
	var currentDoc *commonv1.CurrentDocumentSummary

	if meeting.CurrentAgendaPointID != nil {
		ap, err := s.repo.GetAgendaPointByID(ctx, *meeting.CurrentAgendaPointID)
		if err == nil {
			activeAP = &commonv1.AgendaPointSummary{
				AgendaPointId: strconv.FormatInt(ap.ID, 10),
				Title:         ap.Title,
				IsActive:      true,
			}

			rawSpeakers, err := s.repo.ListSpeakersForAgendaPoint(ctx, ap.ID)
			if err == nil {
				speakers = buildSpeakerSummaries(rawSpeakers, attendeeID)
			}

			currentDoc, err = s.resolveCurrentDoc(ctx, ap)
			if err != nil {
				currentDoc = nil
			}
		}
	}

	caps := s.buildLiveMeetingCapabilities(ctx, meeting, committee, isAttendee)

	eventsURL := fmt.Sprintf("/api/realtime/meetings/%d/events", meetingID)

	view := &meetingsv1.LiveMeetingView{
		CommitteeSlug:      committeeSlug,
		MeetingId:          strconv.FormatInt(meetingID, 10),
		MeetingName:        meeting.Name,
		CommitteeName:      committee.Name,
		Version:            uint64(meeting.Version),
		ActiveAgendaPoint:  activeAP,
		Speakers:           speakers,
		CurrentDocument:    currentDoc,
		Capabilities:       caps,
		EventsUrl:          eventsURL,
	}

	return &meetingsv1.GetLiveMeetingResponse{Meeting: view}, nil
}

func (s *Service) resolveAttendeeID(ctx context.Context, meetingID int64) (int64, bool) {
	sd, ok := session.GetSession(ctx)
	if !ok || sd == nil || sd.IsExpired() {
		return 0, false
	}

	if sd.IsGuestSession() && sd.AttendeeID != nil {
		attendee, err := s.repo.GetAttendeeByID(ctx, *sd.AttendeeID)
		if err == nil && attendee.MeetingID == meetingID {
			return attendee.ID, true
		}
		return 0, false
	}

	if sd.IsAccountSession() && sd.AccountID != nil {
		attendee, err := s.repo.GetAttendeeByUserIDAndMeetingID(ctx, *sd.AccountID, meetingID)
		if err == nil {
			return attendee.ID, true
		}
	}

	return 0, false
}

func (s *Service) buildLiveMeetingCapabilities(ctx context.Context, meeting *model.Meeting, committee *model.Committee, isAttendee bool) *meetingsv1.LiveMeetingCapabilities {
	isActiveMeeting := committee.CurrentMeetingID != nil && *committee.CurrentMeetingID == meeting.ID

	sd, _ := session.GetSession(ctx)
	isAccountSession := sd != nil && !sd.IsExpired() && sd.IsAccountSession()

	return &meetingsv1.LiveMeetingCapabilities{
		CanSelfSignup:               meeting.SignupOpen && !isAttendee && isAccountSession,
		CanAddRegularSpeech:         isAttendee && isActiveMeeting && meeting.CurrentAgendaPointID != nil,
		CanAddPointOfOrderSpeech:    isAttendee && isActiveMeeting && meeting.CurrentAgendaPointID != nil,
		CanVote:                     false, // determined per-vote-definition, not at meeting level
		CanViewCurrentDocument:      isActiveMeeting && meeting.CurrentAgendaPointID != nil,
	}
}

func (s *Service) resolveCurrentDoc(ctx context.Context, ap *model.AgendaPoint) (*commonv1.CurrentDocumentSummary, error) {
	if ap.CurrentAttachmentID == nil {
		return nil, nil
	}
	attachment, err := s.repo.GetAttachmentByID(ctx, *ap.CurrentAttachmentID)
	if err != nil {
		return nil, err
	}
	blob, err := s.repo.GetBlobByID(ctx, attachment.BlobID)
	if err != nil {
		return nil, err
	}
	label := ""
	if attachment.Label != nil {
		label = *attachment.Label
	}
	return &commonv1.CurrentDocumentSummary{
		BlobId:      strconv.FormatInt(blob.ID, 10),
		Filename:    blob.Filename,
		Label:       label,
		ContentType: blob.ContentType,
		DownloadUrl: fmt.Sprintf("/api/blobs/%d/download", blob.ID),
	}, nil
}

func buildSpeakerSummaries(entries []*model.SpeakerEntry, currentAttendeeID int64) []*commonv1.SpeakerSummary {
	summaries := make([]*commonv1.SpeakerSummary, 0, len(entries))
	for _, e := range entries {
		summaries = append(summaries, &commonv1.SpeakerSummary{
			SpeakerId:    strconv.FormatInt(e.ID, 10),
			AttendeeId:   strconv.FormatInt(e.AttendeeID, 10),
			FullName:     e.AttendeeName,
			SpeakerType:  e.Type,
			State:        e.Status,
			Priority:     e.Priority,
			Quoted:       e.GenderQuoted,
			FirstSpeaker: e.FirstSpeaker,
			Mine:         currentAttendeeID != 0 && e.AttendeeID == currentAttendeeID,
		})
	}
	return summaries
}
