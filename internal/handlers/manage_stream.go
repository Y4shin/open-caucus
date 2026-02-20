package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Y4shin/conference-tool/internal/routes"
)

const meetingAttendeesChangedEvent = "meeting-attendees-changed"

// ManageStream streams attendee-related manage-page updates for a meeting.
func (h *Handler) ManageStream(ctx context.Context, r *http.Request, params routes.RouteParams) (<-chan routes.ManageStreamEvent, error) {
	meetingID, err := strconv.ParseInt(params.MeetingId, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid meeting ID")
	}
	clientID := strings.TrimSpace(r.URL.Query().Get("client_id"))

	brokerCh := h.Broker.Subscribe(ctx)
	eventCh := make(chan routes.ManageStreamEvent, 8)

	go func() {
		defer close(eventCh)
		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-brokerCh:
				if !ok {
					return
				}
				if evt.Event != meetingAttendeesChangedEvent {
					continue
				}
				if evt.MeetingID == nil || *evt.MeetingID != meetingID {
					continue
				}
				if clientID != "" && evt.OriginClientID == clientID {
					continue
				}

				attendeePartial, err := h.loadManageAttendeeDependentPartial(ctx, params.Slug, params.MeetingId, meetingID)
				if err != nil {
					continue
				}

				select {
				case eventCh <- routes.ManageAttendeeListUpdatedEvent{Data: *attendeePartial}:
				default:
				}
			}
		}
	}()

	return eventCh, nil
}
