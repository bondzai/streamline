package redis

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	Timeout = 1
)

type (
	Client interface {
		IsConnected() bool
		Close()

		Get(key string, value interface{}) error
		Set(key string, value interface{}) error
		SetWithExpiration(key string, value interface{}, expiration time.Duration) error
		Remove(keys ...string) error

		Publish(channel string, message interface{}) error
		Subscribe(ctx context.Context, channel string) (<-chan *Message, error)
	}

	client struct {
		client *redis.Client
	}

	Message struct {
		Channel      string
		Pattern      string
		Payload      string
		PayloadSlice []string
		Timestamp    time.Time
	}

	Config struct {
		Address  string
		Username string
		Password string
		DB       int
	}
)

func NewClient(config Config) (Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout*time.Second)
	defer cancel()

	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Address,
		Username: config.Username,
		Password: config.Password,
		DB:       config.DB,
	})

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	log.Println("Connected to Redis successfully.")

	return &client{
		client: rdb,
	}, nil
}

func (r *client) IsConnected() bool {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout*time.Second)
	defer cancel()

	if r.client == nil {
		return false
	}

	pong, err := r.client.Ping(ctx).Result()
	log.Fatal(pong, err)
	return err == nil
}

func (r *client) Close() {
	if err := r.client.Close(); err != nil {
		log.Println(err)
		return
	}

	log.Println("Disconnected from Redis.")
}

func (r *client) Publish(channel string, message interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout*time.Second)
	defer cancel()

	return r.client.Publish(ctx, channel, message).Err()
}

func (r *client) Subscribe(ctx context.Context, channel string) (<-chan *Message, error) {
	pubsub := r.client.Subscribe(ctx, channel)
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return nil, err
	}

	ch := make(chan *Message)
	go func() {
		defer close(ch)
		defer pubsub.Close()

		for {
			select {
			case msg := <-pubsub.Channel():
				ch <- &Message{
					Channel:      msg.Channel,
					Pattern:      msg.Pattern,
					Payload:      msg.Payload,
					PayloadSlice: msg.PayloadSlice,
					Timestamp:    time.Now(),
				}

			case <-ctx.Done():
				log.Println("Redis pub/sub channel stopped")
				return
			}
		}
	}()

	return ch, nil
}

func (r *client) Get(key string, value interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout*time.Second)
	defer cancel()

	strValue, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(strValue), value)
	if err != nil {
		return err
	}

	return nil
}

func (r *client) Set(key string, value interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout*time.Second)
	defer cancel()

	bData, _ := json.Marshal(value)
	err := r.client.Set(ctx, key, bData, 0).Err()
	if err != nil {
		return err
	}

	return nil
}

func (r *client) SetWithExpiration(key string, value interface{}, expiration time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout*time.Second)
	defer cancel()

	bData, _ := json.Marshal(value)
	err := r.client.Set(ctx, key, bData, expiration).Err()
	if err != nil {
		return err
	}

	return nil
}

func (r *client) Remove(keys ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout*time.Second)
	defer cancel()

	err := r.client.Del(ctx, keys...).Err()
	if err != nil {
		return err
	}

	return nil
}

func (r *client) Keys(pattern string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout*time.Second)
	defer cancel()

	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	return keys, nil
}

func (r *client) RemovePattern(pattern string) error {
	keys, err := r.Keys(pattern)
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	err = r.Remove(keys...)
	if err != nil {
		return err
	}

	return nil
}
