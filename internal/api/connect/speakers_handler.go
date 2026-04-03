package apiconnect

import (
	"context"

	connect "connectrpc.com/connect"

	speakersv1 "github.com/Y4shin/open-caucus/gen/go/conference/speakers/v1"
	speakersv1connect "github.com/Y4shin/open-caucus/gen/go/conference/speakers/v1/speakersv1connect"
	speakerservice "github.com/Y4shin/open-caucus/internal/services/speakers"
)

type SpeakerHandler struct {
	speakersv1connect.UnimplementedSpeakerServiceHandler
	service *speakerservice.Service
}

func NewSpeakerHandler(service *speakerservice.Service) *SpeakerHandler {
	return &SpeakerHandler{service: service}
}

func (h *SpeakerHandler) ListSpeakers(ctx context.Context, req *connect.Request[speakersv1.ListSpeakersRequest]) (*connect.Response[speakersv1.ListSpeakersResponse], error) {
	resp, err := h.service.ListSpeakers(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *SpeakerHandler) AddSpeaker(ctx context.Context, req *connect.Request[speakersv1.AddSpeakerRequest]) (*connect.Response[speakersv1.AddSpeakerResponse], error) {
	resp, err := h.service.AddSpeaker(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.AttendeeId, req.Msg.SpeakerType)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *SpeakerHandler) RemoveSpeaker(ctx context.Context, req *connect.Request[speakersv1.RemoveSpeakerRequest]) (*connect.Response[speakersv1.RemoveSpeakerResponse], error) {
	resp, err := h.service.RemoveSpeaker(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.SpeakerId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *SpeakerHandler) SetSpeakerSpeaking(ctx context.Context, req *connect.Request[speakersv1.SetSpeakerSpeakingRequest]) (*connect.Response[speakersv1.SetSpeakerSpeakingResponse], error) {
	resp, err := h.service.SetSpeakerSpeaking(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.SpeakerId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *SpeakerHandler) SetSpeakerDone(ctx context.Context, req *connect.Request[speakersv1.SetSpeakerDoneRequest]) (*connect.Response[speakersv1.SetSpeakerDoneResponse], error) {
	resp, err := h.service.SetSpeakerDone(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.SpeakerId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *SpeakerHandler) SetSpeakerPriority(ctx context.Context, req *connect.Request[speakersv1.SetSpeakerPriorityRequest]) (*connect.Response[speakersv1.SetSpeakerPriorityResponse], error) {
	resp, err := h.service.SetSpeakerPriority(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.SpeakerId, req.Msg.Priority)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}
