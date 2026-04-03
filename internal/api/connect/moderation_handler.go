package apiconnect

import (
	"context"

	connect "connectrpc.com/connect"

	moderationv1 "github.com/Y4shin/open-caucus/gen/go/conference/moderation/v1"
	moderationv1connect "github.com/Y4shin/open-caucus/gen/go/conference/moderation/v1/moderationv1connect"
	moderationservice "github.com/Y4shin/open-caucus/internal/services/moderation"
)

type ModerationHandler struct {
	moderationv1connect.UnimplementedModerationServiceHandler
	service *moderationservice.Service
}

func NewModerationHandler(service *moderationservice.Service) *ModerationHandler {
	return &ModerationHandler{service: service}
}

func (h *ModerationHandler) GetModerationView(ctx context.Context, req *connect.Request[moderationv1.GetModerationViewRequest]) (*connect.Response[moderationv1.GetModerationViewResponse], error) {
	resp, err := h.service.GetModerationView(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *ModerationHandler) ToggleSignupOpen(ctx context.Context, req *connect.Request[moderationv1.ToggleSignupOpenRequest]) (*connect.Response[moderationv1.ToggleSignupOpenResponse], error) {
	resp, err := h.service.ToggleSignupOpen(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.DesiredOpen, req.Msg.ExpectedVersion)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *ModerationHandler) SetMeetingQuotation(ctx context.Context, req *connect.Request[moderationv1.SetMeetingQuotationRequest]) (*connect.Response[moderationv1.SetMeetingQuotationResponse], error) {
	resp, err := h.service.SetMeetingQuotation(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.GenderQuotationEnabled, req.Msg.FirstSpeakerQuotationEnabled)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *ModerationHandler) SetMeetingModerator(ctx context.Context, req *connect.Request[moderationv1.SetMeetingModeratorRequest]) (*connect.Response[moderationv1.SetMeetingModeratorResponse], error) {
	resp, err := h.service.SetMeetingModerator(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.ModeratorAttendeeId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}
