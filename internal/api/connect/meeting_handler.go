package apiconnect

import (
	"context"
	"strconv"

	connect "connectrpc.com/connect"

	meetingsv1 "github.com/Y4shin/conference-tool/gen/go/conference/meetings/v1"
	meetingsv1connect "github.com/Y4shin/conference-tool/gen/go/conference/meetings/v1/meetingsv1connect"
	"github.com/Y4shin/conference-tool/internal/broker"
	meetingservice "github.com/Y4shin/conference-tool/internal/services/meetings"
)

type MeetingHandler struct {
	meetingsv1connect.UnimplementedMeetingServiceHandler
	service *meetingservice.Service
	broker  broker.Broker
}

func NewMeetingHandler(service *meetingservice.Service, b broker.Broker) *MeetingHandler {
	return &MeetingHandler{service: service, broker: b}
}

func (h *MeetingHandler) GetJoinMeeting(ctx context.Context, req *connect.Request[meetingsv1.GetJoinMeetingRequest]) (*connect.Response[meetingsv1.GetJoinMeetingResponse], error) {
	resp, err := h.service.GetJoinMeeting(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *MeetingHandler) GetLiveMeeting(ctx context.Context, req *connect.Request[meetingsv1.GetLiveMeetingRequest]) (*connect.Response[meetingsv1.GetLiveMeetingResponse], error) {
	resp, err := h.service.GetLiveMeeting(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

// SubscribeMeetingEvents streams typed invalidation events scoped to a meeting.
// Each MeetingEvent carries a kind that tells the client which view to refetch.
func (h *MeetingHandler) SubscribeMeetingEvents(
	ctx context.Context,
	req *connect.Request[meetingsv1.SubscribeMeetingEventsRequest],
	stream *connect.ServerStream[meetingsv1.MeetingEvent],
) error {
	meetingID, err := strconv.ParseInt(req.Msg.MeetingId, 10, 64)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Send an initial event so the client knows the connection is live.
	if err := stream.Send(&meetingsv1.MeetingEvent{
		Kind: meetingsv1.MeetingEventKind_MEETING_EVENT_KIND_MEETING_UPDATED,
	}); err != nil {
		return err
	}

	ch := h.broker.Subscribe(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil
		case evt, ok := <-ch:
			if !ok {
				return nil
			}
			if evt.MeetingID == nil || *evt.MeetingID != meetingID {
				continue
			}
			kind := brokerEventToKind(evt.Event)
			if kind == meetingsv1.MeetingEventKind_MEETING_EVENT_KIND_UNSPECIFIED {
				continue
			}
			if err := stream.Send(&meetingsv1.MeetingEvent{Kind: kind}); err != nil {
				return err
			}
		}
	}
}

// brokerEventToKind maps the broker SSE event name to a typed MeetingEventKind.
func brokerEventToKind(event string) meetingsv1.MeetingEventKind {
	switch event {
	case "speakers.updated", "speakers-updated":
		return meetingsv1.MeetingEventKind_MEETING_EVENT_KIND_SPEAKERS_UPDATED
	case "votes.updated", "votes-updated":
		return meetingsv1.MeetingEventKind_MEETING_EVENT_KIND_VOTES_UPDATED
	case "meeting-votes-changed":
		return meetingsv1.MeetingEventKind_MEETING_EVENT_KIND_VOTES_UPDATED
	case "agenda.updated", "agenda-updated":
		return meetingsv1.MeetingEventKind_MEETING_EVENT_KIND_AGENDA_UPDATED
	case "attendees.updated", "attendees-updated":
		return meetingsv1.MeetingEventKind_MEETING_EVENT_KIND_ATTENDEES_UPDATED
	case "meeting-attendees-changed":
		return meetingsv1.MeetingEventKind_MEETING_EVENT_KIND_ATTENDEES_UPDATED
	case "moderate-updated", "live-updated":
		return meetingsv1.MeetingEventKind_MEETING_EVENT_KIND_MEETING_UPDATED
	default:
		return meetingsv1.MeetingEventKind_MEETING_EVENT_KIND_UNSPECIFIED
	}
}
