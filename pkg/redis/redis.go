package redis

import (
	"context"
	"log"

	"github.com/go-redis/redis/v8"
)

type (
	Client struct {
		client *redis.Client
		ctx    context.Context
	}

	Message = redis.Message

	Config struct {
		Address  string
		Username string
		Password string
		DB       int
	}
)

func NewClient(config Config) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Address,
		Username: config.Username,
		Password: config.Password,
		DB:       config.DB,
	})

	ctx := context.Background()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	log.Println("Connect to redis successfully.")

	return &Client{
		client: rdb,
		ctx:    ctx,
	}, nil
}

func (c *Client) Publish(channel string, message interface{}) error {
	return c.client.Publish(c.ctx, channel, message).Err()
}

func (c *Client) Subscribe(channel string) (<-chan *redis.Message, error) {
	pubsub := c.client.Subscribe(c.ctx, channel)
	_, err := pubsub.Receive(c.ctx)
	if err != nil {
		return nil, err
	}

	return pubsub.Channel(), nil
}
