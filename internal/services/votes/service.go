package voteservice

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strconv"

	votesv1 "github.com/Y4shin/conference-tool/gen/go/conference/votes/v1"
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

// GetVotesPanel returns the moderator votes panel for the active agenda point.
func (s *Service) GetVotesPanel(ctx context.Context, committeeSlug, meetingIDStr string) (*votesv1.GetVotesPanelResponse, error) {
	if err := s.requireChairperson(ctx, committeeSlug); err != nil {
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

	view := &votesv1.VotesPanelView{
		MeetingId:     meetingIDStr,
		CommitteeSlug: committeeSlug,
	}

	if meeting.CurrentAgendaPointID != nil {
		view.HasActiveAgendaPoint = true
		view.ActiveAgendaPointId = strconv.FormatInt(*meeting.CurrentAgendaPointID, 10)

		if ap, err := s.repo.GetAgendaPointByID(ctx, *meeting.CurrentAgendaPointID); err == nil {
			view.ActiveAgendaPointTitle = ap.Title
		}

		votes, err := s.repo.ListVoteDefinitionsForAgendaPoint(ctx, *meeting.CurrentAgendaPointID)
		if err == nil {
			for _, v := range votes {
				opts, _ := s.repo.ListVoteOptions(ctx, v.ID)
				rec := toVoteDefinitionRecord(v, opts)
				view.Votes = append(view.Votes, rec)

				if v.State == model.VoteStateOpen || v.State == model.VoteStateCounting {
					view.ActiveVote = rec
					if stats, err := s.repo.GetVoteSubmissionStatsLive(ctx, v.ID); err == nil {
						view.ActiveVoteStats = &votesv1.VoteStats{
							EligibleCount: stats.EligibleCount,
							CastCount:     stats.CastCount,
							BallotCount:   stats.BallotCount,
						}
					}
					if tallies, err := s.repo.GetVoteTallies(ctx, v.ID); err == nil {
						for _, t := range tallies {
							view.ActiveVoteTally = append(view.ActiveVoteTally, &votesv1.VoteTallyEntry{
								OptionId: strconv.FormatInt(t.OptionID, 10),
								Label:    t.Label,
								Count:    t.Count,
							})
						}
					}
				}
			}
		}
	}

	return &votesv1.GetVotesPanelResponse{View: view}, nil
}

// GetLiveVotePanel returns the attendee-facing vote panel.
func (s *Service) GetLiveVotePanel(ctx context.Context, committeeSlug, meetingIDStr string) (*votesv1.GetLiveVotePanelResponse, error) {
	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}

	meeting, err := s.repo.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "meeting not found")
	}

	view := &votesv1.LiveVotePanelView{
		MeetingId:     meetingIDStr,
		CommitteeSlug: committeeSlug,
	}

	if meeting.CurrentAgendaPointID == nil {
		return &votesv1.GetLiveVotePanelResponse{View: view}, nil
	}

	votes, err := s.repo.ListVoteDefinitionsForAgendaPoint(ctx, *meeting.CurrentAgendaPointID)
	if err != nil {
		return &votesv1.GetLiveVotePanelResponse{View: view}, nil
	}

	callerAttendeeID := s.callerAttendeeID(ctx, meetingID)

	for _, v := range votes {
		if v.State != model.VoteStateOpen {
			continue
		}
		opts, _ := s.repo.ListVoteOptions(ctx, v.ID)
		view.HasActiveVote = true
		view.ActiveVote = toVoteDefinitionRecord(v, opts)

		if callerAttendeeID != 0 {
			if voters, err := s.repo.ListEligibleVoters(ctx, v.ID); err == nil {
				for _, ev := range voters {
					if ev.AttendeeID == callerAttendeeID {
						view.IsEligible = true
						break
					}
				}
			}
			if casts, err := s.repo.ListVoteCasts(ctx, v.ID); err == nil {
				for _, c := range casts {
					if c.AttendeeID == callerAttendeeID {
						view.AlreadyVoted = true
						break
					}
				}
			}
		}
		break
	}

	return &votesv1.GetLiveVotePanelResponse{View: view}, nil
}

// CreateVote creates a new vote in draft state. Requires chairperson role.
func (s *Service) CreateVote(ctx context.Context, committeeSlug, meetingIDStr, name, visibility string, minSel, maxSel int64, optionLabels []string) (*votesv1.CreateVoteResponse, error) {
	if err := s.requireChairperson(ctx, committeeSlug); err != nil {
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

	if meeting.CurrentAgendaPointID == nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "no active agenda point")
	}

	if name == "" {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "vote name is required")
	}
	if visibility != model.VoteVisibilityOpen && visibility != model.VoteVisibilitySecret {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "visibility must be 'open' or 'secret'")
	}

	vote, err := s.repo.CreateVoteDefinition(ctx, meetingID, *meeting.CurrentAgendaPointID, name, visibility, minSel, maxSel)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to create vote", err)
	}

	options := parseOptions(optionLabels)
	if len(options) > 0 {
		if err := s.repo.ReplaceVoteOptions(ctx, vote.ID, options); err != nil {
			return nil, apierrors.Wrap(apierrors.KindInternal, "failed to set vote options", err)
		}
	}

	opts, _ := s.repo.ListVoteOptions(ctx, vote.ID)
	s.publishVotesUpdated(meetingID)

	return &votesv1.CreateVoteResponse{
		Vote:             toVoteDefinitionRecord(vote, opts),
		InvalidatedViews: []string{"moderation"},
	}, nil
}

// UpdateVoteDraft updates a vote still in draft state. Requires chairperson.
func (s *Service) UpdateVoteDraft(ctx context.Context, committeeSlug, meetingIDStr, voteIDStr, name, visibility string, minSel, maxSel int64, optionLabels []string) (*votesv1.UpdateVoteDraftResponse, error) {
	if err := s.requireChairperson(ctx, committeeSlug); err != nil {
		return nil, err
	}

	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}

	voteID, err := strconv.ParseInt(voteIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid vote id")
	}

	meeting, err := s.repo.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "meeting not found")
	}

	if meeting.CurrentAgendaPointID == nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "no active agenda point")
	}

	vote, err := s.repo.UpdateVoteDefinitionDraft(ctx, voteID, meetingID, *meeting.CurrentAgendaPointID, name, visibility, minSel, maxSel)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to update vote", err)
	}

	options := parseOptions(optionLabels)
	if len(options) > 0 {
		if err := s.repo.ReplaceVoteOptions(ctx, vote.ID, options); err != nil {
			return nil, apierrors.Wrap(apierrors.KindInternal, "failed to update vote options", err)
		}
	}

	opts, _ := s.repo.ListVoteOptions(ctx, vote.ID)
	s.publishVotesUpdated(meetingID)

	return &votesv1.UpdateVoteDraftResponse{
		Vote:             toVoteDefinitionRecord(vote, opts),
		InvalidatedViews: []string{"moderation"},
	}, nil
}

// OpenVote transitions a vote from draft to open. Requires chairperson.
func (s *Service) OpenVote(ctx context.Context, committeeSlug, meetingIDStr, voteIDStr string) (*votesv1.OpenVoteResponse, error) {
	if err := s.requireChairperson(ctx, committeeSlug); err != nil {
		return nil, err
	}

	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}

	voteID, err := strconv.ParseInt(voteIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid vote id")
	}

	attendees, err := s.repo.ListAttendeesForMeeting(ctx, meetingID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to list attendees", err)
	}
	attendeeIDs := make([]int64, len(attendees))
	for i, a := range attendees {
		attendeeIDs[i] = a.ID
	}

	vote, err := s.repo.OpenVoteWithEligibleVoters(ctx, voteID, attendeeIDs)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to open vote", err)
	}

	opts, _ := s.repo.ListVoteOptions(ctx, vote.ID)
	s.publishVotesUpdated(meetingID)

	resp := &votesv1.OpenVoteResponse{
		Vote:             toVoteDefinitionRecord(vote, opts),
		InvalidatedViews: []string{"moderation", "live"},
	}
	if stats, err := s.repo.GetVoteSubmissionStatsLive(ctx, voteID); err == nil {
		resp.Stats = &votesv1.VoteStats{
			EligibleCount: stats.EligibleCount,
			CastCount:     stats.CastCount,
			BallotCount:   stats.BallotCount,
		}
	}
	return resp, nil
}

// CloseVote closes an open vote. Requires chairperson.
func (s *Service) CloseVote(ctx context.Context, committeeSlug, meetingIDStr, voteIDStr string) (*votesv1.CloseVoteResponse, error) {
	if err := s.requireChairperson(ctx, committeeSlug); err != nil {
		return nil, err
	}

	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}

	voteID, err := strconv.ParseInt(voteIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid vote id")
	}

	result, err := s.repo.CloseVote(ctx, voteID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to close vote", err)
	}

	opts, _ := s.repo.ListVoteOptions(ctx, voteID)
	tallies, _ := s.repo.GetVoteTallies(ctx, voteID)
	s.publishVotesUpdated(meetingID)

	resp := &votesv1.CloseVoteResponse{
		Vote:             toVoteDefinitionRecord(result.Vote, opts),
		Outcome:          string(result.Outcome),
		InvalidatedViews: []string{"moderation", "live"},
	}
	for _, t := range tallies {
		resp.Tally = append(resp.Tally, &votesv1.VoteTallyEntry{
			OptionId: strconv.FormatInt(t.OptionID, 10),
			Label:    t.Label,
			Count:    t.Count,
		})
	}
	return resp, nil
}

// ArchiveVote archives a closed vote. Requires chairperson.
func (s *Service) ArchiveVote(ctx context.Context, committeeSlug, meetingIDStr, voteIDStr string) (*votesv1.ArchiveVoteResponse, error) {
	if err := s.requireChairperson(ctx, committeeSlug); err != nil {
		return nil, err
	}

	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}

	voteID, err := strconv.ParseInt(voteIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid vote id")
	}

	vote, err := s.repo.ArchiveVote(ctx, voteID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to archive vote", err)
	}

	opts, _ := s.repo.ListVoteOptions(ctx, vote.ID)
	s.publishVotesUpdated(meetingID)

	return &votesv1.ArchiveVoteResponse{
		Vote:             toVoteDefinitionRecord(vote, opts),
		InvalidatedViews: []string{"moderation"},
	}, nil
}

// SubmitBallot records an attendee ballot. For open votes only in the first
// Phase 2 implementation; secret ballot submission is unimplemented.
func (s *Service) SubmitBallot(ctx context.Context, committeeSlug, meetingIDStr, voteIDStr string, selectedOptionIDStrs []string, onBehalfOfIDStr string) (*votesv1.SubmitBallotResponse, error) {
	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}

	voteID, err := strconv.ParseInt(voteIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid vote id")
	}

	vote, err := s.repo.GetVoteDefinitionByID(ctx, voteID)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "vote not found")
	}

	if vote.State != model.VoteStateOpen {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "vote is not open")
	}

	if vote.Visibility != model.VoteVisibilityOpen {
		return nil, apierrors.New(apierrors.KindUnimplemented, "secret ballot submission via API is not yet supported")
	}

	selectedOptionIDs, err := parseIDList(selectedOptionIDStrs)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid option id")
	}

	attendeeID, err := s.resolveAttendeeForBallot(ctx, meetingID, onBehalfOfIDStr)
	if err != nil {
		return nil, err
	}

	source := model.VoteCastSourceSelfSubmission
	if onBehalfOfIDStr != "" {
		source = model.VoteCastSourceManualSubmission
	}

	token, err := generateReceiptToken()
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to generate receipt token", err)
	}

	ballot, err := s.repo.SubmitOpenBallot(ctx, repository.OpenBallotSubmission{
		VoteDefinitionID: voteID,
		MeetingID:        meetingID,
		AttendeeID:       attendeeID,
		Source:           source,
		ReceiptToken:     token,
		OptionIDs:        selectedOptionIDs,
	})
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to submit ballot", err)
	}

	s.publishVotesUpdated(meetingID)

	return &votesv1.SubmitBallotResponse{
		ReceiptToken:     ballot.ReceiptToken,
		InvalidatedViews: []string{"moderation", "live"},
	}, nil
}

func (s *Service) resolveAttendeeForBallot(ctx context.Context, meetingID int64, onBehalfOfIDStr string) (int64, error) {
	if onBehalfOfIDStr != "" {
		id, err := strconv.ParseInt(onBehalfOfIDStr, 10, 64)
		if err != nil {
			return 0, apierrors.New(apierrors.KindInvalidArgument, "invalid attendee_id")
		}
		return id, nil
	}

	sd, ok := session.GetSession(ctx)
	if !ok || sd == nil {
		return 0, apierrors.New(apierrors.KindUnauthenticated, "session required")
	}

	if sd.IsGuestSession() && sd.AttendeeID != nil {
		return *sd.AttendeeID, nil
	}

	if sd.IsAccountSession() && sd.AccountID != nil {
		return s.findAttendeeByAccount(ctx, meetingID, *sd.AccountID)
	}

	return 0, apierrors.New(apierrors.KindNotFound, "no attendee record found for the current session")
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
		id, _ := s.findAttendeeByAccount(ctx, meetingID, *sd.AccountID)
		return id
	}
	return 0
}

func (s *Service) findAttendeeByAccount(ctx context.Context, meetingID, accountID int64) (int64, error) {
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
			if user.AccountID == accountID {
				return a.ID, nil
			}
		}
	}
	return 0, apierrors.New(apierrors.KindNotFound, "no attendee record found for the current session")
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

func (s *Service) publishVotesUpdated(meetingID int64) {
	mid := meetingID
	s.broker.Publish(broker.SSEEvent{
		Event:     "votes.updated",
		Data:      []byte(`{"type":"votes.updated"}`),
		MeetingID: &mid,
	})
}

func toVoteDefinitionRecord(v *model.VoteDefinition, opts []*model.VoteOption) *votesv1.VoteDefinitionRecord {
	rec := &votesv1.VoteDefinitionRecord{
		VoteId:        strconv.FormatInt(v.ID, 10),
		AgendaPointId: strconv.FormatInt(v.AgendaPointID, 10),
		Name:          v.Name,
		Visibility:    v.Visibility,
		State:         v.State,
		MinSelections: v.MinSelections,
		MaxSelections: v.MaxSelections,
	}
	for _, o := range opts {
		rec.Options = append(rec.Options, &votesv1.VoteOptionRecord{
			OptionId: strconv.FormatInt(o.ID, 10),
			Label:    o.Label,
			Position: o.Position,
		})
	}
	return rec
}

func parseOptions(labels []string) []repository.VoteOptionInput {
	opts := make([]repository.VoteOptionInput, 0, len(labels))
	pos := int64(1)
	for _, l := range labels {
		if l == "" {
			continue
		}
		opts = append(opts, repository.VoteOptionInput{Label: l, Position: pos})
		pos++
	}
	return opts
}

func parseIDList(strs []string) ([]int64, error) {
	ids := make([]int64, 0, len(strs))
	for _, s := range strs {
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func generateReceiptToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
