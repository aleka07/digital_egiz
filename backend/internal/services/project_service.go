package services

import (
	"errors"

	"github.com/digital-egiz/backend/internal/db"
	"github.com/digital-egiz/backend/internal/db/models"
	"github.com/digital-egiz/backend/internal/db/repository"
	"github.com/digital-egiz/backend/internal/utils"
	"go.uber.org/zap"
)

// ProjectService handles project-related business logic
type ProjectService struct {
	db          *db.Database
	logger      *utils.Logger
	projectRepo repository.ProjectRepository
	userRepo    repository.UserRepository
}

// NewProjectService creates a new project service
func NewProjectService(db *db.Database, logger *utils.Logger) *ProjectService {
	repoFactory := repository.NewRepositoryFactory(db.DB)
	return &ProjectService{
		db:          db,
		logger:      logger.Named("project_service"),
		projectRepo: repoFactory.Project(),
		userRepo:    repoFactory.User(),
	}
}

// Create adds a new project and adds the creator as an owner
func (s *ProjectService) Create(project *models.Project) error {
	// Validate project data
	if project.Name == "" {
		return errors.New("project name is required")
	}

	if project.CreatedBy == 0 {
		return errors.New("project creator is required")
	}

	// Verify user exists
	_, err := s.userRepo.GetByID(project.CreatedBy)
	if err != nil {
		s.logger.Error("Failed to verify user exists", zap.Uint("user_id", project.CreatedBy), zap.Error(err))
		return errors.New("invalid creator user")
	}

	// Create project
	err = s.projectRepo.Create(project)
	if err != nil {
		s.logger.Error("Failed to create project", zap.Error(err))
		return errors.New("failed to create project")
	}

	return nil
}

// GetByID retrieves a project by ID
func (s *ProjectService) GetByID(id uint) (*models.Project, error) {
	project, err := s.projectRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errors.New("project not found")
		}
		s.logger.Error("Failed to get project", zap.Uint("id", id), zap.Error(err))
		return nil, errors.New("database error")
	}

	return project, nil
}

// List returns a paginated list of all projects
func (s *ProjectService) List(page, pageSize int) ([]models.Project, int64, error) {
	offset := (page - 1) * pageSize
	projects, total, err := s.projectRepo.List(offset, pageSize)
	if err != nil {
		s.logger.Error("Failed to list projects", zap.Error(err))
		return nil, 0, errors.New("database error")
	}

	return projects, total, nil
}

// ListByUserID returns a paginated list of projects where the user is a member
func (s *ProjectService) ListByUserID(userID uint, page, pageSize int) ([]models.Project, int64, error) {
	offset := (page - 1) * pageSize
	projects, total, err := s.projectRepo.ListByUserID(userID, offset, pageSize)
	if err != nil {
		s.logger.Error("Failed to list projects by user ID", zap.Uint("user_id", userID), zap.Error(err))
		return nil, 0, errors.New("database error")
	}

	return projects, total, nil
}

// Update updates a project's information
func (s *ProjectService) Update(project *models.Project) error {
	// Validate project data
	if project.ID == 0 {
		return errors.New("project ID is required")
	}

	if project.Name == "" {
		return errors.New("project name is required")
	}

	err := s.projectRepo.Update(project)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return errors.New("project not found")
		}
		s.logger.Error("Failed to update project", zap.Uint("id", project.ID), zap.Error(err))
		return errors.New("failed to update project")
	}

	return nil
}

// Delete soft-deletes a project
func (s *ProjectService) Delete(id uint) error {
	err := s.projectRepo.Delete(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return errors.New("project not found")
		}
		s.logger.Error("Failed to delete project", zap.Uint("id", id), zap.Error(err))
		return errors.New("failed to delete project")
	}

	return nil
}

// CheckAccess checks if a user has the required access level for a project
func (s *ProjectService) CheckAccess(projectID, userID uint, minRequiredRole models.ProjectRole) (bool, error) {
	hasAccess, err := s.projectRepo.CheckUserAccess(projectID, userID, minRequiredRole)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return false, errors.New("project not found")
		}
		s.logger.Error("Failed to check project access",
			zap.Uint("project_id", projectID),
			zap.Uint("user_id", userID),
			zap.Error(err))
		return false, errors.New("database error")
	}

	return hasAccess, nil
}

// ListMembers returns all members of a project
func (s *ProjectService) ListMembers(projectID uint) ([]models.ProjectMember, error) {
	members, err := s.projectRepo.ListMembers(projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errors.New("project not found")
		}
		s.logger.Error("Failed to list project members", zap.Uint("project_id", projectID), zap.Error(err))
		return nil, errors.New("database error")
	}

	return members, nil
}

// AddMember adds a user to a project with a specific role
func (s *ProjectService) AddMember(projectID, userID uint, role models.ProjectRole) (*models.ProjectMember, error) {
	// Validate user exists
	_, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errors.New("user not found")
		}
		s.logger.Error("Failed to verify user exists", zap.Uint("user_id", userID), zap.Error(err))
		return nil, errors.New("database error")
	}

	// Add member
	err = s.projectRepo.AddMember(projectID, userID, role)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, errors.New("user already a member of project")
		}
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errors.New("project not found")
		}
		s.logger.Error("Failed to add member to project",
			zap.Uint("project_id", projectID),
			zap.Uint("user_id", userID),
			zap.Error(err))
		return nil, errors.New("failed to add member to project")
	}

	// Get the newly created member
	member, err := s.projectRepo.GetMember(projectID, userID)
	if err != nil {
		s.logger.Error("Failed to get project member after adding",
			zap.Uint("project_id", projectID),
			zap.Uint("user_id", userID),
			zap.Error(err))
		return nil, errors.New("member added but failed to retrieve details")
	}

	return member, nil
}

// UpdateMemberRole updates a member's role in a project
func (s *ProjectService) UpdateMemberRole(projectID, userID uint, role models.ProjectRole) (*models.ProjectMember, error) {
	// Update member role
	err := s.projectRepo.UpdateMemberRole(projectID, userID, role)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errors.New("member not found")
		}
		if errors.Is(err, repository.ErrInvalidInput) {
			return nil, errors.New("cannot demote the last owner")
		}
		s.logger.Error("Failed to update member role",
			zap.Uint("project_id", projectID),
			zap.Uint("user_id", userID),
			zap.Error(err))
		return nil, errors.New("failed to update member role")
	}

	// Get the updated member
	member, err := s.projectRepo.GetMember(projectID, userID)
	if err != nil {
		s.logger.Error("Failed to get project member after update",
			zap.Uint("project_id", projectID),
			zap.Uint("user_id", userID),
			zap.Error(err))
		return nil, errors.New("role updated but failed to retrieve member details")
	}

	return member, nil
}

// RemoveMember removes a user from a project
func (s *ProjectService) RemoveMember(projectID, userID uint) error {
	err := s.projectRepo.RemoveMember(projectID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return errors.New("member not found")
		}
		if errors.Is(err, repository.ErrInvalidInput) {
			return errors.New("cannot remove the last owner")
		}
		s.logger.Error("Failed to remove member from project",
			zap.Uint("project_id", projectID),
			zap.Uint("user_id", userID),
			zap.Error(err))
		return errors.New("failed to remove member from project")
	}

	return nil
}

// CheckUserAccess checks if a user has the required access level for a project
func (s *ProjectService) CheckUserAccess(projectID, userID uint, minRole models.ProjectRole) (bool, error) {
	// Check if project exists
	_, err := s.projectRepo.GetByID(projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return false, errors.New("project not found")
		}
		s.logger.Error("Failed to verify project exists",
			zap.Uint("project_id", projectID),
			zap.Error(err))
		return false, errors.New("database error")
	}

	// Check user's access level
	hasAccess, err := s.projectRepo.CheckUserAccess(projectID, userID, minRole)
	if err != nil {
		s.logger.Error("Failed to check user access to project",
			zap.Uint("project_id", projectID),
			zap.Uint("user_id", userID),
			zap.Error(err))
		return false, errors.New("failed to check project access")
	}

	return hasAccess, nil
}
