package repositories

import (
	"sse-server/pkg/kafka"
)

type (
	KafkaEventRepository interface {
		Publish(topic string, message interface{}) error
		Subscribe(topic []string) (<-chan *kafka.Message, error)
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

func (r *kafkaEventRepository) Subscribe(topic []string) (<-chan *kafka.Message, error) {
	return r.client.Subscribe(topic)
}
