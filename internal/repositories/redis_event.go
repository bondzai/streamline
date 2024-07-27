package repositories

import (
	"context"
	"sse-server/pkg/redis"
)

type (
	EventRepository interface {
		Publish(channel string, message interface{}) error
		Subscribe(ctx context.Context, channel string) (<-chan *redis.Message, error)
	}

	eventRepository struct {
		client redis.Client
	}
)

func NewEventRepository(client redis.Client) EventRepository {
	return &eventRepository{
		client: client,
	}
}

func (r *eventRepository) Publish(channel string, message interface{}) error {
	return r.client.Publish(channel, message)
}

func (r *eventRepository) Subscribe(ctx context.Context, channel string) (<-chan *redis.Message, error) {
	return r.client.Subscribe(ctx, channel)
}
