package services

import (
	"context"
	"fmt"

	"github.com/digital-egiz/backend/internal/config"
	"github.com/digital-egiz/backend/internal/db"
	"github.com/digital-egiz/backend/internal/db/repository"
	"github.com/digital-egiz/backend/internal/ditto"
	"github.com/digital-egiz/backend/internal/kafka"
	"github.com/digital-egiz/backend/internal/utils"
	"go.uber.org/zap"
)

// ServiceProvider manages all services for the application
type ServiceProvider struct {
	logger              *utils.Logger
	config              *config.Config
	database            *db.Database
	kafkaManager        *kafka.Manager
	dittoManager        *ditto.Manager
	kafkaHandler        *KafkaHandler
	historyService      *HistoryService
	notificationService *NotificationService
}

// NewServiceProvider creates a new service provider
func NewServiceProvider(
	logger *utils.Logger,
	config *config.Config,
	database *db.Database,
) *ServiceProvider {
	return &ServiceProvider{
		logger:   logger.Named("services"),
		config:   config,
		database: database,
	}
}

// Initialize initializes all services
func (sp *ServiceProvider) Initialize(ctx context.Context) error {
	var err error

	// Initialize Ditto manager
	sp.dittoManager = ditto.NewManager(&sp.config.Ditto, sp.logger)

	// Connect to Ditto WebSocket
	if err = sp.dittoManager.Connect(); err != nil {
		return fmt.Errorf("failed to connect to Ditto WebSocket: %w", err)
	}
	sp.logger.Info("Connected to Ditto WebSocket")

	// Initialize Kafka manager
	sp.kafkaManager, err = kafka.NewManager(&sp.config.Kafka, sp.logger)
	if err != nil {
		return fmt.Errorf("failed to create Kafka manager: %w", err)
	}

	// Create repository factory
	repoFactory := repository.NewRepositoryFactory(sp.database.DB)

	// Initialize HistoryService
	sp.historyService = NewHistoryService(sp.database, sp.logger)
	sp.logger.Info("History service initialized")

	// Initialize NotificationService
	sp.notificationService = NewNotificationService(sp.logger)
	sp.logger.Info("Notification service initialized")

	// Initialize Kafka handler
	sp.kafkaHandler = NewKafkaHandler(
		sp.logger,
		sp.kafkaManager,
		sp.dittoManager,
		sp.database,
		repoFactory,
	)

	// Initialize Kafka handler
	if err = sp.kafkaHandler.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize Kafka handler: %w", err)
	}

	// Start Kafka manager
	if err = sp.kafkaManager.Start(); err != nil {
		return fmt.Errorf("failed to start Kafka manager: %w", err)
	}
	sp.logger.Info("Kafka manager started")

	// Subscribe to all Ditto events
	if err = sp.dittoManager.SubscribeToThings(""); err != nil {
		return fmt.Errorf("failed to subscribe to Ditto events: %w", err)
	}
	sp.logger.Info("Subscribed to Ditto events")

	sp.logger.Info("All services initialized successfully")
	return nil
}

// Shutdown performs a graceful shutdown of all services
func (sp *ServiceProvider) Shutdown() error {
	sp.logger.Info("Shutting down services")

	// Stop Kafka manager if initialized
	if sp.kafkaManager != nil && sp.kafkaManager.IsRunning() {
		sp.logger.Info("Stopping Kafka manager")
		if err := sp.kafkaManager.Stop(); err != nil {
			sp.logger.Error("Failed to stop Kafka manager", zap.Error(err))
		}
	}

	// Disconnect from Ditto WebSocket if connected
	if sp.dittoManager != nil && sp.dittoManager.IsConnected() {
		sp.logger.Info("Disconnecting from Ditto WebSocket")
		if err := sp.dittoManager.Disconnect(); err != nil {
			sp.logger.Error("Failed to disconnect from Ditto WebSocket", zap.Error(err))
		}
	}

	sp.logger.Info("Services shut down successfully")
	return nil
}

// GetDittoManager returns the Ditto manager
func (sp *ServiceProvider) GetDittoManager() *ditto.Manager {
	return sp.dittoManager
}

// GetKafkaManager returns the Kafka manager
func (sp *ServiceProvider) GetKafkaManager() *kafka.Manager {
	return sp.kafkaManager
}

// GetKafkaHandler returns the Kafka handler
func (sp *ServiceProvider) GetKafkaHandler() *KafkaHandler {
	return sp.kafkaHandler
}

// GetHistoryService returns the history service
func (sp *ServiceProvider) GetHistoryService() *HistoryService {
	return sp.historyService
}

// GetNotificationService returns the notification service
func (sp *ServiceProvider) GetNotificationService() *NotificationService {
	return sp.notificationService
}
