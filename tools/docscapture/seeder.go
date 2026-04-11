package docscapture

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/Y4shin/open-caucus/internal/repository"
	"github.com/Y4shin/open-caucus/internal/repository/model"
	"github.com/Y4shin/open-caucus/internal/storage"
)

// Seeder provides deterministic fixture helpers for docs-capture scripts.
type Seeder struct {
	repo    repository.Repository
	storage storage.Service
}

func NewSeeder(repo repository.Repository, storage storage.Service) *Seeder {
	return &Seeder{
		repo:    repo,
		storage: storage,
	}
}

func (s *Seeder) Storage() storage.Service {
	return s.storage
}

func (s *Seeder) CreateAccount(ctx context.Context, username, password, fullName string) (*model.Account, error) {
	if username == "" || password == "" {
		return nil, fmt.Errorf("username and password are required")
	}
	if fullName == "" {
		fullName = username
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		return nil, fmt.Errorf("hash password for account %q: %w", username, err)
	}

	account, err := s.repo.CreateAccount(ctx, username, fullName, string(hash))
	if err != nil {
		return nil, fmt.Errorf("create account %q: %w", username, err)
	}
	return account, nil
}

func (s *Seeder) CreateAdminAccount(ctx context.Context, username, password, fullName string) (*model.Account, error) {
	account, err := s.CreateAccount(ctx, username, password, fullName)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetAccountIsAdmin(ctx, account.ID, true); err != nil {
		return nil, fmt.Errorf("set account %q as admin: %w", username, err)
	}
	return account, nil
}

func (s *Seeder) CreateOAuthAccount(ctx context.Context, username, fullName string, admin bool) (*model.Account, error) {
	account, err := s.repo.CreateOAuthAccount(ctx, username, fullName)
	if err != nil {
		return nil, fmt.Errorf("create oauth account %q: %w", username, err)
	}
	if admin {
		if err := s.repo.SetAccountIsAdmin(ctx, account.ID, true); err != nil {
			return nil, fmt.Errorf("set oauth account %q as admin: %w", username, err)
		}
	}
	return account, nil
}

func (s *Seeder) CreateCommittee(ctx context.Context, name, slug string) error {
	if err := s.repo.CreateCommitteeWithSlug(ctx, name, slug); err != nil {
		return fmt.Errorf("create committee %q: %w", slug, err)
	}
	return nil
}

func (s *Seeder) CreateCommitteeUser(
	ctx context.Context,
	committeeSlug, username, password, fullName string,
	quoted bool,
	role string,
) error {
	committeeID, err := s.repo.GetCommitteeIDBySlug(ctx, committeeSlug)
	if err != nil {
		return fmt.Errorf("lookup committee %q: %w", committeeSlug, err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		return fmt.Errorf("hash password for user %q: %w", username, err)
	}
	if err := s.repo.CreateUser(ctx, committeeID, username, string(hash), fullName, quoted, role); err != nil {
		return fmt.Errorf("create committee user %q in %q: %w", username, committeeSlug, err)
	}
	return nil
}

func (s *Seeder) CreateMeeting(ctx context.Context, committeeSlug, name, description, secret string, signupOpen bool) error {
	committeeID, err := s.repo.GetCommitteeIDBySlug(ctx, committeeSlug)
	if err != nil {
		return fmt.Errorf("lookup committee %q: %w", committeeSlug, err)
	}
	if secret == "" {
		secret = "docs-capture-meeting-secret"
	}
	if err := s.repo.CreateMeeting(ctx, committeeID, name, description, secret, signupOpen, nil, nil); err != nil {
		return fmt.Errorf("create meeting %q in %q: %w", name, committeeSlug, err)
	}
	return nil
}

func (s *Seeder) CreateOAuthCommitteeRule(ctx context.Context, committeeSlug, groupName, role string) (*model.OAuthCommitteeGroupRule, error) {
	rule, err := s.repo.CreateOAuthCommitteeGroupRuleByCommitteeSlug(ctx, committeeSlug, groupName, role)
	if err != nil {
		return nil, fmt.Errorf("create oauth committee rule for %q: %w", committeeSlug, err)
	}
	return rule, nil
}

func (s *Seeder) CommitteeID(ctx context.Context, committeeSlug string) (int64, error) {
	id, err := s.repo.GetCommitteeIDBySlug(ctx, committeeSlug)
	if err != nil {
		return 0, fmt.Errorf("lookup committee %q: %w", committeeSlug, err)
	}
	return id, nil
}

func (s *Seeder) MeetingByName(ctx context.Context, committeeSlug, name string) (*model.Meeting, error) {
	meetings, err := s.repo.ListMeetingsForCommittee(ctx, committeeSlug, 200, 0)
	if err != nil {
		return nil, fmt.Errorf("list meetings for committee %q: %w", committeeSlug, err)
	}
	normalized := strings.TrimSpace(name)
	for _, meeting := range meetings {
		if meeting != nil && strings.TrimSpace(meeting.Name) == normalized {
			return meeting, nil
		}
	}
	return nil, fmt.Errorf("meeting %q not found in committee %q", name, committeeSlug)
}

func (s *Seeder) SetActiveMeetingByName(ctx context.Context, committeeSlug, meetingName string) (*model.Meeting, error) {
	meeting, err := s.MeetingByName(ctx, committeeSlug, meetingName)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetActiveMeeting(ctx, committeeSlug, &meeting.ID); err != nil {
		return nil, fmt.Errorf("set active meeting %q for committee %q: %w", meetingName, committeeSlug, err)
	}
	return meeting, nil
}

func (s *Seeder) UserByCommitteeAndUsername(ctx context.Context, committeeSlug, username string) (*model.User, error) {
	user, err := s.repo.GetUserByCommitteeAndUsername(ctx, committeeSlug, username)
	if err != nil {
		return nil, fmt.Errorf("lookup user %q in committee %q: %w", username, committeeSlug, err)
	}
	return user, nil
}

func (s *Seeder) CreateAgendaPoint(ctx context.Context, committeeSlug, meetingName, title string) (*model.AgendaPoint, error) {
	meeting, err := s.MeetingByName(ctx, committeeSlug, meetingName)
	if err != nil {
		return nil, err
	}
	agendaPoint, err := s.repo.CreateAgendaPoint(ctx, meeting.ID, title)
	if err != nil {
		return nil, fmt.Errorf("create agenda point %q for meeting %q: %w", title, meetingName, err)
	}
	return agendaPoint, nil
}

func (s *Seeder) CreateSubAgendaPoint(
	ctx context.Context,
	committeeSlug, meetingName string,
	parentID int64,
	title string,
) (*model.AgendaPoint, error) {
	meeting, err := s.MeetingByName(ctx, committeeSlug, meetingName)
	if err != nil {
		return nil, err
	}
	agendaPoint, err := s.repo.CreateSubAgendaPoint(ctx, meeting.ID, parentID, title)
	if err != nil {
		return nil, fmt.Errorf("create sub agenda point %q for meeting %q: %w", title, meetingName, err)
	}
	return agendaPoint, nil
}

func (s *Seeder) SetCurrentAgendaPointByName(ctx context.Context, committeeSlug, meetingName string, agendaPointID int64) error {
	meeting, err := s.MeetingByName(ctx, committeeSlug, meetingName)
	if err != nil {
		return err
	}
	if err := s.repo.SetCurrentAgendaPoint(ctx, meeting.ID, &agendaPointID); err != nil {
		return fmt.Errorf("set current agenda point %d for meeting %q: %w", agendaPointID, meetingName, err)
	}
	return nil
}

func (s *Seeder) CreateGuestAttendee(ctx context.Context, committeeSlug, meetingName, fullName, secret string, quoted bool) (*model.Attendee, error) {
	meeting, err := s.MeetingByName(ctx, committeeSlug, meetingName)
	if err != nil {
		return nil, err
	}
	attendee, err := s.repo.CreateAttendee(ctx, meeting.ID, nil, fullName, secret, quoted)
	if err != nil {
		return nil, fmt.Errorf("create guest attendee %q for meeting %q: %w", fullName, meetingName, err)
	}
	return attendee, nil
}

func (s *Seeder) CreateUserAttendee(ctx context.Context, committeeSlug, meetingName, username string) (*model.Attendee, error) {
	meeting, err := s.MeetingByName(ctx, committeeSlug, meetingName)
	if err != nil {
		return nil, err
	}
	user, err := s.UserByCommitteeAndUsername(ctx, committeeSlug, username)
	if err != nil {
		return nil, err
	}
	attendee, err := s.repo.CreateAttendee(ctx, meeting.ID, &user.ID, user.FullName, "", user.Quoted)
	if err != nil {
		return nil, fmt.Errorf("create attendee for user %q on meeting %q: %w", username, meetingName, err)
	}
	return attendee, nil
}

func (s *Seeder) SetAttendeeChair(ctx context.Context, attendeeID int64, isChair bool) error {
	if err := s.repo.SetAttendeeIsChair(ctx, attendeeID, isChair); err != nil {
		return fmt.Errorf("set attendee %d chair=%t: %w", attendeeID, isChair, err)
	}
	return nil
}

func (s *Seeder) AddSpeaker(ctx context.Context, agendaPointID, attendeeID int64, speakerType string, genderQuoted, firstSpeaker bool) (*model.SpeakerEntry, error) {
	speaker, err := s.repo.AddSpeaker(ctx, agendaPointID, attendeeID, speakerType, genderQuoted, firstSpeaker)
	if err != nil {
		return nil, fmt.Errorf("add speaker attendee=%d agenda-point=%d: %w", attendeeID, agendaPointID, err)
	}
	return speaker, nil
}

func (s *Seeder) SetSpeakerSpeaking(ctx context.Context, speakerID, agendaPointID int64) error {
	if err := s.repo.SetSpeakerSpeaking(ctx, speakerID, agendaPointID); err != nil {
		return fmt.Errorf("set speaker %d speaking: %w", speakerID, err)
	}
	return nil
}

func (s *Seeder) SetSpeakerDone(ctx context.Context, speakerID int64) error {
	if err := s.repo.SetSpeakerDone(ctx, speakerID); err != nil {
		return fmt.Errorf("set speaker %d done: %w", speakerID, err)
	}
	return nil
}

func (s *Seeder) RecomputeSpeakerOrder(ctx context.Context, agendaPointID int64) error {
	return s.repo.RecomputeSpeakerOrder(ctx, agendaPointID)
}

func (s *Seeder) CreateAttachment(
	ctx context.Context,
	agendaPointID int64,
	filename string,
	contentType string,
	content string,
	label *string,
) (*model.AgendaAttachment, error) {
	if strings.TrimSpace(filename) == "" {
		filename = "document.txt"
	}
	if strings.TrimSpace(contentType) == "" {
		contentType = "text/plain"
	}
	storagePath, sizeBytes, err := s.storage.Store(filename, contentType, strings.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("store attachment content %q: %w", filename, err)
	}
	blob, err := s.repo.CreateBlob(ctx, filename, contentType, sizeBytes, storagePath)
	if err != nil {
		return nil, fmt.Errorf("create attachment blob %q: %w", filename, err)
	}
	attachment, err := s.repo.CreateAttachment(ctx, agendaPointID, blob.ID, label)
	if err != nil {
		return nil, fmt.Errorf("create attachment %q: %w", filename, err)
	}
	return attachment, nil
}

func (s *Seeder) SetCurrentAttachment(ctx context.Context, agendaPointID, attachmentID int64) error {
	if err := s.repo.SetCurrentAttachment(ctx, agendaPointID, attachmentID); err != nil {
		return fmt.Errorf("set current attachment %d for agenda point %d: %w", attachmentID, agendaPointID, err)
	}
	return nil
}

func (s *Seeder) CreateVoteDefinition(
	ctx context.Context,
	meetingID int64,
	agendaPointID int64,
	name string,
	visibility string,
	minSelections int64,
	maxSelections int64,
	options []repository.VoteOptionInput,
) (*model.VoteDefinition, error) {
	vote, err := s.repo.CreateVoteDefinition(ctx, meetingID, agendaPointID, name, visibility, minSelections, maxSelections)
	if err != nil {
		return nil, fmt.Errorf("create vote definition %q: %w", name, err)
	}
	if len(options) > 0 {
		if err := s.repo.ReplaceVoteOptions(ctx, vote.ID, options); err != nil {
			return nil, fmt.Errorf("replace vote options for %q: %w", name, err)
		}
	}
	return vote, nil
}

func (s *Seeder) OpenVoteForAttendees(ctx context.Context, voteID int64, attendeeIDs []int64) (*model.VoteDefinition, error) {
	vote, err := s.repo.OpenVoteWithEligibleVoters(ctx, voteID, attendeeIDs)
	if err != nil {
		return nil, fmt.Errorf("open vote %d: %w", voteID, err)
	}
	return vote, nil
}
