package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Y4shin/conference-tool/internal/routes"
	"github.com/Y4shin/conference-tool/internal/templates"
)

// ManageAgendaPointToolsPage renders the per-agenda-point tools page.
// It contains only attachments and motions for a single agenda point.
func (h *Handler) ManageAgendaPointToolsPage(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.MeetingAgendaPointToolsInput, *routes.ResponseMeta, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid meeting ID")
	}
	agendaPointID, err := strconv.ParseInt(params.AgendaPointId, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid agenda point ID")
	}

	committee, err := h.Repository.GetCommitteeBySlug(ctx, params.Slug)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load committee: %w", err)
	}
	meeting, err := h.Repository.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load meeting: %w", err)
	}
	agendaPoint, err := h.Repository.GetAgendaPointByID(ctx, agendaPointID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load agenda point: %w", err)
	}
	if agendaPoint.MeetingID != meetingID {
		return nil, nil, fmt.Errorf("agenda point does not belong to meeting")
	}

	attachments, err := h.loadAttachmentListPartial(ctx, params.Slug, params.MeetingId, agendaPoint)
	if err != nil {
		return nil, nil, err
	}
	motions, err := h.loadMotionListPartial(ctx, params.Slug, params.MeetingId, agendaPoint)
	if err != nil {
		return nil, nil, err
	}

	return &templates.MeetingAgendaPointToolsInput{
		CommitteeName:   committee.Name,
		CommitteeSlug:   committee.Slug,
		MeetingName:     meeting.Name,
		MeetingIDStr:    params.MeetingId,
		AgendaPointIDStr: params.AgendaPointId,
		AgendaPointTitle: agendaPoint.Title,
		Attachments:     *attachments,
		Motions:         *motions,
	}, nil, nil
}
