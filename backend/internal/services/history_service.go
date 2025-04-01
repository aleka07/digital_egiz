package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/digital-egiz/backend/internal/db"
	"github.com/digital-egiz/backend/internal/db/models"
	"github.com/digital-egiz/backend/internal/db/repository"
	"github.com/digital-egiz/backend/internal/utils"
	"go.uber.org/zap"
)

// HistoryService handles time-series data operations
type HistoryService struct {
	db             *db.Database
	logger         *utils.Logger
	timeseriesRepo repository.TimeseriesRepository
	twinRepo       repository.TwinRepository
}

// NewHistoryService creates a new history service
func NewHistoryService(db *db.Database, logger *utils.Logger) *HistoryService {
	repoFactory := repository.NewRepositoryFactory(db.DB)
	return &HistoryService{
		db:             db,
		logger:         logger.Named("history_service"),
		timeseriesRepo: repoFactory.Timeseries(),
		twinRepo:       repoFactory.Twin(),
	}
}

// GetTimeseriesData retrieves time-series data for a specific twin and feature path
func (s *HistoryService) GetTimeseriesData(twinID uint, featurePath string, start, end time.Time, limit int) ([]models.TimeseriesData, error) {
	twin, err := s.twinRepo.GetByID(twinID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errors.New("twin not found")
		}
		s.logger.Error("Failed to verify twin exists", zap.Uint("twin_id", twinID), zap.Error(err))
		return nil, errors.New("database error")
	}

	// Use the Ditto ID for time-series data lookups
	data, err := s.timeseriesRepo.GetTimeseriesData(twin.DittoID, featurePath, start, end, limit)
	if err != nil {
		s.logger.Error("Failed to get time-series data",
			zap.Uint("twin_id", twinID),
			zap.String("ditto_id", twin.DittoID),
			zap.String("feature_path", featurePath),
			zap.Error(err))
		return nil, errors.New("failed to retrieve time-series data")
	}

	return data, nil
}

// GetLatestTimeseriesData retrieves the latest time-series data for a twin and feature path
func (s *HistoryService) GetLatestTimeseriesData(twinID uint, featurePath string) (*models.TimeseriesData, error) {
	twin, err := s.twinRepo.GetByID(twinID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errors.New("twin not found")
		}
		s.logger.Error("Failed to verify twin exists", zap.Uint("twin_id", twinID), zap.Error(err))
		return nil, errors.New("database error")
	}

	data, err := s.timeseriesRepo.GetLatestTimeseriesData(twin.DittoID, featurePath)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errors.New("no data found for the given twin and feature path")
		}
		s.logger.Error("Failed to get latest time-series data",
			zap.Uint("twin_id", twinID),
			zap.String("ditto_id", twin.DittoID),
			zap.String("feature_path", featurePath),
			zap.Error(err))
		return nil, errors.New("database error")
	}

	return data, nil
}

// GetAggregatedData retrieves aggregated time-series data
func (s *HistoryService) GetAggregatedData(twinID uint, featurePath string, start, end time.Time, interval string) ([]models.AggregatedData, error) {
	twin, err := s.twinRepo.GetByID(twinID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errors.New("twin not found")
		}
		s.logger.Error("Failed to verify twin exists", zap.Uint("twin_id", twinID), zap.Error(err))
		return nil, errors.New("database error")
	}

	// Validate interval
	validIntervals := map[string]bool{
		"1m": true, "5m": true, "15m": true, "30m": true,
		"1h": true, "6h": true, "12h": true,
		"1d": true, "1w": true, "1mon": true,
	}

	if !validIntervals[interval] {
		return nil, fmt.Errorf("invalid interval: %s", interval)
	}

	data, err := s.timeseriesRepo.GetAggregatedTimeseriesData(twin.DittoID, featurePath, start, end, interval)
	if err != nil {
		s.logger.Error("Failed to get aggregated time-series data",
			zap.Uint("twin_id", twinID),
			zap.String("ditto_id", twin.DittoID),
			zap.String("feature_path", featurePath),
			zap.String("interval", interval),
			zap.Error(err))
		return nil, errors.New("failed to retrieve aggregated data")
	}

	return data, nil
}

// GetAlertData retrieves alert data for a specific twin
func (s *HistoryService) GetAlertData(twinID uint, start, end time.Time, severity string, limit int) ([]models.AlertData, error) {
	twin, err := s.twinRepo.GetByID(twinID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errors.New("twin not found")
		}
		s.logger.Error("Failed to verify twin exists", zap.Uint("twin_id", twinID), zap.Error(err))
		return nil, errors.New("database error")
	}

	// Validate severity if provided
	if severity != "" {
		validSeverities := map[string]bool{
			"info": true, "warning": true, "error": true, "critical": true,
		}

		if !validSeverities[severity] {
			return nil, fmt.Errorf("invalid severity: %s", severity)
		}
	}

	alerts, err := s.timeseriesRepo.GetAlertData(twin.DittoID, start, end, severity, limit)
	if err != nil {
		s.logger.Error("Failed to get alert data",
			zap.Uint("twin_id", twinID),
			zap.String("ditto_id", twin.DittoID),
			zap.String("severity", severity),
			zap.Error(err))
		return nil, errors.New("failed to retrieve alert data")
	}

	return alerts, nil
}

// AcknowledgeAlert acknowledges an alert
func (s *HistoryService) AcknowledgeAlert(alertID string, userID uint) error {
	// Get user email or name for ack attribution
	var ackBy string
	var user models.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		s.logger.Warn("Failed to get user for alert acknowledgment",
			zap.Uint("user_id", userID),
			zap.Error(err))
		ackBy = fmt.Sprintf("user-%d", userID)
	} else {
		if user.FirstName != "" && user.LastName != "" {
			ackBy = fmt.Sprintf("%s %s", user.FirstName, user.LastName)
		} else {
			ackBy = user.Email
		}
	}

	err := s.timeseriesRepo.AcknowledgeAlert(alertID, ackBy)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return errors.New("alert not found or already acknowledged")
		}
		s.logger.Error("Failed to acknowledge alert",
			zap.String("alert_id", alertID),
			zap.String("ack_by", ackBy),
			zap.Error(err))
		return errors.New("failed to acknowledge alert")
	}

	return nil
}

// GetMLPredictionData retrieves ML prediction data for a specific twin and task
func (s *HistoryService) GetMLPredictionData(twinID uint, taskID string, start, end time.Time, limit int) ([]models.MLPredictionData, error) {
	twin, err := s.twinRepo.GetByID(twinID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errors.New("twin not found")
		}
		s.logger.Error("Failed to verify twin exists", zap.Uint("twin_id", twinID), zap.Error(err))
		return nil, errors.New("database error")
	}

	predictions, err := s.timeseriesRepo.GetMLPredictionData(twin.DittoID, taskID, start, end, limit)
	if err != nil {
		s.logger.Error("Failed to get ML prediction data",
			zap.Uint("twin_id", twinID),
			zap.String("ditto_id", twin.DittoID),
			zap.String("task_id", taskID),
			zap.Error(err))
		return nil, errors.New("failed to retrieve ML prediction data")
	}

	return predictions, nil
}

// GetLatestMLPrediction retrieves the latest ML prediction for a twin and task
func (s *HistoryService) GetLatestMLPrediction(twinID uint, taskID string) (*models.MLPredictionData, error) {
	twin, err := s.twinRepo.GetByID(twinID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errors.New("twin not found")
		}
		s.logger.Error("Failed to verify twin exists", zap.Uint("twin_id", twinID), zap.Error(err))
		return nil, errors.New("database error")
	}

	prediction, err := s.timeseriesRepo.GetLatestMLPrediction(twin.DittoID, taskID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errors.New("no ML prediction found for the given twin and task")
		}
		s.logger.Error("Failed to get latest ML prediction",
			zap.Uint("twin_id", twinID),
			zap.String("ditto_id", twin.DittoID),
			zap.String("task_id", taskID),
			zap.Error(err))
		return nil, errors.New("database error")
	}

	return prediction, nil
}
