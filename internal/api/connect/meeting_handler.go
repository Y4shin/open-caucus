package apiconnect

import (
	"context"

	connect "connectrpc.com/connect"

	meetingsv1 "github.com/Y4shin/conference-tool/gen/go/conference/meetings/v1"
	meetingsv1connect "github.com/Y4shin/conference-tool/gen/go/conference/meetings/v1/meetingsv1connect"
	meetingservice "github.com/Y4shin/conference-tool/internal/services/meetings"
)

type MeetingHandler struct {
	meetingsv1connect.UnimplementedMeetingServiceHandler
	service *meetingservice.Service
}

func NewMeetingHandler(service *meetingservice.Service) *MeetingHandler {
	return &MeetingHandler{service: service}
}

func (h *MeetingHandler) GetLiveMeeting(ctx context.Context, req *connect.Request[meetingsv1.GetLiveMeetingRequest]) (*connect.Response[meetingsv1.GetLiveMeetingResponse], error) {
	resp, err := h.service.GetLiveMeeting(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}
