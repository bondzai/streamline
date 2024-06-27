package usecases

import (
	"context"
	"log"
	"sse-server/internal/entities"
	"sse-server/internal/repositories"
)

type (
	EventUseCase interface {
		PublishEvent(eventId, data string) error
		StreamEventById(ctx context.Context, eventId string, events chan<- entities.Event)
	}

	eventUseCase struct {
		eventRepo repositories.RedisRepository
	}
)

func NewEventUseCase(eventRepo repositories.RedisRepository) EventUseCase {
	return &eventUseCase{eventRepo: eventRepo}
}

func (u *eventUseCase) PublishEvent(eventId, data string) error {
	return u.eventRepo.Publish(eventId, data)
}

func (u *eventUseCase) StreamEventById(ctx context.Context, eventId string, events chan<- entities.Event) {
	msgCh, err := u.eventRepo.Subscribe(eventId)
	if err != nil {
		log.Printf("Error subscribing to Redis: %v\n", err)
		close(events)
		return
	}

	go func() {
		defer close(events)

		events <- entities.Event{
			Id:           eventId,
			LoginSession: nil,
		}

		for {
			select {
			case <-ctx.Done():
				log.Println("Stopped receiving messages from Redis.", eventId)
				return

			case msg, ok := <-msgCh:
				if !ok {
					log.Println("Redis message channel closed.", eventId)
					return
				}

				log.Printf("Received message from channel %s: %s", eventId, msg.Payload)
				events <- entities.Event{
					Id:           eventId,
					LoginSession: &msg.Payload,
				}
			}
		}
	}()
}
