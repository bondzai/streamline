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
		SubscribeAndStreamEvent(ctx context.Context, channel string, events chan<- entities.Event)
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

func (u *eventUseCase) SubscribeAndStreamEvent(ctx context.Context, channel string, events chan<- entities.Event) {
	redisCh, err := u.subscribeRedisEvent(ctx, channel)
	if err != nil {
		log.Printf("Error subscribing to Redis events for channel %s: %v", channel, err)
		close(events)
		return
	}

	kafkaCh, err := u.subscribeKafkaEvent(ctx, []string{channel})
	if err != nil {
		log.Printf("Error subscribing to Kafka events for channel %s: %v", channel, err)
		close(events)
		return
	}

	go u.streamEvent(ctx, channel, redisCh, kafkaCh, events)
}

func (u *eventUseCase) streamEvent(
	ctx context.Context,
	channel string,
	redisCh <-chan *redis.Message,
	kafkaCh <-chan *kafka.Message,
	events chan<- entities.Event,
) {
	defer close(events)
	events <- entities.Event{
		Id:      channel,
		Message: nil,
	}

	for {
		select {
		case <-ctx.Done():
			log.Printf("Context canceled, stopping event stream for channel %s", channel)
			return

		case msg, ok := <-redisCh:
			if !ok {
				log.Printf("Redis channel closed for channel %s", channel)
				return
			}
			if err := u.processRedisMessage(msg, channel, events); err != nil {
				log.Printf("Stopping event stream for channel %s due to error", channel)
				return
			}

		case msg, ok := <-kafkaCh:
			if !ok {
				log.Printf("Kafka topic closed for channel %s", channel)
				return
			}
			if err := u.processKafkaMessage(msg); err != nil {
				log.Printf("Stopping event stream for channel %s due to error", channel)
				return
			}
		}
	}
}

func (u *eventUseCase) processRedisMessage(msg *redis.Message, channel string, events chan<- entities.Event) error {
	var event entities.Event
	if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
		log.Printf("Error unmarshaling Redis message for channel %s: %v", channel, err)
		return err
	}

	event.Id = channel
	events <- event

	return nil
}

func (u *eventUseCase) processKafkaMessage(msg *kafka.Message) error {
	toolbox.PPrint(msg)
	return nil
}

func (u *eventUseCase) subscribeRedisEvent(ctx context.Context, channel string) (<-chan *redis.Message, error) {
	return u.redisEventRepo.Subscribe(ctx, channel)
}

func (u *eventUseCase) subscribeKafkaEvent(ctx context.Context, topics []string) (<-chan *kafka.Message, error) {
	return u.kafkaEventRepo.Subscribe(ctx, topics, 0, consumerGroupName)
}

func (u *eventUseCase) PublishEvent(channel string, message interface{}) error {
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message for channel %s: %v", channel, err)
		return err
	}

	if err := u.redisEventRepo.Publish(channel, jsonMessage); err != nil {
		log.Printf("Error publishing to Redis for channel %s: %v", channel, err)
		return err
	}

	if err := u.kafkaEventRepo.Publish(channel, message); err != nil {
		log.Printf("Error publishing to Kafka for channel %s: %v", channel, err)
		return err
	}

	return nil
}
