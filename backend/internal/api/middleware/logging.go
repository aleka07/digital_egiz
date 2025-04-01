package middleware

import (
	"time"

	"github.com/digital-egiz/backend/internal/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LoggingMiddleware returns a middleware that logs HTTP requests
func LoggingMiddleware(logger *utils.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Stop timer
		end := time.Now()
		latency := end.Sub(start)

		// Get status code and request method
		statusCode := c.Writer.Status()
		method := c.Request.Method
		clientIP := c.ClientIP()

		// Get user ID if authenticated
		userID, exists := c.Get("user_id")

		// Prepare logger fields
		logFields := []zap.Field{
			zap.Int("status", statusCode),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", clientIP),
			zap.Duration("latency", latency),
		}

		// Add user ID if available
		if exists {
			logFields = append(logFields, zap.Any("user_id", userID))
		}

		// Log based on status code
		switch {
		case statusCode >= 500:
			logger.Error("Server error", logFields...)
		case statusCode >= 400:
			logger.Warn("Client error", logFields...)
		default:
			logger.Info("Request completed", logFields...)
		}
	}
}
