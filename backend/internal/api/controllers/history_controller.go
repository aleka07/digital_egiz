package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/digital-egiz/backend/internal/services"
	"github.com/digital-egiz/backend/internal/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TimeseriesRequest defines the query parameters for timeseries data
type TimeseriesRequest struct {
	Start       time.Time `form:"start" time_format:"2006-01-02T15:04:05Z07:00"`
	End         time.Time `form:"end" time_format:"2006-01-02T15:04:05Z07:00"`
	FeaturePath string    `form:"feature_path" binding:"required"`
	Limit       int       `form:"limit"`
}

// AggregatedRequest defines the query parameters for aggregated data
type AggregatedRequest struct {
	Start       time.Time `form:"start" time_format:"2006-01-02T15:04:05Z07:00" binding:"required"`
	End         time.Time `form:"end" time_format:"2006-01-02T15:04:05Z07:00" binding:"required"`
	FeaturePath string    `form:"feature_path" binding:"required"`
	Interval    string    `form:"interval" binding:"required"`
}

// AlertsRequest defines the query parameters for alert data
type AlertsRequest struct {
	Start    time.Time `form:"start" time_format:"2006-01-02T15:04:05Z07:00"`
	End      time.Time `form:"end" time_format:"2006-01-02T15:04:05Z07:00"`
	Severity string    `form:"severity"`
	Limit    int       `form:"limit"`
}

// MLPredictionRequest defines the query parameters for ML prediction data
type MLPredictionRequest struct {
	Start  time.Time `form:"start" time_format:"2006-01-02T15:04:05Z07:00"`
	End    time.Time `form:"end" time_format:"2006-01-02T15:04:05Z07:00"`
	TaskID string    `form:"task_id" binding:"required"`
	Limit  int       `form:"limit"`
}

// AcknowledgeAlertRequest defines the request body for acknowledging an alert
type AcknowledgeAlertRequest struct {
	AlertID string `json:"alert_id" binding:"required"`
}

// HistoryController handles history data requests
type HistoryController struct {
	historyService *services.HistoryService
	logger         *utils.Logger
}

// NewHistoryController creates a new history controller
func NewHistoryController(historyService *services.HistoryService, logger *utils.Logger) *HistoryController {
	return &HistoryController{
		historyService: historyService,
		logger:         logger.Named("history_controller"),
	}
}

// RegisterRoutes registers the history routes
func (c *HistoryController) RegisterRoutes(router *gin.RouterGroup) {
	// Routes under /twins/:id/history
	router.GET("/timeseries", c.GetTimeseriesData)
	router.GET("/timeseries/latest", c.GetLatestTimeseriesData)
	router.GET("/aggregated", c.GetAggregatedData)
	router.GET("/alerts", c.GetAlertData)
	router.POST("/alerts/acknowledge", c.AcknowledgeAlert)
	router.GET("/ml-predictions", c.GetMLPredictionData)
	router.GET("/ml-predictions/latest", c.GetLatestMLPrediction)
}

// GetTimeseriesData returns time-series data for a twin
// @Summary Get time-series data
// @Description Returns time-series data for a twin and feature path
// @Tags history
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Twin ID"
// @Param feature_path query string true "Feature path"
// @Param start query string false "Start time (ISO8601)"
// @Param end query string false "End time (ISO8601)"
// @Param limit query int false "Limit results"
// @Success 200 {array} models.TimeseriesData "Time-series data"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "Twin not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /twins/{id}/history/timeseries [get]
func (c *HistoryController) GetTimeseriesData(ctx *gin.Context) {
	// Get twin ID from URL
	twinID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid twin ID"})
		return
	}

	// Parse query parameters
	var req TimeseriesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default values
	if req.Start.IsZero() {
		req.Start = time.Now().Add(-24 * time.Hour) // Default to last 24 hours
	}
	if req.End.IsZero() {
		req.End = time.Now()
	}
	if req.Limit <= 0 || req.Limit > 1000 {
		req.Limit = 100 // Default limit
	}

	// Get data from service
	data, err := c.historyService.GetTimeseriesData(uint(twinID), req.FeaturePath, req.Start, req.End, req.Limit)
	if err != nil {
		c.logger.Error("Failed to get time-series data",
			zap.Uint64("twin_id", twinID),
			zap.String("feature_path", req.FeaturePath),
			zap.Error(err))

		if err.Error() == "twin not found" {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Twin not found"})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve time-series data"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": data,
		"meta": gin.H{
			"twin_id":      twinID,
			"feature_path": req.FeaturePath,
			"start":        req.Start,
			"end":          req.End,
			"count":        len(data),
		},
	})
}

// GetLatestTimeseriesData returns the latest time-series data point for a twin
// @Summary Get latest time-series data
// @Description Returns the latest time-series data point for a twin and feature path
// @Tags history
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Twin ID"
// @Param feature_path query string true "Feature path"
// @Success 200 {object} models.TimeseriesData "Latest time-series data"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "Twin not found or no data available"
// @Failure 500 {object} map[string]string "Server error"
// @Router /twins/{id}/history/timeseries/latest [get]
func (c *HistoryController) GetLatestTimeseriesData(ctx *gin.Context) {
	// Get twin ID from URL
	twinID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid twin ID"})
		return
	}

	// Parse query parameters
	featurePath := ctx.Query("feature_path")
	if featurePath == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Feature path is required"})
		return
	}

	// Get data from service
	data, err := c.historyService.GetLatestTimeseriesData(uint(twinID), featurePath)
	if err != nil {
		c.logger.Error("Failed to get latest time-series data",
			zap.Uint64("twin_id", twinID),
			zap.String("feature_path", featurePath),
			zap.Error(err))

		if err.Error() == "twin not found" {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Twin not found"})
			return
		}

		if err.Error() == "no data found for the given twin and feature path" {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "No data found for the given twin and feature path"})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve latest time-series data"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": data,
		"meta": gin.H{
			"twin_id":      twinID,
			"feature_path": featurePath,
		},
	})
}

// GetAggregatedData returns aggregated time-series data for a twin
// @Summary Get aggregated time-series data
// @Description Returns aggregated time-series data for a twin and feature path
// @Tags history
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Twin ID"
// @Param feature_path query string true "Feature path"
// @Param start query string true "Start time (ISO8601)"
// @Param end query string true "End time (ISO8601)"
// @Param interval query string true "Aggregation interval (1m, 5m, 15m, 30m, 1h, 6h, 12h, 1d, 1w, 1mon)"
// @Success 200 {array} models.AggregatedData "Aggregated time-series data"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "Twin not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /twins/{id}/history/aggregated [get]
func (c *HistoryController) GetAggregatedData(ctx *gin.Context) {
	// Get twin ID from URL
	twinID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid twin ID"})
		return
	}

	// Parse query parameters
	var req AggregatedRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get data from service
	data, err := c.historyService.GetAggregatedData(uint(twinID), req.FeaturePath, req.Start, req.End, req.Interval)
	if err != nil {
		c.logger.Error("Failed to get aggregated data",
			zap.Uint64("twin_id", twinID),
			zap.String("feature_path", req.FeaturePath),
			zap.String("interval", req.Interval),
			zap.Error(err))

		if err.Error() == "twin not found" {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Twin not found"})
			return
		}

		if err.Error() == "invalid interval" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interval. Supported values: 1m, 5m, 15m, 30m, 1h, 6h, 12h, 1d, 1w, 1mon"})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve aggregated data"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": data,
		"meta": gin.H{
			"twin_id":      twinID,
			"feature_path": req.FeaturePath,
			"start":        req.Start,
			"end":          req.End,
			"interval":     req.Interval,
			"count":        len(data),
		},
	})
}

// GetAlertData returns alert data for a twin
// @Summary Get alert data
// @Description Returns alert data for a twin
// @Tags history
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Twin ID"
// @Param start query string false "Start time (ISO8601)"
// @Param end query string false "End time (ISO8601)"
// @Param severity query string false "Alert severity (info, warning, error, critical)"
// @Param limit query int false "Limit results"
// @Success 200 {array} models.AlertData "Alert data"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "Twin not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /twins/{id}/history/alerts [get]
func (c *HistoryController) GetAlertData(ctx *gin.Context) {
	// Get twin ID from URL
	twinID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid twin ID"})
		return
	}

	// Parse query parameters
	var req AlertsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default values
	if req.Start.IsZero() {
		req.Start = time.Now().Add(-24 * time.Hour) // Default to last 24 hours
	}
	if req.End.IsZero() {
		req.End = time.Now()
	}
	if req.Limit <= 0 || req.Limit > 1000 {
		req.Limit = 100 // Default limit
	}

	// Get data from service
	data, err := c.historyService.GetAlertData(uint(twinID), req.Start, req.End, req.Severity, req.Limit)
	if err != nil {
		c.logger.Error("Failed to get alert data",
			zap.Uint64("twin_id", twinID),
			zap.String("severity", req.Severity),
			zap.Error(err))

		if err.Error() == "twin not found" {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Twin not found"})
			return
		}

		if err.Error() == "invalid severity" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid severity. Supported values: info, warning, error, critical"})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve alert data"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": data,
		"meta": gin.H{
			"twin_id":  twinID,
			"start":    req.Start,
			"end":      req.End,
			"severity": req.Severity,
			"count":    len(data),
		},
	})
}

// AcknowledgeAlert acknowledges an alert
// @Summary Acknowledge alert
// @Description Acknowledges an alert
// @Tags history
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Twin ID"
// @Param request body AcknowledgeAlertRequest true "Acknowledge alert request"
// @Success 200 {object} map[string]string "Alert acknowledged"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "Alert not found or already acknowledged"
// @Failure 500 {object} map[string]string "Server error"
// @Router /twins/{id}/history/alerts/acknowledge [post]
func (c *HistoryController) AcknowledgeAlert(ctx *gin.Context) {
	// Get twin ID from URL (not used in the service method but kept for API consistency)
	_, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid twin ID"})
		return
	}

	// Parse request body
	var req AcknowledgeAlertRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get user ID"})
		return
	}

	// Acknowledge alert
	if err := c.historyService.AcknowledgeAlert(req.AlertID, userID.(uint)); err != nil {
		c.logger.Error("Failed to acknowledge alert",
			zap.String("alert_id", req.AlertID),
			zap.Uint("user_id", userID.(uint)),
			zap.Error(err))

		if err.Error() == "alert not found or already acknowledged" {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Alert not found or already acknowledged"})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to acknowledge alert"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Alert acknowledged"})
}

// GetMLPredictionData returns ML prediction data for a twin
// @Summary Get ML prediction data
// @Description Returns ML prediction data for a twin and task
// @Tags history
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Twin ID"
// @Param task_id query string true "ML task ID"
// @Param start query string false "Start time (ISO8601)"
// @Param end query string false "End time (ISO8601)"
// @Param limit query int false "Limit results"
// @Success 200 {array} models.MLPredictionData "ML prediction data"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "Twin not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /twins/{id}/history/ml-predictions [get]
func (c *HistoryController) GetMLPredictionData(ctx *gin.Context) {
	// Get twin ID from URL
	twinID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid twin ID"})
		return
	}

	// Parse query parameters
	var req MLPredictionRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default values
	if req.Start.IsZero() {
		req.Start = time.Now().Add(-24 * time.Hour) // Default to last 24 hours
	}
	if req.End.IsZero() {
		req.End = time.Now()
	}
	if req.Limit <= 0 || req.Limit > 1000 {
		req.Limit = 100 // Default limit
	}

	// Get data from service
	data, err := c.historyService.GetMLPredictionData(uint(twinID), req.TaskID, req.Start, req.End, req.Limit)
	if err != nil {
		c.logger.Error("Failed to get ML prediction data",
			zap.Uint64("twin_id", twinID),
			zap.String("task_id", req.TaskID),
			zap.Error(err))

		if err.Error() == "twin not found" {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Twin not found"})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve ML prediction data"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": data,
		"meta": gin.H{
			"twin_id": twinID,
			"task_id": req.TaskID,
			"start":   req.Start,
			"end":     req.End,
			"count":   len(data),
		},
	})
}

// GetLatestMLPrediction returns the latest ML prediction for a twin
// @Summary Get latest ML prediction
// @Description Returns the latest ML prediction for a twin and task
// @Tags history
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Twin ID"
// @Param task_id query string true "ML task ID"
// @Success 200 {object} models.MLPredictionData "Latest ML prediction"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "Twin not found or no prediction available"
// @Failure 500 {object} map[string]string "Server error"
// @Router /twins/{id}/history/ml-predictions/latest [get]
func (c *HistoryController) GetLatestMLPrediction(ctx *gin.Context) {
	// Get twin ID from URL
	twinID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid twin ID"})
		return
	}

	// Parse query parameters
	taskID := ctx.Query("task_id")
	if taskID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Task ID is required"})
		return
	}

	// Get data from service
	data, err := c.historyService.GetLatestMLPrediction(uint(twinID), taskID)
	if err != nil {
		c.logger.Error("Failed to get latest ML prediction",
			zap.Uint64("twin_id", twinID),
			zap.String("task_id", taskID),
			zap.Error(err))

		if err.Error() == "twin not found" {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Twin not found"})
			return
		}

		if err.Error() == "no ML prediction found for the given twin and task" {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "No ML prediction found for the given twin and task"})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve latest ML prediction"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": data,
		"meta": gin.H{
			"twin_id": twinID,
			"task_id": taskID,
		},
	})
}
