package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/digital-egiz/backend/internal/db"
	"github.com/digital-egiz/backend/internal/db/models"
	"github.com/digital-egiz/backend/internal/db/repositories"
	"github.com/digital-egiz/backend/internal/ditto"
	"github.com/digital-egiz/backend/internal/kafka"
	"github.com/digital-egiz/backend/internal/utils"
	"go.uber.org/zap"
)

// KafkaHandler implements message handlers for Kafka topics
type KafkaHandler struct {
	logger           *utils.Logger
	kafkaManager     *kafka.Manager
	dittoManager     *ditto.Manager
	timeSeriesRepo   *repositories.TimeSeriesRepository
	twinRepo         *repositories.TwinRepository
	projectRepo      *repositories.ProjectRepository
	dittoEventBuffer chan *DittoEventData
}

// DittoEventData represents processed Ditto event data
type DittoEventData struct {
	ThingID   string          `json:"thingId"`
	Action    string          `json:"action"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// NewKafkaHandler creates a new Kafka message handler service
func NewKafkaHandler(
	logger *utils.Logger,
	kafkaManager *kafka.Manager,
	dittoManager *ditto.Manager,
	db *db.Database,
	repoFactory *repositories.RepositoryFactory,
) *KafkaHandler {
	return &KafkaHandler{
		logger:           logger.Named("kafka_handler"),
		kafkaManager:     kafkaManager,
		dittoManager:     dittoManager,
		timeSeriesRepo:   repoFactory.GetTimeSeriesRepository(),
		twinRepo:         repoFactory.GetTwinRepository(),
		projectRepo:      repoFactory.GetProjectRepository(),
		dittoEventBuffer: make(chan *DittoEventData, 100), // Buffer for processing Ditto events
	}
}

// Initialize sets up Kafka consumers and starts event processing
func (h *KafkaHandler) Initialize(ctx context.Context) error {
	// Register handler for Ditto events from WebSocket
	h.dittoManager.SetEventHandler(h.handleDittoWebSocketEvent)

	// Register handler for Ditto events from Kafka
	if err := h.kafkaManager.RegisterDittoEventHandler("ditto-event-processor", h.handleDittoKafkaEvent); err != nil {
		return fmt.Errorf("failed to register Ditto event handler: %w", err)
	}

	// Register handler for time-series data
	if err := h.kafkaManager.RegisterTimeSeriesDataHandler("timeseries-processor", h.handleTimeSeriesData); err != nil {
		return fmt.Errorf("failed to register time-series data handler: %w", err)
	}

	// Register handler for ML output
	if err := h.kafkaManager.RegisterMLOutputHandler("ml-output-processor", h.handleMLOutput); err != nil {
		return fmt.Errorf("failed to register ML output handler: %w", err)
	}

	// Start event buffer processor
	go h.processDittoEventBuffer(ctx)

	return nil
}

// handleDittoWebSocketEvent handles events from the Ditto WebSocket
func (h *KafkaHandler) handleDittoWebSocketEvent(event *ditto.DittoEvent) {
	h.logger.Debug("Received Ditto WebSocket event",
		zap.String("topic", event.Topic),
		zap.String("path", event.Path),
		zap.String("thingId", event.ThingID),
		zap.String("action", event.Action))

	// Forward event to Kafka for persistence and further processing
	err := h.kafkaManager.ProduceDittoEvent(event.ThingID, event.Action, event.Value)
	if err != nil {
		h.logger.Error("Failed to produce Ditto event to Kafka",
			zap.String("thingId", event.ThingID),
			zap.String("action", event.Action),
			zap.Error(err))
	}

	// Extract feature data for time-series if applicable
	if event.FeatureID != "" && event.Path != "" && event.Value != nil {
		// Path for feature properties: /features/featureId/properties
		if event.Action == "modified" || event.Action == "created" {
			// Forward feature data to time-series topic
			err := h.kafkaManager.ProduceTimeSeriesData(event.ThingID, event.FeatureID, event.Value)
			if err != nil {
				h.logger.Error("Failed to produce time-series data to Kafka",
					zap.String("thingId", event.ThingID),
					zap.String("featureId", event.FeatureID),
					zap.Error(err))
			}
		}
	}
}

// handleDittoKafkaEvent handles Ditto events from Kafka
func (h *KafkaHandler) handleDittoKafkaEvent(thingID, action string, payload json.RawMessage) error {
	h.logger.Debug("Processing Ditto event from Kafka",
		zap.String("thingId", thingID),
		zap.String("action", action))

	// Buffer event for processing
	h.dittoEventBuffer <- &DittoEventData{
		ThingID:   thingID,
		Action:    action,
		Timestamp: time.Now(),
		Payload:   payload,
	}

	return nil
}

// processDittoEventBuffer processes the buffered Ditto events
func (h *KafkaHandler) processDittoEventBuffer(ctx context.Context) {
	h.logger.Info("Starting Ditto event buffer processor")

	for {
		select {
		case <-ctx.Done():
			h.logger.Info("Stopping Ditto event buffer processor")
			return

		case event := <-h.dittoEventBuffer:
			if err := h.processEvent(ctx, event); err != nil {
				h.logger.Error("Failed to process Ditto event",
					zap.String("thingId", event.ThingID),
					zap.String("action", event.Action),
					zap.Error(err))
			}
		}
	}
}

// processEvent processes a single Ditto event
func (h *KafkaHandler) processEvent(ctx context.Context, event *DittoEventData) error {
	// Get the twin from the database
	twin, err := h.twinRepo.FindByThingID(event.ThingID)
	if err != nil {
		// If the twin doesn't exist, we may need to create it
		if event.Action == "created" {
			// Try to create the twin
			return h.handleTwinCreated(ctx, event)
		}
		return fmt.Errorf("twin not found: %w", err)
	}

	// Process the event based on the action
	switch event.Action {
	case "created":
		return h.handleTwinCreated(ctx, event)
	case "modified":
		return h.handleTwinModified(ctx, twin, event)
	case "deleted":
		return h.handleTwinDeleted(ctx, twin, event)
	default:
		return fmt.Errorf("unknown action: %s", event.Action)
	}
}

// handleTwinCreated processes a twin creation event
func (h *KafkaHandler) handleTwinCreated(ctx context.Context, event *DittoEventData) error {
	// Parse thing data
	var thingData struct {
		Attributes map[string]interface{} `json:"attributes"`
	}

	if err := json.Unmarshal(event.Payload, &thingData); err != nil {
		return fmt.Errorf("failed to unmarshal thing data: %w", err)
	}

	// Extract project ID from attributes if available
	var projectID uint
	if projectIDRaw, ok := thingData.Attributes["projectId"]; ok {
		if projectIDFloat, ok := projectIDRaw.(float64); ok {
			projectID = uint(projectIDFloat)
		}
	}

	// If project ID is not found, assign to default project
	if projectID == 0 {
		defaultProject, err := h.projectRepo.FindByName("Default")
		if err != nil {
			return fmt.Errorf("failed to find default project: %w", err)
		}
		projectID = defaultProject.ID
	}

	// Create twin record
	twin := &models.Twin{
		ThingID:     event.ThingID,
		Name:        getStringAttribute(thingData.Attributes, "name", "Unknown Twin"),
		Description: getStringAttribute(thingData.Attributes, "description", ""),
		ProjectID:   projectID,
		Status:      "active",
		CreatedAt:   event.Timestamp,
		UpdatedAt:   event.Timestamp,
	}

	if err := h.twinRepo.Create(twin); err != nil {
		return fmt.Errorf("failed to create twin: %w", err)
	}

	h.logger.Info("Created new twin in database",
		zap.String("thingId", event.ThingID),
		zap.String("name", twin.Name),
		zap.Uint("projectId", projectID))

	return nil
}

// handleTwinModified processes a twin modification event
func (h *KafkaHandler) handleTwinModified(ctx context.Context, twin *models.Twin, event *DittoEventData) error {
	// Parse thing data
	var thingData struct {
		Attributes map[string]interface{} `json:"attributes"`
	}

	if err := json.Unmarshal(event.Payload, &thingData); err != nil {
		return fmt.Errorf("failed to unmarshal thing data: %w", err)
	}

	// Update twin attributes if available
	updated := false
	if thingData.Attributes != nil {
		if name, ok := thingData.Attributes["name"].(string); ok && name != "" {
			twin.Name = name
			updated = true
		}
		if desc, ok := thingData.Attributes["description"].(string); ok {
			twin.Description = desc
			updated = true
		}
		if status, ok := thingData.Attributes["status"].(string); ok && status != "" {
			twin.Status = status
			updated = true
		}
	}

	if updated {
		twin.UpdatedAt = event.Timestamp
		if err := h.twinRepo.Update(twin); err != nil {
			return fmt.Errorf("failed to update twin: %w", err)
		}

		h.logger.Info("Updated twin in database",
			zap.String("thingId", event.ThingID),
			zap.String("name", twin.Name),
			zap.String("status", twin.Status))
	}

	return nil
}

// handleTwinDeleted processes a twin deletion event
func (h *KafkaHandler) handleTwinDeleted(ctx context.Context, twin *models.Twin, event *DittoEventData) error {
	// Soft delete the twin
	twin.Status = "deleted"
	twin.UpdatedAt = event.Timestamp
	twin.DeletedAt = &event.Timestamp

	if err := h.twinRepo.Update(twin); err != nil {
		return fmt.Errorf("failed to delete twin: %w", err)
	}

	h.logger.Info("Deleted twin in database",
		zap.String("thingId", event.ThingID),
		zap.String("name", twin.Name))

	return nil
}

// handleTimeSeriesData handles time-series data from Kafka
func (h *KafkaHandler) handleTimeSeriesData(thingID, featureID string, timestamp time.Time, data json.RawMessage) error {
	h.logger.Debug("Processing time-series data",
		zap.String("thingId", thingID),
		zap.String("featureId", featureID),
		zap.Time("timestamp", timestamp))

	// Store time-series data in TimescaleDB
	timeSeriesData := &models.TimeSeriesData{
		ThingID:    thingID,
		FeatureID:  featureID,
		Time:       timestamp,
		Data:       data,
		SensorType: featureID, // Use feature ID as sensor type for now
	}

	if err := h.timeSeriesRepo.Create(timeSeriesData); err != nil {
		return fmt.Errorf("failed to store time-series data: %w", err)
	}

	// Check if ML analysis is needed
	if isMLEnabledForFeature(featureID) {
		mlInput := map[string]interface{}{
			"thingId":   thingID,
			"featureId": featureID,
			"timestamp": timestamp,
			"data":      data,
		}

		// Forward to ML service
		err := h.kafkaManager.ProduceMLInput(featureID, mlInput)
		if err != nil {
			h.logger.Error("Failed to send data to ML service",
				zap.String("thingId", thingID),
				zap.String("featureId", featureID),
				zap.Error(err))
		}
	}

	return nil
}

// handleMLOutput handles ML output data from Kafka
func (h *KafkaHandler) handleMLOutput(modelID string, timestamp time.Time, output json.RawMessage) error {
	h.logger.Debug("Processing ML output",
		zap.String("modelId", modelID),
		zap.Time("timestamp", timestamp))

	// Parse ML output
	var mlOutput struct {
		ThingID   string          `json:"thingId"`
		FeatureID string          `json:"featureId"`
		Result    json.RawMessage `json:"result"`
		Alert     *MLAlert        `json:"alert,omitempty"`
	}

	if err := json.Unmarshal(output, &mlOutput); err != nil {
		return fmt.Errorf("failed to unmarshal ML output: %w", err)
	}

	// Store ML prediction
	prediction := &models.MLPredictionData{
		ModelID:    modelID,
		ThingID:    mlOutput.ThingID,
		FeatureID:  mlOutput.FeatureID,
		Time:       timestamp,
		Prediction: mlOutput.Result,
	}

	if err := h.timeSeriesRepo.CreateMLPrediction(prediction); err != nil {
		return fmt.Errorf("failed to store ML prediction: %w", err)
	}

	// Handle alerts if present
	if mlOutput.Alert != nil {
		alertData := &models.AlertData{
			ThingID:     mlOutput.ThingID,
			FeatureID:   mlOutput.FeatureID,
			Time:        timestamp,
			AlertType:   mlOutput.Alert.Type,
			Severity:    mlOutput.Alert.Severity,
			Description: mlOutput.Alert.Description,
			Data:        output,
		}

		if err := h.timeSeriesRepo.CreateAlert(alertData); err != nil {
			return fmt.Errorf("failed to store alert: %w", err)
		}

		// TODO: Implement notification for alerts
	}

	return nil
}

// MLAlert represents an alert generated by ML analysis
type MLAlert struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

// Helper functions

// getStringAttribute safely extracts a string attribute with a fallback
func getStringAttribute(attributes map[string]interface{}, key, fallback string) string {
	if val, ok := attributes[key].(string); ok && val != "" {
		return val
	}
	return fallback
}

// isMLEnabledForFeature checks if ML analysis is enabled for a feature
func isMLEnabledForFeature(featureID string) bool {
	// This is a placeholder - in a real implementation, this would
	// check a configuration database or other source to determine
	// if ML analysis should be applied to this feature

	// For now, we'll enable ML for certain feature types
	mlEnabledFeatures := map[string]bool{
		"temperature":  true,
		"pressure":     true,
		"vibration":    true,
		"acceleration": true,
		"flow":         true,
		"level":        true,
	}

	return mlEnabledFeatures[featureID]
}
