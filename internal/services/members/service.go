// Package memberservice handles committee member management (chairperson-accessible).
package memberservice

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	committeesv1 "github.com/Y4shin/open-caucus/gen/go/conference/committees/v1"
	apierrors "github.com/Y4shin/open-caucus/internal/api/errors"
	"github.com/Y4shin/open-caucus/internal/email"
	"github.com/Y4shin/open-caucus/internal/ics"
	"github.com/Y4shin/open-caucus/internal/repository"
	"github.com/Y4shin/open-caucus/internal/repository/model"
	serviceauthz "github.com/Y4shin/open-caucus/internal/services/authz"
)

// Service manages committee members.
type Service struct {
	repo   repository.Repository
	sender email.Sender
}

// New creates a new member management service.
func New(repo repository.Repository, sender email.Sender) *Service {
	return &Service{repo: repo, sender: sender}
}

// EmailEnabled returns whether the email sender is configured.
func (s *Service) EmailEnabled() bool {
	return s.sender.Enabled()
}

func (s *Service) ListCommitteeMembers(ctx context.Context, slug string) (*committeesv1.ListCommitteeMembersResponse, error) {
	if err := serviceauthz.RequireChairperson(ctx, s.repo, slug); err != nil {
		return nil, err
	}
	members, err := s.repo.ListAllMembersForCommittee(ctx, slug)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to list members", err)
	}
	records := make([]*committeesv1.MemberRecord, 0, len(members))
	for _, m := range members {
		records = append(records, memberToProto(m))
	}
	return &committeesv1.ListCommitteeMembersResponse{Members: records}, nil
}

func (s *Service) ListAssignableAccounts(ctx context.Context, slug string) (*committeesv1.ListAssignableAccountsResponse, error) {
	if err := serviceauthz.RequireChairperson(ctx, s.repo, slug); err != nil {
		return nil, err
	}
	committeeID, err := s.repo.GetCommitteeIDBySlug(ctx, slug)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "committee not found")
	}
	accounts, err := s.repo.ListUnassignedAccountsForCommittee(ctx, committeeID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to list assignable accounts", err)
	}
	result := make([]*committeesv1.AssignableAccount, 0, len(accounts))
	for _, a := range accounts {
		result = append(result, &committeesv1.AssignableAccount{
			AccountId: strconv.FormatInt(a.ID, 10),
			Username:  a.Username,
			FullName:  a.FullName,
		})
	}
	return &committeesv1.ListAssignableAccountsResponse{Accounts: result}, nil
}

func (s *Service) AddMemberByEmail(ctx context.Context, slug, emailAddr, fullName, role string, quoted bool) (*committeesv1.AddMemberByEmailResponse, error) {
	if err := serviceauthz.RequireChairperson(ctx, s.repo, slug); err != nil {
		return nil, err
	}
	if emailAddr == "" {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "email is required")
	}
	if fullName == "" {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "full name is required")
	}
	if role != "member" && role != "chairperson" {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "role must be 'member' or 'chairperson'")
	}

	committeeID, err := s.repo.GetCommitteeIDBySlug(ctx, slug)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "committee not found")
	}

	inviteSecret, err := generateInviteSecret()
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to generate invite secret", err)
	}

	if err := s.repo.CreateEmailMember(ctx, committeeID, emailAddr, fullName, quoted, role, inviteSecret); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to create email member", err)
	}

	member, err := s.repo.GetUserByCommitteeAndEmail(ctx, slug, emailAddr)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to load created member", err)
	}

	return &committeesv1.AddMemberByEmailResponse{Member: memberToProto(member)}, nil
}

func (s *Service) AssignAccountToCommittee(ctx context.Context, slug, accountIDStr, role string, quoted bool) (*committeesv1.CommitteeAssignAccountResponse, error) {
	if err := serviceauthz.RequireChairperson(ctx, s.repo, slug); err != nil {
		return nil, err
	}
	if role != "member" && role != "chairperson" {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "role must be 'member' or 'chairperson'")
	}

	accountID, err := strconv.ParseInt(accountIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid account id")
	}

	committeeID, err := s.repo.GetCommitteeIDBySlug(ctx, slug)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "committee not found")
	}

	if err := s.repo.AssignAccountToCommittee(ctx, committeeID, accountID, quoted, role); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to assign account", err)
	}

	membership, err := s.repo.GetUserMembershipByAccountIDAndSlug(ctx, accountID, slug)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to load membership", err)
	}

	return &committeesv1.CommitteeAssignAccountResponse{Member: memberToProto(membership)}, nil
}

func (s *Service) UpdateMember(ctx context.Context, slug, userIDStr, role string, quoted bool) (*committeesv1.UpdateMemberResponse, error) {
	if err := serviceauthz.RequireChairperson(ctx, s.repo, slug); err != nil {
		return nil, err
	}
	if role != "member" && role != "chairperson" {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "role must be 'member' or 'chairperson'")
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid user id")
	}

	if err := s.repo.UpdateUserMembership(ctx, userID, quoted, role); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to update member", err)
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to load updated member", err)
	}

	return &committeesv1.UpdateMemberResponse{Member: memberToProto(user)}, nil
}

func (s *Service) RemoveMember(ctx context.Context, slug, userIDStr string) (*committeesv1.RemoveMemberResponse, error) {
	if err := serviceauthz.RequireChairperson(ctx, s.repo, slug); err != nil {
		return nil, err
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid user id")
	}

	if err := s.repo.DeleteUserByID(ctx, userID); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to remove member", err)
	}

	return &committeesv1.RemoveMemberResponse{}, nil
}

func (s *Service) SendInviteEmails(ctx context.Context, slug, meetingIDStr, baseURL string, memberIDs []string) (*committeesv1.SendInviteEmailsResponse, error) {
	if err := serviceauthz.RequireChairperson(ctx, s.repo, slug); err != nil {
		return nil, err
	}

	if !s.sender.Enabled() {
		return nil, apierrors.New(apierrors.KindUnimplemented, "email sending is not configured")
	}

	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid meeting id")
	}

	meeting, err := s.repo.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "meeting not found")
	}

	committee, err := s.repo.GetCommitteeBySlug(ctx, slug)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "committee not found")
	}

	allMembers, err := s.repo.ListAllMembersForCommittee(ctx, slug)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to list members", err)
	}

	// Load agenda points for the email body.
	agendaPoints, _ := s.repo.ListAgendaPointsForMeeting(ctx, meetingID)

	// Filter to requested member IDs if specified.
	wantIDs := make(map[int64]bool)
	if len(memberIDs) > 0 {
		for _, idStr := range memberIDs {
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				continue
			}
			wantIDs[id] = true
		}
	}

	var sentCount, skippedCount int32
	var errs []string

	for _, m := range allMembers {
		if len(wantIDs) > 0 && !wantIDs[m.ID] {
			continue
		}

		// Determine email address and join URL.
		var toEmail, joinURL string

		if m.AccountID != nil {
			// Account-based member: they log in normally.
			account, err := s.repo.GetAccountByID(ctx, *m.AccountID)
			if err != nil {
				skippedCount++
				continue
			}
			// Use account email if we had one; for now use username as fallback.
			// Account members get a direct meeting link (they authenticate via their account).
			joinURL = fmt.Sprintf("%s/committee/%s/meeting/%d", baseURL, slug, meetingID)
			// We don't have email on accounts — skip if no email on user record either.
			if m.Email != nil && *m.Email != "" {
				toEmail = *m.Email
			} else {
				// Try to construct from username if it looks like an email.
				if isEmailLike(account.Username) {
					toEmail = account.Username
				} else {
					skippedCount++
					continue
				}
			}
		} else if m.Email != nil && *m.Email != "" {
			// Email-only member: personalized invite link.
			toEmail = *m.Email
			secret := ""
			if m.InviteSecret != nil {
				secret = *m.InviteSecret
			}
			joinURL = fmt.Sprintf("%s/committee/%s/meeting/%d/join?invite_secret=%s", baseURL, slug, meetingID, secret)
		} else {
			skippedCount++
			continue
		}

		// Build agenda items for the email.
		agendaItems := make([]email.AgendaItem, 0)
		if agendaPoints != nil {
			for i, ap := range agendaPoints {
				agendaItems = append(agendaItems, email.AgendaItem{
					Number: strconv.Itoa(i + 1),
					Title:  ap.Title,
				})
			}
		}

		// Format datetime strings.
		startStr, endStr := "", ""
		if meeting.StartAt != nil {
			startStr = meeting.StartAt.Format(time.RFC3339)
		}
		if meeting.EndAt != nil {
			endStr = meeting.EndAt.Format(time.RFC3339)
		}

		htmlBody, textBody, err := email.RenderMeetingInvite(email.InviteData{
			MemberName:    m.FullName,
			CommitteeName: committee.Name,
			MeetingName:   meeting.Name,
			MeetingDesc:   meeting.Description,
			JoinURL:       joinURL,
			StartAt:       startStr,
			EndAt:         endStr,
			Agenda:        agendaItems,
		})
		if err != nil {
			errs = append(errs, fmt.Sprintf("render for %s: %v", m.FullName, err))
			continue
		}

		// Generate ICS attachment if meeting has datetime.
		var sendOpts *email.SendOptions
		if meeting.StartAt != nil {
			uid := fmt.Sprintf("meeting-%d@open-caucus", meetingID)
			endTime := time.Time{}
			if meeting.EndAt != nil {
				endTime = *meeting.EndAt
			}
			icsData := ics.GenerateMeetingEvent(uid, meeting.Name, meeting.Description, *meeting.StartAt, endTime)
			sendOpts = &email.SendOptions{ICSData: icsData}
		}

		subject := email.InviteSubject(meeting.Name, committee.Name)
		if err := s.sender.Send(ctx, toEmail, subject, htmlBody, textBody, sendOpts); err != nil {
			errs = append(errs, fmt.Sprintf("send to %s: %v", toEmail, err))
			continue
		}
		sentCount++
	}

	return &committeesv1.SendInviteEmailsResponse{
		SentCount:    sentCount,
		SkippedCount: skippedCount,
		Errors:       errs,
	}, nil
}

func memberToProto(m *model.User) *committeesv1.MemberRecord {
	rec := &committeesv1.MemberRecord{
		UserId:         strconv.FormatInt(m.ID, 10),
		FullName:       m.FullName,
		Role:           m.Role,
		Quoted:         m.Quoted,
		IsOauthManaged: m.OAuthManaged,
		HasAccount:     m.AccountID != nil,
	}
	if m.Email != nil {
		rec.Email = m.Email
	}
	if m.Username != "" {
		rec.Username = &m.Username
	}
	return rec
}

func generateInviteSecret() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func isEmailLike(s string) bool {
	return len(s) > 3 && contains(s, "@") && contains(s, ".")
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
