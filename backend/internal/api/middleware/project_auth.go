package middleware

import (
	"net/http"
	"strconv"

	"github.com/digital-egiz/backend/internal/db/models"
	"github.com/digital-egiz/backend/internal/services"
	"github.com/gin-gonic/gin"
)

// ProjectAuthMiddleware handles project-level authorization
type ProjectAuthMiddleware struct {
	projectService *services.ProjectService
}

// NewProjectAuthMiddleware creates a new project authorization middleware
func NewProjectAuthMiddleware(projectService *services.ProjectService) *ProjectAuthMiddleware {
	return &ProjectAuthMiddleware{
		projectService: projectService,
	}
}

// RequireProjectAccess ensures the user has the required access level to a project
func (pm *ProjectAuthMiddleware) RequireProjectAccess(minRole models.ProjectRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get project ID from URL parameter
		projectID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
			return
		}

		// Get user ID from context (set by RequireAuth middleware)
		userID, exists := c.Get("user_id")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}

		// Check if user has admin role (admins have access to all projects)
		userRole, _ := c.Get("user_role")
		if userRole == string(models.RoleAdmin) {
			c.Next()
			return
		}

		// Check project access for regular users
		hasAccess, err := pm.projectService.CheckUserAccess(uint(projectID), userID.(uint), minRole)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify project access"})
			return
		}

		if !hasAccess {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions for this project",
				"code":  "project_access_denied",
			})
			return
		}

		c.Next()
	}
}

// RequireProjectOwner ensures the user is an owner of the project
func (pm *ProjectAuthMiddleware) RequireProjectOwner() gin.HandlerFunc {
	return pm.RequireProjectAccess(models.ProjectRoleOwner)
}

// RequireProjectEditor ensures the user is at least an editor of the project
func (pm *ProjectAuthMiddleware) RequireProjectEditor() gin.HandlerFunc {
	return pm.RequireProjectAccess(models.ProjectRoleEditor)
}

// RequireProjectViewer ensures the user is at least a viewer of the project
func (pm *ProjectAuthMiddleware) RequireProjectViewer() gin.HandlerFunc {
	return pm.RequireProjectAccess(models.ProjectRoleViewer)
}
