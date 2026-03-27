package authz

import (
	"context"

	apierrors "github.com/Y4shin/conference-tool/internal/api/errors"
	"github.com/Y4shin/conference-tool/internal/repository"
	"github.com/Y4shin/conference-tool/internal/session"
)

func RequireChairperson(ctx context.Context, repo repository.Repository, committeeSlug string) error {
	sd, ok := session.GetSession(ctx)
	if !ok || sd == nil || sd.IsExpired() || !sd.IsAccountSession() || sd.AccountID == nil {
		return apierrors.New(apierrors.KindUnauthenticated, "account session required")
	}

	account, err := repo.GetAccountByID(ctx, *sd.AccountID)
	if err != nil {
		return apierrors.New(apierrors.KindUnauthenticated, "account not found")
	}
	if account.IsAdmin {
		return nil
	}

	membership, err := repo.GetUserMembershipByAccountIDAndSlug(ctx, *sd.AccountID, committeeSlug)
	if err != nil || membership.Role != "chairperson" {
		return apierrors.New(apierrors.KindPermissionDenied, "chairperson role required")
	}
	return nil
}

func RequireModerationAccess(ctx context.Context, repo repository.Repository, committeeSlug string, meetingID int64) error {
	if err := RequireChairperson(ctx, repo, committeeSlug); err == nil {
		return nil
	}

	sd, ok := session.GetSession(ctx)
	if !ok || sd == nil || sd.IsExpired() || !sd.IsGuestSession() || sd.AttendeeID == nil {
		return apierrors.New(apierrors.KindPermissionDenied, "chairperson role required")
	}

	attendee, err := repo.GetAttendeeByID(ctx, *sd.AttendeeID)
	if err != nil || attendee.MeetingID != meetingID {
		return apierrors.New(apierrors.KindPermissionDenied, "chairperson role required")
	}
	if attendee.IsChair {
		return nil
	}

	meeting, err := repo.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return apierrors.New(apierrors.KindPermissionDenied, "chairperson role required")
	}
	if meeting.ModeratorID != nil && *meeting.ModeratorID == attendee.ID {
		return nil
	}

	return apierrors.New(apierrors.KindPermissionDenied, "chairperson role required")
}
