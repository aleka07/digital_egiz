package repository

import (
	"github.com/digital-egiz/backend/internal/db/models"
	"gorm.io/gorm"
)

// TwinRepository defines operations for managing twins
type TwinRepository interface {
	Repository
	Create(twin *models.Twin) error
	GetByID(id uint) (*models.Twin, error)
	GetByDittoID(dittoID string) (*models.Twin, error)
	ListByProjectID(projectID uint, offset, limit int) ([]models.Twin, int64, error)
	Update(twin *models.Twin) error
	Delete(id uint) error

	// Model bindings
	CreateModelBinding(binding *models.ModelBinding) error
	ListModelBindings(twinID uint) ([]models.ModelBinding, error)
	UpdateModelBinding(binding *models.ModelBinding) error
	DeleteModelBinding(id uint) error
}

// twinRepository implements TwinRepository
type twinRepository struct {
	BaseRepository
}

// NewTwinRepository creates a new twin repository
func NewTwinRepository(db *gorm.DB) TwinRepository {
	return &twinRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Create adds a new twin to the database
func (r *twinRepository) Create(twin *models.Twin) error {
	err := r.GetDB().Create(twin).Error
	return r.handleError(err)
}

// GetByID retrieves a twin by ID
func (r *twinRepository) GetByID(id uint) (*models.Twin, error) {
	var twin models.Twin
	err := r.GetDB().Preload("Type").Where("id = ?", id).First(&twin).Error
	if err != nil {
		return nil, r.handleError(err)
	}
	return &twin, nil
}

// GetByDittoID retrieves a twin by Ditto ID
func (r *twinRepository) GetByDittoID(dittoID string) (*models.Twin, error) {
	var twin models.Twin
	err := r.GetDB().Preload("Type").Where("ditto_id = ?", dittoID).First(&twin).Error
	if err != nil {
		return nil, r.handleError(err)
	}
	return &twin, nil
}

// ListByProjectID retrieves a paginated list of twins for a project
func (r *twinRepository) ListByProjectID(projectID uint, offset, limit int) ([]models.Twin, int64, error) {
	var twins []models.Twin
	var total int64

	// Get total count
	if err := r.GetDB().Model(&models.Twin{}).Where("project_id = ?", projectID).Count(&total).Error; err != nil {
		return nil, 0, r.handleError(err)
	}

	// Get paginated twins
	err := r.GetDB().Preload("Type").
		Where("project_id = ?", projectID).
		Offset(offset).Limit(limit).
		Order("id asc").
		Find(&twins).Error

	if err != nil {
		return nil, 0, r.handleError(err)
	}

	return twins, total, nil
}

// Update updates a twin's information
func (r *twinRepository) Update(twin *models.Twin) error {
	// Check if twin exists
	var existingTwin models.Twin
	if err := r.GetDB().Where("id = ?", twin.ID).First(&existingTwin).Error; err != nil {
		return r.handleError(err)
	}

	// Update the twin
	err := r.GetDB().Model(twin).Updates(map[string]interface{}{
		"name":        twin.Name,
		"description": twin.Description,
		"ditto_id":    twin.DittoID,
		"type_id":     twin.TypeID,
		"model_url":   twin.ModelURL,
		"metadata":    twin.Metadata,
	}).Error

	return r.handleError(err)
}

// Delete soft-deletes a twin
func (r *twinRepository) Delete(id uint) error {
	result := r.GetDB().Delete(&models.Twin{}, id)
	if result.Error != nil {
		return r.handleError(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateModelBinding adds a new model binding to the database
func (r *twinRepository) CreateModelBinding(binding *models.ModelBinding) error {
	err := r.GetDB().Create(binding).Error
	return r.handleError(err)
}

// ListModelBindings retrieves all model bindings for a twin
func (r *twinRepository) ListModelBindings(twinID uint) ([]models.ModelBinding, error) {
	var bindings []models.ModelBinding
	err := r.GetDB().Where("twin_id = ?", twinID).Find(&bindings).Error
	if err != nil {
		return nil, r.handleError(err)
	}
	return bindings, nil
}

// UpdateModelBinding updates a model binding
func (r *twinRepository) UpdateModelBinding(binding *models.ModelBinding) error {
	// Check if binding exists
	var existingBinding models.ModelBinding
	if err := r.GetDB().Where("id = ?", binding.ID).First(&existingBinding).Error; err != nil {
		return r.handleError(err)
	}

	// Update the binding
	err := r.GetDB().Model(binding).Updates(map[string]interface{}{
		"part_id":      binding.PartID,
		"feature_path": binding.FeaturePath,
		"binding_type": binding.BindingType,
		"properties":   binding.Properties,
	}).Error

	return r.handleError(err)
}

// DeleteModelBinding deletes a model binding
func (r *twinRepository) DeleteModelBinding(id uint) error {
	result := r.GetDB().Delete(&models.ModelBinding{}, id)
	if result.Error != nil {
		return r.handleError(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
