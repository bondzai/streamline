package repositories

import (
	"context"
	"streamline-sse/pkg/redis"
)

type (
	RedisEventRepository interface {
		Publish(channel string, message interface{}) error
		Subscribe(ctx context.Context, channel string) (<-chan *redis.Message, error)
	}

	redisEventRepository struct {
		client redis.Client
	}
)

func NewRedisEventRepository(client redis.Client) RedisEventRepository {
	return &redisEventRepository{
		client: client,
	}
}

func (r *redisEventRepository) Publish(channel string, message interface{}) error {
	return r.client.Publish(channel, message)
}

func (r *redisEventRepository) Subscribe(ctx context.Context, channel string) (<-chan *redis.Message, error) {
	return r.client.Subscribe(ctx, channel)
}
