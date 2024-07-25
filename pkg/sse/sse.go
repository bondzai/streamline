package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// SSE header constants
const (
	ContentTypeHeader      = "Content-Type"
	ContentTypeValue       = "text/event-stream"
	CacheControlHeader     = "Cache-Control"
	CacheControlValue      = "no-cache"
	ConnectionHeader       = "Connection"
	ConnectionValue        = "keep-alive"
	TransferEncodingHeader = "Transfer-Encoding"
	TransferEncodingValue  = "chunked"
)

// Error messages
const (
	ErrorEncodingEventData = "Error encoding event data"
	ErrorWritingToClient   = "Error writing to client"
	ResponseWriterError    = "Response writer does not support flushing"
	EventChannelClosed     = "Event channel closed"
	ClientConnectionClosed = "Client connection closed"
)

// setSSEHeaders sets the necessary headers for Server-Sent Events.
func setSSEHeaders(w http.ResponseWriter) {
	w.Header().Set(ContentTypeHeader, ContentTypeValue)
	w.Header().Set(CacheControlHeader, CacheControlValue)
	w.Header().Set(ConnectionHeader, ConnectionValue)
	w.Header().Set(TransferEncodingHeader, TransferEncodingValue)
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
		log.Fatal(ResponseWriterError)
	}

	flusher.Flush()

	for {
		select {
		case event, ok := <-events:
			if !ok {
				log.Println(EventChannelClosed)
				return
			}

			eventData, err := json.Marshal(event)
			if err != nil {
				logError(ErrorEncodingEventData, err)
				continue
			}

			eventStr := fmt.Sprintf("data: %s\n\n", eventData)

			_, err = fmt.Fprint(w, eventStr)
			if err != nil {
				logError(ErrorWritingToClient, err)
				return
			}

			flusher.Flush()

		case <-ctx.Done():
			log.Println(ClientConnectionClosed)
			return
		}
	}
}
