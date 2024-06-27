package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type Event struct {
	Id           string  `json:"-"`
	LoginSession *string `json:"loginSession"`
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/event/{id}", serveServerSentEvents)
	http.Handle("/", r)

	port := ":4400"
	fmt.Printf("Server listening on port %s\n", port)
	log.Fatal(http.ListenAndServe(port, r))
}

func serveServerSentEvents(w http.ResponseWriter, r *http.Request) {
	// Extract event ID from URL path
	vars := mux.Vars(r)
	eventID := vars["id"]

	// Check if the client supports flushing
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Set headers for Server-Sent Events
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Channel to send events to clients
	events := make(chan Event)
	defer close(events)

	// Start generating events in a goroutine
	go generateEvents(r.Context(), events, eventID)

	// Continuously send events to the client
	for event := range events {
		eventData, err := json.Marshal(event)
		if err != nil {
			log.Printf("Error encoding event data: %v\n", err)
			continue
		}

		// Format the Server-Sent Event string
		eventStr := fmt.Sprintf("data: %s\n\n", eventData)

		// Send the event string and flush
		fmt.Fprint(w, eventStr)
		flusher.Flush()
	}

	log.Println("Finished sending event updates...")
}

func generateEvents(ctx context.Context, events chan<- Event, eventID string) {
	r := rand.New(rand.NewSource(time.Now().Unix()))

	// Send initial event with nil login session
	initialEvent := Event{
		Id:           fmt.Sprintf("event-%d", r.Intn(100)),
		LoginSession: nil,
	}
	events <- initialEvent

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Event generation stopped.", eventID)
			return
		case <-ticker.C:
			// Update login session randomly every 5 seconds
			newSession := fmt.Sprintf("session-%d", r.Intn(1000))
			event := Event{
				Id:           fmt.Sprintf("event-%d", r.Intn(100)),
				LoginSession: &newSession,
			}
			events <- event
		}
	}
}
