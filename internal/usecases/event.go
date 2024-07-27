package usecases

import (
	"context"
	"encoding/json"
	"log"
	"sse-server/internal/entities"
	"sse-server/internal/repositories"
	"sse-server/pkg/kafka"
	"sse-server/pkg/redis"

	"github.com/bondzai/gogear/toolbox"
)

const consumerGroupName = "consumerGroup1"

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

func (u *eventUseCase) StreamEventById(ctx context.Context, channel string, events chan<- entities.Event) {
	redisMessageChannel, err := u.subscribeRedisEvent(ctx, channel)
	if err != nil {
		close(events)
		return
	}

	kafkaMessageTopics, err := u.subscribeKafkaEvent(ctx, []string{channel})
	if err != nil {
		close(events)
		return
	}

	go func() {
		var event entities.Event
		events <- event

		for {
			select {
			case <-ctx.Done():
				log.Println("Stopped receiving messages from source.", channel)
				return

			case msg, ok := <-redisMessageChannel:
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

			case msg, ok := <-kafkaMessageTopics:
				if !ok {
					log.Println("Kafka message topics closed.", channel)
					return
				}

				toolbox.PPrint(msg)
			}
		}
	}()
}

func (u *eventUseCase) subscribeRedisEvent(ctx context.Context, channel string) (<-chan *redis.Message, error) {
	messageChannel, err := u.eventRepo.Subscribe(ctx, channel)
	if err != nil {
		log.Println("Subscribe Redis event error: ", err)
		return nil, err
	}

	return messageChannel, nil
}

func (u *eventUseCase) subscribeKafkaEvent(ctx context.Context, topic []string) (<-chan *kafka.Message, error) {
	messageTopic, err := u.kafkaEventRepo.Subscribe(ctx, topic, 0, consumerGroupName)
	if err != nil {
		log.Println("Subscribe Kafka event error: ", err)
		return nil, err
	}

	return messageTopic, nil
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
