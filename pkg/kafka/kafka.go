package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/IBM/sarama"
)

const (
	Timeout = 1
)

type (
	Client interface {
		IsConnected() bool
		Publish(topic string, message interface{}) error
		Subscribe(topics []string) (<-chan *sarama.ConsumerMessage, error)
	}

	client struct {
		producer sarama.SyncProducer
		consumer sarama.Consumer
		brokers  []string
	}

	Config struct {
		Brokers []string
	}
)

func NewClient(config Config) (Client, error) {
	producer, err := newProducer(config.Brokers)
	if err != nil {
		return nil, err
	}

	consumer, err := newConsumer(config.Brokers)
	if err != nil {
		return nil, err
	}

	log.Println("Connected to Kafka successfully.")

	return &client{
		producer: producer,
		consumer: consumer,
		brokers:  config.Brokers,
	}, nil
}

func newProducer(brokers []string) (sarama.SyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}
	return producer, nil
}

func newConsumer(brokers []string) (sarama.Consumer, error) {
	consumer, err := sarama.NewConsumer(brokers, nil)
	if err != nil {
		return nil, err
	}
	return consumer, nil
}

func (r *client) IsConnected() bool {
	// Check if the producer is nil
	if r.producer == nil {
		return false
	}

	// Check if the consumer is nil
	if r.consumer == nil {
		return false
	}

	// Optionally, you could send a ping message to check the connection.
	// For simplicity, we assume if producer and consumer are not nil, the connection is established.
	return true
}

func (r *client) Publish(topic string, message interface{}) error {
	_, cancel := context.WithTimeout(context.Background(), Timeout*time.Second)
	defer cancel()

	bData, err := json.Marshal(message)
	if err != nil {
		return err
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(bData),
	}

	_, _, err = r.producer.SendMessage(msg)
	return err
}

func (r *client) Subscribe(topics []string) (<-chan *sarama.ConsumerMessage, error) {
	// Create a channel to receive messages
	messages := make(chan *sarama.ConsumerMessage)

	for _, topic := range topics {
		partitions, err := r.consumer.Partitions(topic)
		if err != nil {
			return nil, err
		}

		for _, partition := range partitions {
			pc, err := r.consumer.ConsumePartition(topic, partition, sarama.OffsetNewest)
			if err != nil {
				return nil, err
			}

			go func(pc sarama.PartitionConsumer) {
				for message := range pc.Messages() {
					messages <- message
				}
			}(pc)
		}
	}

	return messages, nil
}
