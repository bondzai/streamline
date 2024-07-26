package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/IBM/sarama"
)

const (
	Timeout          = 1
	OffsetFromLatest = iota
	OffsetFromEarliest
)

type (
	Client interface {
		Publish(topic string, message interface{}) error
		Subscribe(topics []string, offsetOption int, consumerGroup string) (<-chan *Message, error)
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

	log.Println("Connected to Kafka successfully.")

	return &client{
		producer: producer,
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

func newConsumerGroup(config Config, group string, offsetOption int) (sarama.ConsumerGroup, error) {
	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Consumer.Return.Errors = true

	switch offsetOption {
	case OffsetFromLatest:
		kafkaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	case OffsetFromEarliest:
		kafkaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	}

	if config.Username != "" && config.Password != "" {
		kafkaConfig.Net.SASL.Enable = true
		kafkaConfig.Net.SASL.User = config.Username
		kafkaConfig.Net.SASL.Password = config.Password
		kafkaConfig.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	}

	if config.UseTLS {
		kafkaConfig.Net.TLS.Enable = true
	}

	consumerGroup, err := sarama.NewConsumerGroup(config.Brokers, group, kafkaConfig)
	if err != nil {
		return nil, err
	}

	return consumerGroup, nil
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

func (r *client) Subscribe(topics []string, offsetOption int, consumerGroup string) (<-chan *Message, error) {
	messages := make(chan *Message)
	ctx := context.Background()

	consumerGroupClient, err := newConsumerGroup(Config{Brokers: r.brokers}, consumerGroup, offsetOption)
	if err != nil {
		return nil, err
	}

	consumer := &consumerGroupHandler{
		messages:     messages,
		offsetOption: offsetOption,
	}

	go func() {
		for {
			if err := consumerGroupClient.Consume(ctx, topics, consumer); err != nil {
				log.Printf("Error from consumer: %v", err)
			}

			if ctx.Err() != nil {
				return
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
