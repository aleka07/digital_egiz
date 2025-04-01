package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/digital-egiz/backend/internal/api"
	"github.com/digital-egiz/backend/internal/config"
	"github.com/digital-egiz/backend/internal/db"
	"github.com/digital-egiz/backend/internal/kafka"
	"github.com/digital-egiz/backend/internal/services"
	"github.com/digital-egiz/backend/internal/utils"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, err := utils.NewLogger(&config.LogConfig{Level: "info"})
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting Digital Egiz backend service")

	// Load configuration
	cfg, err := config.LoadConfig("")
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Print startup configuration (excluding sensitive data)
	logger.Info("Configuration loaded",
		zap.String("environment", cfg.Server.Environment),
		zap.Int("port", cfg.Server.Port),
		zap.Bool("development", cfg.Server.IsDevelopment()),
	)

	// Initialize database
	database, err := initDatabase(logger, cfg)
	if err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}
	defer closeDatabase(logger, database)

	// Check database connectivity
	if err := checkDatabaseConnection(database); err != nil {
		logger.Fatal("Database connectivity check failed", zap.Error(err))
	}
	logger.Info("Database connection established successfully")

	// Initialize Kafka
	kafkaManager, err := initKafka(logger, cfg)
	if err != nil {
		logger.Fatal("Failed to initialize Kafka", zap.Error(err))
	}
	defer closeKafka(logger, kafkaManager)

	// Initialize service provider
	serviceProvider := initServiceProvider(logger, database, kafkaManager, cfg)

	// Initialize API router
	router := api.NewRouter(cfg, logger, database, serviceProvider)
	router.SetupRoutes()

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router.GetEngine(),
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting HTTP server", zap.Int("port", cfg.Server.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	// Graceful shutdown
	gracefulShutdown(logger, server, kafkaManager)
}

// initDatabase initializes the database connection
func initDatabase(logger *utils.Logger, cfg *config.Config) (*db.Database, error) {
	logger.Info("Initializing database connection",
		zap.String("host", cfg.Database.Host),
		zap.Int("port", cfg.Database.Port),
		zap.String("name", cfg.Database.Name),
		zap.String("user", cfg.Database.User),
	)

	database, err := db.NewDatabase(cfg.Database, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return database, nil
}

// closeDatabase closes the database connection
func closeDatabase(logger *utils.Logger, database *db.Database) {
	logger.Info("Closing database connection")
	if err := database.Close(); err != nil {
		logger.Error("Error closing database connection", zap.Error(err))
	}
}

// checkDatabaseConnection verifies the database connection works
func checkDatabaseConnection(database *db.Database) error {
	sqlDB, err := database.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

// initKafka initializes the Kafka connection
func initKafka(logger *utils.Logger, cfg *config.Config) (*kafka.Manager, error) {
	logger.Info("Initializing Kafka connection",
		zap.Strings("brokers", cfg.Kafka.Brokers),
		zap.String("consumerGroup", cfg.Kafka.ConsumerGroup),
	)

	kafkaManager, err := kafka.NewManager(cfg.Kafka, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Kafka: %w", err)
	}

	return kafkaManager, nil
}

// closeKafka closes the Kafka connection
func closeKafka(logger *utils.Logger, kafkaManager *kafka.Manager) {
	logger.Info("Closing Kafka connection")
	if err := kafkaManager.Close(); err != nil {
		logger.Error("Error closing Kafka connection", zap.Error(err))
	}
}

// initServiceProvider initializes all services
func initServiceProvider(logger *utils.Logger, database *db.Database, kafkaManager *kafka.Manager, cfg *config.Config) *services.ServiceProvider {
	logger.Info("Initializing service provider")

	// Create repository factory
	repositoryFactory := db.NewRepositoryFactory(database)

	// Create service provider
	serviceProvider := services.NewServiceProvider(
		database,
		kafkaManager,
		cfg,
		logger,
		repositoryFactory,
	)

	// Initialize Kafka handlers
	kafkaHandler := services.NewKafkaHandler(
		serviceProvider.GetTwinService(),
		serviceProvider.GetProjectService(),
		serviceProvider.GetHistoryService(),
		database,
		logger,
	)

	// Add consumer handlers
	err := kafkaManager.RegisterConsumerHandlers(kafkaHandler)
	if err != nil {
		logger.Error("Failed to register Kafka handlers", zap.Error(err))
	}

	// Start Kafka consumers
	kafkaManager.StartConsumers()

	return serviceProvider
}

// gracefulShutdown handles graceful shutdown of the server
func gracefulShutdown(logger *utils.Logger, server *http.Server, kafkaManager *kafka.Manager) {
	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	sig := <-quit
	logger.Info("Received shutdown signal", zap.String("signal", sig.String()))

	// Create a deadline for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First stop Kafka consumers to prevent processing new messages
	logger.Info("Stopping Kafka consumers")
	kafkaManager.StopConsumers()

	// Then shutdown the server
	logger.Info("Shutting down HTTP server")
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("HTTP server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server gracefully stopped")
}
