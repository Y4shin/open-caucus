package voteservice

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	votesv1 "github.com/Y4shin/open-caucus/gen/go/conference/votes/v1"
	apierrors "github.com/Y4shin/open-caucus/internal/api/errors"
	"github.com/Y4shin/open-caucus/internal/broker"
	"github.com/Y4shin/open-caucus/internal/repository"
	"github.com/Y4shin/open-caucus/internal/repository/model"
	serviceauthz "github.com/Y4shin/open-caucus/internal/services/authz"
	"github.com/Y4shin/open-caucus/internal/session"
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
	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}
	if err := serviceauthz.RequireModerationAccess(ctx, s.repo, committeeSlug, meetingID); err != nil {
		return nil, err
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
				if stats, err := s.repo.GetVoteSubmissionStatsLive(ctx, v.ID); err == nil {
					rec.Stats = &votesv1.VoteStats{
						EligibleCount: stats.EligibleCount,
						CastCount:     stats.CastCount,
						BallotCount:   stats.BallotCount,
					}
				}
				if v.State == model.VoteStateClosed || v.State == model.VoteStateArchived {
					if tallies, err := s.repo.GetVoteTallies(ctx, v.ID); err == nil {
						for _, t := range tallies {
							rec.Tally = append(rec.Tally, &votesv1.VoteTallyEntry{
								OptionId: strconv.FormatInt(t.OptionID, 10),
								Label:    t.Label,
								Count:    t.Count,
							})
						}
					}
				}
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
	view.HasActiveAgenda = true

	votes, err := s.repo.ListVoteDefinitionsForAgendaPoint(ctx, *meeting.CurrentAgendaPointID)
	if err != nil {
		return &votesv1.GetLiveVotePanelResponse{View: view}, nil
	}

	callerAttendeeID := s.callerAttendeeID(ctx, meetingID)
	now := time.Now()

	for _, v := range votes {
		opts, _ := s.repo.ListVoteOptions(ctx, v.ID)
		record := toVoteDefinitionRecord(v, opts)
		card := &votesv1.LiveVoteCardView{
			Vote: record,
		}

		if callerAttendeeID != 0 {
			if voters, err := s.repo.ListEligibleVoters(ctx, v.ID); err == nil {
				for _, ev := range voters {
					if ev.AttendeeID == callerAttendeeID {
						card.IsEligible = true
						break
					}
				}
			}
			if casts, err := s.repo.ListVoteCasts(ctx, v.ID); err == nil {
				for _, c := range casts {
					if c.AttendeeID == callerAttendeeID {
						card.AlreadyVoted = true
						break
					}
				}
			}
		}

		if stats, err := s.repo.GetVoteSubmissionStatsLive(ctx, v.ID); err == nil {
			card.Stats = &votesv1.VoteStats{
				EligibleCount: stats.EligibleCount,
				CastCount:     stats.CastCount,
				BallotCount:   stats.BallotCount,
			}
		}

		include := false
		switch v.State {
		case model.VoteStateOpen:
			include = true
			view.HasActiveVote = true
			view.ActiveVote = record
			view.IsEligible = card.IsEligible
			view.AlreadyVoted = card.AlreadyVoted
		case model.VoteStateCounting:
			include = true
			card.ResultsBlockedCounting = true
		case model.VoteStateClosed:
			if v.ClosedAt != nil {
				until := v.ClosedAt.Add(30 * time.Second)
				if now.Before(until) {
					if tallies, err := s.repo.GetVoteTallies(ctx, v.ID); err == nil {
						card.HasTimedResults = true
						card.ResultsUntilUnix = until.Unix()
						card.ResultsRemainingSeconds = int64(until.Sub(now).Seconds()) + 1
						for _, row := range tallies {
							card.TimedResults = append(card.TimedResults, &votesv1.VoteTallyEntry{
								OptionId: strconv.FormatInt(row.OptionID, 10),
								Label:    row.Label,
								Count:    row.Count,
							})
						}
						include = true
					}
				}
			}
		}

		if include {
			view.Votes = append(view.Votes, card)
		}
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

// SubmitBallot records an attendee ballot for either open or secret votes.
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

	selectedOptionIDs, err := parseIDList(selectedOptionIDStrs)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid option id")
	}

	numSelected := int64(len(selectedOptionIDs))
	if numSelected < vote.MinSelections {
		return nil, apierrors.New(apierrors.KindInvalidArgument, fmt.Sprintf("too few selections: got %d, minimum is %d", numSelected, vote.MinSelections))
	}
	if numSelected > vote.MaxSelections {
		return nil, apierrors.New(apierrors.KindInvalidArgument, fmt.Sprintf("too many selections: got %d, maximum is %d", numSelected, vote.MaxSelections))
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

	switch vote.Visibility {
	case model.VoteVisibilityOpen:
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
	case model.VoteVisibilitySecret:
		if _, err := s.repo.RegisterVoteCast(ctx, voteID, meetingID, attendeeID, source); err != nil {
			return nil, apierrors.Wrap(apierrors.KindInternal, "failed to register secret vote cast", err)
		}

		ballot, err := s.repo.SubmitSecretBallot(ctx, repository.SecretBallotSubmission{
			VoteDefinitionID:    voteID,
			ReceiptToken:        token,
			EncryptedCommitment: []byte(fmt.Sprintf("%d:%v:%s", attendeeID, selectedOptionIDs, token)),
			CommitmentCipher:    "xchacha20poly1305",
			CommitmentVersion:   1,
			OptionIDs:           selectedOptionIDs,
		})
		if err != nil {
			return nil, apierrors.Wrap(apierrors.KindInternal, "failed to submit secret ballot", err)
		}

		s.publishVotesUpdated(meetingID)

		return &votesv1.SubmitBallotResponse{
			ReceiptToken:     ballot.ReceiptToken,
			InvalidatedViews: []string{"moderation", "live"},
		}, nil
	default:
		return nil, apierrors.New(apierrors.KindInvalidArgument, "unsupported vote visibility")
	}
}

func (s *Service) VerifyOpenReceipt(ctx context.Context, voteIDStr, receiptToken, attendeeIDStr string) (*votesv1.VerifyOpenReceiptResponse, error) {
	voteID, err := strconv.ParseInt(voteIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid vote id")
	}

	verification, err := s.repo.VerifyOpenBallotByReceipt(ctx, voteID, receiptToken)
	if err != nil {
		return nil, mapVerifyError(err)
	}

	if attendeeIDStr != "" {
		attendeeID, err := strconv.ParseInt(attendeeIDStr, 10, 64)
		if err != nil {
			return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid attendee id")
		}
		if verification.AttendeeID != attendeeID {
			return nil, apierrors.New(apierrors.KindNotFound, "ballot not found for attendee")
		}
	}

	return &votesv1.VerifyOpenReceiptResponse{
		VoteId:          strconv.FormatInt(verification.VoteDefinitionID, 10),
		VoteName:        verification.VoteName,
		AttendeeId:      strconv.FormatInt(verification.AttendeeID, 10),
		AttendeeNumber:  strconv.FormatInt(verification.AttendeeNumber, 10),
		ReceiptToken:    verification.ReceiptToken,
		ChoiceLabels:    verification.ChoiceLabels,
		ChoiceOptionIds: formatInt64Slice(verification.ChoiceOptionIDs),
	}, nil
}

func (s *Service) VerifySecretReceipt(ctx context.Context, voteIDStr, receiptToken string) (*votesv1.VerifySecretReceiptResponse, error) {
	voteID, err := strconv.ParseInt(voteIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid vote id")
	}

	verification, err := s.repo.VerifySecretBallotByReceipt(ctx, voteID, receiptToken)
	if err != nil {
		return nil, mapVerifyError(err)
	}

	choiceOptionIDs := make([]string, len(verification.ChoiceOptionIDs))
	for i, id := range verification.ChoiceOptionIDs {
		choiceOptionIDs[i] = strconv.FormatInt(id, 10)
	}

	return &votesv1.VerifySecretReceiptResponse{
		VoteId:                 strconv.FormatInt(verification.VoteDefinitionID, 10),
		VoteName:               verification.VoteName,
		ReceiptToken:           verification.ReceiptToken,
		EncryptedCommitmentB64: base64.StdEncoding.EncodeToString(verification.EncryptedCommitment),
		CommitmentCipher:       verification.CommitmentCipher,
		CommitmentVersion:      verification.CommitmentVersion,
		ChoiceLabels:           verification.ChoiceLabels,
		ChoiceOptionIds:        choiceOptionIDs,
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

func formatInt64Slice(values []int64) []string {
	if len(values) == 0 {
		return nil
	}

	formatted := make([]string, 0, len(values))
	for _, value := range values {
		formatted = append(formatted, strconv.FormatInt(value, 10))
	}

	return formatted
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
			if user.AccountID != nil && *user.AccountID == accountID {
				return a.ID, nil
			}
		}
	}
	return 0, apierrors.New(apierrors.KindNotFound, "no attendee record found for the current session")
}

func (s *Service) requireChairperson(ctx context.Context, committeeSlug string) error {
	return serviceauthz.RequireChairperson(ctx, s.repo, committeeSlug)
}

func (s *Service) publishVotesUpdated(meetingID int64) {
	mid := meetingID
	s.broker.Publish(broker.SSEEvent{
		Event:     "votes.updated",
		Data:      []byte(`{"type":"votes.updated"}`),
		MeetingID: &mid,
	})
}

func mapVerifyError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(strings.ToLower(msg), "not found"):
		return apierrors.New(apierrors.KindNotFound, msg)
	case strings.Contains(strings.ToLower(msg), "counting"):
		return apierrors.New(apierrors.KindConflict, msg)
	case strings.Contains(strings.ToLower(msg), "invalid"):
		return apierrors.New(apierrors.KindInvalidArgument, msg)
	default:
		return apierrors.New(apierrors.KindInvalidArgument, msg)
	}
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

// RegisterCast marks an attendee as having submitted a physical ballot for a secret vote.
func (s *Service) RegisterCast(ctx context.Context, committeeSlug, meetingIDStr, voteIDStr, attendeeIDStr string) (*votesv1.RegisterCastResponse, error) {
	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}
	if err := serviceauthz.RequireModerationAccess(ctx, s.repo, committeeSlug, meetingID); err != nil {
		return nil, err
	}
	voteID, err := strconv.ParseInt(voteIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid vote id")
	}
	attendeeID, err := strconv.ParseInt(attendeeIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid attendee id")
	}
	if _, err := s.repo.RegisterVoteCast(ctx, voteID, meetingID, attendeeID, model.VoteCastSourceManualSubmission); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInvalidArgument, "failed to register cast: "+err.Error(), err)
	}
	s.publishVotesUpdated(meetingID)
	view, err := s.getVotesPanelView(ctx, committeeSlug, meetingIDStr, meetingID)
	if err != nil {
		return nil, err
	}
	return &votesv1.RegisterCastResponse{View: view}, nil
}

// CountSecretBallot counts a physical secret ballot.
func (s *Service) CountSecretBallot(ctx context.Context, committeeSlug, meetingIDStr, voteIDStr, receiptToken string, selectedOptionIDStrs []string) (*votesv1.CountSecretBallotResponse, error) {
	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}
	if err := serviceauthz.RequireModerationAccess(ctx, s.repo, committeeSlug, meetingID); err != nil {
		return nil, err
	}
	voteID, err := strconv.ParseInt(voteIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid vote id")
	}
	optionIDs, err := parseIDList(selectedOptionIDStrs)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid option id")
	}
	vote, err := s.repo.GetVoteDefinitionByID(ctx, voteID)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "vote not found")
	}
	numSelected := int64(len(optionIDs))
	if numSelected < vote.MinSelections {
		return nil, apierrors.New(apierrors.KindInvalidArgument, fmt.Sprintf("too few selections: got %d, minimum is %d", numSelected, vote.MinSelections))
	}
	if numSelected > vote.MaxSelections {
		return nil, apierrors.New(apierrors.KindInvalidArgument, fmt.Sprintf("too many selections: got %d, maximum is %d", numSelected, vote.MaxSelections))
	}
	if receiptToken == "" {
		receiptToken, _ = generateReceiptToken()
	}
	payload := []byte("manual:" + receiptToken)
	if _, err := s.repo.SubmitSecretBallot(ctx, repository.SecretBallotSubmission{
		VoteDefinitionID:    voteID,
		ReceiptToken:        receiptToken,
		EncryptedCommitment: payload,
		CommitmentCipher:    "xchacha20poly1305",
		CommitmentVersion:   1,
		OptionIDs:           optionIDs,
	}); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInvalidArgument, "failed to count secret ballot: "+err.Error(), err)
	}
	s.publishVotesUpdated(meetingID)
	view, err := s.getVotesPanelView(ctx, committeeSlug, meetingIDStr, meetingID)
	if err != nil {
		return nil, err
	}
	return &votesv1.CountSecretBallotResponse{View: view}, nil
}

// CountOpenBallot manually records an open ballot on behalf of an attendee.
func (s *Service) CountOpenBallot(ctx context.Context, committeeSlug, meetingIDStr, voteIDStr, attendeeIDStr string, selectedOptionIDStrs []string) (*votesv1.CountOpenBallotResponse, error) {
	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}
	if err := serviceauthz.RequireModerationAccess(ctx, s.repo, committeeSlug, meetingID); err != nil {
		return nil, err
	}
	voteID, err := strconv.ParseInt(voteIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid vote id")
	}
	attendeeID, err := strconv.ParseInt(attendeeIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid attendee id")
	}
	optionIDs, err := parseIDList(selectedOptionIDStrs)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid option id")
	}
	vote, err := s.repo.GetVoteDefinitionByID(ctx, voteID)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "vote not found")
	}
	numSelected := int64(len(optionIDs))
	if numSelected < vote.MinSelections {
		return nil, apierrors.New(apierrors.KindInvalidArgument, fmt.Sprintf("too few selections: got %d, minimum is %d", numSelected, vote.MinSelections))
	}
	if numSelected > vote.MaxSelections {
		return nil, apierrors.New(apierrors.KindInvalidArgument, fmt.Sprintf("too many selections: got %d, maximum is %d", numSelected, vote.MaxSelections))
	}
	receiptToken, _ := generateReceiptToken()
	if _, err := s.repo.SubmitOpenBallot(ctx, repository.OpenBallotSubmission{
		VoteDefinitionID: voteID,
		MeetingID:        meetingID,
		AttendeeID:       attendeeID,
		Source:           model.VoteCastSourceManualSubmission,
		ReceiptToken:     receiptToken,
		OptionIDs:        optionIDs,
	}); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInvalidArgument, "failed to count open ballot: "+err.Error(), err)
	}
	s.publishVotesUpdated(meetingID)
	view, err := s.getVotesPanelView(ctx, committeeSlug, meetingIDStr, meetingID)
	if err != nil {
		return nil, err
	}
	return &votesv1.CountOpenBallotResponse{View: view}, nil
}

// getVotesPanelView is a helper that returns the current VotesPanelView for the given meeting.
func (s *Service) getVotesPanelView(ctx context.Context, committeeSlug, meetingIDStr string, meetingID int64) (*votesv1.VotesPanelView, error) {
	res, err := s.GetVotesPanel(ctx, committeeSlug, meetingIDStr)
	if err != nil {
		return nil, err
	}
	_ = meetingID
	return res.View, nil
}
