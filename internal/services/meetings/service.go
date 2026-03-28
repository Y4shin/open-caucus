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

func (s *Service) GetJoinMeeting(ctx context.Context, committeeSlug, meetingIDStr string) (*meetingsv1.GetJoinMeetingResponse, error) {
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

	currentAttendee, isAttendee := s.resolveCurrentAttendee(ctx, committeeSlug, meetingID)
	view := &meetingsv1.JoinMeetingView{
		CommitteeSlug:   committeeSlug,
		MeetingId:       strconv.FormatInt(meetingID, 10),
		MeetingName:     meeting.Name,
		CommitteeName:   committee.Name,
		SignupOpen:      meeting.SignupOpen,
		CurrentAttendee: currentAttendee,
		Capabilities:    s.buildJoinMeetingCapabilities(ctx, committeeSlug, meeting.SignupOpen, isAttendee),
	}

	return &meetingsv1.GetJoinMeetingResponse{Meeting: view}, nil
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
	if err := s.requireLiveMeetingAccess(ctx, committeeSlug, committee, meeting); err != nil {
		return nil, err
	}

	// Resolve actor context for capability decisions
	attendeeID, isAttendee := s.resolveAttendeeID(ctx, committeeSlug, meetingID)

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

	view := &meetingsv1.LiveMeetingView{
		CommitteeSlug:     committeeSlug,
		MeetingId:         strconv.FormatInt(meetingID, 10),
		MeetingName:       meeting.Name,
		CommitteeName:     committee.Name,
		Version:           uint64(meeting.Version),
		ActiveAgendaPoint: activeAP,
		Speakers:          speakers,
		CurrentDocument:   currentDoc,
		Capabilities:      caps,
	}

	return &meetingsv1.GetLiveMeetingResponse{Meeting: view}, nil
}

func (s *Service) resolveCurrentAttendee(ctx context.Context, committeeSlug string, meetingID int64) (*commonv1.AttendeeSummary, bool) {
	sd, ok := session.GetSession(ctx)
	if !ok || sd == nil || sd.IsExpired() {
		return nil, false
	}

	if sd.IsGuestSession() && sd.AttendeeID != nil {
		attendee, err := s.repo.GetAttendeeByID(ctx, *sd.AttendeeID)
		if err == nil && attendee.MeetingID == meetingID {
			return toAttendeeSummary(attendee), true
		}
		return nil, false
	}

	if sd.IsAccountSession() && sd.AccountID != nil {
		membership, err := s.repo.GetUserMembershipByAccountIDAndSlug(ctx, *sd.AccountID, committeeSlug)
		if err != nil {
			return nil, false
		}

		attendee, err := s.repo.GetAttendeeByUserIDAndMeetingID(ctx, membership.ID, meetingID)
		if err == nil {
			return toAttendeeSummary(attendee), true
		}
	}

	return nil, false
}

func (s *Service) resolveAttendeeID(ctx context.Context, committeeSlug string, meetingID int64) (int64, bool) {
	attendee, ok := s.resolveCurrentAttendee(ctx, committeeSlug, meetingID)
	if !ok || attendee == nil {
		return 0, false
	}
	id, err := strconv.ParseInt(attendee.AttendeeId, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

func (s *Service) buildLiveMeetingCapabilities(ctx context.Context, meeting *model.Meeting, committee *model.Committee, isAttendee bool) *meetingsv1.LiveMeetingCapabilities {
	isActiveMeeting := committee.CurrentMeetingID != nil && *committee.CurrentMeetingID == meeting.ID

	sd, _ := session.GetSession(ctx)
	isAccountSession := sd != nil && !sd.IsExpired() && sd.IsAccountSession()

	return &meetingsv1.LiveMeetingCapabilities{
		CanSelfSignup:            meeting.SignupOpen && !isAttendee && isAccountSession,
		CanAddRegularSpeech:      isAttendee && isActiveMeeting && meeting.CurrentAgendaPointID != nil,
		CanAddPointOfOrderSpeech: isAttendee && isActiveMeeting && meeting.CurrentAgendaPointID != nil,
		CanVote:                  false, // determined per-vote-definition, not at meeting level
		CanViewCurrentDocument:   isActiveMeeting && meeting.CurrentAgendaPointID != nil,
	}
}

func (s *Service) requireLiveMeetingAccess(ctx context.Context, committeeSlug string, committee *model.Committee, meeting *model.Meeting) error {
	sd, ok := session.GetSession(ctx)
	if !ok || sd == nil || sd.IsExpired() {
		return nil
	}

	if sd.IsAccountSession() && sd.AccountID != nil {
		account, err := s.repo.GetAccountByID(ctx, *sd.AccountID)
		if err == nil && account.IsAdmin {
			return nil
		}

		membership, err := s.repo.GetUserMembershipByAccountIDAndSlug(ctx, *sd.AccountID, committeeSlug)
		if err != nil {
			return apierrors.New(apierrors.KindPermissionDenied, "not a member of this committee")
		}
		if membership.Role == "member" && (committee.CurrentMeetingID == nil || *committee.CurrentMeetingID != meeting.ID) {
			return apierrors.New(apierrors.KindPermissionDenied, "meeting is not currently active")
		}
	}

	if sd.IsGuestSession() && sd.AttendeeID != nil {
		attendee, err := s.repo.GetAttendeeByID(ctx, *sd.AttendeeID)
		if err != nil || attendee.MeetingID != meeting.ID {
			return apierrors.New(apierrors.KindPermissionDenied, "attendee is not signed in to this meeting")
		}
	}

	return nil
}

func (s *Service) buildJoinMeetingCapabilities(ctx context.Context, committeeSlug string, signupOpen bool, isAttendee bool) *meetingsv1.JoinMeetingCapabilities {
	sd, _ := session.GetSession(ctx)
	isAuthenticated := sd != nil && !sd.IsExpired()
	isAccountSession := isAuthenticated && sd.IsAccountSession() && sd.AccountID != nil

	// Members can self-signup regardless of whether signup_open is set — signupOpen only gates guests.
	canSelfSignup := false
	if isAccountSession && !isAttendee {
		if _, err := s.repo.GetUserMembershipByAccountIDAndSlug(ctx, *sd.AccountID, committeeSlug); err == nil {
			canSelfSignup = true
		}
	}

	return &meetingsv1.JoinMeetingCapabilities{
		CanSelfSignup:       canSelfSignup,
		CanGuestJoin:        signupOpen && !isAuthenticated,
		AlreadyJoined:       isAttendee,
		CanUseAttendeeLogin: !isAccountSession && !isAttendee,
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

func toAttendeeSummary(attendee *model.Attendee) *commonv1.AttendeeSummary {
	return &commonv1.AttendeeSummary{
		AttendeeId:     strconv.FormatInt(attendee.ID, 10),
		FullName:       attendee.FullName,
		AttendeeNumber: attendee.AttendeeNumber,
		IsChair:        attendee.IsChair,
		IsGuest:        attendee.UserID == nil,
		Quoted:         attendee.Quoted,
	}
}
