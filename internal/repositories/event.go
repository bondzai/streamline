package repositories

import (
	"encoding/json"
	"sse-server/pkg/redis"
)

type EventRepository interface {
	Publish(channel string, message interface{}) error
	Subscribe(channel string) (<-chan *redis.Message, error)
}

type eventRepository struct {
	client redis.Client
}

func NewEventRepository(client redis.Client) EventRepository {
	return &eventRepository{
		client: client,
	}
}

func (r *eventRepository) Publish(channel string, message interface{}) error {
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return r.client.Publish(channel, jsonMessage)
}

func (r *eventRepository) Subscribe(channel string) (<-chan *redis.Message, error) {
	return r.client.Subscribe(channel)
}
