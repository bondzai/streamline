package kafka

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"log"
	"time"

	"github.com/IBM/sarama"
)

const (
	Timeout = 1

	OffsetFromLatest = iota
	OffsetFromEarliest
)

type (
	Client interface {
		Produce(topic string, message interface{}) error
		Consume(ctx context.Context, topics []string, offsetOption int, consumerGroup string) (<-chan *Message, error)
		Close() error
	}

	client struct {
		producer      sarama.SyncProducer
		consumerGroup sarama.ConsumerGroup
		brokers       []string
	}

	Message struct {
		Topic     string
		Partition int32
		Offset    int64
		Key       []byte
		Value     []byte
		Timestamp time.Time
	}

	Config struct {
		Brokers   []string
		Username  string
		Password  string
		UseTLS    bool
		TLSConfig *tls.Config // Optional custom TLS configuration
	}
)

func NewClient(config Config) (Client, error) {
	producer, err := newProducer(config)
	if err != nil {
		return nil, err
	}

	log.Println("Connected to Kafka successfully.")

	return &client{
		producer: producer,
		brokers:  config.Brokers,
	}, nil
}

func newSaramaConfig(config Config) *sarama.Config {
	kafkaConfig := sarama.NewConfig()

	if config.Username != "" && config.Password != "" {
		kafkaConfig.Net.SASL.Enable = true
		kafkaConfig.Net.SASL.User = config.Username
		kafkaConfig.Net.SASL.Password = config.Password
		kafkaConfig.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	}

	if config.UseTLS {
		kafkaConfig.Net.TLS.Enable = true
		kafkaConfig.Net.TLS.Config = config.TLSConfig
	}

	return kafkaConfig
}

// newProducer creates a new Kafka sync producer.
func newProducer(config Config) (sarama.SyncProducer, error) {
	kafkaConfig := newSaramaConfig(config)
	kafkaConfig.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer(config.Brokers, kafkaConfig)
	if err != nil {
		return nil, err
	}

	return producer, nil
}

// newConsumerGroup creates a new Kafka consumer group.
func newConsumerGroup(config Config, group string, offsetOption int) (sarama.ConsumerGroup, error) {
	kafkaConfig := newSaramaConfig(config)
	kafkaConfig.Consumer.Return.Errors = true

	switch offsetOption {
	case OffsetFromLatest:
		kafkaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	case OffsetFromEarliest:
		kafkaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	default:
		kafkaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	}

	consumerGroup, err := sarama.NewConsumerGroup(config.Brokers, group, kafkaConfig)
	if err != nil {
		return nil, err
	}

	return consumerGroup, nil
}

func (r *client) Produce(topic string, message interface{}) error {
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

func (r *client) Consume(ctx context.Context, topics []string, offsetOption int, consumerGroup string) (<-chan *Message, error) {
	messages := make(chan *Message)

	consumerGroupClient, err := newConsumerGroup(Config{Brokers: r.brokers}, consumerGroup, offsetOption)
	if err != nil {
		return nil, err
	}

	consumer := &consumerGroupHandler{
		messages:     messages,
		offsetOption: offsetOption,
	}

	go func() {
		defer func() {
			consumerGroupClient.Close()
			close(messages)
		}()

		for {
			select {
			case <-ctx.Done():
				return

			default:
				if err := consumerGroupClient.Consume(ctx, topics, consumer); err != nil {
					log.Printf("Error from consumer: %v", err)
					if ctx.Err() != nil {
						log.Println("context error detected, stopping consumption.")
						return
					}
				}
			}
		}
	}()

	r.consumerGroup = consumerGroupClient

	return messages, nil
}

func (r *client) Close() error {
	if err := r.producer.Close(); err != nil {
		return err
	}

	if r.consumerGroup != nil {
		if err := r.consumerGroup.Close(); err != nil {
			return err
		}
	}

	log.Println("Disconnected from Kafka.")

	return nil
}

type consumerGroupHandler struct {
	messages     chan<- *Message
	offsetOption int
}

func (h *consumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *consumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *consumerGroupHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		h.messages <- &Message{
			Topic:     msg.Topic,
			Partition: msg.Partition,
			Offset:    msg.Offset,
			Key:       msg.Key,
			Value:     msg.Value,
			Timestamp: msg.Timestamp,
		}
		sess.MarkMessage(msg, "")
	}

	return nil
}
