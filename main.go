package main

import (
	"log"
	"sse-server/config"
	"sse-server/internal/handlers"
	"sse-server/internal/repositories"
	"sse-server/internal/usecases"
	"sse-server/pkg/redis"

	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
)

func init() {
	err := config.LoadConfig()
	if err != nil {
		log.Println(err)
	}
}

func main() {
	redisClient, err := redis.NewClient(redis.Config{
		Address:  viper.GetString("redis.host"),
		Password: viper.GetString("redis.pass"),
		DB:       viper.GetInt("redis.db"),
	})
	if err != nil {
		log.Fatalf("Failed to setup Redis: %v", err)
	}

	eventRepo := repositories.NewEventRepository(redisClient)
	eventUseCase := usecases.NewEventUseCase(eventRepo)
	eventHandler := handlers.NewEventHandler(eventUseCase)

	app := fiber.New()

	app.Get("/api/v1/event/:id", eventHandler.StreamEvent)
	app.Patch("/api/v1/event/:id", eventHandler.PatchEvent)

	if err := app.Listen(":" + viper.GetString("app.port")); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
