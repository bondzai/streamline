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

	http.HandleFunc("/api/v1/event/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			eventHandler.StreamEvent(w, r)
		case http.MethodPatch:
			eventHandler.PatchEvent(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	serverAddr := ":" + config.AppConfig.AppPort
	log.Printf("Server listening on %s\n", serverAddr)
	if err := http.ListenAndServe(serverAddr, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
