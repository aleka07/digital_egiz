package controllers

import (
	"net/http"
	"strconv"

	"github.com/digital-egiz/backend/internal/db/models"
	"github.com/digital-egiz/backend/internal/services"
	"github.com/digital-egiz/backend/internal/utils"
	"github.com/gin-gonic/gin"
)

// UserIDKey is the context key for the user ID
const UserIDKey = "user_id"

// TwinController handles HTTP requests for twin operations
type TwinController struct {
	twinService *services.TwinService
	logger      *utils.Logger
}

// NewTwinController creates a new twin controller
func NewTwinController(twinService *services.TwinService, logger *utils.Logger) *TwinController {
	return &TwinController{
		twinService: twinService,
		logger:      logger.Named("twin_controller"),
	}
}

// RegisterRoutes registers the twin routes
func (c *TwinController) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("", c.CreateTwin)
	router.GET("", c.ListTwins)
	router.GET("/:id", c.GetTwin)
	router.PUT("/:id", c.UpdateTwin)
	router.DELETE("/:id", c.DeleteTwin)

	// Model bindings routes
	router.POST("/:id/bindings", c.CreateModelBinding)
	router.GET("/:id/bindings", c.ListModelBindings)
	router.PUT("/bindings/:bindingId", c.UpdateModelBinding)
	router.DELETE("/bindings/:bindingId", c.DeleteModelBinding)
}

// CreateTwinRequest defines the request body for creating a twin
type CreateTwinRequest struct {
	Name      string `json:"name" binding:"required"`
	DittoID   string `json:"dittoId" binding:"required"`
	TypeID    uint   `json:"typeId" binding:"required"`
	ProjectID uint   `json:"projectId" binding:"required"`
	// Optional fields
	Description string `json:"description"`
	ModelURL    string `json:"modelUrl"`
}

// CreateTwin handles creating a new twin
func (c *TwinController) CreateTwin(ctx *gin.Context) {
	// Parse the request body
	var req CreateTwinRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// Get user ID from context
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get user ID"})
		return
	}

	// Create twin object
	twin := &models.Twin{
		Name:        req.Name,
		DittoID:     req.DittoID,
		TypeID:      req.TypeID,
		ProjectID:   req.ProjectID,
		Description: req.Description,
		ModelURL:    req.ModelURL,
		CreatedBy:   userID.(uint),
	}

	// Create the twin
	if err := c.twinService.Create(twin); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, twin)
}

// GetTwin handles getting a twin by ID
func (c *TwinController) GetTwin(ctx *gin.Context) {
	// Get twin ID from URL
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid twin ID"})
		return
	}

	// Get twin
	twin, err := c.twinService.GetByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, twin)
}

// ListTwinsResponse defines the response for listing twins
type ListTwinsResponse struct {
	Twins []models.Twin `json:"twins"`
	Total int64         `json:"total"`
	Page  int           `json:"page"`
	Size  int           `json:"size"`
}

// ListTwins handles listing twins for a project
func (c *TwinController) ListTwins(ctx *gin.Context) {
	// Parse query parameters
	projectID, err := strconv.ParseUint(ctx.Query("projectId"), 10, 64)
	if err != nil || projectID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or missing project ID"})
		return
	}

	page, err := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	size, err := strconv.Atoi(ctx.DefaultQuery("size", "20"))
	if err != nil || size < 1 || size > 100 {
		size = 20 // Default size
	}

	// Get twins
	twins, total, err := c.twinService.ListByProject(uint(projectID), page, size)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Prepare response
	response := ListTwinsResponse{
		Twins: twins,
		Total: total,
		Page:  page,
		Size:  size,
	}

	ctx.JSON(http.StatusOK, response)
}

// UpdateTwinRequest defines the request body for updating a twin
type UpdateTwinRequest struct {
	Name        string `json:"name" binding:"required"`
	DittoID     string `json:"dittoId" binding:"required"`
	Description string `json:"description"`
	ModelURL    string `json:"modelUrl"`
}

// UpdateTwin handles updating a twin
func (c *TwinController) UpdateTwin(ctx *gin.Context) {
	// Get twin ID from URL
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid twin ID"})
		return
	}

	// Parse the request body
	var req UpdateTwinRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// Get existing twin
	existingTwin, err := c.twinService.GetByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Update fields
	existingTwin.Name = req.Name
	existingTwin.DittoID = req.DittoID
	existingTwin.Description = req.Description
	existingTwin.ModelURL = req.ModelURL

	// Update twin
	if err := c.twinService.Update(existingTwin); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, existingTwin)
}

// DeleteTwin handles deleting a twin
func (c *TwinController) DeleteTwin(ctx *gin.Context) {
	// Get twin ID from URL
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid twin ID"})
		return
	}

	// Delete twin
	if err := c.twinService.Delete(uint(id)); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

// ModelBindingRequest defines the request body for creating/updating a model binding
type ModelBindingRequest struct {
	PartID      string `json:"partId" binding:"required"`
	FeaturePath string `json:"featurePath" binding:"required"`
	BindingType string `json:"bindingType" binding:"required"`
	Properties  string `json:"properties"` // JSON-encoded properties
}

// CreateModelBinding handles creating a model binding
func (c *TwinController) CreateModelBinding(ctx *gin.Context) {
	// Get twin ID from URL
	twinID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid twin ID"})
		return
	}

	// Parse the request body
	var req ModelBindingRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// Create binding object
	var properties models.JSON
	if req.Properties != "" {
		if err := properties.UnmarshalJSON([]byte(req.Properties)); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid properties JSON"})
			return
		}
	}

	binding := &models.ModelBinding{
		TwinID:      uint(twinID),
		PartID:      req.PartID,
		FeaturePath: req.FeaturePath,
		BindingType: req.BindingType,
		Properties:  properties,
	}

	// Create the binding
	if err := c.twinService.CreateModelBinding(binding); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, binding)
}

// ListModelBindings handles listing model bindings for a twin
func (c *TwinController) ListModelBindings(ctx *gin.Context) {
	// Get twin ID from URL
	twinID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid twin ID"})
		return
	}

	// Get bindings
	bindings, err := c.twinService.ListModelBindings(uint(twinID))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, bindings)
}

// UpdateModelBinding handles updating a model binding
func (c *TwinController) UpdateModelBinding(ctx *gin.Context) {
	// Get binding ID from URL
	bindingID, err := strconv.ParseUint(ctx.Param("bindingId"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid binding ID"})
		return
	}

	// Parse the request body
	var req ModelBindingRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// Create binding object
	var properties models.JSON
	if req.Properties != "" {
		if err := properties.UnmarshalJSON([]byte(req.Properties)); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid properties JSON"})
			return
		}
	}

	binding := &models.ModelBinding{
		ID:          uint(bindingID),
		PartID:      req.PartID,
		FeaturePath: req.FeaturePath,
		BindingType: req.BindingType,
		Properties:  properties,
	}

	// Update the binding
	if err := c.twinService.UpdateModelBinding(binding); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, binding)
}

// DeleteModelBinding handles deleting a model binding
func (c *TwinController) DeleteModelBinding(ctx *gin.Context) {
	// Get binding ID from URL
	bindingID, err := strconv.ParseUint(ctx.Param("bindingId"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid binding ID"})
		return
	}

	// Delete binding
	if err := c.twinService.DeleteModelBinding(uint(bindingID)); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}
