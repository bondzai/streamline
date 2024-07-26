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
	// Initialize Redis client
	redisClient, err := redis.NewClient(redis.Config{
		Address:  config.Env.RedisURL,
		Password: config.Env.RedisPassword,
		DB:       config.Env.RedisDatabase,
	})
	if err != nil {
		log.Fatalf("Failed to setup Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize Kafka client
	kafkaClient, err := kafka.NewClient(kafka.Config{
		Brokers: []string{config.Env.KafkaUrl},
	})
	if err != nil {
		log.Fatalf("Failed to setup Kafka client: %v", err)
	}
	defer kafkaClient.Close()

	// Setup repositories and use case
	kafkaEventRepo := repositories.KafkaEventRepository(kafkaClient)
	eventRepo := repositories.NewEventRepository(redisClient)
	eventUseCase := usecases.NewEventUseCase(eventRepo, kafkaEventRepo)
	eventHandler := handlers.NewEventHandler(eventUseCase)

	// Create a new Fiber app
	app := fiber.New()

	// Define routes
	app.Get("/api/v1/event/:id", eventHandler.StreamEvent)
	app.Patch("/api/v1/event/:id", eventHandler.PatchEvent)

	// Start the server
	serverAddr := ":" + config.Env.AppPort
	log.Printf("Server listening on %s\n", serverAddr)
	if err := app.Listen(serverAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
