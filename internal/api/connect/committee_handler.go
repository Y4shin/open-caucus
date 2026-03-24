package apiconnect

import (
	"context"

	connect "connectrpc.com/connect"

	committeesv1 "github.com/Y4shin/conference-tool/gen/go/conference/committees/v1"
	committeesv1connect "github.com/Y4shin/conference-tool/gen/go/conference/committees/v1/committeesv1connect"
	committeeservice "github.com/Y4shin/conference-tool/internal/services/committees"
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
