package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Y4shin/conference-tool/internal/repository/model"
	"github.com/Y4shin/conference-tool/internal/routes"
	"github.com/Y4shin/conference-tool/internal/session"
	"github.com/Y4shin/conference-tool/internal/templates"
)

// buildProtocolAgendaPointItems converts model agenda points to protocol template items.
func buildProtocolAgendaPointItems(aps []*model.AgendaPoint) []templates.ProtocolAgendaPointItem {
	items := make([]templates.ProtocolAgendaPointItem, len(aps))
	for i, ap := range aps {
		items[i] = templates.ProtocolAgendaPointItem{
			ID:       ap.ID,
			IDString: strconv.FormatInt(ap.ID, 10),
			Position: ap.Position,
			Title:    ap.Title,
			Protocol: ap.Protocol,
		}
	}
	return items
}

// MeetingProtocolPage renders the protocol writer page for the assigned attendee.
func (h *Handler) MeetingProtocolPage(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.MeetingProtocolInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}

	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load meeting: %w", err)
	}

	sd, _ := session.GetSession(ctx)
	if meeting.ProtocolWriterID == nil || sd.AttendeeID == nil || *meeting.ProtocolWriterID != *sd.AttendeeID {
		return nil, nil, fmt.Errorf("not the protocol writer")
	}

	committee, err := h.Repository.GetCommitteeBySlug(ctx, params.Slug)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load committee: %w", err)
	}

	agendaPoints, err := h.Repository.ListAgendaPointsForMeeting(ctx, meetingID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load agenda points: %w", err)
	}

	return &templates.MeetingProtocolInput{
		CommitteeName: committee.Name,
		CommitteeSlug: committee.Slug,
		MeetingName:   meeting.Name,
		IDString:      params.MeetingId,
		AgendaPoints:  buildProtocolAgendaPointItems(agendaPoints),
	}, nil, nil
}

// ProtocolSaveAgendaPoint saves the protocol text for one agenda point.
func (h *Handler) ProtocolSaveAgendaPoint(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.ProtocolAgendaPointPartialInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}

	apID, err := strconv.ParseInt(params.AgendaPointId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid agenda point ID")
	}

	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load meeting: %w", err)
	}

	sd, _ := session.GetSession(ctx)
	if meeting.ProtocolWriterID == nil || sd.AttendeeID == nil || *meeting.ProtocolWriterID != *sd.AttendeeID {
		return nil, nil, fmt.Errorf("not the protocol writer")
	}

	if err := r.ParseForm(); err != nil {
		return nil, nil, fmt.Errorf("failed to parse form: %w", err)
	}

	protocol := strings.TrimSpace(r.FormValue("protocol"))
	if err := h.Repository.UpdateAgendaPointProtocol(ctx, apID, protocol); err != nil {
		return nil, nil, fmt.Errorf("failed to save protocol: %w", err)
	}

	ap, err := h.Repository.GetAgendaPointByID(ctx, apID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to reload agenda point: %w", err)
	}

	return &templates.ProtocolAgendaPointPartialInput{
		CommitteeSlug: params.Slug,
		IDString:      params.MeetingId,
		AgendaPoint: templates.ProtocolAgendaPointItem{
			ID:       ap.ID,
			IDString: strconv.FormatInt(ap.ID, 10),
			Position: ap.Position,
			Title:    ap.Title,
			Protocol: ap.Protocol,
		},
		Saved: true,
	}, nil, nil
}
