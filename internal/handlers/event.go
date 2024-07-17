package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"sse-server/internal/entities"
	"sse-server/internal/usecases"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

type (
	EventHandler interface {
		PatchEvent(c *fiber.Ctx) error
		StreamEvent(c *fiber.Ctx) error
	}

	eventHandler struct {
		eventUseCase usecases.EventUseCase
	}
)

func NewEventHandler(eventUseCase usecases.EventUseCase) EventHandler {
	return &eventHandler{eventUseCase: eventUseCase}
}

func (h eventHandler) PatchEvent(c *fiber.Ctx) error {
	var request entities.Event
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Can not parse request.")
	}

	if err := h.eventUseCase.PublishEvent(c.Params("id"), request); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Unexpected error")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h eventHandler) StreamEvent(c *fiber.Ctx) error {
	channelId := c.Params("id")

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		events := make(chan entities.Event)
		defer close(events)

		go h.eventUseCase.StreamEventById(ctx, channelId, events)

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

				err = w.Flush()
				if err != nil {
					fmt.Printf("Error while flushing: %v. Closing HTTP connection.\n", err)
					return
				}
			case <-ctx.Done():
				fmt.Println("Client connection closed.")
				return
			}
		}
	}))

	return nil
}
