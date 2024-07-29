package usecases

import (
	"context"
	"encoding/json"
	"log"

	"streamline/internal/entities"
	"streamline/internal/repositories"
	"streamline/pkg/kafka"
	"streamline/pkg/redis"

	"github.com/bondzai/gogear/toolbox"
)

const (
	consumerGroupName = "consumerGroup1"

	errCtxDone        = "Error Context canceled, stopping event stream for channel %s"
	errSubscribeRedis = "Error subscribing to Redis events for channel %s: %v"
	errSubscribeKafka = "Error subscribing to Kafka events for channel %s: %v"
	errStreamEvent    = "Error streaming events for channel %s: %v"
	errRedisClosed    = "Error Redis channel closed for channel %s"
	errKafkaClosed    = "Error Kafka topic closed for channel %s"
	errUnmarshalRedis = "Error unmarshaling Redis message for channel %s: %v"
	errProcessKafka   = "Error processing Kafka message for channel %s: %v"
	errMarshalMessage = "Error marshaling message for channel %s: %v"
	errPublishRedis   = "Error publishing to Redis for channel %s: %v"
	errPublishKafka   = "Error publishing to Kafka for channel %s: %v"
)

type (
	EventUseCase interface {
		PublishEvent(channel string, message interface{}) error
		SubscribeAndStreamEvent(ctx context.Context, chID string, eventCh chan<- entities.Event) error
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

func (u *eventUseCase) SubscribeAndStreamEvent(ctx context.Context, chID string, eventCh chan<- entities.Event) error {
	redisCh, err := u.redisEventRepo.Subscribe(ctx, chID)
	if err != nil {
		log.Printf(errSubscribeRedis, chID, err)
		return err
	}

	kafkaCh, err := u.kafkaEventRepo.Subscribe(ctx, []string{chID}, kafka.OffsetFromLatest, consumerGroupName)
	if err != nil {
		log.Printf(errSubscribeKafka, chID, err)
		return err
	}

	if err := u.streamEvent(ctx, chID, redisCh, kafkaCh, eventCh); err != nil {
		log.Printf(errStreamEvent, chID, err)
		return err
	}

	return nil
}

func (u *eventUseCase) streamEvent(
	ctx context.Context,
	chID string,
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

		var event entities.Event
		event.Id = chID
		eventCh <- event

		for {
			select {
			case <-ctx.Done():
				log.Printf(errCtxDone, chID)
				errCh <- ctx.Err()
				return

			case msg, ok := <-redisCh:
				if !ok {
					log.Printf(errRedisClosed, chID)
					errCh <- nil
					return
				}

				event, err := u.processRedisMessage(msg, event)
				if err != nil {
					log.Printf(errUnmarshalRedis, chID, err)
					errCh <- err
					return
				}
				eventCh <- *event

			case msg, ok := <-kafkaCh:
				if !ok {
					log.Printf(errKafkaClosed, chID)
					errCh <- nil
					return
				}

				if err := u.processKafkaMessage(msg); err != nil {
					log.Printf(errProcessKafka, chID, err)
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

func (u *eventUseCase) processRedisMessage(msg *redis.Message, event entities.Event) (*entities.Event, error) {
	if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
		return nil, err
	}

	return &event, nil
}

func (u *eventUseCase) processKafkaMessage(msg *kafka.Message) error {
	toolbox.PPrint(msg)
	return nil
}

func (u *eventUseCase) PublishEvent(chID string, message interface{}) error {
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		log.Printf(errMarshalMessage, chID, err)
		return err
	}

	if err := u.redisEventRepo.Publish(chID, jsonMessage); err != nil {
		log.Printf(errPublishRedis, chID, err)
		return err
	}

	if err := u.kafkaEventRepo.Publish(chID, message); err != nil {
		log.Printf(errPublishKafka, chID, err)
		return err
	}

	return nil
}
