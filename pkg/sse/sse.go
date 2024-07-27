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
	MsgErrorEncodingEventData = "SSE error encoding event data"
	MsgErrorWritingToClient   = "SSE error writing to client"
	MsgResponseWriterError    = "SSE response writer does not support flushing"
	MsgEventChannelClosed     = "SSE event channel closed"
	MsgClientConnectionClosed = "SSE client connection closed"
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

// isEmptySliceOrArray checks if the given event is an empty slice or array.
func isEmptySliceOrArray(event interface{}) bool {
	v := reflect.ValueOf(event)
	return (v.Kind() == reflect.Slice || v.Kind() == reflect.Array) && v.Len() == 0
}

// validateData prepares and validates the data for the SSE response.
func validateData[T any](event T) ([]byte, error) {
	if isEmptySliceOrArray(event) {
		return []byte("[]"), nil
	}
	return json.Marshal(event)
}

// sendResponse handles sending the response to the client and flushing.
func sendResponse(w http.ResponseWriter, flusher http.Flusher, data []byte) {
	if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
		logError(MsgErrorWritingToClient, err)
		return
	}
	flusher.Flush()
}

// StreamSSE handles Server-Sent Events for the given context and events channel.
// It streams events from the provided channel to the HTTP response writer.
func Stream[T any](ctx context.Context, w http.ResponseWriter, events chan T) {
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

			data, err := validateData(event)
			if err != nil {
				logError(MsgErrorEncodingEventData, err)
				continue
			}

			sendResponse(w, flusher, data)

		case <-ctx.Done():
			log.Println(MsgClientConnectionClosed)
			return
		}
	}
}
