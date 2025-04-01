package utils

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ValidationError represents a structured validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrorResponse is the standard response for validation errors
type ValidationErrorResponse struct {
	Errors []ValidationError `json:"errors"`
}

// HandleValidationErrors processes validation errors and returns a standardized response
func HandleValidationErrors(ctx *gin.Context, err error) {
	// Check if the error is a validator error
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		// If not a validation error, return a generic error
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create a slice to hold the validation errors
	var errors []ValidationError

	// Process each validation error
	for _, fieldError := range validationErrors {
		// Create a human-readable error message
		message := getValidationErrorMessage(fieldError)

		// Convert the field name to snake_case for consistent API responses
		fieldName := toSnakeCase(fieldError.Field())

		// Add the error to the slice
		errors = append(errors, ValidationError{
			Field:   fieldName,
			Message: message,
		})
	}

	// Return the validation errors
	ctx.JSON(http.StatusBadRequest, ValidationErrorResponse{
		Errors: errors,
	})
}

// getValidationErrorMessage returns a human-readable message for a validation error
func getValidationErrorMessage(fieldError validator.FieldError) string {
	switch fieldError.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Invalid email format"
	case "min":
		if fieldError.Type().Kind().String() == "string" {
			return "Must be at least " + fieldError.Param() + " characters long"
		}
		return "Must be at least " + fieldError.Param()
	case "max":
		if fieldError.Type().Kind().String() == "string" {
			return "Must be at most " + fieldError.Param() + " characters long"
		}
		return "Must be at most " + fieldError.Param()
	case "len":
		if fieldError.Type().Kind().String() == "string" {
			return "Must be exactly " + fieldError.Param() + " characters long"
		}
		return "Must be exactly " + fieldError.Param()
	case "eq":
		return "Must be equal to " + fieldError.Param()
	case "ne":
		return "Must not be equal to " + fieldError.Param()
	case "gt":
		return "Must be greater than " + fieldError.Param()
	case "gte":
		return "Must be greater than or equal to " + fieldError.Param()
	case "lt":
		return "Must be less than " + fieldError.Param()
	case "lte":
		return "Must be less than or equal to " + fieldError.Param()
	case "alpha":
		return "Must contain only alphabetic characters"
	case "alphanum":
		return "Must contain only alphanumeric characters"
	case "numeric":
		return "Must be a valid numeric value"
	case "uuid":
		return "Must be a valid UUID"
	case "url":
		return "Must be a valid URL"
	case "datetime":
		return "Must be a valid datetime in format " + fieldError.Param()
	default:
		return "Invalid value for this field"
	}
}

// toSnakeCase converts a string from camelCase to snake_case
func toSnakeCase(s string) string {
	// If the string is already snake_case, return it as is
	if strings.Contains(s, "_") {
		return s
	}

	var result strings.Builder
	for i, r := range s {
		if i > 0 && 'A' <= r && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}
