package usecases

import (
	"context"
	"encoding/json"
	"log"
	"streamline-sse/internal/entities"
	"streamline-sse/internal/repositories"
	"streamline-sse/pkg/kafka"
	"streamline-sse/pkg/redis"

	"github.com/bondzai/gogear/toolbox"
)

const consumerGroupName = "consumerGroup1"

type (
	EventUseCase interface {
		PublishEvent(channel string, message interface{}) error
		StreamEvent(ctx context.Context, channel string, events chan<- entities.Event)
	}

	eventUseCase struct {
		redisEventRepo repositories.RedisEventRepository
		kafkaEventRepo repositories.KafkaEventRepository
	}
)

func NewEventUseCase(redisEventRepo repositories.RedisEventRepository, kafkaEventRepo repositories.KafkaEventRepository) EventUseCase {
	return &eventUseCase{
		redisEventRepo: redisEventRepo,
		kafkaEventRepo: kafkaEventRepo,
	}
}

func (u *eventUseCase) StreamEvent(ctx context.Context, channel string, events chan<- entities.Event) {
	redisMessageChannel, err := u.subscribeRedisEvent(ctx, channel)
	if err != nil {
		log.Println("Error subscribe event from Redis: ", err)
		close(events)
		return
	}

	kafkaMessageTopics, err := u.subscribeKafkaEvent(ctx, []string{channel})
	if err != nil {
		log.Println("Error subscribe event from Kafka: ", err)
		close(events)
		return
	}

	go func() {
		var event entities.Event
		events <- event

		for {
			select {
			case <-ctx.Done():
				log.Println("Stopped receiving messages from source: ", channel)
				return

			case msg, ok := <-redisMessageChannel:
				if !ok {
					log.Println("Redis message channel closed: ", channel)
					return
				}

				if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
					log.Println("Error unmarshaling message from Redis: ", err)
					continue
				}

				event.Id = channel
				events <- event

			case msg, ok := <-kafkaMessageTopics:
				if !ok {
					log.Println("Kafka message topics closed: ", channel)
					return
				}

				toolbox.PPrint(msg)
			}
		}
	}()
}

func (u *eventUseCase) subscribeRedisEvent(ctx context.Context, channel string) (<-chan *redis.Message, error) {
	messageChannel, err := u.redisEventRepo.Subscribe(ctx, channel)
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

	err = u.redisEventRepo.Publish(channel, jsonMessage)
	if err != nil {
		log.Println("Publish event to Redis error: ", err)
		return err
	}

	err = u.kafkaEventRepo.Publish(channel, message)
	if err != nil {
		log.Println("Publish event to Kafka error: ", err)
	}

	return nil
}
