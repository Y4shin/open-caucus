// Package memberservice handles committee member management (chairperson-accessible).
package memberservice

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
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

func (s *Service) SendInviteEmails(ctx context.Context, slug, meetingIDStr, baseURL string, memberIDs []string, customMessage, language, timezone string) (*committeesv1.SendInviteEmailsResponse, error) {
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

	// Load agenda points (top-level + sub-points) for the email body.
	agendaPoints, _ := s.repo.ListAgendaPointsForMeeting(ctx, meetingID)
	subPoints, _ := s.repo.ListSubAgendaPointsForMeeting(ctx, meetingID)

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
			// Account members get a direct meeting link (they authenticate via their account).
			joinURL = fmt.Sprintf("%s/committee/%s/meeting/%d", baseURL, slug, meetingID)
			// Use account email, then user email, then skip.
			if account.Email != "" {
				toEmail = account.Email
			} else if m.Email != nil && *m.Email != "" {
				toEmail = *m.Email
			} else {
				slog.Info("email invite skipped", "member", m.FullName, "reason", "account member has no email", "account_id", *m.AccountID, "account_email", account.Email)
				skippedCount++
				continue
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
			slog.Info("email invite skipped", "member", m.FullName, "reason", "no email and no account")
			skippedCount++
			continue
		}

		// Build agenda items with sub-points for the email.
		agendaItems := make([]email.AgendaItem, 0)
		if agendaPoints != nil {
			// Build a map of parent ID → children.
			childrenOf := make(map[int64][]*model.AgendaPoint)
			if subPoints != nil {
				for _, sp := range subPoints {
					if sp.ParentID != nil {
						childrenOf[*sp.ParentID] = append(childrenOf[*sp.ParentID], sp)
					}
				}
			}
			for i, ap := range agendaPoints {
				item := email.AgendaItem{
					Number: strconv.Itoa(i + 1),
					Title:  ap.Title,
				}
				for j, child := range childrenOf[ap.ID] {
					item.Children = append(item.Children, email.AgendaItem{
						Number: fmt.Sprintf("%d.%d", i+1, j+1),
						Title:  child.Title,
					})
				}
				agendaItems = append(agendaItems, item)
			}
		}

		// Format datetime strings in the requested timezone.
		loc := time.UTC
		if timezone != "" {
			if parsed, err := time.LoadLocation(timezone); err == nil {
				loc = parsed
			}
		}
		dtFmt := "Mon, 2 Jan 2006 15:04 (MST)"
		startStr, endStr := "", ""
		if meeting.StartAt != nil {
			startStr = meeting.StartAt.In(loc).Format(dtFmt)
		}
		if meeting.EndAt != nil {
			endStr = meeting.EndAt.In(loc).Format(dtFmt)
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
			CustomMessage: customMessage,
			Language:      language,
		})
		if err != nil {
			errs = append(errs, fmt.Sprintf("render for %s: %v", m.FullName, err))
			continue
		}

		// Generate Message-ID and look up References for threading.
		messageUUID, _ := generateInviteSecret() // reuse for UUID-like unique string
		fromDomain := s.sender.FromDomain()
		messageID := fmt.Sprintf("<%s@%s>", messageUUID, fromDomain)

		var references []string
		if prevIDs, err := s.repo.ListSentEmailMessageIDs(ctx, toEmail, committee.ID); err == nil && len(prevIDs) > 0 {
			references = prevIDs
		}

		// Generate ICS attachment if meeting has datetime.
		sendOpts := &email.SendOptions{
			MessageID:  messageID,
			References: references,
		}
		if meeting.StartAt != nil {
			uid := fmt.Sprintf("meeting-%d@open-caucus", meetingID)
			endTime := time.Time{}
			if meeting.EndAt != nil {
				endTime = *meeting.EndAt
			}
			sendOpts.ICSData = ics.GenerateMeetingEvent(uid, meeting.Name, meeting.Description, *meeting.StartAt, endTime)
		}

		subject := email.InviteSubject(meeting.Name, committee.Name)
		slog.Info("email sending invite", "to", toEmail, "member", m.FullName, "meeting", meeting.Name, "message_id", messageID, "references_count", len(references), "has_ics", sendOpts.ICSData != "")
		if err := s.sender.Send(ctx, toEmail, subject, htmlBody, textBody, sendOpts); err != nil {
			slog.Error("email send failed", "to", toEmail, "err", err)
			errs = append(errs, fmt.Sprintf("send to %s: %v", toEmail, err))
			continue
		}

		// Record sent email for future threading.
		cID := committee.ID
		if err := s.repo.InsertSentEmail(ctx, messageID, toEmail, &cID, &meetingID, subject); err != nil {
			slog.Warn("failed to record sent email", "message_id", messageID, "err", err)
		}

		slog.Info("email sent successfully", "to", toEmail, "message_id", messageID)
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

