package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/skip2/go-qrcode"

	"github.com/Y4shin/conference-tool/internal/routes"
	"github.com/Y4shin/conference-tool/internal/templates"
)

// ManageAttendeeRecoveryPage renders a recovery URL and QR code for a guest attendee.
func (h *Handler) ManageAttendeeRecoveryPage(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.MeetingAttendeeRecoveryInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	attendeeID, err := strconv.ParseInt(params.AttendeeId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid attendee ID")
	}

	committee, err := h.Repository.GetCommitteeBySlug(ctx, params.Slug)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load committee: %w", err)
	}
	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load meeting: %w", err)
	}
	attendee, err := h.Repository.GetAttendeeByID(ctx, attendeeID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load attendee: %w", err)
	}
	if attendee.MeetingID != meetingID {
		return nil, nil, fmt.Errorf("attendee does not belong to meeting")
	}
	if attendee.UserID != nil {
		return nil, nil, fmt.Errorf("recovery link is only available for guests")
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwardedProto := r.Header.Get("X-Forwarded-Proto"); forwardedProto != "" {
		scheme = forwardedProto
	}

	loginURL := url.URL{
		Scheme: scheme,
		Host:   r.Host,
		Path:   fmt.Sprintf("/committee/%s/meeting/%s/attendee-login", params.Slug, params.MeetingId),
	}
	query := loginURL.Query()
	query.Set("secret", attendee.Secret)
	loginURL.RawQuery = query.Encode()
	loginURLStr := loginURL.String()

	png, err := qrcode.Encode(loginURLStr, qrcode.Medium, 320)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate attendee recovery QR code: %w", err)
	}
	qrCodeDataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)

	return &templates.MeetingAttendeeRecoveryInput{
		CommitteeName: committee.Name,
		CommitteeSlug: committee.Slug,
		MeetingName:   meeting.Name,
		MeetingIDStr:  params.MeetingId,
		AttendeeName:  attendee.FullName,
		LoginURL:      loginURLStr,
		QRCodeDataURL: qrCodeDataURL,
	}, nil, nil
}
