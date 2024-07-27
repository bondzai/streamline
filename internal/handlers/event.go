package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"streamline-sse/internal/entities"
	"streamline-sse/internal/usecases"
	"streamline-sse/pkg/sse"

	"github.com/bondzai/gogear/toolbox"
	"github.com/gorilla/mux"
)

const (
	MsgCanNotParseRequest = "Cannot parse request"
	MsgMissingEventID     = "Missing event ID"
	MsgUnexpectedErr      = "Unexpected error"
)

type EventHandler interface {
	StreamEvent(w http.ResponseWriter, r *http.Request)
	PatchEvent(w http.ResponseWriter, r *http.Request)
}

type eventHandler struct {
	eventUseCase usecases.EventUseCase
}

func NewEventHandler(eventUseCase usecases.EventUseCase) EventHandler {
	return &eventHandler{eventUseCase: eventUseCase}
}

func (h *eventHandler) StreamEvent(w http.ResponseWriter, r *http.Request) {
	chanID := mux.Vars(r)["id"]
	if chanID == "" {
		http.Error(w, MsgMissingEventID, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	events := make(chan entities.Event)

	// The use case is responsible for closing the 'events' channel
	h.eventUseCase.SubscribeAndStreamEvent(ctx, chanID, events)

	if err := sse.Stream(ctx, w, events); err != nil {
		http.Error(w, MsgUnexpectedErr, http.StatusInternalServerError)
		return
	}

	toolbox.TrackRoutines()
}

func (h *eventHandler) PatchEvent(w http.ResponseWriter, r *http.Request) {
	var request entities.Event
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, MsgCanNotParseRequest+err.Error(), http.StatusBadRequest)
		return
	}

	chanID := mux.Vars(r)["id"]
	if chanID == "" {
		http.Error(w, MsgMissingEventID, http.StatusBadRequest)
		return
	}

	err = h.eventUseCase.PublishEvent(chanID, request)
	if err != nil {
		http.Error(w, MsgUnexpectedErr, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
