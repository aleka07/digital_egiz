package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/digital-egiz/backend/internal/db/models"
	"github.com/digital-egiz/backend/internal/services"
	"github.com/digital-egiz/backend/internal/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TwinTypeResponse represents a twin type in responses
type TwinTypeResponse struct {
	ID          uint            `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Version     string          `json:"version"`
	SchemaJSON  json.RawMessage `json:"schema_json"`
	CreatedBy   uint            `json:"created_by"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
}

// CreateTwinTypeRequest represents the request to create a twin type
type CreateTwinTypeRequest struct {
	Name        string          `json:"name" binding:"required"`
	Description string          `json:"description"`
	Version     string          `json:"version" binding:"required"`
	SchemaJSON  json.RawMessage `json:"schema_json" binding:"required"`
}

// UpdateTwinTypeRequest represents the request to update a twin type
type UpdateTwinTypeRequest struct {
	Name        string          `json:"name" binding:"required"`
	Description string          `json:"description"`
	Version     string          `json:"version" binding:"required"`
	SchemaJSON  json.RawMessage `json:"schema_json" binding:"required"`
}

// TwinTypeController handles twin type management endpoints
type TwinTypeController struct {
	twinTypeService *services.TwinTypeService
	logger          *utils.Logger
}

// NewTwinTypeController creates a new twin type controller
func NewTwinTypeController(twinTypeService *services.TwinTypeService, logger *utils.Logger) *TwinTypeController {
	return &TwinTypeController{
		twinTypeService: twinTypeService,
		logger:          logger.Named("twin_type_controller"),
	}
}

// RegisterRoutes registers the controller's routes with the router group
func (tc *TwinTypeController) RegisterRoutes(router *gin.RouterGroup) {
	twinTypes := router.Group("/twin-types")
	{
		twinTypes.GET("", tc.ListTwinTypes)
		twinTypes.POST("", tc.CreateTwinType)
		twinTypes.GET("/:id", tc.GetTwinType)
		twinTypes.PUT("/:id", tc.UpdateTwinType)
		twinTypes.DELETE("/:id", tc.DeleteTwinType)
	}
}

// ListTwinTypes returns a paginated list of twin types
// @Summary Get a list of twin types
// @Description Returns a paginated list of twin types
// @Tags twin-types
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "Page number (1-based)" default(1)
// @Param limit query int false "Page size" default(20)
// @Success 200 {object} []TwinTypeResponse "Twin type list"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Router /twin-types [get]
func (tc *TwinTypeController) ListTwinTypes(c *gin.Context) {
	// Parse pagination parameters
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	twinTypes, total, err := tc.twinTypeService.List(page, limit)
	if err != nil {
		tc.logger.Error("Failed to list twin types", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve twin types"})
		return
	}

	// Map twin types to response objects
	response := make([]TwinTypeResponse, len(twinTypes))
	for i, twinType := range twinTypes {
		response[i] = TwinTypeResponse{
			ID:          twinType.ID,
			Name:        twinType.Name,
			Description: twinType.Description,
			Version:     twinType.Version,
			SchemaJSON:  json.RawMessage(twinType.SchemaJSON),
			CreatedBy:   twinType.CreatedBy,
			CreatedAt:   twinType.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   twinType.UpdatedAt.Format(time.RFC3339),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"twin_types": response,
		"pagination": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// CreateTwinType creates a new twin type
// @Summary Create a new twin type
// @Description Creates a new twin type
// @Tags twin-types
// @Accept json
// @Produce json
// @Security Bearer
// @Param twin_type body CreateTwinTypeRequest true "Twin type information"
// @Success 201 {object} TwinTypeResponse "Created twin type"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Router /twin-types [post]
func (tc *TwinTypeController) CreateTwinType(c *gin.Context) {
	// Get current user ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req CreateTwinTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate JSON schema (must be valid JSON)
	if !json.Valid(req.SchemaJSON) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON schema"})
		return
	}

	// Create new twin type
	twinType := &models.TwinType{
		Name:        req.Name,
		Description: req.Description,
		Version:     req.Version,
		SchemaJSON:  models.JSON(req.SchemaJSON),
		CreatedBy:   userID.(uint),
	}

	// Save twin type to database
	if err := tc.twinTypeService.Create(twinType); err != nil {
		tc.logger.Error("Failed to create twin type", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, TwinTypeResponse{
		ID:          twinType.ID,
		Name:        twinType.Name,
		Description: twinType.Description,
		Version:     twinType.Version,
		SchemaJSON:  json.RawMessage(twinType.SchemaJSON),
		CreatedBy:   twinType.CreatedBy,
		CreatedAt:   twinType.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   twinType.UpdatedAt.Format(time.RFC3339),
	})
}

// GetTwinType returns a twin type by ID
// @Summary Get twin type by ID
// @Description Returns a twin type by ID
// @Tags twin-types
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Twin type ID"
// @Success 200 {object} TwinTypeResponse "Twin type details"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Twin type not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /twin-types/{id} [get]
func (tc *TwinTypeController) GetTwinType(c *gin.Context) {
	// Parse twin type ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid twin type ID"})
		return
	}

	twinType, err := tc.twinTypeService.GetByID(uint(id))
	if err != nil {
		if err.Error() == "twin type not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Twin type not found"})
			return
		}
		tc.logger.Error("Failed to get twin type", zap.Uint("id", uint(id)), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve twin type"})
		return
	}

	c.JSON(http.StatusOK, TwinTypeResponse{
		ID:          twinType.ID,
		Name:        twinType.Name,
		Description: twinType.Description,
		Version:     twinType.Version,
		SchemaJSON:  json.RawMessage(twinType.SchemaJSON),
		CreatedBy:   twinType.CreatedBy,
		CreatedAt:   twinType.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   twinType.UpdatedAt.Format(time.RFC3339),
	})
}

// UpdateTwinType updates a twin type by ID
// @Summary Update twin type
// @Description Updates a twin type by ID
// @Tags twin-types
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Twin type ID"
// @Param twin_type body UpdateTwinTypeRequest true "Twin type information"
// @Success 200 {object} TwinTypeResponse "Updated twin type"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Twin type not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /twin-types/{id} [put]
func (tc *TwinTypeController) UpdateTwinType(c *gin.Context) {
	// Parse twin type ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid twin type ID"})
		return
	}

	var req UpdateTwinTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate JSON schema (must be valid JSON)
	if !json.Valid(req.SchemaJSON) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON schema"})
		return
	}

	// Get existing twin type
	twinType, err := tc.twinTypeService.GetByID(uint(id))
	if err != nil {
		if err.Error() == "twin type not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Twin type not found"})
			return
		}
		tc.logger.Error("Failed to get twin type", zap.Uint("id", uint(id)), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve twin type"})
		return
	}

	// Update twin type fields
	twinType.Name = req.Name
	twinType.Description = req.Description
	twinType.Version = req.Version
	twinType.SchemaJSON = models.JSON(req.SchemaJSON)

	// Save twin type to database
	if err := tc.twinTypeService.Update(twinType); err != nil {
		tc.logger.Error("Failed to update twin type", zap.Uint("id", uint(id)), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, TwinTypeResponse{
		ID:          twinType.ID,
		Name:        twinType.Name,
		Description: twinType.Description,
		Version:     twinType.Version,
		SchemaJSON:  json.RawMessage(twinType.SchemaJSON),
		CreatedBy:   twinType.CreatedBy,
		CreatedAt:   twinType.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   twinType.UpdatedAt.Format(time.RFC3339),
	})
}

// DeleteTwinType deletes a twin type by ID
// @Summary Delete twin type
// @Description Deletes a twin type by ID
// @Tags twin-types
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Twin type ID"
// @Success 200 {object} map[string]string "Twin type deleted successfully"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Twin type not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /twin-types/{id} [delete]
func (tc *TwinTypeController) DeleteTwinType(c *gin.Context) {
	// Parse twin type ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid twin type ID"})
		return
	}

	// Delete twin type
	if err := tc.twinTypeService.Delete(uint(id)); err != nil {
		if err.Error() == "twin type not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Twin type not found"})
			return
		}
		tc.logger.Error("Failed to delete twin type", zap.Uint("id", uint(id)), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete twin type"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Twin type deleted successfully"})
}
