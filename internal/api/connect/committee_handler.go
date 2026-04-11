package apiconnect

import (
	"context"
	"time"

	connect "connectrpc.com/connect"

	committeesv1 "github.com/Y4shin/open-caucus/gen/go/conference/committees/v1"
	committeesv1connect "github.com/Y4shin/open-caucus/gen/go/conference/committees/v1/committeesv1connect"
	committeeservice "github.com/Y4shin/open-caucus/internal/services/committees"
)

type CommitteeHandler struct {
	committeesv1connect.UnimplementedCommitteeServiceHandler
	service *committeeservice.Service
}

func NewCommitteeHandler(service *committeeservice.Service) *CommitteeHandler {
	return &CommitteeHandler{service: service}
}

func (h *CommitteeHandler) ListMyCommittees(ctx context.Context, _ *connect.Request[committeesv1.ListMyCommitteesRequest]) (*connect.Response[committeesv1.ListMyCommitteesResponse], error) {
	resp, err := h.service.ListMyCommittees(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *CommitteeHandler) GetCommitteeOverview(ctx context.Context, req *connect.Request[committeesv1.GetCommitteeOverviewRequest]) (*connect.Response[committeesv1.GetCommitteeOverviewResponse], error) {
	resp, err := h.service.GetCommitteeOverview(ctx, req.Msg.CommitteeSlug)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *CommitteeHandler) CreateMeeting(ctx context.Context, req *connect.Request[committeesv1.CreateMeetingRequest]) (*connect.Response[committeesv1.CreateMeetingResponse], error) {
	startAt := parseOptionalUTCTime(req.Msg.StartAt)
	endAt := parseOptionalUTCTime(req.Msg.EndAt)
	resp, err := h.service.CreateMeeting(ctx, req.Msg.CommitteeSlug, req.Msg.Name, req.Msg.Description, startAt, endAt)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func parseOptionalUTCTime(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil
	}
	t = t.UTC()
	return &t
}

func (h *CommitteeHandler) DeleteMeeting(ctx context.Context, req *connect.Request[committeesv1.DeleteMeetingRequest]) (*connect.Response[committeesv1.DeleteMeetingResponse], error) {
	resp, err := h.service.DeleteMeeting(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *CommitteeHandler) ToggleMeetingActive(ctx context.Context, req *connect.Request[committeesv1.ToggleMeetingActiveRequest]) (*connect.Response[committeesv1.ToggleMeetingActiveResponse], error) {
	resp, err := h.service.ToggleMeetingActive(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}
