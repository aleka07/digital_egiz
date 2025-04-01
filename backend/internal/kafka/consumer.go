package kafka

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/digital-egiz/backend/internal/config"
	"github.com/digital-egiz/backend/internal/utils"
	"go.uber.org/zap"
)

// MessageHandler is a function that processes a Kafka message
type MessageHandler func(msg *kafka.Message) error

// Consumer provides functionality to consume messages from Kafka topics
type Consumer struct {
	consumer       *kafka.Consumer
	logger         *utils.Logger
	config         *config.KafkaConfig
	handlers       map[string][]MessageHandler
	dlqProducer    *Producer
	signalChannel  chan os.Signal
	stopChannel    chan struct{}
	runningChannel chan struct{}
	isRunning      bool
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(cfg *config.KafkaConfig, logger *utils.Logger, dlqProducer *Producer) (*Consumer, error) {
	kafkaLogger := logger.Named("kafka_consumer")

	// Create Kafka configuration
	kafkaConfig := &kafka.ConfigMap{
		"bootstrap.servers":       cfg.Brokers,
		"group.id":                cfg.ConsumerGroup,
		"auto.offset.reset":       "earliest",
		"enable.auto.commit":      true,
		"auto.commit.interval.ms": 5000,
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

	// Create Kafka consumer
	consumer, err := kafka.NewConsumer(kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	return &Consumer{
		consumer:       consumer,
		logger:         kafkaLogger,
		config:         cfg,
		handlers:       make(map[string][]MessageHandler),
		dlqProducer:    dlqProducer,
		signalChannel:  make(chan os.Signal, 1),
		stopChannel:    make(chan struct{}),
		runningChannel: make(chan struct{}),
		isRunning:      false,
	}, nil
}

// RegisterHandler registers a message handler for a specific topic
func (c *Consumer) RegisterHandler(topic string, handler MessageHandler) {
	if handlers, ok := c.handlers[topic]; ok {
		c.handlers[topic] = append(handlers, handler)
	} else {
		c.handlers[topic] = []MessageHandler{handler}
	}

	c.logger.Info("Registered handler for topic", zap.String("topic", topic))
}

// Start starts consuming messages from registered topics
func (c *Consumer) Start(ctx context.Context) error {
	if c.isRunning {
		return fmt.Errorf("consumer is already running")
	}

	// Get topics from handlers
	topics := make([]string, 0, len(c.handlers))
	for topic := range c.handlers {
		topics = append(topics, topic)
	}

	if len(topics) == 0 {
		return fmt.Errorf("no topics registered")
	}

	// Subscribe to topics
	if err := c.consumer.SubscribeTopics(topics, nil); err != nil {
		return fmt.Errorf("failed to subscribe to topics: %w", err)
	}

	c.logger.Info("Subscribed to topics", zap.Strings("topics", topics))

	// Set up signal handling
	signal.Notify(c.signalChannel, syscall.SIGINT, syscall.SIGTERM)

	// Mark consumer as running
	c.isRunning = true

	// Start consumer loop in a goroutine
	go c.consumeLoop(ctx)

	return nil
}

// consumeLoop runs the main consumption loop
func (c *Consumer) consumeLoop(ctx context.Context) {
	defer close(c.runningChannel)

	c.logger.Info("Starting Kafka consumer loop")

	// Notify that consumer is running
	c.runningChannel <- struct{}{}

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Context canceled, stopping consumer")
			c.isRunning = false
			_ = c.consumer.Close()
			return

		case <-c.stopChannel:
			c.logger.Info("Received stop signal, stopping consumer")
			c.isRunning = false
			_ = c.consumer.Close()
			return

		case sig := <-c.signalChannel:
			c.logger.Info("Received signal, stopping consumer", zap.String("signal", sig.String()))
			c.isRunning = false
			_ = c.consumer.Close()
			return

		default:
			// Poll for messages
			msg, err := c.consumer.ReadMessage(100 * time.Millisecond)
			if err != nil {
				// Ignore timeout errors
				if err.(kafka.Error).Code() == kafka.ErrTimedOut {
					continue
				}

				c.logger.Error("Error reading message from Kafka", zap.Error(err))
				continue
			}

			// Process message
			c.processMessage(msg)
		}
	}
}

// processMessage processes a Kafka message using registered handlers
func (c *Consumer) processMessage(msg *kafka.Message) {
	if msg == nil || msg.TopicPartition.Topic == nil {
		return
	}

	topic := *msg.TopicPartition.Topic
	handlers, ok := c.handlers[topic]
	if !ok || len(handlers) == 0 {
		c.logger.Warn("No handlers registered for topic", zap.String("topic", topic))
		return
	}

	c.logger.Debug("Processing message",
		zap.String("topic", topic),
		zap.Int32("partition", msg.TopicPartition.Partition),
		zap.Int64("offset", int64(msg.TopicPartition.Offset)),
		zap.Time("timestamp", msg.Timestamp),
	)

	// Process message with all registered handlers
	for i, handler := range handlers {
		if err := handler(msg); err != nil {
			c.logger.Error("Handler failed to process message",
				zap.String("topic", topic),
				zap.Int("handler_index", i),
				zap.Error(err),
			)

			// Send to DLQ if a producer is available
			if c.dlqProducer != nil {
				dlqTopic := fmt.Sprintf("%s.dlq", topic)
				headers := make(map[string]string)
				headers["error"] = err.Error()
				headers["original_topic"] = topic

				// Create DLQ message
				dlqMessage := &Message{
					Key:       string(msg.Key),
					Value:     msg.Value,
					Timestamp: time.Now(),
					Headers:   headers,
				}

				if err := c.dlqProducer.Produce(dlqTopic, dlqMessage); err != nil {
					c.logger.Error("Failed to send message to DLQ",
						zap.String("dlq_topic", dlqTopic),
						zap.Error(err),
					)
				}
			}
		}
	}
}

// Stop stops the consumer
func (c *Consumer) Stop() {
	if c.isRunning {
		close(c.stopChannel)
		<-c.runningChannel
	}
	c.logger.Info("Kafka consumer stopped")
}

// Close closes the consumer
func (c *Consumer) Close() error {
	c.Stop()
	return c.consumer.Close()
}
