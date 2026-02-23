package templates

import (
	"context"

	"github.com/Y4shin/conference-tool/internal/session"
)

func CurrentDisplayName(ctx context.Context) string {
	// On meeting pages the attendee full name is more meaningful than the username.
	if attendee, ok := session.GetCurrentAttendee(ctx); ok && attendee.FullName != "" {
		return attendee.FullName
	}
	if user, ok := session.GetCurrentUser(ctx); ok && user.Username != "" {
		return user.Username
	}
	return ""
}

func CurrentUserRole(ctx context.Context) string {
	if user, ok := session.GetCurrentUser(ctx); ok {
		return user.Role
	}
	return ""
}
