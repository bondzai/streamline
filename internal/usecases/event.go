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
		SubscribeAndStreamEvent(ctx context.Context, channel string, eventCh chan<- entities.Event) error
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

func (u *eventUseCase) SubscribeAndStreamEvent(ctx context.Context, chName string, eventCh chan<- entities.Event) error {
	redisCh, err := u.redisEventRepo.Subscribe(ctx, chName)
	if err != nil {
		log.Printf("Error subscribing to Redis events for channel %s: %v", chName, err)
		return err
	}

	kafkaCh, err := u.kafkaEventRepo.Subscribe(ctx, []string{chName}, 0, consumerGroupName)
	if err != nil {
		log.Printf("Error subscribing to Kafka events for channel %s: %v", chName, err)
		return err
	}

	if err := u.streamEvent(ctx, chName, redisCh, kafkaCh, eventCh); err != nil {
		log.Printf("Error streaming events for channel %s: %v", chName, err)
		return err
	}

	return nil
}

func (u *eventUseCase) streamEvent(
	ctx context.Context,
	chName string,
	redisCh <-chan *redis.Message,
	kafkaCh <-chan *kafka.Message,
	eventCh chan<- entities.Event,
) error {
	errCh := make(chan error, 1)

	// Stream processing function
	processStreams := func() {
		defer func() {
			close(errCh)
			close(eventCh)
		}()

		eventCh <- entities.Event{
			Id:      chName,
			Message: nil,
		}

		for {
			select {
			case <-ctx.Done():
				log.Printf("Context canceled, stopping event stream for channel %s", chName)
				errCh <- ctx.Err()
				return

			case msg, ok := <-redisCh:
				if !ok {
					log.Printf("Redis channel closed for channel %s", chName)
					errCh <- nil
					return
				}

				event, err := u.processRedisMessage(msg, chName)
				if err != nil {
					log.Printf("Error processing Redis message for channel %s: %v", chName, err)
					errCh <- err
					return
				}
				eventCh <- *event

			case msg, ok := <-kafkaCh:
				if !ok {
					log.Printf("Kafka topic closed for channel %s", chName)
					errCh <- nil
					return
				}

				if err := u.processKafkaMessage(msg); err != nil {
					log.Printf("Error processing Kafka message for channel %s: %v", chName, err)
					errCh <- err
					return
				}
			}
		}
	}

	go processStreams()

	select {
	case err := <-errCh:
		return err

	case <-ctx.Done():
		return ctx.Err()

	default:
		return nil
	}
}

func (u *eventUseCase) processRedisMessage(msg *redis.Message, chName string) (*entities.Event, error) {
	var event entities.Event
	event.Id = chName

	if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
		log.Printf("Error unmarshaling Redis message for channel %s: %v", chName, err)
		return nil, err
	}

	return &event, nil
}

func (u *eventUseCase) processKafkaMessage(msg *kafka.Message) error {
	toolbox.PPrint(msg)
	return nil
}

func (u *eventUseCase) PublishEvent(chName string, message interface{}) error {
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message for channel %s: %v", chName, err)
		return err
	}

	if err := u.redisEventRepo.Publish(chName, jsonMessage); err != nil {
		log.Printf("Error publishing to Redis for channel %s: %v", chName, err)
		return err
	}

	if err := u.kafkaEventRepo.Publish(chName, message); err != nil {
		log.Printf("Error publishing to Kafka for channel %s: %v", chName, err)
		return err
	}

	return nil
}
