package utils

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Common error types for consistent handling
var (
	ErrNotFound           = errors.New("resource not found")
	ErrAlreadyExists      = errors.New("resource already exists")
	ErrUnauthorized       = errors.New("unauthorized access")
	ErrForbidden          = errors.New("access forbidden")
	ErrBadRequest         = errors.New("invalid request")
	ErrInternalServer     = errors.New("internal server error")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrValidation         = errors.New("validation error")
)

// ErrorResponse represents a standardized API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// HandleError processes an error and returns the appropriate HTTP response
func HandleError(ctx *gin.Context, err error, logger *Logger) {
	// Determine the error type and status code
	status, response := processError(err)

	// If it's a server error, log it
	if status >= 500 {
		logger.Error("Server error",
			zap.Error(err),
			zap.String("path", ctx.Request.URL.Path),
			zap.String("method", ctx.Request.Method),
			zap.String("ip", ctx.ClientIP()),
		)
	}

	// Return the error response
	ctx.JSON(status, response)
}

// processError determines the appropriate HTTP status code and response for an error
func processError(err error) (int, ErrorResponse) {
	// Handle common errors first
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: err.Error(),
		}
	case errors.Is(err, ErrAlreadyExists):
		return http.StatusConflict, ErrorResponse{
			Error:   "already_exists",
			Message: err.Error(),
		}
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: err.Error(),
		}
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: err.Error(),
		}
	case errors.Is(err, ErrBadRequest):
		return http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		}
	case errors.Is(err, ErrValidation):
		return http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		}
	case errors.Is(err, ErrServiceUnavailable):
		return http.StatusServiceUnavailable, ErrorResponse{
			Error:   "service_unavailable",
			Message: err.Error(),
		}
	default:
		// Default to internal server error
		return http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_server_error",
			Message: "An unexpected error occurred",
		}
	}
}

// NewError creates a new error with a custom error code
type ErrorWithCode struct {
	Err  error
	Code string
}

// Error returns the error message
func (e *ErrorWithCode) Error() string {
	return e.Err.Error()
}

// Unwrap returns the wrapped error
func (e *ErrorWithCode) Unwrap() error {
	return e.Err
}

// NewErrorWithCode creates a new error with a custom error code
func NewErrorWithCode(err error, code string) error {
	return &ErrorWithCode{
		Err:  err,
		Code: code,
	}
}

// IsNotFoundError checks if an error is a "not found" error
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsBadRequestError checks if an error is a "bad request" error
func IsBadRequestError(err error) bool {
	return errors.Is(err, ErrBadRequest)
}

// IsUnauthorizedError checks if an error is an "unauthorized" error
func IsUnauthorizedError(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

// IsForbiddenError checks if an error is a "forbidden" error
func IsForbiddenError(err error) bool {
	return errors.Is(err, ErrForbidden)
}

// IsInternalServerError checks if an error is an "internal server error"
func IsInternalServerError(err error) bool {
	return errors.Is(err, ErrInternalServer)
}

// IsAlreadyExistsError checks if an error is an "already exists" error
func IsAlreadyExistsError(err error) bool {
	return errors.Is(err, ErrAlreadyExists)
}

// IsValidationError checks if an error is a "validation error"
func IsValidationError(err error) bool {
	return errors.Is(err, ErrValidation)
}

// IsServiceUnavailableError checks if an error is a "service unavailable" error
func IsServiceUnavailableError(err error) bool {
	return errors.Is(err, ErrServiceUnavailable)
}
