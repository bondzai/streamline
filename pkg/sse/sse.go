package sse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// Error messages for various events and errors.
var (
	ErrResponseWriterNotFlushable = errors.New("response writer does not support flushing")
)

// responseWriter abstracts http.ResponseWriter and http.Flusher.
type responseWriter interface {
	http.ResponseWriter
	http.Flusher
}

// setSSEHeaders sets the necessary headers for Server-Sent Events.
func setSSEHeaders(w http.ResponseWriter) {
	w.Header().Set(ContentTypeHeader, ContentTypeValue)
	w.Header().Set(CacheControlHeader, CacheControlValue)
	w.Header().Set(ConnectionHeader, ConnectionValue)
	w.Header().Set(TransferEncodingHeader, TransferEncodingValue)
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
func sendResponse(w responseWriter, data []byte) error {
	if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
		return err
	}
	w.Flush()
	return nil
}

// Stream handles Server-Sent Events for the given context and events channel.
// It streams events from the provided channel to the HTTP response writer.
func Stream[T any](ctx context.Context, w http.ResponseWriter, events chan T) error {
	setSSEHeaders(w)

	flusher, ok := w.(responseWriter)
	if !ok {
		return ErrResponseWriterNotFlushable
	}
	flusher.Flush()

	for {
		select {
		case event, ok := <-events:
			if !ok {
				return nil
			}

			data, err := validateData(event)
			if err != nil {
				return fmt.Errorf("encoding event data: %w", err)
			}

			if err := sendResponse(flusher, data); err != nil {
				return fmt.Errorf("writing to client: %w", err)
			}

		case <-ctx.Done():
			if ctx.Err() == context.Canceled {
				return nil
			}
			return ctx.Err()
		}
	}
}
