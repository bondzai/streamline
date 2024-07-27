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
		SubscribeAndStreamEvent(ctx context.Context, channel string, events chan<- entities.Event) error
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

func (u *eventUseCase) SubscribeAndStreamEvent(ctx context.Context, channel string, events chan<- entities.Event) error {
	redisCh, err := u.subscribeRedisEvent(ctx, channel)
	if err != nil {
		log.Printf("Error subscribing to Redis events for channel %s: %v", channel, err)
		return err
	}

	kafkaCh, err := u.subscribeKafkaEvent(ctx, []string{channel})
	if err != nil {
		log.Printf("Error subscribing to Kafka events for channel %s: %v", channel, err)
		return err
	}

	errCh := make(chan error, 1)
	go u.streamEvent(ctx, channel, redisCh, kafkaCh, events, errCh)

	// Wait for the goroutine to report an error or complete
	select {
	case err := <-errCh:
		if err != nil {
			log.Printf("Error streaming events for channel %s: %v", channel, err)
			return err
		}

	case <-ctx.Done():
		log.Printf("Context canceled while streaming events for channel %s", channel)
		return ctx.Err()
	}

	return nil
}

func (u *eventUseCase) streamEvent(
	ctx context.Context,
	channel string,
	redisCh <-chan *redis.Message,
	kafkaCh <-chan *kafka.Message,
	events chan<- entities.Event,
	errCh chan<- error,
) {
	defer close(errCh)

	events <- entities.Event{
		Id:      channel,
		Message: nil,
	}

	for {
		select {
		case <-ctx.Done():
			log.Printf("Context canceled, stopping event stream for channel %s", channel)
			errCh <- ctx.Err()
			return

		case msg, ok := <-redisCh:
			if !ok {
				log.Printf("Redis channel closed for channel %s", channel)
				errCh <- nil
				return
			}
			if err := u.processRedisMessage(msg, channel, events); err != nil {
				log.Printf("Error processing Redis message for channel %s: %v", channel, err)
				errCh <- err
				return
			}

		case msg, ok := <-kafkaCh:
			if !ok {
				log.Printf("Kafka topic closed for channel %s", channel)
				errCh <- nil
				return
			}
			if err := u.processKafkaMessage(msg); err != nil {
				log.Printf("Error processing Kafka message for channel %s: %v", channel, err)
				errCh <- err
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
