package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"

	"sse-server/internal/entities"
	"sse-server/internal/usecases"
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
	eventID := r.URL.Query().Get("id")

	fmt.Printf("Number of Running Goroutines: %d\n", runtime.NumGoroutine())

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	w.(http.Flusher).Flush()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	events := make(chan entities.Event)

	h.eventUseCase.StreamEventById(ctx, eventID, events)

	for {
		select {
		case event := <-events:
			eventData, err := json.Marshal(event)
			if err != nil {
				fmt.Printf("Error encoding event data: %v\n", err)
				continue
			}

			eventStr := fmt.Sprintf("data: %s\n\n", eventData)

			_, err = fmt.Fprint(w, eventStr)
			if err != nil {
				fmt.Printf("Error writing to client: %v. Closing HTTP connection.\n", err)
				return
			}

			w.(http.Flusher).Flush()

		case <-ctx.Done():
			fmt.Println("Client connection closed.")
			return
		}
	}
}
