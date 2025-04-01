package services

import (
	"errors"

	"github.com/digital-egiz/backend/internal/db"
	"github.com/digital-egiz/backend/internal/db/models"
	"github.com/digital-egiz/backend/internal/db/repository"
	"github.com/digital-egiz/backend/internal/utils"
	"go.uber.org/zap"
)

// TwinTypeService handles twin type-related business logic
type TwinTypeService struct {
	db           *db.Database
	logger       *utils.Logger
	twinTypeRepo repository.TwinTypeRepository
	userRepo     repository.UserRepository
}

// NewTwinTypeService creates a new twin type service
func NewTwinTypeService(db *db.Database, logger *utils.Logger) *TwinTypeService {
	repoFactory := repository.NewRepositoryFactory(db.DB)
	return &TwinTypeService{
		db:           db,
		logger:       logger.Named("twin_type_service"),
		twinTypeRepo: repoFactory.TwinType(),
		userRepo:     repoFactory.User(),
	}
}

// Create adds a new twin type
func (s *TwinTypeService) Create(twinType *models.TwinType) error {
	// Validate twin type data
	if twinType.Name == "" {
		return errors.New("twin type name is required")
	}

	if twinType.Version == "" {
		return errors.New("twin type version is required")
	}

	if twinType.CreatedBy == 0 {
		return errors.New("twin type creator is required")
	}

	// Verify user exists
	_, err := s.userRepo.GetByID(twinType.CreatedBy)
	if err != nil {
		s.logger.Error("Failed to verify user exists", zap.Uint("user_id", twinType.CreatedBy), zap.Error(err))
		return errors.New("invalid creator user")
	}

	// Check if twin type with same name already exists
	_, err = s.twinTypeRepo.GetByName(twinType.Name)
	if err == nil {
		return errors.New("twin type with this name already exists")
	} else if !errors.Is(err, repository.ErrNotFound) {
		s.logger.Error("Error checking twin type existence", zap.String("name", twinType.Name), zap.Error(err))
		return errors.New("database error")
	}

	// Create twin type
	err = s.twinTypeRepo.Create(twinType)
	if err != nil {
		s.logger.Error("Failed to create twin type", zap.Error(err))
		return errors.New("failed to create twin type")
	}

	return nil
}

// GetByID retrieves a twin type by ID
func (s *TwinTypeService) GetByID(id uint) (*models.TwinType, error) {
	twinType, err := s.twinTypeRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errors.New("twin type not found")
		}
		s.logger.Error("Failed to get twin type", zap.Uint("id", id), zap.Error(err))
		return nil, errors.New("database error")
	}

	return twinType, nil
}

// GetByName retrieves a twin type by name
func (s *TwinTypeService) GetByName(name string) (*models.TwinType, error) {
	twinType, err := s.twinTypeRepo.GetByName(name)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errors.New("twin type not found")
		}
		s.logger.Error("Failed to get twin type by name", zap.String("name", name), zap.Error(err))
		return nil, errors.New("database error")
	}

	return twinType, nil
}

// List returns a paginated list of twin types
func (s *TwinTypeService) List(page, pageSize int) ([]models.TwinType, int64, error) {
	offset := (page - 1) * pageSize
	twinTypes, total, err := s.twinTypeRepo.List(offset, pageSize)
	if err != nil {
		s.logger.Error("Failed to list twin types", zap.Error(err))
		return nil, 0, errors.New("database error")
	}

	return twinTypes, total, nil
}

// Update updates a twin type's information
func (s *TwinTypeService) Update(twinType *models.TwinType) error {
	// Validate twin type data
	if twinType.ID == 0 {
		return errors.New("twin type ID is required")
	}

	if twinType.Name == "" {
		return errors.New("twin type name is required")
	}

	if twinType.Version == "" {
		return errors.New("twin type version is required")
	}

	// Check if twin type exists
	existingTwinType, err := s.twinTypeRepo.GetByID(twinType.ID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return errors.New("twin type not found")
		}
		s.logger.Error("Failed to get twin type", zap.Uint("id", twinType.ID), zap.Error(err))
		return errors.New("database error")
	}

	// If name is changed, check if new name already exists
	if twinType.Name != existingTwinType.Name {
		existingWithName, err := s.twinTypeRepo.GetByName(twinType.Name)
		if err == nil && existingWithName.ID != twinType.ID {
			return errors.New("twin type with this name already exists")
		} else if err != nil && !errors.Is(err, repository.ErrNotFound) {
			s.logger.Error("Error checking twin type existence", zap.String("name", twinType.Name), zap.Error(err))
			return errors.New("database error")
		}
	}

	// Update twin type
	err = s.twinTypeRepo.Update(twinType)
	if err != nil {
		s.logger.Error("Failed to update twin type", zap.Uint("id", twinType.ID), zap.Error(err))
		return errors.New("failed to update twin type")
	}

	return nil
}

// Delete soft-deletes a twin type
func (s *TwinTypeService) Delete(id uint) error {
	// Check if twin type exists
	_, err := s.twinTypeRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return errors.New("twin type not found")
		}
		s.logger.Error("Failed to get twin type", zap.Uint("id", id), zap.Error(err))
		return errors.New("database error")
	}

	// TODO: Check if there are any twins using this twin type
	// If there are, we should either prevent deletion or cascade delete

	// Delete twin type
	err = s.twinTypeRepo.Delete(id)
	if err != nil {
		s.logger.Error("Failed to delete twin type", zap.Uint("id", id), zap.Error(err))
		return errors.New("failed to delete twin type")
	}

	return nil
}
