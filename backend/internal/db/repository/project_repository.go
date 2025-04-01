package repository

import (
	"github.com/digital-egiz/backend/internal/db/models"
	"gorm.io/gorm"
)

// ProjectRepository defines operations for managing projects
type ProjectRepository interface {
	Repository
	Create(project *models.Project) error
	GetByID(id uint) (*models.Project, error)
	List(offset, limit int) ([]models.Project, int64, error)
	ListByUserID(userID uint, offset, limit int) ([]models.Project, int64, error)
	Update(project *models.Project) error
	Delete(id uint) error

	// Project members methods
	AddMember(projectID, userID uint, role models.ProjectRole) error
	UpdateMemberRole(projectID, userID uint, role models.ProjectRole) error
	RemoveMember(projectID, userID uint) error
	ListMembers(projectID uint) ([]models.ProjectMember, error)
	GetMember(projectID, userID uint) (*models.ProjectMember, error)
	CheckUserAccess(projectID, userID uint, minRequiredRole models.ProjectRole) (bool, error)
}

// projectRepository implements ProjectRepository
type projectRepository struct {
	BaseRepository
}

// NewProjectRepository creates a new project repository
func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &projectRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Create adds a new project to the database
func (r *projectRepository) Create(project *models.Project) error {
	// Start a transaction
	tx := r.GetDB().Begin()
	if tx.Error != nil {
		return r.handleError(tx.Error)
	}

	// Create the project
	if err := tx.Create(project).Error; err != nil {
		tx.Rollback()
		return r.handleError(err)
	}

	// Add the creator as an owner
	member := models.ProjectMember{
		ProjectID: project.ID,
		UserID:    project.CreatedBy,
		Role:      models.ProjectRoleOwner,
	}

	if err := tx.Create(&member).Error; err != nil {
		tx.Rollback()
		return r.handleError(err)
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return r.handleError(err)
	}

	return nil
}

// GetByID retrieves a project by ID
func (r *projectRepository) GetByID(id uint) (*models.Project, error) {
	var project models.Project
	err := r.GetDB().Where("id = ?", id).First(&project).Error
	if err != nil {
		return nil, r.handleError(err)
	}
	return &project, nil
}

// List retrieves a paginated list of projects
func (r *projectRepository) List(offset, limit int) ([]models.Project, int64, error) {
	var projects []models.Project
	var total int64

	// Get total count
	if err := r.GetDB().Model(&models.Project{}).Count(&total).Error; err != nil {
		return nil, 0, r.handleError(err)
	}

	// Get paginated projects
	err := r.GetDB().Offset(offset).Limit(limit).Order("id asc").Find(&projects).Error
	if err != nil {
		return nil, 0, r.handleError(err)
	}

	return projects, total, nil
}

// ListByUserID retrieves projects where the user is a member
func (r *projectRepository) ListByUserID(userID uint, offset, limit int) ([]models.Project, int64, error) {
	var projects []models.Project
	var total int64

	// Get total count of projects the user is a member of
	query := r.GetDB().Model(&models.Project{}).
		Joins("JOIN project_members ON project_members.project_id = projects.id").
		Where("project_members.user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, r.handleError(err)
	}

	// Get paginated projects the user is a member of
	err := query.Offset(offset).Limit(limit).Order("projects.id asc").Find(&projects).Error
	if err != nil {
		return nil, 0, r.handleError(err)
	}

	return projects, total, nil
}

// Update updates a project's information
func (r *projectRepository) Update(project *models.Project) error {
	// Check if project exists
	var existingProject models.Project
	if err := r.GetDB().Where("id = ?", project.ID).First(&existingProject).Error; err != nil {
		return r.handleError(err)
	}

	// Update only allowed fields
	err := r.GetDB().Model(project).Updates(map[string]interface{}{
		"name":        project.Name,
		"description": project.Description,
	}).Error

	return r.handleError(err)
}

// Delete soft-deletes a project
func (r *projectRepository) Delete(id uint) error {
	result := r.GetDB().Delete(&models.Project{}, id)
	if result.Error != nil {
		return r.handleError(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// AddMember adds a user to a project with the specified role
func (r *projectRepository) AddMember(projectID, userID uint, role models.ProjectRole) error {
	// Check if project exists
	if _, err := r.GetByID(projectID); err != nil {
		return err
	}

	// Check if user is already a member
	var count int64
	if err := r.GetDB().Model(&models.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Count(&count).Error; err != nil {
		return r.handleError(err)
	}

	if count > 0 {
		return ErrConflict
	}

	// Add member
	member := models.ProjectMember{
		ProjectID: projectID,
		UserID:    userID,
		Role:      role,
	}

	err := r.GetDB().Create(&member).Error
	return r.handleError(err)
}

// UpdateMemberRole updates a user's role in a project
func (r *projectRepository) UpdateMemberRole(projectID, userID uint, role models.ProjectRole) error {
	result := r.GetDB().Model(&models.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Update("role", role)

	if result.Error != nil {
		return r.handleError(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// RemoveMember removes a user from a project
func (r *projectRepository) RemoveMember(projectID, userID uint) error {
	// First check if this is the last owner
	var ownerCount int64
	if err := r.GetDB().Model(&models.ProjectMember{}).
		Where("project_id = ? AND role = ?", projectID, models.ProjectRoleOwner).
		Count(&ownerCount).Error; err != nil {
		return r.handleError(err)
	}

	// Check if we're removing an owner
	var targetMember models.ProjectMember
	if err := r.GetDB().Where("project_id = ? AND user_id = ?", projectID, userID).
		First(&targetMember).Error; err != nil {
		return r.handleError(err)
	}

	// Can't remove the last owner
	if ownerCount == 1 && targetMember.Role == models.ProjectRoleOwner {
		return ErrInvalidInput
	}

	// Remove member
	result := r.GetDB().Where("project_id = ? AND user_id = ?", projectID, userID).
		Delete(&models.ProjectMember{})

	if result.Error != nil {
		return r.handleError(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ListMembers lists all members of a project
func (r *projectRepository) ListMembers(projectID uint) ([]models.ProjectMember, error) {
	var members []models.ProjectMember
	err := r.GetDB().Preload("User").
		Where("project_id = ?", projectID).
		Find(&members).Error
	if err != nil {
		return nil, r.handleError(err)
	}
	return members, nil
}

// GetMember gets a specific member from a project
func (r *projectRepository) GetMember(projectID, userID uint) (*models.ProjectMember, error) {
	var member models.ProjectMember
	err := r.GetDB().Preload("User").
		Where("project_id = ? AND user_id = ?", projectID, userID).
		First(&member).Error
	if err != nil {
		return nil, r.handleError(err)
	}
	return &member, nil
}

// CheckUserAccess checks if a user has the required access level for a project
func (r *projectRepository) CheckUserAccess(projectID, userID uint, minRequiredRole models.ProjectRole) (bool, error) {
	var member models.ProjectMember
	err := r.GetDB().Where("project_id = ? AND user_id = ?", projectID, userID).
		First(&member).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, r.handleError(err)
	}

	// Check if user's role has sufficient permissions
	switch minRequiredRole {
	case models.ProjectRoleViewer:
		return true, nil
	case models.ProjectRoleEditor:
		return member.Role == models.ProjectRoleEditor || member.Role == models.ProjectRoleOwner, nil
	case models.ProjectRoleOwner:
		return member.Role == models.ProjectRoleOwner, nil
	default:
		return false, nil
	}
}
