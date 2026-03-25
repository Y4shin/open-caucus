package apiconnect

import (
	"context"

	connect "connectrpc.com/connect"

	attendeesv1 "github.com/Y4shin/conference-tool/gen/go/conference/attendees/v1"
	attendeesv1connect "github.com/Y4shin/conference-tool/gen/go/conference/attendees/v1/attendeesv1connect"
	attendeeservice "github.com/Y4shin/conference-tool/internal/services/attendees"
)

type AttendeeHandler struct {
	attendeesv1connect.UnimplementedAttendeeServiceHandler
	service *attendeeservice.Service
}

func NewAttendeeHandler(service *attendeeservice.Service) *AttendeeHandler {
	return &AttendeeHandler{service: service}
}

func (h *AttendeeHandler) ListAttendees(ctx context.Context, req *connect.Request[attendeesv1.ListAttendeesRequest]) (*connect.Response[attendeesv1.ListAttendeesResponse], error) {
	resp, err := h.service.ListAttendees(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AttendeeHandler) SelfSignup(ctx context.Context, req *connect.Request[attendeesv1.SelfSignupRequest]) (*connect.Response[attendeesv1.SelfSignupResponse], error) {
	resp, err := h.service.SelfSignup(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AttendeeHandler) GuestJoin(ctx context.Context, req *connect.Request[attendeesv1.GuestJoinRequest]) (*connect.Response[attendeesv1.GuestJoinResponse], error) {
	resp, err := h.service.GuestJoin(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.FullName, req.Msg.MeetingSecret, req.Msg.GenderQuoted)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AttendeeHandler) AttendeeLogin(ctx context.Context, req *connect.Request[attendeesv1.AttendeeLoginRequest]) (*connect.Response[attendeesv1.AttendeeLoginResponse], error) {
	resp, cookie, err := h.service.AttendeeLogin(ctx, req.Msg.MeetingId, req.Msg.AttendeeSecret)
	if err != nil {
		return nil, err
	}
	connectResp := connect.NewResponse(resp)
	addCookie(connectResp, cookie)
	return connectResp, nil
}
