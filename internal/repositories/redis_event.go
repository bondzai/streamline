package repositories

import (
	"context"

	"streamline/pkg/redis"
)

type (
	RedisEventRepository interface {
		Publish(chID string, message interface{}) error
		Subscribe(ctx context.Context, chID string) (<-chan *redis.Message, error)
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

func (r *redisEventRepository) Publish(chID string, message interface{}) error {
	return r.client.Publish(chID, message)
}

func (r *redisEventRepository) Subscribe(ctx context.Context, chID string) (<-chan *redis.Message, error) {
	return r.client.Subscribe(ctx, chID)
}
