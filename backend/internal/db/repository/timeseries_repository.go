package repository

import (
	"time"

	"github.com/digital-egiz/backend/internal/db/models"
	"gorm.io/gorm"
)

// TimeseriesRepository defines operations for managing time-series data
type TimeseriesRepository interface {
	Repository
	// Timeseries data operations
	InsertTimeseriesData(data *models.TimeseriesData) error
	InsertTimeseriesBatch(data []models.TimeseriesData) error
	GetTimeseriesData(twinID string, featurePath string, start, end time.Time, limit int) ([]models.TimeseriesData, error)
	GetLatestTimeseriesData(twinID string, featurePath string) (*models.TimeseriesData, error)
	GetAggregatedTimeseriesData(twinID string, featurePath string, start, end time.Time, interval string) ([]models.AggregatedData, error)
	DeleteTimeseriesData(twinID string, featurePath string, start, end time.Time) error

	// Aggregated data operations
	InsertAggregatedData(data *models.AggregatedData) error
	InsertAggregatedBatch(data []models.AggregatedData) error

	// Alert data operations
	InsertAlertData(alert *models.AlertData) error
	GetAlertData(twinID string, start, end time.Time, severity string, limit int) ([]models.AlertData, error)
	AcknowledgeAlert(alertID string, ackBy string) error
	DeleteAlertData(alertID string) error

	// ML prediction data operations
	InsertMLPredictionData(prediction *models.MLPredictionData) error
	InsertMLPredictionBatch(predictions []models.MLPredictionData) error
	GetMLPredictionData(twinID string, taskID string, start, end time.Time, limit int) ([]models.MLPredictionData, error)
	GetLatestMLPrediction(twinID string, taskID string) (*models.MLPredictionData, error)
	DeleteMLPredictionData(twinID string, taskID string, start, end time.Time) error
}

// timeseriesRepository implements TimeseriesRepository
type timeseriesRepository struct {
	BaseRepository
}

// NewTimeseriesRepository creates a new time-series repository
func NewTimeseriesRepository(db *gorm.DB) TimeseriesRepository {
	return &timeseriesRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// InsertTimeseriesData inserts a single time-series data point
func (r *timeseriesRepository) InsertTimeseriesData(data *models.TimeseriesData) error {
	err := r.GetDB().Create(data).Error
	return r.handleError(err)
}

// InsertTimeseriesBatch inserts multiple time-series data points in a batch
func (r *timeseriesRepository) InsertTimeseriesBatch(data []models.TimeseriesData) error {
	// Use a transaction for batches to ensure atomicity
	tx := r.GetDB().Begin()
	if tx.Error != nil {
		return r.handleError(tx.Error)
	}

	// Create batch insert
	if err := tx.CreateInBatches(data, 100).Error; err != nil {
		tx.Rollback()
		return r.handleError(err)
	}

	return r.handleError(tx.Commit().Error)
}

// GetTimeseriesData retrieves time-series data for a specific twin and feature path
func (r *timeseriesRepository) GetTimeseriesData(twinID string, featurePath string, start, end time.Time, limit int) ([]models.TimeseriesData, error) {
	var data []models.TimeseriesData

	query := r.GetDB().Where("twin_id = ? AND feature_path = ? AND time >= ? AND time <= ?", twinID, featurePath, start, end)

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Order("time desc").Find(&data).Error
	if err != nil {
		return nil, r.handleError(err)
	}

	return data, nil
}

// GetLatestTimeseriesData retrieves the latest time-series data for a twin and feature path
func (r *timeseriesRepository) GetLatestTimeseriesData(twinID string, featurePath string) (*models.TimeseriesData, error) {
	var data models.TimeseriesData
	err := r.GetDB().Where("twin_id = ? AND feature_path = ?", twinID, featurePath).
		Order("time desc").
		Limit(1).
		First(&data).Error

	if err != nil {
		return nil, r.handleError(err)
	}

	return &data, nil
}

// GetAggregatedTimeseriesData retrieves aggregated time-series data
func (r *timeseriesRepository) GetAggregatedTimeseriesData(twinID string, featurePath string, start, end time.Time, interval string) ([]models.AggregatedData, error) {
	var data []models.AggregatedData

	intervalTypeMap := map[string]string{
		"1m":   "minute",
		"5m":   "minute",
		"15m":  "minute",
		"30m":  "minute",
		"1h":   "hour",
		"6h":   "hour",
		"12h":  "hour",
		"1d":   "day",
		"1w":   "week",
		"1mon": "month",
	}

	intervalType, ok := intervalTypeMap[interval]
	if !ok {
		intervalType = "hour" // Default to hourly aggregation
	}

	// Check if aggregated data exists
	query := r.GetDB().Where("twin_id = ? AND feature_path = ? AND time_interval >= ? AND time_interval <= ? AND interval_type = ?",
		twinID, featurePath, start, end, intervalType)

	err := query.Order("time_interval desc").Find(&data).Error
	if err != nil {
		return nil, r.handleError(err)
	}

	if len(data) > 0 {
		return data, nil
	}

	// If no aggregated data, generate it on-the-fly (using TimescaleDB time_bucket function)
	query = r.GetDB().Raw(`
		SELECT 
			time_bucket(?::interval, time) as time_interval,
			? as twin_id,
			? as feature_path,
			? as interval_type,
			MIN(value_num) as min,
			MAX(value_num) as max,
			AVG(value_num) as avg,
			SUM(value_num) as sum,
			COUNT(*) as count,
			MIN(time) as first_time,
			MAX(time) as last_time
		FROM timeseries_data
		WHERE twin_id = ? AND feature_path = ? AND time >= ? AND time <= ? AND value_type = 'number'
		GROUP BY time_bucket(?::interval, time)
		ORDER BY time_interval DESC
	`, interval, twinID, featurePath, intervalType, twinID, featurePath, start, end, interval)

	err = query.Scan(&data).Error
	if err != nil {
		return nil, r.handleError(err)
	}

	return data, nil
}

// DeleteTimeseriesData deletes time-series data for a twin and feature path within a time range
func (r *timeseriesRepository) DeleteTimeseriesData(twinID string, featurePath string, start, end time.Time) error {
	result := r.GetDB().Where("twin_id = ? AND feature_path = ? AND time >= ? AND time <= ?",
		twinID, featurePath, start, end).
		Delete(&models.TimeseriesData{})

	return r.handleError(result.Error)
}

// InsertAggregatedData inserts a single aggregated data point
func (r *timeseriesRepository) InsertAggregatedData(data *models.AggregatedData) error {
	err := r.GetDB().Create(data).Error
	return r.handleError(err)
}

// InsertAggregatedBatch inserts multiple aggregated data points in a batch
func (r *timeseriesRepository) InsertAggregatedBatch(data []models.AggregatedData) error {
	// Use a transaction for batches to ensure atomicity
	tx := r.GetDB().Begin()
	if tx.Error != nil {
		return r.handleError(tx.Error)
	}

	// Create batch insert
	if err := tx.CreateInBatches(data, 100).Error; err != nil {
		tx.Rollback()
		return r.handleError(err)
	}

	return r.handleError(tx.Commit().Error)
}

// InsertAlertData inserts alert data
func (r *timeseriesRepository) InsertAlertData(alert *models.AlertData) error {
	err := r.GetDB().Create(alert).Error
	return r.handleError(err)
}

// GetAlertData retrieves alert data for a specific twin
func (r *timeseriesRepository) GetAlertData(twinID string, start, end time.Time, severity string, limit int) ([]models.AlertData, error) {
	var alerts []models.AlertData

	query := r.GetDB().Where("twin_id = ? AND time >= ? AND time <= ?", twinID, start, end)

	if severity != "" {
		query = query.Where("severity = ?", severity)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Order("time desc").Find(&alerts).Error
	if err != nil {
		return nil, r.handleError(err)
	}

	return alerts, nil
}

// AcknowledgeAlert acknowledges an alert
func (r *timeseriesRepository) AcknowledgeAlert(alertID string, ackBy string) error {
	result := r.GetDB().Model(&models.AlertData{}).
		Where("alert_id = ? AND acknowledged = false", alertID).
		Updates(map[string]interface{}{
			"acknowledged": true,
			"ack_by":       ackBy,
			"ack_time":     time.Now(),
		})

	if result.Error != nil {
		return r.handleError(result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// DeleteAlertData deletes an alert
func (r *timeseriesRepository) DeleteAlertData(alertID string) error {
	result := r.GetDB().Where("alert_id = ?", alertID).Delete(&models.AlertData{})

	if result.Error != nil {
		return r.handleError(result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// InsertMLPredictionData inserts ML prediction data
func (r *timeseriesRepository) InsertMLPredictionData(prediction *models.MLPredictionData) error {
	err := r.GetDB().Create(prediction).Error
	return r.handleError(err)
}

// InsertMLPredictionBatch inserts multiple ML prediction data points in a batch
func (r *timeseriesRepository) InsertMLPredictionBatch(predictions []models.MLPredictionData) error {
	// Use a transaction for batches to ensure atomicity
	tx := r.GetDB().Begin()
	if tx.Error != nil {
		return r.handleError(tx.Error)
	}

	// Create batch insert
	if err := tx.CreateInBatches(predictions, 100).Error; err != nil {
		tx.Rollback()
		return r.handleError(err)
	}

	return r.handleError(tx.Commit().Error)
}

// GetMLPredictionData retrieves ML prediction data for a specific twin and task
func (r *timeseriesRepository) GetMLPredictionData(twinID string, taskID string, start, end time.Time, limit int) ([]models.MLPredictionData, error) {
	var predictions []models.MLPredictionData

	query := r.GetDB().Where("twin_id = ? AND task_id = ? AND time >= ? AND time <= ?",
		twinID, taskID, start, end)

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Order("time desc").Find(&predictions).Error
	if err != nil {
		return nil, r.handleError(err)
	}

	return predictions, nil
}

// GetLatestMLPrediction retrieves the latest ML prediction for a twin and task
func (r *timeseriesRepository) GetLatestMLPrediction(twinID string, taskID string) (*models.MLPredictionData, error) {
	var prediction models.MLPredictionData
	err := r.GetDB().Where("twin_id = ? AND task_id = ?", twinID, taskID).
		Order("time desc").
		Limit(1).
		First(&prediction).Error

	if err != nil {
		return nil, r.handleError(err)
	}

	return &prediction, nil
}

// DeleteMLPredictionData deletes ML prediction data for a twin and task within a time range
func (r *timeseriesRepository) DeleteMLPredictionData(twinID string, taskID string, start, end time.Time) error {
	result := r.GetDB().Where("twin_id = ? AND task_id = ? AND time >= ? AND time <= ?",
		twinID, taskID, start, end).
		Delete(&models.MLPredictionData{})

	return r.handleError(result.Error)
}
