package apihttp

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/Y4shin/conference-tool/internal/broker"
)

// MeetingEventsHandler streams meeting-scoped JSON invalidation events to the
// client over Server-Sent Events.
//
// URL pattern (registered on the stripped /api mux):
//
//	GET /realtime/meetings/{meetingId}/events
//
// Any connected subscriber receives an event whenever a moderation mutation
// publishes a broker event scoped to that meeting. The payload is a JSON
// MeetingInvalidationEvent. The client should use each event as a signal to
// refetch the relevant typed read model.
type MeetingEventsHandler struct {
	broker broker.Broker
}

func NewMeetingEventsHandler(b broker.Broker) *MeetingEventsHandler {
	return &MeetingEventsHandler{broker: b}
}

func (h *MeetingEventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	meetingIDStr := r.PathValue("meetingId")
	meetingID, err := strconv.ParseInt(meetingIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid meeting id", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	// Send an initial ping so the client knows the connection is live.
	fmt.Fprintf(w, "event: connected\ndata: {\"meetingId\":\"%d\"}\n\n", meetingID)
	flusher.Flush()

	ch := h.broker.Subscribe(r.Context())
	for {
		select {
		case <-r.Context().Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			// Filter: only forward events scoped to this meeting.
			if evt.MeetingID == nil || *evt.MeetingID != meetingID {
				continue
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Event, evt.Data)
			flusher.Flush()
		}
	}
}
