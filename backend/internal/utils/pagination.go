package utils

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DefaultLimit is the default number of items per page
const DefaultLimit = 20

// MaxLimit is the maximum number of items per page
const MaxLimit = 100

// PaginationRequest holds pagination parameters
type PaginationRequest struct {
	Page  int `form:"page"`
	Limit int `form:"limit"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination holds pagination metadata
type Pagination struct {
	CurrentPage int `json:"current_page"`
	TotalPages  int `json:"total_pages"`
	TotalItems  int `json:"total_items"`
	PerPage     int `json:"per_page"`
}

// GetPaginationFromContext extracts pagination parameters from the gin context
func GetPaginationFromContext(ctx *gin.Context) PaginationRequest {
	var pagination PaginationRequest

	// Get page from query parameter
	pageStr := ctx.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	// Get limit from query parameter
	limitStr := ctx.DefaultQuery("limit", strconv.Itoa(DefaultLimit))
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = DefaultLimit
	}

	// Cap limit to maximum value
	if limit > MaxLimit {
		limit = MaxLimit
	}

	pagination.Page = page
	pagination.Limit = limit

	return pagination
}

// ApplyPagination applies pagination to a GORM query
func ApplyPagination(query *gorm.DB, pagination PaginationRequest) *gorm.DB {
	offset := (pagination.Page - 1) * pagination.Limit
	return query.Offset(offset).Limit(pagination.Limit)
}

// NewPaginatedResponse creates a new paginated response
func NewPaginatedResponse(data interface{}, pagination PaginationRequest, totalItems int) PaginatedResponse {
	totalPages := calculateTotalPages(totalItems, pagination.Limit)

	return PaginatedResponse{
		Data: data,
		Pagination: Pagination{
			CurrentPage: pagination.Page,
			TotalPages:  totalPages,
			TotalItems:  totalItems,
			PerPage:     pagination.Limit,
		},
	}
}

// calculateTotalPages calculates the total number of pages
func calculateTotalPages(totalItems, perPage int) int {
	if perPage == 0 {
		return 0
	}

	totalPages := totalItems / perPage
	if totalItems%perPage > 0 {
		totalPages++
	}

	return totalPages
}
