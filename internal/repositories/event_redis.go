package repositories

import (
	"context"
	"log"

	"github.com/go-redis/redis/v8"
)

type RedisRepository interface {
	Publish(channel, message string) error
	Subscribe(channel string) (<-chan *redis.Message, error)
}
type redisRepository struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisRepository(addr, username, password string, db int) (RedisRepository, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Username: username,
		Password: password,
		DB:       db,
	})

	ctx := context.Background()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	log.Println("connect to redis successfully.")

	return &redisRepository{
		client: rdb,
		ctx:    ctx,
	}, nil
}

func (r *redisRepository) Publish(channel, message string) error {
	return r.client.Publish(r.ctx, channel, message).Err()
}

func (r *redisRepository) Subscribe(channel string) (<-chan *redis.Message, error) {
	pubsub := r.client.Subscribe(r.ctx, channel)
	_, err := pubsub.Receive(r.ctx)
	if err != nil {
		return nil, err
	}

	return pubsub.Channel(), nil
}
