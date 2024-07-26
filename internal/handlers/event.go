package handlers

import (
	"fmt"
	"runtime"

	"sse-server/internal/entities"
	"sse-server/internal/usecases"
	"sse-server/pkg/sse"

	"github.com/gofiber/fiber/v2"
)

type EventHandler interface {
	PatchEvent(c *fiber.Ctx) error
	StreamEvent(c *fiber.Ctx) error
}

type eventHandler struct {
	eventUseCase usecases.EventUseCase
}

func NewEventHandler(eventUseCase usecases.EventUseCase) EventHandler {
	return &eventHandler{eventUseCase: eventUseCase}
}

func (h *eventHandler) PatchEvent(c *fiber.Ctx) error {
	var request entities.Event
	if err := c.BodyParser(&request); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Cannot parse request: "+err.Error())
	}

	eventID := c.Params("id")
	if eventID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Missing event ID.")
	}

	if err := h.eventUseCase.PublishEvent(eventID, request); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Unexpected error")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *eventHandler) StreamEvent(c *fiber.Ctx) error {
	fmt.Printf("Number of Running Goroutines: %d\n", runtime.NumGoroutine())

	eventID := c.Params("id")

	events := make(chan entities.Event)
	go h.eventUseCase.StreamEventById(c.Context(), eventID, events)

	return sse.StreamSSE(c, events)
}
