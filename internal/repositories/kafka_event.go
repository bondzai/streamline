package repositories

import (
	"context"
	"sse-server/pkg/kafka"
)

type (
	KafkaEventRepository interface {
		Publish(topic string, message interface{}) error
		Subscribe(ctx context.Context, topic []string, offsetOption int, consumerGroup string) (<-chan *kafka.Message, error)
	}

	kafkaEventRepository struct {
		client kafka.Client
	}
)

func NewKafkaEventRepository(client kafka.Client) KafkaEventRepository {
	return &kafkaEventRepository{
		client: client,
	}
}
func (r *kafkaEventRepository) Publish(topic string, message interface{}) error {
	return r.client.Publish(topic, message)
}

func (r *kafkaEventRepository) Subscribe(ctx context.Context, topic []string, offsetOption int, consumerGroup string) (<-chan *kafka.Message, error) {
	return r.client.Subscribe(ctx, topic, offsetOption, consumerGroup)
}
