package usecases

import (
	"context"
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
	return u.eventRepo.Publish(channel, message)
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

		events <- entities.Event{
			Id:           channel,
			LoginSession: nil,
			DeviceId:     nil,
		}

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
				events <- entities.Event{
					Id:           channel,
					LoginSession: &msg.Payload,
					DeviceId:     &msg.Payload,
				}
			}
		}
	}()
}
