package apiconnect

import (
	"context"

	connect "connectrpc.com/connect"

	votesv1 "github.com/Y4shin/conference-tool/gen/go/conference/votes/v1"
	votesv1connect "github.com/Y4shin/conference-tool/gen/go/conference/votes/v1/votesv1connect"
	voteservice "github.com/Y4shin/conference-tool/internal/services/votes"
)

type VoteHandler struct {
	votesv1connect.UnimplementedVoteServiceHandler
	service *voteservice.Service
}

func NewVoteHandler(service *voteservice.Service) *VoteHandler {
	return &VoteHandler{service: service}
}

func (h *VoteHandler) GetVotesPanel(ctx context.Context, req *connect.Request[votesv1.GetVotesPanelRequest]) (*connect.Response[votesv1.GetVotesPanelResponse], error) {
	resp, err := h.service.GetVotesPanel(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *VoteHandler) GetLiveVotePanel(ctx context.Context, req *connect.Request[votesv1.GetLiveVotePanelRequest]) (*connect.Response[votesv1.GetLiveVotePanelResponse], error) {
	resp, err := h.service.GetLiveVotePanel(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *VoteHandler) CreateVote(ctx context.Context, req *connect.Request[votesv1.CreateVoteRequest]) (*connect.Response[votesv1.CreateVoteResponse], error) {
	resp, err := h.service.CreateVote(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.Name, req.Msg.Visibility, req.Msg.MinSelections, req.Msg.MaxSelections, req.Msg.OptionLabels)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *VoteHandler) UpdateVoteDraft(ctx context.Context, req *connect.Request[votesv1.UpdateVoteDraftRequest]) (*connect.Response[votesv1.UpdateVoteDraftResponse], error) {
	resp, err := h.service.UpdateVoteDraft(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.VoteId, req.Msg.Name, req.Msg.Visibility, req.Msg.MinSelections, req.Msg.MaxSelections, req.Msg.OptionLabels)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *VoteHandler) OpenVote(ctx context.Context, req *connect.Request[votesv1.OpenVoteRequest]) (*connect.Response[votesv1.OpenVoteResponse], error) {
	resp, err := h.service.OpenVote(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.VoteId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *VoteHandler) CloseVote(ctx context.Context, req *connect.Request[votesv1.CloseVoteRequest]) (*connect.Response[votesv1.CloseVoteResponse], error) {
	resp, err := h.service.CloseVote(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.VoteId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *VoteHandler) ArchiveVote(ctx context.Context, req *connect.Request[votesv1.ArchiveVoteRequest]) (*connect.Response[votesv1.ArchiveVoteResponse], error) {
	resp, err := h.service.ArchiveVote(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.VoteId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *VoteHandler) SubmitBallot(ctx context.Context, req *connect.Request[votesv1.SubmitBallotRequest]) (*connect.Response[votesv1.SubmitBallotResponse], error) {
	resp, err := h.service.SubmitBallot(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.VoteId, req.Msg.SelectedOptionIds, req.Msg.AttendeeId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *VoteHandler) VerifyOpenReceipt(ctx context.Context, req *connect.Request[votesv1.VerifyOpenReceiptRequest]) (*connect.Response[votesv1.VerifyOpenReceiptResponse], error) {
	resp, err := h.service.VerifyOpenReceipt(ctx, req.Msg.VoteId, req.Msg.ReceiptToken, req.Msg.AttendeeId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *VoteHandler) VerifySecretReceipt(ctx context.Context, req *connect.Request[votesv1.VerifySecretReceiptRequest]) (*connect.Response[votesv1.VerifySecretReceiptResponse], error) {
	resp, err := h.service.VerifySecretReceipt(ctx, req.Msg.VoteId, req.Msg.ReceiptToken)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *VoteHandler) RegisterCast(ctx context.Context, req *connect.Request[votesv1.RegisterCastRequest]) (*connect.Response[votesv1.RegisterCastResponse], error) {
	resp, err := h.service.RegisterCast(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.VoteId, req.Msg.AttendeeId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *VoteHandler) CountSecretBallot(ctx context.Context, req *connect.Request[votesv1.CountSecretBallotRequest]) (*connect.Response[votesv1.CountSecretBallotResponse], error) {
	resp, err := h.service.CountSecretBallot(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.VoteId, req.Msg.ReceiptToken, req.Msg.SelectedOptionIds)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *VoteHandler) CountOpenBallot(ctx context.Context, req *connect.Request[votesv1.CountOpenBallotRequest]) (*connect.Response[votesv1.CountOpenBallotResponse], error) {
	resp, err := h.service.CountOpenBallot(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.VoteId, req.Msg.AttendeeId, req.Msg.SelectedOptionIds)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}
