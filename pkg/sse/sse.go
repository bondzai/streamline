package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
)

// Header constants for Server-Sent Events.
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

// Log messages for various events and errors.
const (
	MsgErrorEncodingEventData = "error encoding event data"
	MsgErrorWritingToClient   = "error writing to client"
	MsgResponseWriterError    = "response writer does not support flushing"
	MsgEventChannelClosed     = "event channel closed"
	MsgClientConnectionClosed = "client connection closed"
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
		log.Printf("ERROR: %s: %v", message, err)
	}
}

// StreamSSE handles Server-Sent Events for the given context and events channel.
// It streams events from the provided channel to the HTTP response writer.
func StreamSSE[T any](ctx context.Context, w http.ResponseWriter, events chan T) {
	setSSEHeaders(w)

	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Fatal(MsgResponseWriterError)
	}

	flusher.Flush()

	for {
		select {
		case event, ok := <-events:
			if !ok {
				log.Println(MsgEventChannelClosed)
				return
			}

			if reflect.ValueOf(event).Kind() == reflect.Slice || reflect.ValueOf(event).Kind() == reflect.Array {
				if reflect.ValueOf(event).Len() == 0 {
					fmt.Fprint(w, "data: []\n\n")
					flusher.Flush()
					continue
				}
			}

			eventData, err := json.Marshal(event)
			if err != nil {
				logError(MsgErrorEncodingEventData, err)
				continue
			}

			if _, err := fmt.Fprintf(w, "data: %s\n\n", eventData); err != nil {
				logError(MsgErrorWritingToClient, err)
				return
			}

			flusher.Flush()

		case <-ctx.Done():
			log.Println(MsgClientConnectionClosed)
			return
		}
	}
}
