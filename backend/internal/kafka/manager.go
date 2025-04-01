package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/digital-egiz/backend/internal/config"
	"github.com/digital-egiz/backend/internal/utils"
	"go.uber.org/zap"
)

// Topic constants for the application
const (
	TopicDittoEvents      = "ditto-events"
	TopicTimeSeriesData   = "timeseries-data"
	TopicMLInput          = "ml-input"
	TopicMLOutput         = "ml-output"
	TopicDigitalTwinState = "twin-state"
	TopicAlerts           = "alerts"
)

// Manager coordinates Kafka producers and consumers
type Manager struct {
	config           *config.KafkaConfig
	logger           *utils.Logger
	mainProducer     *Producer
	dlqProducer      *Producer
	consumers        map[string]*Consumer
	consumerCtx      context.Context
	consumerCancel   context.CancelFunc
	wg               sync.WaitGroup
	mu               sync.Mutex
	isRunning        bool
	messageProcessed chan struct{}
}

// NewManager creates a new Kafka manager
func NewManager(cfg *config.KafkaConfig, logger *utils.Logger) (*Manager, error) {
	kafkaLogger := logger.Named("kafka_manager")

	// Create main producer
	mainProducer, err := NewProducer(cfg, kafkaLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create main producer: %w", err)
	}

	// Create DLQ producer
	dlqProducer, err := NewProducer(cfg, kafkaLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create DLQ producer: %w", err)
	}

	// Create context for consumers
	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		config:           cfg,
		logger:           kafkaLogger,
		mainProducer:     mainProducer,
		dlqProducer:      dlqProducer,
		consumers:        make(map[string]*Consumer),
		consumerCtx:      ctx,
		consumerCancel:   cancel,
		messageProcessed: make(chan struct{}, 100), // Buffer for processing signals
		isRunning:        false,
	}, nil
}

// Start initializes and starts all registered consumers
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return fmt.Errorf("kafka manager is already running")
	}

	// Start consumers
	for name, consumer := range m.consumers {
		m.logger.Info("Starting consumer", zap.String("name", name))
		if err := consumer.Start(m.consumerCtx); err != nil {
			m.logger.Error("Failed to start consumer",
				zap.String("name", name),
				zap.Error(err))
			// Stop any consumers that were already started
			m.stopAllConsumers()
			return fmt.Errorf("failed to start consumer %s: %w", name, err)
		}
	}

	// Start message processed monitor
	m.wg.Add(1)
	go m.monitorProcessing()

	m.isRunning = true
	m.logger.Info("Kafka manager started")
	return nil
}

// AddConsumer creates and registers a consumer with specific handlers
func (m *Manager) AddConsumer(name string, topics []string, handlers map[string][]MessageHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return fmt.Errorf("cannot add consumer while manager is running")
	}

	// Check if consumer with this name already exists
	if _, exists := m.consumers[name]; exists {
		return fmt.Errorf("consumer with name %s already exists", name)
	}

	// Create consumer
	consumer, err := NewConsumer(m.config, m.logger, m.dlqProducer)
	if err != nil {
		return fmt.Errorf("failed to create consumer %s: %w", name, err)
	}

	// Register handlers
	for topic, topicHandlers := range handlers {
		for _, handler := range topicHandlers {
			consumer.RegisterHandler(topic, m.wrapHandler(handler))
		}
	}

	// Store consumer
	m.consumers[name] = consumer
	m.logger.Info("Added consumer",
		zap.String("name", name),
		zap.Strings("topics", topics))

	return nil
}

// wrapHandler wraps a message handler to signal when processing is complete
func (m *Manager) wrapHandler(handler MessageHandler) MessageHandler {
	return func(msg *kafka.Message) error {
		defer func() {
			select {
			case m.messageProcessed <- struct{}{}:
				// Signal sent
			default:
				// Channel buffer full, which is fine in high throughput
			}
		}()

		return handler(msg)
	}
}

// ProduceMessage sends a message to the specified topic
func (m *Manager) ProduceMessage(topic string, key string, value interface{}, headers map[string]string) error {
	message := &Message{
		Key:       key,
		Value:     value,
		Timestamp: time.Now(),
		Headers:   headers,
	}

	return m.mainProducer.Produce(topic, message)
}

// ProduceDittoEvent publishes a Ditto event to Kafka
func (m *Manager) ProduceDittoEvent(thingID string, action string, payload interface{}) error {
	event := map[string]interface{}{
		"thingId":   thingID,
		"action":    action,
		"timestamp": time.Now().Format(time.RFC3339),
		"payload":   payload,
	}

	return m.ProduceMessage(TopicDittoEvents, thingID, event, nil)
}

// ProduceTimeSeriesData publishes time-series data to Kafka
func (m *Manager) ProduceTimeSeriesData(thingID string, featureID string, data interface{}) error {
	tsData := map[string]interface{}{
		"thingId":   thingID,
		"featureId": featureID,
		"timestamp": time.Now().Format(time.RFC3339),
		"data":      data,
	}

	return m.ProduceMessage(TopicTimeSeriesData, thingID, tsData, nil)
}

// ProduceMLInput publishes ML input data to Kafka
func (m *Manager) ProduceMLInput(modelID string, input interface{}) error {
	mlInput := map[string]interface{}{
		"modelId":   modelID,
		"timestamp": time.Now().Format(time.RFC3339),
		"input":     input,
	}

	return m.ProduceMessage(TopicMLInput, modelID, mlInput, nil)
}

// RegisterDittoEventHandler registers a handler for Ditto events
func (m *Manager) RegisterDittoEventHandler(name string, handler func(thingID, action string, payload json.RawMessage) error) error {
	msgHandler := func(msg *kafka.Message) error {
		var event struct {
			ThingID   string          `json:"thingId"`
			Action    string          `json:"action"`
			Timestamp string          `json:"timestamp"`
			Payload   json.RawMessage `json:"payload"`
		}

		if err := json.Unmarshal(msg.Value, &event); err != nil {
			return fmt.Errorf("failed to unmarshal Ditto event: %w", err)
		}

		return handler(event.ThingID, event.Action, event.Payload)
	}

	return m.AddConsumer(
		fmt.Sprintf("%s-ditto-events", name),
		[]string{TopicDittoEvents},
		map[string][]MessageHandler{
			TopicDittoEvents: {msgHandler},
		},
	)
}

// RegisterTimeSeriesDataHandler registers a handler for time-series data
func (m *Manager) RegisterTimeSeriesDataHandler(name string, handler func(thingID, featureID string, timestamp time.Time, data json.RawMessage) error) error {
	msgHandler := func(msg *kafka.Message) error {
		var tsData struct {
			ThingID   string          `json:"thingId"`
			FeatureID string          `json:"featureId"`
			Timestamp string          `json:"timestamp"`
			Data      json.RawMessage `json:"data"`
		}

		if err := json.Unmarshal(msg.Value, &tsData); err != nil {
			return fmt.Errorf("failed to unmarshal time-series data: %w", err)
		}

		timestamp, err := time.Parse(time.RFC3339, tsData.Timestamp)
		if err != nil {
			return fmt.Errorf("failed to parse timestamp: %w", err)
		}

		return handler(tsData.ThingID, tsData.FeatureID, timestamp, tsData.Data)
	}

	return m.AddConsumer(
		fmt.Sprintf("%s-timeseries-data", name),
		[]string{TopicTimeSeriesData},
		map[string][]MessageHandler{
			TopicTimeSeriesData: {msgHandler},
		},
	)
}

// RegisterMLOutputHandler registers a handler for ML output data
func (m *Manager) RegisterMLOutputHandler(name string, handler func(modelID string, timestamp time.Time, output json.RawMessage) error) error {
	msgHandler := func(msg *kafka.Message) error {
		var mlOutput struct {
			ModelID   string          `json:"modelId"`
			Timestamp string          `json:"timestamp"`
			Output    json.RawMessage `json:"output"`
		}

		if err := json.Unmarshal(msg.Value, &mlOutput); err != nil {
			return fmt.Errorf("failed to unmarshal ML output data: %w", err)
		}

		timestamp, err := time.Parse(time.RFC3339, mlOutput.Timestamp)
		if err != nil {
			return fmt.Errorf("failed to parse timestamp: %w", err)
		}

		return handler(mlOutput.ModelID, timestamp, mlOutput.Output)
	}

	return m.AddConsumer(
		fmt.Sprintf("%s-ml-output", name),
		[]string{TopicMLOutput},
		map[string][]MessageHandler{
			TopicMLOutput: {msgHandler},
		},
	)
}

// monitorProcessing tracks and logs message processing metrics
func (m *Manager) monitorProcessing() {
	defer m.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	messageCount := 0

	for {
		select {
		case <-m.consumerCtx.Done():
			m.logger.Info("Message processing monitor stopped")
			return

		case <-m.messageProcessed:
			messageCount++

		case <-ticker.C:
			if messageCount > 0 {
				m.logger.Info("Message processing statistics",
					zap.Int("processed_messages", messageCount),
					zap.String("interval", "1m"))
				messageCount = 0
			}
		}
	}
}

// stopAllConsumers stops all consumers
func (m *Manager) stopAllConsumers() {
	for name, consumer := range m.consumers {
		m.logger.Info("Stopping consumer", zap.String("name", name))
		consumer.Stop()
	}
}

// Stop stops the Kafka manager and all consumers
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return fmt.Errorf("kafka manager is not running")
	}

	// Cancel consumer context
	m.consumerCancel()

	// Stop all consumers
	m.stopAllConsumers()

	// Wait for all goroutines to finish
	m.wg.Wait()

	// Flush and close producers
	m.mainProducer.Flush(5000)
	m.mainProducer.Close()
	m.dlqProducer.Flush(5000)
	m.dlqProducer.Close()

	m.isRunning = false
	m.logger.Info("Kafka manager stopped")
	return nil
}

// IsRunning returns whether the Kafka manager is running
func (m *Manager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.isRunning
}
