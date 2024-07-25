package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"

	"sse-server/internal/entities"
	"sse-server/internal/usecases"
	"sse-server/pkg/sse"
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
		http.Error(w, "Can not parse request.", http.StatusBadRequest)
		return
	}

	eventID := r.URL.Query().Get("id")
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
	fmt.Printf("Number of Running Goroutines: %d\n", runtime.NumGoroutine())

	eventID := r.URL.Query().Get("id")

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	events := make(chan entities.Event)
	go h.eventUseCase.StreamEventById(ctx, eventID, events)

	sse.StreamSSE(ctx, w, events)
}
