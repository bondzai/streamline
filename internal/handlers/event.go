package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"streamline/internal/entities"
	"streamline/internal/usecases"
	"streamline/pkg/sse"

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
	chID := mux.Vars(r)["id"]
	if chID == "" {
		http.Error(w, MsgMissingEventID, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	eventCh := make(chan entities.Event)

	// The use case is responsible for closing the 'eventCh' channel
	if err := h.eventUseCase.SubscribeAndStreamEvent(ctx, chID, eventCh); err != nil {
		http.Error(w, MsgUnexpectedErr, http.StatusInternalServerError)
		return
	}

	if err := sse.Stream(ctx, w, eventCh); err != nil {
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

	chID := mux.Vars(r)["id"]
	if chID == "" {
		http.Error(w, MsgMissingEventID, http.StatusBadRequest)
		return
	}

	err = h.eventUseCase.PublishEvent(chID, request)
	if err != nil {
		http.Error(w, MsgUnexpectedErr, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
