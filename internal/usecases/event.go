package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"sse-server/internal/entities"
	"sse-server/internal/repositories"
	"sse-server/pkg/kafka"
	"sse-server/pkg/redis"
)

type (
	EventUseCase interface {
		PublishEvent(channel string, message interface{}) error
		StreamEventById(ctx context.Context, channel string, events chan<- entities.Event)
	}

	eventUseCase struct {
		eventRepo      repositories.EventRepository
		kafkaEventRepo repositories.KafkaEventRepository
	}
)

func NewEventUseCase(eventRepo repositories.EventRepository, kafkaEventRepo repositories.KafkaEventRepository) EventUseCase {
	return &eventUseCase{
		eventRepo:      eventRepo,
		kafkaEventRepo: kafkaEventRepo,
	}
}

func (u *eventUseCase) PublishEvent(channel string, message interface{}) error {
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		log.Println("Json marshal error: ", err)
		return err
	}

	err = u.eventRepo.Publish(channel, jsonMessage)
	if err != nil {
		log.Println("Publish event error: ", err)
		return err
	}

	err = u.kafkaEventRepo.Publish(channel, message)
	if err != nil {
		log.Println("Publish event to kafka error: ", err)
	}

	return nil
}

func (u *eventUseCase) subscribeRedisEvent(channel string) (<-chan *redis.Message, error) {
	messageChannel, err := u.eventRepo.Subscribe(channel)
	if err != nil {
		log.Println("Subscribe redis event error: ", err)
		return nil, err
	}

	return messageChannel, nil
}

func (u *eventUseCase) subscribeKafkaEvent(topic []string) (<-chan *kafka.Message, error) {
	messageTopic, err := u.kafkaEventRepo.Subscribe(topic)
	if err != nil {
		log.Println("Subscribe kafka event error: ", err)
		return nil, err
	}

	return messageTopic, nil
}

func (u *eventUseCase) StreamEventById(ctx context.Context, channel string, events chan<- entities.Event) {
	messageChannel, err := u.subscribeRedisEvent(channel)
	if err != nil {
		close(events)
		return
	}

	messageTopic, err := u.subscribeKafkaEvent([]string{channel})
	if err != nil {
		close(events)
		return
	}

	// Get the number of running Goroutines
	numGoroutines := runtime.NumGoroutine()
	fmt.Printf("Number of Running Goroutines: %d\n", numGoroutines)

	go func() {
		defer close(events)

		var event entities.Event
		events <- event

		for {
			select {
			case <-ctx.Done():
				log.Println("Stopped receiving messages from Redis.", channel)
				return

			case msg, ok := <-messageChannel:
				if !ok {
					log.Println("Redis message channel closed.", channel)
					return
				}

				if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
					log.Printf("Error unmarshaling message from Redis: %v", err)
					continue
				}

				event.Id = channel
				events <- event

			case msgKafka, ok := <-messageTopic:
				if !ok {
					return
				}

				if err := json.Unmarshal(msgKafka.Value, &event); err != nil {
					log.Printf("Error unmarshaling message from Kafka: %v", err)
					continue
				}

				log.Printf("Kafka received messages: %s", string(msgKafka.Value))

				event.Id = channel
				events <- event
			}
		}
	}()
}
