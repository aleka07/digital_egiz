package repository

import (
	"github.com/digital-egiz/backend/internal/db/models"
	"gorm.io/gorm"
)

// TwinTypeRepository defines operations for managing twin types
type TwinTypeRepository interface {
	Repository
	Create(twinType *models.TwinType) error
	GetByID(id uint) (*models.TwinType, error)
	GetByName(name string) (*models.TwinType, error)
	List(offset, limit int) ([]models.TwinType, int64, error)
	Update(twinType *models.TwinType) error
	Delete(id uint) error
}

// twinTypeRepository implements TwinTypeRepository
type twinTypeRepository struct {
	BaseRepository
}

// NewTwinTypeRepository creates a new twin type repository
func NewTwinTypeRepository(db *gorm.DB) TwinTypeRepository {
	return &twinTypeRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Create adds a new twin type to the database
func (r *twinTypeRepository) Create(twinType *models.TwinType) error {
	err := r.GetDB().Create(twinType).Error
	return r.handleError(err)
}

// GetByID retrieves a twin type by ID
func (r *twinTypeRepository) GetByID(id uint) (*models.TwinType, error) {
	var twinType models.TwinType
	err := r.GetDB().Where("id = ?", id).First(&twinType).Error
	if err != nil {
		return nil, r.handleError(err)
	}
	return &twinType, nil
}

// GetByName retrieves a twin type by name
func (r *twinTypeRepository) GetByName(name string) (*models.TwinType, error) {
	var twinType models.TwinType
	err := r.GetDB().Where("name = ?", name).First(&twinType).Error
	if err != nil {
		return nil, r.handleError(err)
	}
	return &twinType, nil
}

// List retrieves a paginated list of twin types
func (r *twinTypeRepository) List(offset, limit int) ([]models.TwinType, int64, error) {
	var twinTypes []models.TwinType
	var total int64

	// Get total count
	if err := r.GetDB().Model(&models.TwinType{}).Count(&total).Error; err != nil {
		return nil, 0, r.handleError(err)
	}

	// Get paginated twin types
	err := r.GetDB().Offset(offset).Limit(limit).Order("id asc").Find(&twinTypes).Error
	if err != nil {
		return nil, 0, r.handleError(err)
	}

	return twinTypes, total, nil
}

// Update updates a twin type's information
func (r *twinTypeRepository) Update(twinType *models.TwinType) error {
	// Check if twin type exists
	var existingTwinType models.TwinType
	if err := r.GetDB().Where("id = ?", twinType.ID).First(&existingTwinType).Error; err != nil {
		return r.handleError(err)
	}

	// Update the twin type
	err := r.GetDB().Model(twinType).Updates(map[string]interface{}{
		"name":        twinType.Name,
		"description": twinType.Description,
		"version":     twinType.Version,
		"schema_json": twinType.SchemaJSON,
	}).Error

	return r.handleError(err)
}

// Delete soft-deletes a twin type
func (r *twinTypeRepository) Delete(id uint) error {
	result := r.GetDB().Delete(&models.TwinType{}, id)
	if result.Error != nil {
		return r.handleError(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
