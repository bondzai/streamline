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

	evenHandler struct {
		eventUseCase usecases.EventUseCase
	}
)

func NewEventHandler(eventUseCase usecases.EventUseCase) EventHandler {
	return &evenHandler{eventUseCase: eventUseCase}
}

func (h evenHandler) PatchEvent(c *fiber.Ctx) error {
	customerId := c.Params("id")
	var request entities.Event

	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Can not parse request.")
	}

	if err := h.eventUseCase.PublishEvent(customerId, *request.LoginSession); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Can not parse request.")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h evenHandler) StreamEvent(c *fiber.Ctx) error {
	customerId := c.Params("id")

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		c, cancel := context.WithCancel(context.Background())
		defer cancel()

		events := make(chan entities.Event)

		h.eventUseCase.StreamEventById(c, customerId, events)

		for event := range events {
			eventData, err := json.Marshal(event)
			if err != nil {
				fmt.Printf("Error encoding event data: %v\n", err)
				continue
			}

			eventStr := fmt.Sprintf("data: %s\n\n", eventData)
			fmt.Fprint(w, eventStr)

			err = w.Flush()
			if err != nil {
				fmt.Printf("Error while flushing: %v. Closing HTTP connection.\n", err)
				break
			}
		}
	}))

	return nil
}
