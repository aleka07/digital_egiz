package controllers

import (
	"net/http"
	"strconv"

	"github.com/digital-egiz/backend/internal/db/models"
	"github.com/digital-egiz/backend/internal/services"
	"github.com/digital-egiz/backend/internal/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// UserResponse represents a user in responses
type UserResponse struct {
	ID        uint   `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
	Active    bool   `json:"active"`
}

// UpdateUserRequest represents the request to update a user
type UpdateUserRequest struct {
	FirstName string      `json:"first_name"`
	LastName  string      `json:"last_name"`
	Role      models.Role `json:"role,omitempty"`
	Active    *bool       `json:"active,omitempty"`
}

// ChangePasswordRequest represents the request to change a password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// UserController handles user management endpoints
type UserController struct {
	userService *services.UserService
	logger      *utils.Logger
}

// NewUserController creates a new user controller
func NewUserController(userService *services.UserService, logger *utils.Logger) *UserController {
	return &UserController{
		userService: userService,
		logger:      logger.Named("user_controller"),
	}
}

// RegisterRoutes registers the controller's routes with the router group
func (uc *UserController) RegisterRoutes(router *gin.RouterGroup) {
	users := router.Group("/users")
	{
		users.GET("", uc.ListUsers)
		users.GET("/me", uc.GetCurrentUser)
		users.GET("/:id", uc.GetUser)
		users.PUT("/me", uc.UpdateCurrentUser)
		users.PUT("/:id", uc.UpdateUser)
		users.POST("/change-password", uc.ChangePassword)
		users.DELETE("/:id", uc.DeleteUser)
	}
}

// ListUsers returns a paginated list of users
// @Summary Get a list of users
// @Description Returns a paginated list of users
// @Tags users
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "Page number (1-based)" default(1)
// @Param limit query int false "Page size" default(20)
// @Success 200 {object} []UserResponse "User list"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Router /users [get]
func (uc *UserController) ListUsers(c *gin.Context) {
	// Parse pagination parameters
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	users, total, err := uc.userService.List(page, limit)
	if err != nil {
		uc.logger.Error("Failed to list users", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users"})
		return
	}

	// Map users to response objects
	response := make([]UserResponse, len(users))
	for i, user := range users {
		response[i] = UserResponse{
			ID:        user.ID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Role:      string(user.Role),
			Active:    user.Active,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"users": response,
		"pagination": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// GetCurrentUser returns the current authenticated user
// @Summary Get current user
// @Description Returns the current authenticated user's profile
// @Tags users
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} UserResponse "User profile"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Router /users/me [get]
func (uc *UserController) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	user, err := uc.userService.GetByID(userID.(uint))
	if err != nil {
		uc.logger.Error("Failed to get user", zap.Uint("user_id", userID.(uint)), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user profile"})
		return
	}

	c.JSON(http.StatusOK, UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      string(user.Role),
		Active:    user.Active,
	})
}

// GetUser returns a user by ID
// @Summary Get user by ID
// @Description Returns a user by ID
// @Tags users
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "User ID"
// @Success 200 {object} UserResponse "User profile"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /users/{id} [get]
func (uc *UserController) GetUser(c *gin.Context) {
	// Parse user ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := uc.userService.GetByID(uint(id))
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		uc.logger.Error("Failed to get user", zap.Uint("user_id", uint(id)), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		return
	}

	c.JSON(http.StatusOK, UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      string(user.Role),
		Active:    user.Active,
	})
}

// UpdateCurrentUser updates the current user's profile
// @Summary Update current user
// @Description Updates the current authenticated user's profile
// @Tags users
// @Accept json
// @Produce json
// @Security Bearer
// @Param user body UpdateUserRequest true "User information"
// @Success 200 {object} UserResponse "Updated user profile"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Router /users/me [put]
func (uc *UserController) UpdateCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current user
	user, err := uc.userService.GetByID(userID.(uint))
	if err != nil {
		uc.logger.Error("Failed to get user for update", zap.Uint("user_id", userID.(uint)), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user profile"})
		return
	}

	// Update fields
	user.FirstName = req.FirstName
	user.LastName = req.LastName

	// Regular users can't change their role or active status
	if user.Role == models.RoleAdmin {
		if req.Role != "" {
			user.Role = req.Role
		}
		if req.Active != nil {
			user.Active = *req.Active
		}
	}

	if err := uc.userService.Update(user); err != nil {
		uc.logger.Error("Failed to update user", zap.Uint("user_id", user.ID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user profile"})
		return
	}

	c.JSON(http.StatusOK, UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      string(user.Role),
		Active:    user.Active,
	})
}

// UpdateUser updates a user by ID (admin only)
// @Summary Update user
// @Description Updates a user by ID (admin only)
// @Tags users
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "User ID"
// @Param user body UpdateUserRequest true "User information"
// @Success 200 {object} UserResponse "Updated user profile"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /users/{id} [put]
func (uc *UserController) UpdateUser(c *gin.Context) {
	// Check if admin
	role, exists := c.Get("user_role")
	if !exists || role != string(models.RoleAdmin) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	// Parse user ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user
	user, err := uc.userService.GetByID(uint(id))
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		uc.logger.Error("Failed to get user for update", zap.Uint("user_id", uint(id)), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		return
	}

	// Update fields
	user.FirstName = req.FirstName
	user.LastName = req.LastName
	if req.Role != "" {
		user.Role = req.Role
	}
	if req.Active != nil {
		user.Active = *req.Active
	}

	if err := uc.userService.Update(user); err != nil {
		uc.logger.Error("Failed to update user", zap.Uint("user_id", user.ID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      string(user.Role),
		Active:    user.Active,
	})
}

// ChangePassword changes the current user's password
// @Summary Change password
// @Description Changes the current authenticated user's password
// @Tags users
// @Accept json
// @Produce json
// @Security Bearer
// @Param password body ChangePasswordRequest true "Password information"
// @Success 200 {object} map[string]string "Password changed successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Router /users/change-password [post]
func (uc *UserController) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := uc.userService.ChangePassword(userID.(uint), req.CurrentPassword, req.NewPassword)
	if err != nil {
		switch err.Error() {
		case "current password is incorrect":
			c.JSON(http.StatusBadRequest, gin.H{"error": "Current password is incorrect"})
		default:
			uc.logger.Error("Failed to change password", zap.Uint("user_id", userID.(uint)), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to change password"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// DeleteUser deletes a user by ID (admin only)
// @Summary Delete user
// @Description Deletes a user by ID (admin only)
// @Tags users
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "User ID"
// @Success 200 {object} map[string]string "User deleted successfully"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /users/{id} [delete]
func (uc *UserController) DeleteUser(c *gin.Context) {
	// Check if admin
	role, exists := c.Get("user_role")
	if !exists || role != string(models.RoleAdmin) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	// Parse user ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Check if user exists
	if _, err := uc.userService.GetByID(uint(id)); err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		uc.logger.Error("Failed to get user for deletion", zap.Uint("user_id", uint(id)), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		return
	}

	// Delete user
	if err := uc.userService.Delete(uint(id)); err != nil {
		uc.logger.Error("Failed to delete user", zap.Uint("user_id", uint(id)), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}
