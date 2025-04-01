package middleware

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/digital-egiz/backend/internal/db/models"
	"github.com/digital-egiz/backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ProjectAuthMiddleware provides middleware for project-based authorization
type ProjectAuthMiddleware struct {
	projectService *services.ProjectService
	logger         *zap.Logger
}

// NewProjectAuthMiddleware creates a new project authorization middleware
func NewProjectAuthMiddleware(projectService *services.ProjectService) *ProjectAuthMiddleware {
	return &ProjectAuthMiddleware{
		projectService: projectService,
	}
}

// RequireProjectRole middleware ensures the user has the required role in the project
func (pa *ProjectAuthMiddleware) RequireProjectRole(requiredRoles ...models.ProjectRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get project ID from URL
		projectIDStr := c.Param("id")
		if projectIDStr == "" {
			// Try to get from query parameters
			projectIDStr = c.Query("projectId")
			if projectIDStr == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
				c.Abort()
				return
			}
		}

		projectID, err := strconv.ParseUint(projectIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
			c.Abort()
			return
		}

		// Get user ID from context
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User is not authenticated"})
			c.Abort()
			return
		}

		// Check if user is admin (admins have access to all projects)
		userRole, _ := c.Get("user_role")
		if userRole == string(models.RoleAdmin) {
			// Admin has all permissions
			c.Set("project_role", string(models.ProjectRoleOwner))
			c.Next()
			return
		}

		// Get project to check membership
		_, err = pa.projectService.GetByID(uint(projectID))
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
				c.Abort()
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check project"})
			c.Abort()
			return
		}

		// Check if user is a member and get their role
		members, err := pa.projectService.ListMembers(uint(projectID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check project membership"})
			c.Abort()
			return
		}

		// Find user's role in the project
		var userProjectRole models.ProjectRole
		userFound := false

		for _, member := range members {
			if member.UserID == userID.(uint) {
				userProjectRole = member.Role
				userFound = true
				break
			}
		}

		if !userFound {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have access to this project"})
			c.Abort()
			return
		}

		// Check if user has one of the required roles
		hasRole := false
		for _, requiredRole := range requiredRoles {
			if userProjectRole == requiredRole || userProjectRole == models.ProjectRoleOwner {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions for this project"})
			c.Abort()
			return
		}

		// Add project role to context for potential use downstream
		c.Set("project_role", string(userProjectRole))

		// Continue with the request
		c.Next()
	}
}

// RequireProjectOwner middleware ensures the user is the project owner
func (pa *ProjectAuthMiddleware) RequireProjectOwner() gin.HandlerFunc {
	return pa.RequireProjectRole(models.ProjectRoleOwner)
}

// RequireProjectEditor middleware ensures the user is at least a project editor
func (pa *ProjectAuthMiddleware) RequireProjectEditor() gin.HandlerFunc {
	return pa.RequireProjectRole(models.ProjectRoleOwner, models.ProjectRoleEditor)
}

// RequireProjectViewer middleware ensures the user has any level of access to the project
func (pa *ProjectAuthMiddleware) RequireProjectViewer() gin.HandlerFunc {
	return pa.RequireProjectRole(models.ProjectRoleOwner, models.ProjectRoleEditor, models.ProjectRoleViewer)
}
