package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/digital-egiz/backend/internal/config"
	"github.com/digital-egiz/backend/internal/kafka"
	"github.com/digital-egiz/backend/internal/utils"
	"go.uber.org/zap"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "./config", "Path to the configuration directory")
	mode := flag.String("mode", "both", "Mode to run: producer, consumer, or both")
	topic := flag.String("topic", "test-topic", "Topic to use for testing")
	messageCount := flag.Int("messages", 10, "Number of messages to produce")
	interval := flag.Int("interval", 1, "Interval between messages in seconds")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Set up logging
	logger, err := utils.NewLogger(&cfg.Log)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Create Kafka manager
	kafkaManager, err := kafka.NewManager(&cfg.Kafka, logger)
	if err != nil {
		logger.Fatal("Failed to create Kafka manager", zap.Error(err))
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-signals
		logger.Info("Received signal, shutting down", zap.String("signal", sig.String()))
		cancel()
	}()

	// Handle consumer if needed
	if *mode == "consumer" || *mode == "both" {
		// Register message handler
		err = kafkaManager.AddConsumer(
			"test-consumer",
			[]string{*topic},
			map[string][]kafka.MessageHandler{
				*topic: {func(msg *kafka.Message) error {
					keyStr := ""
					if msg.Key != nil {
						keyStr = string(msg.Key)
					}

					valueStr := ""
					if msg.Value != nil {
						// Try to convert message value to bytes
						if valueBytes, ok := msg.Value.([]byte); ok {
							valueStr = string(valueBytes)
						} else {
							valueStr = fmt.Sprintf("%v", msg.Value)
						}
					}

					logger.Info("Received message",
						zap.String("topic", *topic),
						zap.String("key", keyStr),
						zap.String("value", valueStr),
						zap.Time("timestamp", msg.Timestamp))
					return nil
				}},
			},
		)
		if err != nil {
			logger.Fatal("Failed to register message handler", zap.Error(err))
		}
	}

	// Start Kafka manager
	if err := kafkaManager.Start(); err != nil {
		logger.Fatal("Failed to start Kafka manager", zap.Error(err))
	}
	logger.Info("Kafka manager started")

	// Create wait group
	var wg sync.WaitGroup

	// Handle producer if needed
	if *mode == "producer" || *mode == "both" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			produceMessages(ctx, kafkaManager, logger, *topic, *messageCount, *interval)
		}()
	}

	// Wait for context cancellation or completion
	select {
	case <-ctx.Done():
		logger.Info("Context canceled, shutting down")
	case <-waitForCompletion(&wg):
		logger.Info("Production completed")
	}

	// Stop Kafka manager
	if err := kafkaManager.Stop(); err != nil {
		logger.Error("Failed to stop Kafka manager", zap.Error(err))
	}
	logger.Info("Kafka manager stopped")
}

// produceMessages produces test messages to the specified topic
func produceMessages(ctx context.Context, kafkaManager *kafka.Manager, logger *utils.Logger, topic string, count, interval int) {
	logger.Info("Starting message production",
		zap.String("topic", topic),
		zap.Int("count", count),
		zap.Int("interval", interval))

	for i := 0; i < count; i++ {
		select {
		case <-ctx.Done():
			logger.Info("Context canceled, stopping message production")
			return
		default:
			// Create message
			key := fmt.Sprintf("key-%d", i)
			value := map[string]interface{}{
				"message_id": i,
				"content":    fmt.Sprintf("Test message %d", i),
				"timestamp":  time.Now().Format(time.RFC3339),
			}

			// Produce message
			err := kafkaManager.ProduceMessage(topic, key, value, nil)
			if err != nil {
				logger.Error("Failed to produce message",
					zap.Int("message_id", i),
					zap.Error(err))
			} else {
				logger.Info("Produced message",
					zap.String("topic", topic),
					zap.String("key", key),
					zap.Int("message_id", i))
			}

			// Sleep for interval
			if i < count-1 && interval > 0 {
				time.Sleep(time.Duration(interval) * time.Second)
			}
		}
	}

	logger.Info("Message production completed",
		zap.String("topic", topic),
		zap.Int("count", count))
}

// waitForCompletion returns a channel that is closed when the wait group is done
func waitForCompletion(wg *sync.WaitGroup) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		wg.Wait()
		close(ch)
	}()
	return ch
}
