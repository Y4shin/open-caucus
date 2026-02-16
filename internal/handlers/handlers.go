package handlers

import (
	"fmt"
	"net/http"

	"github.com/Y4shin/conference-tool/internal/broker"
	"github.com/Y4shin/conference-tool/internal/templates"
)

type Handler struct {
	Broker broker.Broker
}

func NewHandler(b broker.Broker) *Handler {
	return &Handler{Broker: b}
}

func (h *Handler) HomePage(w http.ResponseWriter, r *http.Request) (*templates.HomePageInput, error) {
	return &templates.HomePageInput{
		Counter: h.Broker.Counter(),
	}, nil
}

func (h *Handler) Increment(w http.ResponseWriter, r *http.Request) (*templates.CounterInput, error) {
	h.Broker.Increment()
	return &templates.CounterInput{
		Value: h.Broker.Counter(),
	}, nil
}

func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) error {
	ch := h.Broker.Subscribe(r.Context())
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported")
	}

	for {
		select {
		case <-r.Context().Done():
			return nil
		case evt, ok := <-ch:
			if !ok {
				return nil
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Event, evt.Data)
			flusher.Flush()
		}
	}
}
