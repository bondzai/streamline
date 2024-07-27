package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"sse-server/internal/entities"
	"sse-server/internal/usecases"
	"sse-server/pkg/sse"

	"github.com/gorilla/mux"
)

type EventHandler interface {
	PatchEvent(w http.ResponseWriter, r *http.Request)
	StreamEvent(w http.ResponseWriter, r *http.Request)
}

type eventHandler struct {
	eventUseCase usecases.EventUseCase
}

func NewEventHandler(eventUseCase usecases.EventUseCase) EventHandler {
	return &eventHandler{eventUseCase: eventUseCase}
}

func (h *eventHandler) PatchEvent(w http.ResponseWriter, r *http.Request) {
	var request entities.Event
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Cannot parse request: "+err.Error(), http.StatusBadRequest)
		return
	}

	eventID := mux.Vars(r)["id"]
	if eventID == "" {
		http.Error(w, "Missing event ID.", http.StatusBadRequest)
		return
	}

	err = h.eventUseCase.PublishEvent(eventID, request)
	if err != nil {
		http.Error(w, "Unexpected error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *eventHandler) StreamEvent(w http.ResponseWriter, r *http.Request) {
	eventID := mux.Vars(r)["id"]

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	events := make(chan entities.Event)
	h.eventUseCase.StreamEventById(ctx, eventID, events)

	sse.StreamSSE(ctx, w, events)
}
