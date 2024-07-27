package main

import (
	"log"
	"net/http"
	"sse-server/config"
	"sse-server/internal/handlers"
	"sse-server/internal/repositories"
	"sse-server/internal/usecases"
	"sse-server/pkg/kafka"
	"sse-server/pkg/redis"

	"github.com/gofiber/fiber/v2"
	"github.com/gorilla/mux"
)

func init() {
	err := config.LoadConfig()
	if err != nil {
		log.Println(err)
	}
}

func main() {
	redisClient, err := redis.NewClient(redis.Config{
		Address:  config.Env.RedisURL,
		Password: config.Env.RedisPassword,
		DB:       config.Env.RedisDatabase,
	})
	if err != nil {
		log.Fatalf("Failed to setup Redis: %v", err)
	}
	defer redisClient.Close()

	kafkaClient, err := kafka.NewClient(kafka.Config{
		Brokers: []string{config.Env.KafkaUrl},
	})
	if err != nil {
		log.Fatalf("Failed to setup Kafka client: %v", err)
	}
	defer kafkaClient.Close()

	kafkaEventRepo := repositories.KafkaEventRepository(kafkaClient)
	eventRepo := repositories.NewEventRepository(redisClient)
	eventUseCase := usecases.NewEventUseCase(eventRepo, kafkaEventRepo)
	eventHandler := handlers.NewEventHandler(eventUseCase)

	app := fiber.New()

	app.Get("/api/v1/fiber", func(c *fiber.Ctx) error {
		return c.SendString("Response from Fiber!")
	})

	go func() {
		if err := app.Listen(":3001"); err != nil {
			log.Fatalf("Fiber server failed to start: %v", err)
		}
	}()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/event/{id:[^/]+}", eventHandler.StreamEvent).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/event/{id:[^/]+}", eventHandler.PatchEvent).Methods(http.MethodPatch)

	serverAddr := ":" + config.Env.AppPort
	log.Printf("Server listening on %s\n", serverAddr)
	if err := http.ListenAndServe(serverAddr, router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
