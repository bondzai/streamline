package main

import (
	"log"
	"net/http"

	"streamline-sse/config"
	"streamline-sse/internal/handlers"
	"streamline-sse/internal/repositories"
	"streamline-sse/internal/usecases"
	"streamline-sse/pkg/kafka"
	"streamline-sse/pkg/redis"

	"github.com/gofiber/fiber/v2"
	"github.com/gorilla/mux"
)

func init() {
	err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
}

func main() {
	redisClient, err := redis.NewClient(redis.Config{
		Address:  config.Env.RedisURL,
		Password: config.Env.RedisPassword,
		DB:       config.Env.RedisDatabase,
	})
	if err != nil {
		log.Fatalf("Failed to setup Redis client: %v", err)
	}
	defer redisClient.Close()

	kafkaClient, err := kafka.NewClient(kafka.Config{
		Brokers: []string{config.Env.KafkaUrl},
	})
	if err != nil {
		log.Fatalf("Failed to setup Kafka client: %v", err)
	}
	defer kafkaClient.Close()

	kafkaEventRepo := repositories.NewKafkaEventRepository(kafkaClient)
	redisEventRepo := repositories.NewRedisEventRepository(redisClient)
	eventUseCase := usecases.NewEventUseCase(redisEventRepo, kafkaEventRepo)
	eventHandler := handlers.NewEventHandler(eventUseCase)

	app := fiber.New()

	app.Get("/api/v1/fiber", func(c *fiber.Ctx) error {
		return c.SendString("Response from Fiber!")
	})

	go func() {
		if err := app.Listen(":" + config.Env.FiberPort); err != nil {
			log.Fatalf("Fiber server failed to start: %v", err)
		}
	}()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/event/{id:[^/]+}", eventHandler.StreamEvent).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/event/{id:[^/]+}", eventHandler.PatchEvent).Methods(http.MethodPatch)

	netServerAddress := ":" + config.Env.NetPort
	log.Printf("Net/http server listening on %s\n", netServerAddress)
	if err := http.ListenAndServe(netServerAddress, router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
