package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/digital-egiz/backend/internal/config"
	"github.com/digital-egiz/backend/internal/ditto"
	"github.com/digital-egiz/backend/internal/utils"
	"go.uber.org/zap"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "./config", "Path to the configuration directory")
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

	// Create Ditto manager
	manager := ditto.NewManager(&cfg.Ditto, logger)

	// Create a context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-signalChan
		logger.Info(fmt.Sprintf("Received signal %s, shutting down...", sig))
		cancel()
	}()

	// Register an event handler
	manager.SetEventHandler(func(event *ditto.DittoEvent) {
		logger.Info("Received Ditto event",
			zap.String("topic", event.Topic),
			zap.String("path", event.Path),
			zap.String("thingID", event.ThingID),
			zap.String("action", event.Action))
	})

	// Connect to WebSocket
	if err := manager.Connect(); err != nil {
		logger.Error("Failed to connect to Ditto WebSocket", zap.Error(err))
		os.Exit(1)
	}
	defer manager.Disconnect()

	// Subscribe to all thing events
	if err := manager.SubscribeToThings(""); err != nil {
		logger.Error("Failed to subscribe to things", zap.Error(err))
		os.Exit(1)
	}

	// Create a test thing
	testThing := &ditto.Thing{
		ThingID:    "org.example:test-thing-" + fmt.Sprintf("%d", time.Now().Unix()),
		Definition: "org.example:TestThing:1.0.0",
		Attributes: map[string]interface{}{
			"name":        "Test Thing",
			"description": "A test thing created by the Digital EGIZ platform",
			"createdAt":   time.Now().Format(time.RFC3339),
		},
		Features: map[string]ditto.Feature{
			"status": {
				Properties: map[string]interface{}{
					"state":      "active",
					"lastUpdate": time.Now().Format(time.RFC3339),
				},
			},
		},
	}

	createdThing, err := manager.CreateThing(ctx, testThing)
	if err != nil {
		logger.Error("Failed to create thing", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("Created thing",
		zap.String("thingID", createdThing.ThingID),
		zap.Int64("revision", createdThing.Revision))

	// Wait for a moment to receive WebSocket events
	logger.Info("Waiting for WebSocket events... Press Ctrl+C to exit")
	<-ctx.Done()

	logger.Info("Shutting down")
}
