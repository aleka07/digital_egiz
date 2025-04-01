package services

import (
	"errors"

	"github.com/digital-egiz/backend/internal/db"
	"github.com/digital-egiz/backend/internal/db/models"
	"github.com/digital-egiz/backend/internal/db/repository"
	"github.com/digital-egiz/backend/internal/utils"
	"go.uber.org/zap"
)

// TwinService handles twin-related business logic
type TwinService struct {
	db           *db.Database
	logger       *utils.Logger
	twinRepo     repository.TwinRepository
	twinTypeRepo repository.TwinTypeRepository
	projectRepo  repository.ProjectRepository
	userRepo     repository.UserRepository
}

// NewTwinService creates a new twin service
func NewTwinService(db *db.Database, logger *utils.Logger) *TwinService {
	repoFactory := repository.NewRepositoryFactory(db.DB)
	return &TwinService{
		db:           db,
		logger:       logger.Named("twin_service"),
		twinRepo:     repoFactory.Twin(),
		twinTypeRepo: repoFactory.TwinType(),
		projectRepo:  repoFactory.Project(),
		userRepo:     repoFactory.User(),
	}
}

// Create adds a new twin
func (s *TwinService) Create(twin *models.Twin) error {
	// Validate twin data
	if twin.Name == "" {
		return errors.New("twin name is required")
	}

	if twin.DittoID == "" {
		return errors.New("ditto ID is required")
	}

	if twin.TypeID == 0 {
		return errors.New("twin type is required")
	}

	if twin.ProjectID == 0 {
		return errors.New("project is required")
	}

	if twin.CreatedBy == 0 {
		return errors.New("creator is required")
	}

	// Verify user exists
	_, err := s.userRepo.GetByID(twin.CreatedBy)
	if err != nil {
		s.logger.Error("Failed to verify user exists", zap.Uint("user_id", twin.CreatedBy), zap.Error(err))
		return errors.New("invalid creator user")
	}

	// Verify twin type exists
	_, err = s.twinTypeRepo.GetByID(twin.TypeID)
	if err != nil {
		s.logger.Error("Failed to verify twin type exists", zap.Uint("type_id", twin.TypeID), zap.Error(err))
		return errors.New("invalid twin type")
	}

	// Verify project exists
	_, err = s.projectRepo.GetByID(twin.ProjectID)
	if err != nil {
		s.logger.Error("Failed to verify project exists", zap.Uint("project_id", twin.ProjectID), zap.Error(err))
		return errors.New("invalid project")
	}

	// Check if twin with same Ditto ID already exists
	_, err = s.twinRepo.GetByDittoID(twin.DittoID)
	if err == nil {
		return errors.New("twin with this Ditto ID already exists")
	} else if !errors.Is(err, repository.ErrNotFound) {
		s.logger.Error("Error checking twin existence", zap.String("ditto_id", twin.DittoID), zap.Error(err))
		return errors.New("database error")
	}

	// Create twin
	err = s.twinRepo.Create(twin)
	if err != nil {
		s.logger.Error("Failed to create twin", zap.Error(err))
		return errors.New("failed to create twin")
	}

	return nil
}

// GetByID retrieves a twin by ID
func (s *TwinService) GetByID(id uint) (*models.Twin, error) {
	twin, err := s.twinRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errors.New("twin not found")
		}
		s.logger.Error("Failed to get twin", zap.Uint("id", id), zap.Error(err))
		return nil, errors.New("database error")
	}

	return twin, nil
}

// GetByDittoID retrieves a twin by Ditto ID
func (s *TwinService) GetByDittoID(dittoID string) (*models.Twin, error) {
	twin, err := s.twinRepo.GetByDittoID(dittoID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errors.New("twin not found")
		}
		s.logger.Error("Failed to get twin by Ditto ID", zap.String("ditto_id", dittoID), zap.Error(err))
		return nil, errors.New("database error")
	}

	return twin, nil
}

// ListByProject returns a paginated list of twins in a project
func (s *TwinService) ListByProject(projectID uint, page, pageSize int) ([]models.Twin, int64, error) {
	// Verify project exists
	_, err := s.projectRepo.GetByID(projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, 0, errors.New("project not found")
		}
		s.logger.Error("Failed to verify project exists", zap.Uint("project_id", projectID), zap.Error(err))
		return nil, 0, errors.New("database error")
	}

	offset := (page - 1) * pageSize
	twins, total, err := s.twinRepo.ListByProjectID(projectID, offset, pageSize)
	if err != nil {
		s.logger.Error("Failed to list twins", zap.Uint("project_id", projectID), zap.Error(err))
		return nil, 0, errors.New("database error")
	}

	return twins, total, nil
}

// Update updates a twin's information
func (s *TwinService) Update(twin *models.Twin) error {
	// Validate twin data
	if twin.ID == 0 {
		return errors.New("twin ID is required")
	}

	if twin.Name == "" {
		return errors.New("twin name is required")
	}

	if twin.DittoID == "" {
		return errors.New("ditto ID is required")
	}

	// Check if twin exists
	existingTwin, err := s.twinRepo.GetByID(twin.ID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return errors.New("twin not found")
		}
		s.logger.Error("Failed to get twin", zap.Uint("id", twin.ID), zap.Error(err))
		return errors.New("database error")
	}

	// If Ditto ID is changed, check if new ID already exists
	if twin.DittoID != existingTwin.DittoID {
		existingWithDittoID, err := s.twinRepo.GetByDittoID(twin.DittoID)
		if err == nil && existingWithDittoID.ID != twin.ID {
			return errors.New("twin with this Ditto ID already exists")
		} else if err != nil && !errors.Is(err, repository.ErrNotFound) {
			s.logger.Error("Error checking twin existence", zap.String("ditto_id", twin.DittoID), zap.Error(err))
			return errors.New("database error")
		}
	}

	// Maintain original values for fields that shouldn't be updated
	twin.CreatedBy = existingTwin.CreatedBy
	twin.CreatedAt = existingTwin.CreatedAt

	// Update twin
	err = s.twinRepo.Update(twin)
	if err != nil {
		s.logger.Error("Failed to update twin", zap.Uint("id", twin.ID), zap.Error(err))
		return errors.New("failed to update twin")
	}

	return nil
}

// Delete soft-deletes a twin
func (s *TwinService) Delete(id uint) error {
	// Check if twin exists
	_, err := s.twinRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return errors.New("twin not found")
		}
		s.logger.Error("Failed to get twin", zap.Uint("id", id), zap.Error(err))
		return errors.New("database error")
	}

	// TODO: Consider deleting from Ditto as well if needed

	// Delete twin
	err = s.twinRepo.Delete(id)
	if err != nil {
		s.logger.Error("Failed to delete twin", zap.Uint("id", id), zap.Error(err))
		return errors.New("failed to delete twin")
	}

	return nil
}

// CreateModelBinding creates a model binding for a twin
func (s *TwinService) CreateModelBinding(binding *models.ModelBinding) error {
	if binding.TwinID == 0 {
		return errors.New("twin ID is required")
	}

	if binding.PartID == "" {
		return errors.New("part ID is required")
	}

	if binding.FeaturePath == "" {
		return errors.New("feature path is required")
	}

	if binding.BindingType == "" {
		return errors.New("binding type is required")
	}

	// Verify twin exists
	_, err := s.twinRepo.GetByID(binding.TwinID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return errors.New("twin not found")
		}
		s.logger.Error("Failed to verify twin exists", zap.Uint("twin_id", binding.TwinID), zap.Error(err))
		return errors.New("database error")
	}

	// Create binding
	err = s.twinRepo.CreateModelBinding(binding)
	if err != nil {
		s.logger.Error("Failed to create model binding", zap.Error(err))
		return errors.New("failed to create model binding")
	}

	return nil
}

// ListModelBindings lists model bindings for a twin
func (s *TwinService) ListModelBindings(twinID uint) ([]models.ModelBinding, error) {
	// Verify twin exists
	_, err := s.twinRepo.GetByID(twinID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errors.New("twin not found")
		}
		s.logger.Error("Failed to verify twin exists", zap.Uint("twin_id", twinID), zap.Error(err))
		return nil, errors.New("database error")
	}

	// Get bindings
	bindings, err := s.twinRepo.ListModelBindings(twinID)
	if err != nil {
		s.logger.Error("Failed to list model bindings", zap.Uint("twin_id", twinID), zap.Error(err))
		return nil, errors.New("failed to retrieve model bindings")
	}

	return bindings, nil
}

// UpdateModelBinding updates a model binding
func (s *TwinService) UpdateModelBinding(binding *models.ModelBinding) error {
	if binding.ID == 0 {
		return errors.New("binding ID is required")
	}

	if binding.PartID == "" {
		return errors.New("part ID is required")
	}

	if binding.FeaturePath == "" {
		return errors.New("feature path is required")
	}

	if binding.BindingType == "" {
		return errors.New("binding type is required")
	}

	// Update binding
	err := s.twinRepo.UpdateModelBinding(binding)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return errors.New("binding not found")
		}
		s.logger.Error("Failed to update model binding", zap.Uint("id", binding.ID), zap.Error(err))
		return errors.New("failed to update model binding")
	}

	return nil
}

// DeleteModelBinding deletes a model binding
func (s *TwinService) DeleteModelBinding(id uint) error {
	err := s.twinRepo.DeleteModelBinding(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return errors.New("binding not found")
		}
		s.logger.Error("Failed to delete model binding", zap.Uint("id", id), zap.Error(err))
		return errors.New("failed to delete model binding")
	}

	return nil
}
