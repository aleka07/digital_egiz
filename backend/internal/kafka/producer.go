package kafka

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/digital-egiz/backend/internal/config"
	"github.com/digital-egiz/backend/internal/utils"
	"go.uber.org/zap"
)

// Producer provides functionality to produce messages to Kafka topics
type Producer struct {
	producer *kafka.Producer
	logger   *utils.Logger
	config   *config.KafkaConfig
}

// NewProducer creates a new Kafka producer
func NewProducer(cfg *config.KafkaConfig, logger *utils.Logger) (*Producer, error) {
	kafkaLogger := logger.Named("kafka_producer")

	// Create Kafka configuration
	kafkaConfig := &kafka.ConfigMap{
		"bootstrap.servers": cfg.Brokers,
		"client.id":         "digital-egiz-producer",
		"acks":              "all",
	}

	// Add security configuration if enabled
	if cfg.SecurityEnable {
		err := kafkaConfig.SetKey("security.protocol", "SASL_SSL")
		if err != nil {
			return nil, fmt.Errorf("failed to set security protocol: %w", err)
		}

		err = kafkaConfig.SetKey("sasl.mechanisms", "PLAIN")
		if err != nil {
			return nil, fmt.Errorf("failed to set SASL mechanism: %w", err)
		}

		err = kafkaConfig.SetKey("sasl.username", cfg.SecurityUser)
		if err != nil {
			return nil, fmt.Errorf("failed to set SASL username: %w", err)
		}

		err = kafkaConfig.SetKey("sasl.password", cfg.SecurityPass)
		if err != nil {
			return nil, fmt.Errorf("failed to set SASL password: %w", err)
		}
	}

	// Create Kafka producer
	producer, err := kafka.NewProducer(kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	// Start delivery report goroutine
	go func() {
		for e := range producer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					kafkaLogger.Error("Failed to deliver message",
						zap.String("topic", *ev.TopicPartition.Topic),
						zap.Error(ev.TopicPartition.Error),
					)
				} else {
					kafkaLogger.Debug("Message delivered",
						zap.String("topic", *ev.TopicPartition.Topic),
						zap.Int32("partition", ev.TopicPartition.Partition),
						zap.Int64("offset", int64(ev.TopicPartition.Offset)),
					)
				}
			}
		}
	}()

	return &Producer{
		producer: producer,
		logger:   kafkaLogger,
		config:   cfg,
	}, nil
}

// Message represents a message to be sent to Kafka
type Message struct {
	Key       string
	Value     interface{}
	Timestamp time.Time
	Headers   map[string]string
}

// Produce sends a message to a Kafka topic
func (p *Producer) Produce(topic string, message *Message) error {
	// Marshal the message value to JSON
	valueBytes, err := json.Marshal(message.Value)
	if err != nil {
		return fmt.Errorf("failed to marshal message value: %w", err)
	}

	// Create Kafka message
	kafkaMessage := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          valueBytes,
		Timestamp:      message.Timestamp,
	}

	// Add key if provided
	if message.Key != "" {
		kafkaMessage.Key = []byte(message.Key)
	}

	// Add headers if provided
	if len(message.Headers) > 0 {
		kafkaMessage.Headers = make([]kafka.Header, 0, len(message.Headers))
		for k, v := range message.Headers {
			kafkaMessage.Headers = append(kafkaMessage.Headers, kafka.Header{
				Key:   k,
				Value: []byte(v),
			})
		}
	}

	// Produce message
	p.logger.Debug("Producing message",
		zap.String("topic", topic),
		zap.String("key", message.Key),
		zap.Time("timestamp", message.Timestamp),
	)

	if err := p.producer.Produce(kafkaMessage, nil); err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	return nil
}

// ProduceSync sends a message to a Kafka topic and waits for the delivery report
func (p *Producer) ProduceSync(topic string, message *Message) error {
	// Marshal the message value to JSON
	valueBytes, err := json.Marshal(message.Value)
	if err != nil {
		return fmt.Errorf("failed to marshal message value: %w", err)
	}

	// Create Kafka message
	kafkaMessage := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          valueBytes,
		Timestamp:      message.Timestamp,
	}

	// Add key if provided
	if message.Key != "" {
		kafkaMessage.Key = []byte(message.Key)
	}

	// Add headers if provided
	if len(message.Headers) > 0 {
		kafkaMessage.Headers = make([]kafka.Header, 0, len(message.Headers))
		for k, v := range message.Headers {
			kafkaMessage.Headers = append(kafkaMessage.Headers, kafka.Header{
				Key:   k,
				Value: []byte(v),
			})
		}
	}

	// Produce message and wait for delivery report
	p.logger.Debug("Producing message (sync)",
		zap.String("topic", topic),
		zap.String("key", message.Key),
		zap.Time("timestamp", message.Timestamp),
	)

	deliveryChan := make(chan kafka.Event)
	if err := p.producer.Produce(kafkaMessage, deliveryChan); err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	// Wait for delivery report
	e := <-deliveryChan
	m := e.(*kafka.Message)

	if m.TopicPartition.Error != nil {
		return fmt.Errorf("failed to deliver message: %w", m.TopicPartition.Error)
	}

	return nil
}

// Flush flushes the producer's message queue
func (p *Producer) Flush(timeoutMs int) int {
	return p.producer.Flush(timeoutMs)
}

// Close closes the producer and waits for any outstanding messages to be delivered
func (p *Producer) Close() {
	// Flush any remaining messages
	p.logger.Info("Flushing producer before closing")
	remaining := p.producer.Flush(5000)
	if remaining > 0 {
		p.logger.Warn("Failed to deliver all messages during flush", zap.Int("remaining", remaining))
	}

	// Close the producer
	p.producer.Close()
	p.logger.Info("Kafka producer closed")
}
