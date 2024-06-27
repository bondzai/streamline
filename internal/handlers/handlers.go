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
		PatchEvent(ctx *fiber.Ctx) error
		StreamEvent(ctx *fiber.Ctx) error
	}

	evenHandler struct {
		eventUseCase usecases.EventUseCase
	}
)

func NewEventHandler() EventHandler {
	return &evenHandler{}
}

func (c evenHandler) PatchEvent(ctx *fiber.Ctx) error {
	customerId := ctx.Params("id")
	var request entities.Event

	if err := ctx.BodyParser(&request); err != nil {
		return ctx.Status(fiber.StatusBadRequest).SendString("Can not parse request.")
	}

	if err := c.eventUseCase.PublishEvent(customerId, *request.LoginSession); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).SendString("Can not parse request.")
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

func (c evenHandler) StreamEvent(ctx *fiber.Ctx) error {
	customerId := ctx.Params("id")

	ctx.Set("Content-Type", "text/event-stream")
	ctx.Set("Cache-Control", "no-cache")
	ctx.Set("Connection", "keep-alive")
	ctx.Set("Transfer-Encoding", "chunked")

	ctx.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		events := make(chan entities.Event)

		c.eventUseCase.StreamEventById(ctx, customerId, events)

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
