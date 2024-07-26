package sse

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

const (
	// constants
	ConnCheckInterval = 10

	// header constants for Server-Sent Events.
	ContentTypeHeader      = "Content-Type"
	ContentTypeValue       = "text/event-stream"
	CacheControlHeader     = "Cache-Control"
	CacheControlValue      = "no-cache"
	ConnectionHeader       = "Connection"
	ConnectionValue        = "keep-alive"
	TransferEncodingHeader = "Transfer-Encoding"
	TransferEncodingValue  = "chunked"

	// log messages for various events and errors.
	MsgErrorEncodingEventData = "SSE error encoding event data"
	MsgErrorWritingToClient   = "SSE error writing to client"
	MsgEventChannelClosed     = "SSE event channel closed"
	MsgClientConnectionClosed = "SSE client connection closed"
)

// logError logs errors with a consistent format.
func logError(message string, err error) {
	if err != nil {
		log.Printf("ERROR: %s: %v", message, err)
	}
}

// validateData prepares and validates the data for the SSE response.
func validateData[T any](event T) ([]byte, error) {
	v := reflect.ValueOf(event)
	if (v.Kind() == reflect.Slice || v.Kind() == reflect.Array) && v.Len() == 0 {
		return []byte("[]"), nil
	}

	return json.Marshal(event)
}

// sendResponse handles sending the response to the client and flushing.
func sendResponse(writer *bufio.Writer, data []byte) error {
	if _, err := fmt.Fprintf(writer, "data: %s\n\n", data); err != nil {
		return err
	}

	return writer.Flush()
}

// StreamSSE handles Server-Sent Events for the given context and events channel.
// It streams events from the provided channel to the HTTP response writer using fasthttp.
func StreamSSE[T any](ctx *fiber.Ctx, events chan T) error {
	ctx.Set(ContentTypeHeader, ContentTypeValue)
	ctx.Set(CacheControlHeader, CacheControlValue)
	ctx.Set(ConnectionHeader, ConnectionValue)
	ctx.Set(TransferEncodingHeader, TransferEncodingValue)

	ctx.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(writer *bufio.Writer) {
		ticker := time.NewTicker(ConnCheckInterval * time.Second)
		defer ticker.Stop()

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

				if err := sendResponse(writer, data); err != nil {
					logError(MsgErrorWritingToClient, err)
					return
				}

			case <-ticker.C:
				if err := writer.Flush(); err != nil {
					logError(MsgClientConnectionClosed, err)
					return
				}
			}
		}
	}))

	return nil
}
