package repositories

import (
	"sse-server/pkg/redis"
)

type RedisRepository interface {
	Publish(channel, message string) error
	Subscribe(channel string) (<-chan *redis.Message, error)
}

type redisRepository struct {
	client *redis.Client
}

func NewRedisRepository(client *redis.Client) RedisRepository {
	return &redisRepository{
		client: client,
	}
}

func (r *redisRepository) Publish(channel, message string) error {
	return r.client.Publish(channel, message)
}

func (r *redisRepository) Subscribe(channel string) (<-chan *redis.Message, error) {
	return r.client.Subscribe(channel)
}
