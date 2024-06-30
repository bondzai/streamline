package usecases

import (
	"context"
	"encoding/json"
	"log"
	"sse-server/internal/entities"
	"sse-server/internal/repositories"
	"sse-server/pkg/redis"
)

type (
	EventUseCase interface {
		PublishEvent(channel string, message interface{}) error
		StreamEventById(ctx context.Context, channel string, events chan<- entities.Event)
	}

	eventUseCase struct {
		eventRepo repositories.EventRepository
	}
)

func NewEventUseCase(eventRepo repositories.EventRepository) EventUseCase {
	return &eventUseCase{eventRepo: eventRepo}
}

func (u *eventUseCase) PublishEvent(channel string, message interface{}) error {
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		return err
	}

	err = u.eventRepo.Publish(channel, jsonMessage)
	if err != nil {
		log.Println("Publish event error: ", err)
		return err
	}

	return nil
}

func (u *eventUseCase) SubscribeEvent(channel string) (<-chan *redis.Message, error) {
	messageChannel, err := u.eventRepo.Subscribe(channel)
	if err != nil {
		return nil, err
	}

	return messageChannel, nil
}

func (u *eventUseCase) StreamEventById(ctx context.Context, channel string, events chan<- entities.Event) {
	msgCh, err := u.SubscribeEvent(channel)
	if err != nil {
		log.Printf("Error subscribing to Redis: %v\n", err)
		close(events)
		return
	}

	go func() {
		defer close(events)

		var event entities.Event
		events <- event

		for {
			select {
			case <-ctx.Done():
				log.Println("Stopped receiving messages from Redis.", channel)
				return

			case msg, ok := <-msgCh:
				if !ok {
					log.Println("Redis message channel closed.", channel)
					return
				}

				log.Printf("Received message from channel %s: %s", channel, msg.Payload)

				if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
					log.Printf("Error unmarshaling message from Redis: %v", err)
					continue
				}

				event.Id = channel
				events <- event
			}
		}
	}()
}
