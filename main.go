package main

import (
	"log"
	"sse-server/config"
	"sse-server/internal/handlers"
	"sse-server/internal/repositories"
	"sse-server/internal/usecases"
	"sse-server/pkg/kafka"
	"sse-server/pkg/redis"

	"github.com/gofiber/fiber/v2"
)

func init() {
	err := config.LoadConfig()
	if err != nil {
		log.Println(err)
	}
}

func main() {
	redisClient, err := redis.NewClient(redis.Config{
		Address:  config.AppConfig.RedisURL,
		Password: config.AppConfig.RedisPassword,
		DB:       config.AppConfig.RedisDatabase,
	})
	if err != nil {
		log.Fatalf("Failed to setup Redis: %v", err)
	}

	kafkaClient, err := kafka.NewClient(kafka.Config{
		Brokers: []string{config.AppConfig.KafkaUrl},
	})
	if err != nil {
		log.Fatalf("Failed to setup Kafka client: %v", err)
	}

	kafkaEventRepo := repositories.KafkaEventRepository(kafkaClient)
	eventRepo := repositories.NewEventRepository(redisClient)
	eventUseCase := usecases.NewEventUseCase(eventRepo, kafkaEventRepo)
	eventHandler := handlers.NewEventHandler(eventUseCase)

	app := fiber.New()

	v1 := app.Group("/api/v1")

	event := v1.Group("event")
	event.Get("/:id", eventHandler.StreamEvent)
	event.Patch("/:id", eventHandler.PatchEvent)

	log.Println("Shutting down...")

	if err := app.Listen(":" + config.AppConfig.AppPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
