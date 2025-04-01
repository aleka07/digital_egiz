package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/digital-egiz/backend/internal/db/models"
	"github.com/digital-egiz/backend/internal/services"
	"github.com/digital-egiz/backend/internal/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ProjectResponse represents a project in responses
type ProjectResponse struct {
	ID          uint                    `json:"id"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	CreatedBy   uint                    `json:"created_by"`
	CreatedAt   string                  `json:"created_at"`
	UpdatedAt   string                  `json:"updated_at"`
	Members     []ProjectMemberResponse `json:"members,omitempty"`
}

// ProjectMemberResponse represents a project member in responses
type ProjectMemberResponse struct {
	ID        uint   `json:"id"`
	ProjectID uint   `json:"project_id"`
	UserID    uint   `json:"user_id"`
	Role      string `json:"role"`
	Email     string `json:"email,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

// CreateProjectRequest represents the request to create a project
type CreateProjectRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// UpdateProjectRequest represents the request to update a project
type UpdateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// AddMemberRequest represents the request to add a member to a project
type AddMemberRequest struct {
	UserID uint               `json:"user_id" binding:"required"`
	Role   models.ProjectRole `json:"role" binding:"required"`
}

// UpdateMemberRequest represents the request to update a member's role
type UpdateMemberRequest struct {
	Role models.ProjectRole `json:"role" binding:"required"`
}

// ProjectController handles project management endpoints
type ProjectController struct {
	projectService *services.ProjectService
	logger         *utils.Logger
}

// NewProjectController creates a new project controller
func NewProjectController(projectService *services.ProjectService, logger *utils.Logger) *ProjectController {
	return &ProjectController{
		projectService: projectService,
		logger:         logger.Named("project_controller"),
	}
}

// RegisterRoutes registers the controller's routes with the router group
func (pc *ProjectController) RegisterRoutes(router *gin.RouterGroup) {
	projects := router.Group("/projects")
	{
		projects.GET("", pc.ListProjects)
		projects.POST("", pc.CreateProject)
		projects.GET("/:id", pc.GetProject)
		projects.PUT("/:id", pc.UpdateProject)
		projects.DELETE("/:id", pc.DeleteProject)

		// Project members
		projects.GET("/:id/members", pc.ListMembers)
		projects.POST("/:id/members", pc.AddMember)
		projects.PUT("/:id/members/:user_id", pc.UpdateMember)
		projects.DELETE("/:id/members/:user_id", pc.RemoveMember)
	}
}

// ListProjects returns a paginated list of projects the user has access to
// @Summary Get a list of projects
// @Description Returns a paginated list of projects the user has access to
// @Tags projects
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "Page number (1-based)" default(1)
// @Param limit query int false "Page size" default(20)
// @Success 200 {object} []ProjectResponse "Project list"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Router /projects [get]
func (pc *ProjectController) ListProjects(c *gin.Context) {
	// Get current user ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	// Parse pagination parameters
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	// Check if admin (admins see all projects)
	userRole, _ := c.Get("user_role")
	isAdmin := userRole == string(models.RoleAdmin)

	var projects []models.Project
	var total int64
	var listErr error

	if isAdmin {
		// Admins see all projects
		projects, total, listErr = pc.projectService.List(page, limit)
	} else {
		// Regular users see only projects they're members of
		projects, total, listErr = pc.projectService.ListByUserID(userID.(uint), page, limit)
	}

	if listErr != nil {
		pc.logger.Error("Failed to list projects", zap.Error(listErr))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve projects"})
		return
	}

	// Map projects to response objects
	response := make([]ProjectResponse, len(projects))
	for i, project := range projects {
		response[i] = ProjectResponse{
			ID:          project.ID,
			Name:        project.Name,
			Description: project.Description,
			CreatedBy:   project.CreatedBy,
			CreatedAt:   project.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   project.UpdatedAt.Format(time.RFC3339),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"projects": response,
		"pagination": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// CreateProject creates a new project
// @Summary Create a new project
// @Description Creates a new project with the current user as owner
// @Tags projects
// @Accept json
// @Produce json
// @Security Bearer
// @Param project body CreateProjectRequest true "Project information"
// @Success 201 {object} ProjectResponse "Created project"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Router /projects [post]
func (pc *ProjectController) CreateProject(c *gin.Context) {
	// Get current user ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create new project
	project := &models.Project{
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   userID.(uint),
	}

	// Save project to database
	if err := pc.projectService.Create(project); err != nil {
		pc.logger.Error("Failed to create project", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		return
	}

	c.JSON(http.StatusCreated, ProjectResponse{
		ID:          project.ID,
		Name:        project.Name,
		Description: project.Description,
		CreatedBy:   project.CreatedBy,
		CreatedAt:   project.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   project.UpdatedAt.Format(time.RFC3339),
	})
}

// GetProject returns a project by ID
// @Summary Get project by ID
// @Description Returns a project by ID if the user has access
// @Tags projects
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Project ID"
// @Success 200 {object} ProjectResponse "Project details"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "Project not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /projects/{id} [get]
func (pc *ProjectController) GetProject(c *gin.Context) {
	// Get project ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	// Get current user ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	// Get the project
	project, err := pc.projectService.GetByID(uint(id))
	if err != nil {
		if err.Error() == "project not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		pc.logger.Error("Failed to get project", zap.Uint("project_id", uint(id)), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve project"})
		return
	}

	// Check if user has access to the project
	userRole, _ := c.Get("user_role")
	isAdmin := userRole == string(models.RoleAdmin)

	if !isAdmin {
		hasAccess, err := pc.projectService.CheckAccess(uint(id), userID.(uint), models.ProjectRoleViewer)
		if err != nil {
			pc.logger.Error("Failed to check project access", zap.Uint("project_id", uint(id)), zap.Uint("user_id", userID.(uint)), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check project access"})
			return
		}

		if !hasAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have access to this project"})
			return
		}
	}

	// Get project members
	members, err := pc.projectService.ListMembers(uint(id))
	if err != nil {
		pc.logger.Error("Failed to get project members", zap.Uint("project_id", uint(id)), zap.Error(err))
		// Don't return an error, continue with the project data
	}

	// Map members to response objects
	memberResponses := make([]ProjectMemberResponse, 0)
	if len(members) > 0 {
		for _, member := range members {
			memberResponses = append(memberResponses, ProjectMemberResponse{
				ID:        member.ID,
				ProjectID: member.ProjectID,
				UserID:    member.UserID,
				Role:      string(member.Role),
				Email:     member.User.Email,
				FirstName: member.User.FirstName,
				LastName:  member.User.LastName,
			})
		}
	}

	c.JSON(http.StatusOK, ProjectResponse{
		ID:          project.ID,
		Name:        project.Name,
		Description: project.Description,
		CreatedBy:   project.CreatedBy,
		CreatedAt:   project.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   project.UpdatedAt.Format(time.RFC3339),
		Members:     memberResponses,
	})
}

// UpdateProject updates a project by ID
// @Summary Update project
// @Description Updates a project by ID if the user has appropriate access
// @Tags projects
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Project ID"
// @Param project body UpdateProjectRequest true "Project information"
// @Success 200 {object} ProjectResponse "Updated project"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "Project not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /projects/{id} [put]
func (pc *ProjectController) UpdateProject(c *gin.Context) {
	// Get project ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	// Get current user ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the project
	project, err := pc.projectService.GetByID(uint(id))
	if err != nil {
		if err.Error() == "project not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		pc.logger.Error("Failed to get project", zap.Uint("project_id", uint(id)), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve project"})
		return
	}

	// Check if user has editor or owner access to the project
	userRole, _ := c.Get("user_role")
	isAdmin := userRole == string(models.RoleAdmin)

	if !isAdmin {
		hasAccess, err := pc.projectService.CheckAccess(uint(id), userID.(uint), models.ProjectRoleEditor)
		if err != nil {
			pc.logger.Error("Failed to check project access", zap.Uint("project_id", uint(id)), zap.Uint("user_id", userID.(uint)), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check project access"})
			return
		}

		if !hasAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to update this project"})
			return
		}
	}

	// Update project fields
	if req.Name != "" {
		project.Name = req.Name
	}
	project.Description = req.Description

	// Save project to database
	if err := pc.projectService.Update(project); err != nil {
		pc.logger.Error("Failed to update project", zap.Uint("project_id", uint(id)), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project"})
		return
	}

	c.JSON(http.StatusOK, ProjectResponse{
		ID:          project.ID,
		Name:        project.Name,
		Description: project.Description,
		CreatedBy:   project.CreatedBy,
		CreatedAt:   project.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   project.UpdatedAt.Format(time.RFC3339),
	})
}

// DeleteProject deletes a project by ID
// @Summary Delete project
// @Description Deletes a project by ID if the user is the owner or an admin
// @Tags projects
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Project ID"
// @Success 200 {object} map[string]string "Project deleted successfully"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "Project not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /projects/{id} [delete]
func (pc *ProjectController) DeleteProject(c *gin.Context) {
	// Get project ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	// Get current user ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	// Check if project exists
	if _, err := pc.projectService.GetByID(uint(id)); err != nil {
		if err.Error() == "project not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		pc.logger.Error("Failed to get project", zap.Uint("project_id", uint(id)), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve project"})
		return
	}

	// Check if user is admin or project owner
	userRole, _ := c.Get("user_role")
	isAdmin := userRole == string(models.RoleAdmin)

	if !isAdmin {
		hasAccess, err := pc.projectService.CheckAccess(uint(id), userID.(uint), models.ProjectRoleOwner)
		if err != nil {
			pc.logger.Error("Failed to check project access", zap.Uint("project_id", uint(id)), zap.Uint("user_id", userID.(uint)), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check project access"})
			return
		}

		if !hasAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only project owners can delete projects"})
			return
		}
	}

	// Delete project
	if err := pc.projectService.Delete(uint(id)); err != nil {
		pc.logger.Error("Failed to delete project", zap.Uint("project_id", uint(id)), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete project"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Project deleted successfully"})
}

// ListMembers returns the members of a project
// @Summary List project members
// @Description Returns the members of a project if the user has access
// @Tags projects
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Project ID"
// @Success 200 {object} []ProjectMemberResponse "Project members"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "Project not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /projects/{id}/members [get]
func (pc *ProjectController) ListMembers(c *gin.Context) {
	// Get project ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	// Get current user ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	// Check if project exists
	if _, err := pc.projectService.GetByID(uint(id)); err != nil {
		if err.Error() == "project not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		pc.logger.Error("Failed to get project", zap.Uint("project_id", uint(id)), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve project"})
		return
	}

	// Check if user has access to the project
	userRole, _ := c.Get("user_role")
	isAdmin := userRole == string(models.RoleAdmin)

	if !isAdmin {
		hasAccess, err := pc.projectService.CheckAccess(uint(id), userID.(uint), models.ProjectRoleViewer)
		if err != nil {
			pc.logger.Error("Failed to check project access", zap.Uint("project_id", uint(id)), zap.Uint("user_id", userID.(uint)), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check project access"})
			return
		}

		if !hasAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have access to this project"})
			return
		}
	}

	// Get project members
	members, err := pc.projectService.ListMembers(uint(id))
	if err != nil {
		pc.logger.Error("Failed to list project members", zap.Uint("project_id", uint(id)), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve project members"})
		return
	}

	// Map members to response objects
	response := make([]ProjectMemberResponse, len(members))
	for i, member := range members {
		response[i] = ProjectMemberResponse{
			ID:        member.ID,
			ProjectID: member.ProjectID,
			UserID:    member.UserID,
			Role:      string(member.Role),
			Email:     member.User.Email,
			FirstName: member.User.FirstName,
			LastName:  member.User.LastName,
		}
	}

	c.JSON(http.StatusOK, gin.H{"members": response})
}

// AddMember adds a member to a project
// @Summary Add project member
// @Description Adds a member to a project if the user has appropriate access
// @Tags projects
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Project ID"
// @Param member body AddMemberRequest true "Member information"
// @Success 201 {object} ProjectMemberResponse "Added member"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "Project not found"
// @Failure 409 {object} map[string]string "Member already exists"
// @Failure 500 {object} map[string]string "Server error"
// @Router /projects/{id}/members [post]
func (pc *ProjectController) AddMember(c *gin.Context) {
	// Get project ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	// Get current user ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if project exists
	if _, err := pc.projectService.GetByID(uint(id)); err != nil {
		if err.Error() == "project not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		pc.logger.Error("Failed to get project", zap.Uint("project_id", uint(id)), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve project"})
		return
	}

	// Check if user has owner access to the project
	userRole, _ := c.Get("user_role")
	isAdmin := userRole == string(models.RoleAdmin)

	if !isAdmin {
		hasAccess, err := pc.projectService.CheckAccess(uint(id), userID.(uint), models.ProjectRoleOwner)
		if err != nil {
			pc.logger.Error("Failed to check project access", zap.Uint("project_id", uint(id)), zap.Uint("user_id", userID.(uint)), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check project access"})
			return
		}

		if !hasAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only project owners can add members"})
			return
		}
	}

	// Add member to project
	member, err := pc.projectService.AddMember(uint(id), req.UserID, req.Role)
	if err != nil {
		if err.Error() == "user already a member of project" {
			c.JSON(http.StatusConflict, gin.H{"error": "User is already a member of this project"})
			return
		}
		pc.logger.Error("Failed to add member to project",
			zap.Uint("project_id", uint(id)),
			zap.Uint("user_id", req.UserID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add member to project"})
		return
	}

	c.JSON(http.StatusCreated, ProjectMemberResponse{
		ID:        member.ID,
		ProjectID: member.ProjectID,
		UserID:    member.UserID,
		Role:      string(member.Role),
		Email:     member.User.Email,
		FirstName: member.User.FirstName,
		LastName:  member.User.LastName,
	})
}

// UpdateMember updates a member's role in a project
// @Summary Update project member
// @Description Updates a member's role in a project if the user has appropriate access
// @Tags projects
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Project ID"
// @Param user_id path int true "User ID"
// @Param member body UpdateMemberRequest true "Member information"
// @Success 200 {object} ProjectMemberResponse "Updated member"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "Project or member not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /projects/{id}/members/{user_id} [put]
func (pc *ProjectController) UpdateMember(c *gin.Context) {
	// Get project ID and user ID
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	memberUserID, err := strconv.ParseUint(c.Param("user_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get current user ID
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req UpdateMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if project exists
	if _, err := pc.projectService.GetByID(uint(projectID)); err != nil {
		if err.Error() == "project not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		pc.logger.Error("Failed to get project", zap.Uint("project_id", uint(projectID)), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve project"})
		return
	}

	// Check if user has owner access to the project
	userRole, _ := c.Get("user_role")
	isAdmin := userRole == string(models.RoleAdmin)

	if !isAdmin {
		hasAccess, err := pc.projectService.CheckAccess(uint(projectID), currentUserID.(uint), models.ProjectRoleOwner)
		if err != nil {
			pc.logger.Error("Failed to check project access",
				zap.Uint("project_id", uint(projectID)),
				zap.Uint("user_id", currentUserID.(uint)),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check project access"})
			return
		}

		if !hasAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only project owners can update member roles"})
			return
		}
	}

	// Update member role
	member, err := pc.projectService.UpdateMemberRole(uint(projectID), uint(memberUserID), req.Role)
	if err != nil {
		if err.Error() == "member not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Member not found in project"})
			return
		}
		if err.Error() == "cannot demote the last owner" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot change the role of the last owner"})
			return
		}
		pc.logger.Error("Failed to update member role",
			zap.Uint("project_id", uint(projectID)),
			zap.Uint("user_id", uint(memberUserID)),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update member role"})
		return
	}

	c.JSON(http.StatusOK, ProjectMemberResponse{
		ID:        member.ID,
		ProjectID: member.ProjectID,
		UserID:    member.UserID,
		Role:      string(member.Role),
		Email:     member.User.Email,
		FirstName: member.User.FirstName,
		LastName:  member.User.LastName,
	})
}

// RemoveMember removes a member from a project
// @Summary Remove project member
// @Description Removes a member from a project if the user has appropriate access
// @Tags projects
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Project ID"
// @Param user_id path int true "User ID"
// @Success 200 {object} map[string]string "Member removed successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "Project or member not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /projects/{id}/members/{user_id} [delete]
func (pc *ProjectController) RemoveMember(c *gin.Context) {
	// Get project ID and user ID
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	memberUserID, err := strconv.ParseUint(c.Param("user_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get current user ID
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	// Check if project exists
	if _, err := pc.projectService.GetByID(uint(projectID)); err != nil {
		if err.Error() == "project not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		pc.logger.Error("Failed to get project", zap.Uint("project_id", uint(projectID)), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve project"})
		return
	}

	// Check if user has owner access to the project
	userRole, _ := c.Get("user_role")
	isAdmin := userRole == string(models.RoleAdmin)

	if !isAdmin {
		hasAccess, err := pc.projectService.CheckAccess(uint(projectID), currentUserID.(uint), models.ProjectRoleOwner)
		if err != nil {
			pc.logger.Error("Failed to check project access",
				zap.Uint("project_id", uint(projectID)),
				zap.Uint("user_id", currentUserID.(uint)),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check project access"})
			return
		}

		if !hasAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only project owners can remove members"})
			return
		}
	}

	// Remove member from project
	if err := pc.projectService.RemoveMember(uint(projectID), uint(memberUserID)); err != nil {
		if err.Error() == "member not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Member not found in project"})
			return
		}
		if err.Error() == "cannot remove the last owner" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot remove the last owner from the project"})
			return
		}
		pc.logger.Error("Failed to remove member",
			zap.Uint("project_id", uint(projectID)),
			zap.Uint("user_id", uint(memberUserID)),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove member from project"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member removed successfully"})
}
