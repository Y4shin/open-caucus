package apiconnect

import (
	"context"

	connect "connectrpc.com/connect"

	attendeesv1 "github.com/Y4shin/open-caucus/gen/go/conference/attendees/v1"
	attendeesv1connect "github.com/Y4shin/open-caucus/gen/go/conference/attendees/v1/attendeesv1connect"
	attendeeservice "github.com/Y4shin/open-caucus/internal/services/attendees"
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

func (h *AttendeeHandler) CreateAttendee(ctx context.Context, req *connect.Request[attendeesv1.CreateAttendeeRequest]) (*connect.Response[attendeesv1.CreateAttendeeResponse], error) {
	resp, err := h.service.CreateAttendee(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.FullName, req.Msg.GenderQuoted)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AttendeeHandler) DeleteAttendee(ctx context.Context, req *connect.Request[attendeesv1.DeleteAttendeeRequest]) (*connect.Response[attendeesv1.DeleteAttendeeResponse], error) {
	resp, err := h.service.DeleteAttendee(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.AttendeeId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AttendeeHandler) SetChairperson(ctx context.Context, req *connect.Request[attendeesv1.SetChairpersonRequest]) (*connect.Response[attendeesv1.SetChairpersonResponse], error) {
	resp, err := h.service.SetChairperson(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.AttendeeId, req.Msg.IsChair)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AttendeeHandler) SetQuoted(ctx context.Context, req *connect.Request[attendeesv1.SetQuotedRequest]) (*connect.Response[attendeesv1.SetQuotedResponse], error) {
	resp, err := h.service.SetQuoted(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.AttendeeId, req.Msg.Quoted)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AttendeeHandler) GetAttendeeRecovery(ctx context.Context, req *connect.Request[attendeesv1.GetAttendeeRecoveryRequest]) (*connect.Response[attendeesv1.GetAttendeeRecoveryResponse], error) {
	resp, err := h.service.GetAttendeeRecovery(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.AttendeeId, req.Msg.BaseUrl)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AttendeeHandler) InviteSecretJoin(ctx context.Context, req *connect.Request[attendeesv1.InviteSecretJoinRequest]) (*connect.Response[attendeesv1.InviteSecretJoinResponse], error) {
	resp, cookie, err := h.service.InviteSecretJoin(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.InviteSecret)
	if err != nil {
		return nil, err
	}
	out := connect.NewResponse(resp)
	if cookie != nil {
		out.Header().Add("Set-Cookie", cookie.String())
	}
	return out, nil
}
