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
	redisClient, err := redis.NewClient(
		viper.GetString("redis.host"),
		viper.GetString("redis.user"),
		viper.GetString("redis.pass"),
		viper.GetInt("redis.db"),
	)
	if err != nil {
		log.Println(err)
	}

	redisRepo := repositories.NewEventRepository(redisClient)
	eventUseCase := usecases.NewEventUseCase(redisRepo)
	eventHandler := handlers.NewEventHandler(eventUseCase)

	app := fiber.New()

	app.Get("/api/v1/event/:id", eventHandler.StreamEvent)
	app.Patch("/api/v1/event/:id", eventHandler.PatchEvent)

	app.Listen(":" + viper.GetString("app.port"))
}
