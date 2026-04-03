package moderationservice

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	commonv1 "github.com/Y4shin/open-caucus/gen/go/conference/common/v1"
	moderationv1 "github.com/Y4shin/open-caucus/gen/go/conference/moderation/v1"
	apierrors "github.com/Y4shin/open-caucus/internal/api/errors"
	"github.com/Y4shin/open-caucus/internal/broker"
	"github.com/Y4shin/open-caucus/internal/repository"
	serviceauthz "github.com/Y4shin/open-caucus/internal/services/authz"
)

// MeetingInvalidationEvent is the JSON payload sent over the SSE stream when
// meeting-scoped state changes.
type MeetingInvalidationEvent struct {
	Type       string   `json:"type"`
	MeetingID  string   `json:"meetingId"`
	Scope      []string `json:"scope"`
	Version    uint64   `json:"version"`
	OccurredAt string   `json:"occurredAt"`
}

type Service struct {
	repo   repository.Repository
	broker broker.Broker
}

func New(repo repository.Repository, b broker.Broker) *Service {
	return &Service{repo: repo, broker: b}
}

func (s *Service) GetModerationView(ctx context.Context, committeeSlug, meetingIDStr string) (*moderationv1.GetModerationViewResponse, error) {
	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
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

	attendees, err := s.repo.ListAttendeesForMeeting(ctx, meetingID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to list attendees", err)
	}

	var totalCount, guestCount, chairCount int32
	for _, a := range attendees {
		totalCount++
		if a.UserID == nil {
			guestCount++
		}
		if a.IsChair {
			chairCount++
		}
	}

	var activeAP *commonv1.AgendaPointSummary
	var speakerSummary *moderationv1.ModerationSpeakerSummaryBlock

	if meeting.CurrentAgendaPointID != nil {
		ap, err := s.repo.GetAgendaPointByID(ctx, *meeting.CurrentAgendaPointID)
		if err == nil {
			activeAP = &commonv1.AgendaPointSummary{
				AgendaPointId: strconv.FormatInt(ap.ID, 10),
				Title:         ap.Title,
				IsActive:      true,
			}

			speakers, err := s.repo.ListSpeakersForAgendaPoint(ctx, ap.ID)
			if err == nil {
				hasActive := false
				waitingCount := int32(0)
				for _, sp := range speakers {
					if sp.Status == "SPEAKING" {
						hasActive = true
					}
					if sp.Status == "WAITING" {
						waitingCount++
					}
				}
				speakerSummary = &moderationv1.ModerationSpeakerSummaryBlock{
					TotalCount:       int32(len(speakers)),
					HasActiveSpeaker: hasActive,
					WaitingCount:     waitingCount,
				}
			}
		}
	}

	if speakerSummary == nil {
		speakerSummary = &moderationv1.ModerationSpeakerSummaryBlock{}
	}

	caps := []*commonv1.Capability{
		{Key: "moderation.view", Allowed: true},
		{Key: "moderation.manage_attendees", Allowed: true},
		{Key: "moderation.manage_speakers", Allowed: true},
		{Key: "moderation.toggle_signup", Allowed: true},
		{Key: "moderation.manage_agenda", Allowed: true},
	}

	isActiveMeeting := committee.CurrentMeetingID != nil && *committee.CurrentMeetingID == meetingID

	view := &moderationv1.ModerationView{
		Meeting: &moderationv1.ModerationMeetingSummary{
			CommitteeSlug: committeeSlug,
			MeetingId:     strconv.FormatInt(meetingID, 10),
			MeetingName:   meeting.Name,
			CommitteeName: committee.Name,
		},
		Version: uint64(meeting.Version),
		Attendees: &moderationv1.ModerationAttendeeSummaryBlock{
			SignupOpen:     meeting.SignupOpen,
			TotalCount:     totalCount,
			GuestCount:     guestCount,
			ChairCount:     chairCount,
			ShowSelfSignup: isActiveMeeting && meeting.SignupOpen,
		},
		ActiveAgendaPoint: activeAP,
		Speakers:          speakerSummary,
		Capabilities:      caps,
		Settings: &moderationv1.ModerationMeetingSettingsBlock{
			GenderQuotationEnabled:       meeting.GenderQuotationEnabled,
			FirstSpeakerQuotationEnabled: meeting.FirstSpeakerQuotationEnabled,
			ModeratorAttendeeId:          optionalInt64String(meeting.ModeratorID),
		},
	}

	return &moderationv1.GetModerationViewResponse{View: view}, nil
}

func (s *Service) ToggleSignupOpen(ctx context.Context, committeeSlug, meetingIDStr string, desiredOpen bool, expectedVersion uint64) (*moderationv1.ToggleSignupOpenResponse, error) {
	if err := serviceauthz.RequireChairperson(ctx, s.repo, committeeSlug); err != nil {
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

	// Version check: expectedVersion == 0 means skip check
	if expectedVersion != 0 && uint64(meeting.Version) != expectedVersion {
		return nil, apierrors.New(apierrors.KindConflict, "meeting has been modified since last read; please refresh")
	}

	newVersion, err := s.repo.SetMeetingSignupOpenWithVersion(ctx, meetingID, desiredOpen)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to toggle signup", err)
	}

	invalidatedViews := []string{"moderation", "live"}

	s.publishInvalidation(meetingID, "attendees.updated", invalidatedViews, uint64(newVersion))

	return &moderationv1.ToggleSignupOpenResponse{
		MeetingId:        strconv.FormatInt(meetingID, 10),
		SignupOpen:       desiredOpen,
		Version:          uint64(newVersion),
		InvalidatedViews: invalidatedViews,
	}, nil
}

func (s *Service) SetMeetingQuotation(ctx context.Context, committeeSlug, meetingIDStr string, genderQuotationEnabled, firstSpeakerQuotationEnabled bool) (*moderationv1.SetMeetingQuotationResponse, error) {
	if err := serviceauthz.RequireChairperson(ctx, s.repo, committeeSlug); err != nil {
		return nil, err
	}

	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}

	if err := s.repo.SetMeetingGenderQuotation(ctx, meetingID, genderQuotationEnabled); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to set gender quotation", err)
	}
	if err := s.repo.SetMeetingFirstSpeakerQuotation(ctx, meetingID, firstSpeakerQuotationEnabled); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to set first-speaker quotation", err)
	}

	meeting, err := s.repo.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "meeting not found")
	}
	if meeting.CurrentAgendaPointID != nil {
		if err := s.repo.RecomputeSpeakerOrder(ctx, *meeting.CurrentAgendaPointID); err != nil {
			return nil, apierrors.Wrap(apierrors.KindInternal, "failed to recompute speaker order", err)
		}
	}

	invalidatedViews := []string{"moderation", "live"}
	s.publishInvalidation(meetingID, "speakers.updated", invalidatedViews, uint64(meeting.Version))

	return &moderationv1.SetMeetingQuotationResponse{
		MeetingId:                    strconv.FormatInt(meetingID, 10),
		GenderQuotationEnabled:       genderQuotationEnabled,
		FirstSpeakerQuotationEnabled: firstSpeakerQuotationEnabled,
		Version:                      uint64(meeting.Version),
		InvalidatedViews:             invalidatedViews,
	}, nil
}

func (s *Service) SetMeetingModerator(ctx context.Context, committeeSlug, meetingIDStr, moderatorAttendeeIDStr string) (*moderationv1.SetMeetingModeratorResponse, error) {
	if err := serviceauthz.RequireChairperson(ctx, s.repo, committeeSlug); err != nil {
		return nil, err
	}

	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}

	var moderatorID *int64
	if moderatorAttendeeIDStr != "" {
		id, err := strconv.ParseInt(moderatorAttendeeIDStr, 10, 64)
		if err != nil {
			return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid moderator attendee id")
		}
		moderatorID = &id
	}

	if err := s.repo.SetMeetingModerator(ctx, meetingID, moderatorID); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to set meeting moderator", err)
	}

	meeting, err := s.repo.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "meeting not found")
	}

	invalidatedViews := []string{"moderation", "live"}
	s.publishInvalidation(meetingID, "meeting.updated", invalidatedViews, uint64(meeting.Version))

	return &moderationv1.SetMeetingModeratorResponse{
		MeetingId:           strconv.FormatInt(meetingID, 10),
		ModeratorAttendeeId: moderatorAttendeeIDStr,
		Version:             uint64(meeting.Version),
		InvalidatedViews:    invalidatedViews,
	}, nil
}

func (s *Service) publishInvalidation(meetingID int64, eventType string, scope []string, version uint64) {
	evt := MeetingInvalidationEvent{
		Type:       eventType,
		MeetingID:  strconv.FormatInt(meetingID, 10),
		Scope:      scope,
		Version:    version,
		OccurredAt: time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.Marshal(evt)
	if err != nil {
		return
	}
	mid := meetingID
	s.broker.Publish(broker.SSEEvent{
		Event:     eventType,
		Data:      data,
		MeetingID: &mid,
	})
}

func (s *Service) requireChairperson(ctx context.Context, committeeSlug string) error {
	return serviceauthz.RequireChairperson(ctx, s.repo, committeeSlug)
}

func optionalInt64String(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}
