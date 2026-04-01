package apiconnect

import (
	"context"

	connect "connectrpc.com/connect"

	agendav1 "github.com/Y4shin/conference-tool/gen/go/conference/agenda/v1"
	agendav1connect "github.com/Y4shin/conference-tool/gen/go/conference/agenda/v1/agendav1connect"
	agendaservice "github.com/Y4shin/conference-tool/internal/services/agenda"
)

type AgendaHandler struct {
	agendav1connect.UnimplementedAgendaServiceHandler
	service *agendaservice.Service
}

func NewAgendaHandler(service *agendaservice.Service) *AgendaHandler {
	return &AgendaHandler{service: service}
}

func (h *AgendaHandler) ListAgendaPoints(ctx context.Context, req *connect.Request[agendav1.ListAgendaPointsRequest]) (*connect.Response[agendav1.ListAgendaPointsResponse], error) {
	resp, err := h.service.ListAgendaPoints(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AgendaHandler) GetAgendaPointTools(ctx context.Context, req *connect.Request[agendav1.GetAgendaPointToolsRequest]) (*connect.Response[agendav1.GetAgendaPointToolsResponse], error) {
	resp, err := h.service.GetAgendaPointTools(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.AgendaPointId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AgendaHandler) CreateAgendaPoint(ctx context.Context, req *connect.Request[agendav1.CreateAgendaPointRequest]) (*connect.Response[agendav1.CreateAgendaPointResponse], error) {
	resp, err := h.service.CreateAgendaPoint(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.Title, req.Msg.ParentAgendaPointId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AgendaHandler) DeleteAgendaPoint(ctx context.Context, req *connect.Request[agendav1.DeleteAgendaPointRequest]) (*connect.Response[agendav1.DeleteAgendaPointResponse], error) {
	resp, err := h.service.DeleteAgendaPoint(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.AgendaPointId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AgendaHandler) MoveAgendaPoint(ctx context.Context, req *connect.Request[agendav1.MoveAgendaPointRequest]) (*connect.Response[agendav1.MoveAgendaPointResponse], error) {
	resp, err := h.service.MoveAgendaPoint(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.AgendaPointId, req.Msg.Direction)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AgendaHandler) ActivateAgendaPoint(ctx context.Context, req *connect.Request[agendav1.ActivateAgendaPointRequest]) (*connect.Response[agendav1.ActivateAgendaPointResponse], error) {
	resp, err := h.service.ActivateAgendaPoint(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.AgendaPointId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AgendaHandler) SetCurrentAttachment(ctx context.Context, req *connect.Request[agendav1.SetCurrentAttachmentRequest]) (*connect.Response[agendav1.SetCurrentAttachmentResponse], error) {
	resp, err := h.service.SetCurrentAttachment(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.AgendaPointId, req.Msg.AttachmentId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AgendaHandler) ClearCurrentDocument(ctx context.Context, req *connect.Request[agendav1.ClearCurrentDocumentRequest]) (*connect.Response[agendav1.ClearCurrentDocumentResponse], error) {
	resp, err := h.service.ClearCurrentDocument(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.AgendaPointId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AgendaHandler) DeleteAttachment(ctx context.Context, req *connect.Request[agendav1.DeleteAttachmentRequest]) (*connect.Response[agendav1.DeleteAttachmentResponse], error) {
	resp, err := h.service.DeleteAttachment(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.AgendaPointId, req.Msg.AttachmentId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AgendaHandler) UpdateAgendaPoint(ctx context.Context, req *connect.Request[agendav1.UpdateAgendaPointRequest]) (*connect.Response[agendav1.UpdateAgendaPointResponse], error) {
	resp, err := h.service.UpdateAgendaPoint(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.AgendaPointId, req.Msg.Title)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}
