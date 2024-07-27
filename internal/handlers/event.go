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
	MsgMissingEventId     = "Missing event ID"
	MsgUnExpectedErr      = "Unexpected error"
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
		http.Error(w, MsgCanNotParseRequest+err.Error(), http.StatusBadRequest)
		return
	}

	chanID := mux.Vars(r)["id"]
	if chanID == "" {
		http.Error(w, MsgMissingEventId, http.StatusBadRequest)
		return
	}

	err = h.eventUseCase.PublishEvent(chanID, request)
	if err != nil {
		http.Error(w, MsgUnExpectedErr, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *eventHandler) StreamEvent(w http.ResponseWriter, r *http.Request) {
	toolbox.TrackRoutines()

	chanID := mux.Vars(r)["id"]

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	events := make(chan entities.Event)
	h.eventUseCase.StreamEvent(ctx, chanID, events)

	sse.StreamSSE(ctx, w, events)
}
