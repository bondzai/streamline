package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// setSSEHeaders sets the necessary headers for Server-Sent Events.
func setSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")
}

// logError logs errors with a consistent format.
func logError(message string, err error) {
	if err != nil {
		log.Printf("ERROR: %s: %v\n", message, err)
	}
}

// StreamSSE handles Server-Sent Events for the given context and events channel.
func StreamSSE[T any](ctx context.Context, w http.ResponseWriter, events chan T) {
	setSSEHeaders(w)

	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Fatal("Response writer does not support flushing")
	}

	flusher.Flush()

	for {
		select {
		case event, ok := <-events:
			if !ok {
				log.Println("Event channel closed")
				return
			}

			eventData, err := json.Marshal(event)
			if err != nil {
				logError("Error encoding event data", err)
				continue
			}

			eventStr := fmt.Sprintf("data: %s\n\n", eventData)

			_, err = fmt.Fprint(w, eventStr)
			if err != nil {
				logError("Error writing to client", err)
				return
			}

			flusher.Flush()

		case <-ctx.Done():
			log.Println("Client connection closed.")
			return
		}
	}
}
