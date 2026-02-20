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

// CommitteeMeetingManageJoinQR renders a QR code for guest join URLs with a prefilled meeting secret.
func (h *Handler) CommitteeMeetingManageJoinQR(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.MeetingJoinQRInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}

	committee, err := h.Repository.GetCommitteeBySlug(ctx, params.Slug)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load committee: %w", err)
	}

	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load meeting: %w", err)
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwardedProto := r.Header.Get("X-Forwarded-Proto"); forwardedProto != "" {
		scheme = forwardedProto
	}

	joinURL := url.URL{
		Scheme: scheme,
		Host:   r.Host,
		Path:   fmt.Sprintf("/committee/%s/meeting/%s/join", params.Slug, params.MeetingId),
	}
	query := joinURL.Query()
	query.Set("meeting_secret", meeting.Secret)
	joinURL.RawQuery = query.Encode()
	joinURLStr := joinURL.String()

	png, err := qrcode.Encode(joinURLStr, qrcode.Medium, 320)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate join QR code: %w", err)
	}
	qrCodeDataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)

	return &templates.MeetingJoinQRInput{
		CommitteeName: committee.Name,
		CommitteeSlug: committee.Slug,
		MeetingName:   meeting.Name,
		IDString:      params.MeetingId,
		JoinURL:       joinURLStr,
		QRCodeDataURL: qrCodeDataURL,
	}, nil, nil
}
