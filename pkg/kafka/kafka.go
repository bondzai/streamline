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
		Brokers  []string
		Username string
		Password string
		UseTLS   bool
	}
)

func NewClient(config Config) (Client, error) {
	producer, err := newProducer(config)
	if err != nil {
		return nil, err
	}

	consumer, err := newConsumer(config)
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

func newProducer(config Config) (sarama.SyncProducer, error) {
	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Producer.Return.Successes = true

	if config.Username != "" && config.Password != "" {
		kafkaConfig.Net.SASL.Enable = true
		kafkaConfig.Net.SASL.User = config.Username
		kafkaConfig.Net.SASL.Password = config.Password
		kafkaConfig.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	}

	if config.UseTLS {
		kafkaConfig.Net.TLS.Enable = true
	}

	producer, err := sarama.NewSyncProducer(config.Brokers, kafkaConfig)
	if err != nil {
		return nil, err
	}
	return producer, nil
}

func newConsumer(config Config) (sarama.Consumer, error) {
	kafkaConfig := sarama.NewConfig()

	if config.Username != "" && config.Password != "" {
		kafkaConfig.Net.SASL.Enable = true
		kafkaConfig.Net.SASL.User = config.Username
		kafkaConfig.Net.SASL.Password = config.Password
		kafkaConfig.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	}

	if config.UseTLS {
		kafkaConfig.Net.TLS.Enable = true
	}

	consumer, err := sarama.NewConsumer(config.Brokers, kafkaConfig)
	if err != nil {
		return nil, err
	}
	return consumer, nil
}

func (r *client) IsConnected() bool {
	if r.producer == nil || r.consumer == nil {
		return false
	}
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
