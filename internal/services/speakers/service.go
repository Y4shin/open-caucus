package speakerservice

import (
	"context"
	"strconv"

	commonv1 "github.com/Y4shin/open-caucus/gen/go/conference/common/v1"
	speakersv1 "github.com/Y4shin/open-caucus/gen/go/conference/speakers/v1"
	apierrors "github.com/Y4shin/open-caucus/internal/api/errors"
	"github.com/Y4shin/open-caucus/internal/broker"
	"github.com/Y4shin/open-caucus/internal/repository"
	"github.com/Y4shin/open-caucus/internal/repository/model"
	"github.com/Y4shin/open-caucus/internal/session"
)

type Service struct {
	repo   repository.Repository
	broker broker.Broker
}

func New(repo repository.Repository, b broker.Broker) *Service {
	return &Service{repo: repo, broker: b}
}

// ListSpeakers returns the speaker queue for the active agenda point.
func (s *Service) ListSpeakers(ctx context.Context, committeeSlug, meetingIDStr string) (*speakersv1.ListSpeakersResponse, error) {
	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}

	meeting, err := s.repo.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "meeting not found")
	}

	view, err := s.buildQueueView(ctx, committeeSlug, meetingID, meeting)
	if err != nil {
		return nil, err
	}
	return &speakersv1.ListSpeakersResponse{View: view}, nil
}

// AddSpeaker adds an attendee to the speaker queue. Attendees may add
// themselves; chairpersons may add any attendee.
func (s *Service) AddSpeaker(ctx context.Context, committeeSlug, meetingIDStr, attendeeIDStr, speakerType string) (*speakersv1.AddSpeakerResponse, error) {
	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}

	meeting, err := s.repo.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "meeting not found")
	}

	if meeting.CurrentAgendaPointID == nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "no active agenda point")
	}

	attendeeID, err := s.resolveAttendeeID(ctx, meetingID, attendeeIDStr)
	if err != nil {
		return nil, err
	}

	ap, err := s.repo.GetAgendaPointByID(ctx, *meeting.CurrentAgendaPointID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to load active agenda point", err)
	}

	effectiveOrder := meeting.QuotationOrder
	if ap.QuotationOrder != nil {
		effectiveOrder = *ap.QuotationOrder
	}

	attendee, err := s.repo.GetAttendeeByID(ctx, attendeeID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to load attendee", err)
	}

	genderQuoted := attendee.Quoted && containsQuotationType(effectiveOrder, "gender")
	firstSpeaker := false
	if hasPrev, err := s.repo.HasAttendeeSpokenOnAgendaPoint(ctx, ap.ID, attendeeID); err == nil {
		firstSpeaker = !hasPrev
	}

	if speakerType == "" {
		speakerType = "regular"
	}

	if _, err := s.repo.AddSpeaker(ctx, ap.ID, attendeeID, speakerType, genderQuoted, firstSpeaker); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to add speaker", err)
	}

	if err := s.repo.RecomputeSpeakerOrder(ctx, ap.ID); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to recompute speaker order", err)
	}

	s.publishSpeakersUpdated(meetingID)

	view, err := s.buildQueueView(ctx, committeeSlug, meetingID, meeting)
	if err != nil {
		return nil, err
	}

	return &speakersv1.AddSpeakerResponse{
		View:             view,
		InvalidatedViews: []string{"moderation", "live"},
	}, nil
}

// RemoveSpeaker removes a speaker entry from the queue.
func (s *Service) RemoveSpeaker(ctx context.Context, committeeSlug, meetingIDStr, speakerIDStr string) (*speakersv1.RemoveSpeakerResponse, error) {
	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}

	speakerID, err := strconv.ParseInt(speakerIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid speaker id")
	}

	entry, err := s.repo.GetSpeakerEntryByID(ctx, speakerID)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "speaker entry not found")
	}

	if err := s.repo.SetSpeakerWithdrawn(ctx, speakerID); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to remove speaker", err)
	}

	if err := s.repo.RecomputeSpeakerOrder(ctx, entry.AgendaPointID); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to recompute speaker order", err)
	}

	s.publishSpeakersUpdated(meetingID)

	meeting, err := s.repo.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to reload meeting", err)
	}

	view, err := s.buildQueueView(ctx, committeeSlug, meetingID, meeting)
	if err != nil {
		return nil, err
	}

	return &speakersv1.RemoveSpeakerResponse{
		View:             view,
		InvalidatedViews: []string{"moderation", "live"},
	}, nil
}

// SetSpeakerSpeaking marks a speaker entry as currently speaking.
func (s *Service) SetSpeakerSpeaking(ctx context.Context, committeeSlug, meetingIDStr, speakerIDStr string) (*speakersv1.SetSpeakerSpeakingResponse, error) {
	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}

	speakerID, err := strconv.ParseInt(speakerIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid speaker id")
	}

	entry, err := s.repo.GetSpeakerEntryByID(ctx, speakerID)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "speaker entry not found")
	}

	if err := s.repo.SetSpeakerSpeaking(ctx, speakerID, entry.AgendaPointID); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to set speaker speaking", err)
	}

	s.publishSpeakersUpdated(meetingID)

	meeting, err := s.repo.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to reload meeting", err)
	}

	view, err := s.buildQueueView(ctx, committeeSlug, meetingID, meeting)
	if err != nil {
		return nil, err
	}

	return &speakersv1.SetSpeakerSpeakingResponse{
		View:             view,
		InvalidatedViews: []string{"moderation", "live"},
	}, nil
}

// SetSpeakerDone marks a speaker entry as done.
func (s *Service) SetSpeakerDone(ctx context.Context, committeeSlug, meetingIDStr, speakerIDStr string) (*speakersv1.SetSpeakerDoneResponse, error) {
	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}

	speakerID, err := strconv.ParseInt(speakerIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid speaker id")
	}

	if err := s.repo.SetSpeakerDone(ctx, speakerID); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to set speaker done", err)
	}

	s.publishSpeakersUpdated(meetingID)

	meeting, err := s.repo.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to reload meeting", err)
	}

	view, err := s.buildQueueView(ctx, committeeSlug, meetingID, meeting)
	if err != nil {
		return nil, err
	}

	return &speakersv1.SetSpeakerDoneResponse{
		View:             view,
		InvalidatedViews: []string{"moderation", "live"},
	}, nil
}

// SetSpeakerPriority sets or clears the priority flag for a speaker entry.
func (s *Service) SetSpeakerPriority(ctx context.Context, committeeSlug, meetingIDStr, speakerIDStr string, priority bool) (*speakersv1.SetSpeakerPriorityResponse, error) {
	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}

	speakerID, err := strconv.ParseInt(speakerIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid speaker id")
	}

	entry, err := s.repo.GetSpeakerEntryByID(ctx, speakerID)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "speaker entry not found")
	}

	if err := s.repo.SetSpeakerPriority(ctx, speakerID, priority); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to set speaker priority", err)
	}

	if err := s.repo.RecomputeSpeakerOrder(ctx, entry.AgendaPointID); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to recompute speaker order", err)
	}

	s.publishSpeakersUpdated(meetingID)

	meeting, err := s.repo.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to reload meeting", err)
	}

	view, err := s.buildQueueView(ctx, committeeSlug, meetingID, meeting)
	if err != nil {
		return nil, err
	}

	return &speakersv1.SetSpeakerPriorityResponse{
		View:             view,
		InvalidatedViews: []string{"moderation", "live"},
	}, nil
}

func (s *Service) buildQueueView(ctx context.Context, committeeSlug string, meetingID int64, meeting *model.Meeting) (*speakersv1.SpeakerQueueView, error) {
	view := &speakersv1.SpeakerQueueView{
		MeetingId:     strconv.FormatInt(meetingID, 10),
		CommitteeSlug: committeeSlug,
	}

	if meeting.CurrentAgendaPointID == nil {
		return view, nil
	}

	ap, err := s.repo.GetAgendaPointByID(ctx, *meeting.CurrentAgendaPointID)
	if err != nil {
		return view, nil
	}

	view.ActiveAgendaPointId = strconv.FormatInt(ap.ID, 10)
	view.ActiveAgendaPointTitle = ap.Title
	effectiveOrder := meeting.QuotationOrder
	if ap.QuotationOrder != nil {
		effectiveOrder = *ap.QuotationOrder
	}
	view.QuotationOrder = stringsToQuotationTypes(effectiveOrder)

	// Determine whether the calling actor can add themselves.
	view.CanAddSelf = s.canAddSelf(ctx, meetingID)

	entries, err := s.repo.ListSpeakersForAgendaPoint(ctx, ap.ID)
	if err != nil {
		return view, nil
	}

	callerAttendeeID := s.callerAttendeeID(ctx, meetingID)

	speakers := make([]*commonv1.SpeakerSummary, 0, len(entries))
	for _, e := range entries {
		if e.Status == "WITHDRAWN" {
			continue
		}
		mine := callerAttendeeID != 0 && e.AttendeeID == callerAttendeeID
		speakers = append(speakers, &commonv1.SpeakerSummary{
			SpeakerId:       strconv.FormatInt(e.ID, 10),
			AttendeeId:      strconv.FormatInt(e.AttendeeID, 10),
			FullName:        e.AttendeeName,
			SpeakerType:     e.Type,
			State:           e.Status,
			Priority:        e.Priority,
			Quoted:          e.GenderQuoted,
			FirstSpeaker:    e.FirstSpeaker,
			Mine:            mine,
			DurationSeconds: e.DurationSeconds,
		})
	}
	view.Speakers = speakers

	return view, nil
}

func (s *Service) resolveAttendeeID(ctx context.Context, meetingID int64, attendeeIDStr string) (int64, error) {
	if attendeeIDStr != "" {
		id, err := strconv.ParseInt(attendeeIDStr, 10, 64)
		if err != nil {
			return 0, apierrors.New(apierrors.KindInvalidArgument, "invalid attendee id")
		}
		return id, nil
	}

	// Use the caller's attendee record.
	sd, ok := session.GetSession(ctx)
	if !ok || sd == nil {
		return 0, apierrors.New(apierrors.KindUnauthenticated, "session required to add self as speaker")
	}

	if sd.IsGuestSession() && sd.AttendeeID != nil {
		return *sd.AttendeeID, nil
	}

	if sd.IsAccountSession() && sd.AccountID != nil {
		// Look up the attendee record for this user in the meeting.
		attendees, err := s.repo.ListAttendeesForMeeting(ctx, meetingID)
		if err != nil {
			return 0, apierrors.Wrap(apierrors.KindInternal, "failed to list attendees", err)
		}
		for _, a := range attendees {
			if a.UserID != nil {
				user, uErr := s.repo.GetUserByID(ctx, *a.UserID)
				if uErr != nil {
					continue
				}
				if user.AccountID != nil && *user.AccountID == *sd.AccountID {
					return a.ID, nil
				}
			}
		}
	}

	return 0, apierrors.New(apierrors.KindNotFound, "no attendee record found for the current session")
}

func (s *Service) canAddSelf(ctx context.Context, meetingID int64) bool {
	id := s.callerAttendeeID(ctx, meetingID)
	return id != 0
}

func (s *Service) callerAttendeeID(ctx context.Context, meetingID int64) int64 {
	sd, ok := session.GetSession(ctx)
	if !ok || sd == nil {
		return 0
	}
	if sd.IsGuestSession() && sd.AttendeeID != nil {
		return *sd.AttendeeID
	}
	if sd.IsAccountSession() && sd.AccountID != nil {
		attendees, err := s.repo.ListAttendeesForMeeting(ctx, meetingID)
		if err != nil {
			return 0
		}
		for _, a := range attendees {
			if a.UserID != nil {
				user, uErr := s.repo.GetUserByID(ctx, *a.UserID)
				if uErr != nil {
					continue
				}
				if user.AccountID != nil && *user.AccountID == *sd.AccountID {
					return a.ID
				}
			}
		}
	}
	return 0
}

func (s *Service) publishSpeakersUpdated(meetingID int64) {
	mid := meetingID
	s.broker.Publish(broker.SSEEvent{
		Event:     "speakers.updated",
		Data:      []byte(`{"type":"speakers.updated"}`),
		MeetingID: &mid,
	})
}

func containsQuotationType(order []string, qt string) bool {
	for _, s := range order {
		if s == qt {
			return true
		}
	}
	return false
}

func stringsToQuotationTypes(order []string) []commonv1.QuotationType {
	result := make([]commonv1.QuotationType, 0, len(order))
	for _, s := range order {
		switch s {
		case "gender":
			result = append(result, commonv1.QuotationType_QUOTATION_TYPE_GENDER)
		case "first_speaker":
			result = append(result, commonv1.QuotationType_QUOTATION_TYPE_FIRST_SPEAKER)
		}
	}
	return result
}
