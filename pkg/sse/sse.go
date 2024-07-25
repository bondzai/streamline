package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// StreamSSE handles Server-Sent Events for the given context and events channel.
func StreamSSE[T any](ctx context.Context, w http.ResponseWriter, events chan T) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	w.(http.Flusher).Flush()

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
