package repository

import (
	"github.com/digital-egiz/backend/internal/db/models"
	"gorm.io/gorm"
)

// TwinRepository defines operations for managing digital twins
type TwinRepository interface {
	Repository
	// Twin type operations
	CreateTwinType(twinType *models.TwinType) error
	GetTwinTypeByID(id uint) (*models.TwinType, error)
	ListTwinTypes(offset, limit int) ([]models.TwinType, int64, error)
	UpdateTwinType(twinType *models.TwinType) error
	DeleteTwinType(id uint) error

	// Twin instance operations
	CreateTwin(twin *models.Twin) error
	GetTwinByID(id uint) (*models.Twin, error)
	GetTwinByDittoID(dittoID string) (*models.Twin, error)
	ListTwins(offset, limit int) ([]models.Twin, int64, error)
	ListTwinsByProjectID(projectID uint, offset, limit int) ([]models.Twin, int64, error)
	ListTwinsByTypeID(typeID uint, offset, limit int) ([]models.Twin, int64, error)
	UpdateTwin(twin *models.Twin) error
	DeleteTwin(id uint) error

	// 3D model operations
	SaveTwinModel3D(model *models.TwinModel3D) error
	GetTwinModel3D(twinID uint) (*models.TwinModel3D, error)
	UpdateTwinModel3D(model *models.TwinModel3D) error
	DeleteTwinModel3D(twinID uint) error

	// Data binding operations
	CreateDataBinding3D(binding *models.DataBinding3D) error
	GetDataBinding3DByID(id uint) (*models.DataBinding3D, error)
	ListDataBindings3D(twinID uint) ([]models.DataBinding3D, error)
	UpdateDataBinding3D(binding *models.DataBinding3D) error
	DeleteDataBinding3D(id uint) error
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

// CreateTwinType adds a new twin type to the database
func (r *twinRepository) CreateTwinType(twinType *models.TwinType) error {
	// Check if twin type with the same name already exists
	var count int64
	if err := r.GetDB().Model(&models.TwinType{}).Where("name = ?", twinType.Name).Count(&count).Error; err != nil {
		return r.handleError(err)
	}

	if count > 0 {
		return ErrConflict
	}

	err := r.GetDB().Create(twinType).Error
	return r.handleError(err)
}

// GetTwinTypeByID retrieves a twin type by ID
func (r *twinRepository) GetTwinTypeByID(id uint) (*models.TwinType, error) {
	var twinType models.TwinType
	err := r.GetDB().Where("id = ?", id).First(&twinType).Error
	if err != nil {
		return nil, r.handleError(err)
	}
	return &twinType, nil
}

// ListTwinTypes retrieves a paginated list of twin types
func (r *twinRepository) ListTwinTypes(offset, limit int) ([]models.TwinType, int64, error) {
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

// UpdateTwinType updates a twin type's information
func (r *twinRepository) UpdateTwinType(twinType *models.TwinType) error {
	// Check if twin type exists
	var existingTwinType models.TwinType
	if err := r.GetDB().Where("id = ?", twinType.ID).First(&existingTwinType).Error; err != nil {
		return r.handleError(err)
	}

	// Check name uniqueness if it was changed
	if existingTwinType.Name != twinType.Name {
		var count int64
		if err := r.GetDB().Model(&models.TwinType{}).Where("name = ? AND id != ?", twinType.Name, twinType.ID).Count(&count).Error; err != nil {
			return r.handleError(err)
		}

		if count > 0 {
			return ErrConflict
		}
	}

	// Update only allowed fields
	err := r.GetDB().Model(twinType).Updates(map[string]interface{}{
		"name":        twinType.Name,
		"description": twinType.Description,
		"version":     twinType.Version,
		"schema_json": twinType.SchemaJSON,
	}).Error

	return r.handleError(err)
}

// DeleteTwinType soft-deletes a twin type
func (r *twinRepository) DeleteTwinType(id uint) error {
	// Check if any twins of this type exist
	var count int64
	if err := r.GetDB().Model(&models.Twin{}).Where("type_id = ?", id).Count(&count).Error; err != nil {
		return r.handleError(err)
	}

	if count > 0 {
		return ErrInvalidInput
	}

	result := r.GetDB().Delete(&models.TwinType{}, id)
	if result.Error != nil {
		return r.handleError(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateTwin adds a new twin instance to the database
func (r *twinRepository) CreateTwin(twin *models.Twin) error {
	// Check if twin with the same Ditto ID already exists
	var count int64
	if err := r.GetDB().Model(&models.Twin{}).Where("ditto_thing_id = ?", twin.DittoThingID).Count(&count).Error; err != nil {
		return r.handleError(err)
	}

	if count > 0 {
		return ErrConflict
	}

	err := r.GetDB().Create(twin).Error
	return r.handleError(err)
}

// GetTwinByID retrieves a twin by ID
func (r *twinRepository) GetTwinByID(id uint) (*models.Twin, error) {
	var twin models.Twin
	err := r.GetDB().Preload("Type").Where("id = ?", id).First(&twin).Error
	if err != nil {
		return nil, r.handleError(err)
	}
	return &twin, nil
}

// GetTwinByDittoID retrieves a twin by its Ditto Thing ID
func (r *twinRepository) GetTwinByDittoID(dittoID string) (*models.Twin, error) {
	var twin models.Twin
	err := r.GetDB().Preload("Type").Where("ditto_thing_id = ?", dittoID).First(&twin).Error
	if err != nil {
		return nil, r.handleError(err)
	}
	return &twin, nil
}

// ListTwins retrieves a paginated list of twins
func (r *twinRepository) ListTwins(offset, limit int) ([]models.Twin, int64, error) {
	var twins []models.Twin
	var total int64

	// Get total count
	if err := r.GetDB().Model(&models.Twin{}).Count(&total).Error; err != nil {
		return nil, 0, r.handleError(err)
	}

	// Get paginated twins with type preloaded
	err := r.GetDB().Preload("Type").Offset(offset).Limit(limit).Order("id asc").Find(&twins).Error
	if err != nil {
		return nil, 0, r.handleError(err)
	}

	return twins, total, nil
}

// ListTwinsByProjectID retrieves twins by project ID
func (r *twinRepository) ListTwinsByProjectID(projectID uint, offset, limit int) ([]models.Twin, int64, error) {
	var twins []models.Twin
	var total int64

	// Get total count
	if err := r.GetDB().Model(&models.Twin{}).Where("project_id = ?", projectID).Count(&total).Error; err != nil {
		return nil, 0, r.handleError(err)
	}

	// Get paginated twins with type preloaded
	err := r.GetDB().Preload("Type").Where("project_id = ?", projectID).
		Offset(offset).Limit(limit).Order("id asc").Find(&twins).Error
	if err != nil {
		return nil, 0, r.handleError(err)
	}

	return twins, total, nil
}

// ListTwinsByTypeID retrieves twins by type ID
func (r *twinRepository) ListTwinsByTypeID(typeID uint, offset, limit int) ([]models.Twin, int64, error) {
	var twins []models.Twin
	var total int64

	// Get total count
	if err := r.GetDB().Model(&models.Twin{}).Where("type_id = ?", typeID).Count(&total).Error; err != nil {
		return nil, 0, r.handleError(err)
	}

	// Get paginated twins with type preloaded
	err := r.GetDB().Preload("Type").Where("type_id = ?", typeID).
		Offset(offset).Limit(limit).Order("id asc").Find(&twins).Error
	if err != nil {
		return nil, 0, r.handleError(err)
	}

	return twins, total, nil
}

// UpdateTwin updates a twin's information
func (r *twinRepository) UpdateTwin(twin *models.Twin) error {
	// Check if twin exists
	var existingTwin models.Twin
	if err := r.GetDB().Where("id = ?", twin.ID).First(&existingTwin).Error; err != nil {
		return r.handleError(err)
	}

	// Check Ditto ID uniqueness if it was changed
	if existingTwin.DittoThingID != twin.DittoThingID {
		var count int64
		if err := r.GetDB().Model(&models.Twin{}).Where("ditto_thing_id = ? AND id != ?", twin.DittoThingID, twin.ID).Count(&count).Error; err != nil {
			return r.handleError(err)
		}

		if count > 0 {
			return ErrConflict
		}
	}

	// Update only allowed fields
	err := r.GetDB().Model(twin).Updates(map[string]interface{}{
		"name":           twin.Name,
		"description":    twin.Description,
		"ditto_thing_id": twin.DittoThingID,
		"metadata_json":  twin.MetadataJSON,
		"has_3d_model":   twin.Has3DModel,
	}).Error

	return r.handleError(err)
}

// DeleteTwin soft-deletes a twin
func (r *twinRepository) DeleteTwin(id uint) error {
	// Delete in a transaction to also delete related data
	tx := r.GetDB().Begin()
	if tx.Error != nil {
		return r.handleError(tx.Error)
	}

	// Delete 3D model (if exists) - hard delete
	if err := tx.Where("twin_id = ?", id).Delete(&models.TwinModel3D{}).Error; err != nil {
		tx.Rollback()
		return r.handleError(err)
	}

	// Delete data bindings - hard delete
	if err := tx.Where("twin_id = ?", id).Delete(&models.DataBinding3D{}).Error; err != nil {
		tx.Rollback()
		return r.handleError(err)
	}

	// Soft delete the twin
	result := tx.Delete(&models.Twin{}, id)
	if result.Error != nil {
		tx.Rollback()
		return r.handleError(result.Error)
	}

	if result.RowsAffected == 0 {
		tx.Rollback()
		return ErrNotFound
	}

	return tx.Commit().Error
}

// SaveTwinModel3D saves a 3D model for a twin
func (r *twinRepository) SaveTwinModel3D(model *models.TwinModel3D) error {
	// Check if twin exists
	var twin models.Twin
	if err := r.GetDB().Where("id = ?", model.TwinID).First(&twin).Error; err != nil {
		return r.handleError(err)
	}

	// Check if model already exists
	var count int64
	if err := r.GetDB().Model(&models.TwinModel3D{}).Where("twin_id = ?", model.TwinID).Count(&count).Error; err != nil {
		return r.handleError(err)
	}

	// Start transaction
	tx := r.GetDB().Begin()
	if tx.Error != nil {
		return r.handleError(tx.Error)
	}

	if count > 0 {
		// Update existing
		if err := tx.Model(&models.TwinModel3D{}).Where("twin_id = ?", model.TwinID).
			Updates(map[string]interface{}{
				"model_url":    model.ModelURL,
				"model_format": model.ModelFormat,
				"version_tag":  model.VersionTag,
			}).Error; err != nil {
			tx.Rollback()
			return r.handleError(err)
		}
	} else {
		// Create new
		if err := tx.Create(model).Error; err != nil {
			tx.Rollback()
			return r.handleError(err)
		}
	}

	// Update twin to indicate it has a 3D model
	if !twin.Has3DModel {
		if err := tx.Model(&twin).Update("has_3d_model", true).Error; err != nil {
			tx.Rollback()
			return r.handleError(err)
		}
	}

	return tx.Commit().Error
}

// GetTwinModel3D retrieves a 3D model for a twin
func (r *twinRepository) GetTwinModel3D(twinID uint) (*models.TwinModel3D, error) {
	var model models.TwinModel3D
	err := r.GetDB().Where("twin_id = ?", twinID).First(&model).Error
	if err != nil {
		return nil, r.handleError(err)
	}
	return &model, nil
}

// UpdateTwinModel3D updates a 3D model for a twin
func (r *twinRepository) UpdateTwinModel3D(model *models.TwinModel3D) error {
	err := r.GetDB().Model(model).Updates(map[string]interface{}{
		"model_url":    model.ModelURL,
		"model_format": model.ModelFormat,
		"version_tag":  model.VersionTag,
	}).Error
	return r.handleError(err)
}

// DeleteTwinModel3D deletes a 3D model for a twin
func (r *twinRepository) DeleteTwinModel3D(twinID uint) error {
	// Start transaction
	tx := r.GetDB().Begin()
	if tx.Error != nil {
		return r.handleError(tx.Error)
	}

	// Delete the model
	result := tx.Where("twin_id = ?", twinID).Delete(&models.TwinModel3D{})
	if result.Error != nil {
		tx.Rollback()
		return r.handleError(result.Error)
	}

	if result.RowsAffected == 0 {
		tx.Rollback()
		return ErrNotFound
	}

	// Update twin to indicate it no longer has a 3D model
	if err := tx.Model(&models.Twin{}).Where("id = ?", twinID).
		Update("has_3d_model", false).Error; err != nil {
		tx.Rollback()
		return r.handleError(err)
	}

	return tx.Commit().Error
}

// CreateDataBinding3D creates a new 3D data binding
func (r *twinRepository) CreateDataBinding3D(binding *models.DataBinding3D) error {
	err := r.GetDB().Create(binding).Error
	return r.handleError(err)
}

// GetDataBinding3DByID retrieves a 3D data binding by ID
func (r *twinRepository) GetDataBinding3DByID(id uint) (*models.DataBinding3D, error) {
	var binding models.DataBinding3D
	err := r.GetDB().Where("id = ?", id).First(&binding).Error
	if err != nil {
		return nil, r.handleError(err)
	}
	return &binding, nil
}

// ListDataBindings3D lists all 3D data bindings for a twin
func (r *twinRepository) ListDataBindings3D(twinID uint) ([]models.DataBinding3D, error) {
	var bindings []models.DataBinding3D
	err := r.GetDB().Where("twin_id = ?", twinID).Find(&bindings).Error
	if err != nil {
		return nil, r.handleError(err)
	}
	return bindings, nil
}

// UpdateDataBinding3D updates a 3D data binding
func (r *twinRepository) UpdateDataBinding3D(binding *models.DataBinding3D) error {
	err := r.GetDB().Model(binding).Updates(map[string]interface{}{
		"object_name":       binding.ObjectName,
		"ditto_path":        binding.DittoPath,
		"binding_type":      binding.BindingType,
		"binding_value_map": binding.BindingValueMap,
		"description":       binding.Description,
	}).Error
	return r.handleError(err)
}

// DeleteDataBinding3D deletes a 3D data binding
func (r *twinRepository) DeleteDataBinding3D(id uint) error {
	result := r.GetDB().Delete(&models.DataBinding3D{}, id)
	if result.Error != nil {
		return r.handleError(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
