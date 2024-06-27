package main

import (
	"log"
	"sse-server/internal/handlers"
	"sse-server/internal/repositories"
	"sse-server/internal/usecases"

	"github.com/gofiber/fiber/v2"
)

func main() {
	redisRepo, err := repositories.NewRedisRepository("localhost:6379", "", "", 0)
	if err != nil {
		log.Println(err)
	}

	eventUseCase := usecases.NewEventUseCase(redisRepo)
	eventHandler := handlers.NewEventHandler(eventUseCase)

	app := fiber.New()

	app.Get("/event/:id", eventHandler.StreamEvent)
	app.Patch("/event/:id", eventHandler.PatchEvent)
}
