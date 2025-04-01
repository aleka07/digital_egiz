package repository

import (
	"github.com/digital-egiz/backend/internal/db/models"
	"gorm.io/gorm"
)

// MLRepository defines operations for managing ML tasks and models
type MLRepository interface {
	Repository
	// ML task operations
	CreateMLTask(task *models.MLTask) error
	GetMLTaskByID(id uint) (*models.MLTask, error)
	ListMLTasks(offset, limit int) ([]models.MLTask, int64, error)
	UpdateMLTask(task *models.MLTask) error
	DeleteMLTask(id uint) error
	ActivateMLTask(id uint, active bool) error

	// ML task binding operations
	CreateMLTaskBinding(binding *models.MLTaskBinding) error
	GetMLTaskBindingByID(id uint) (*models.MLTaskBinding, error)
	ListMLTaskBindingsByTaskID(taskID uint) ([]models.MLTaskBinding, error)
	ListMLTaskBindingsByTwinID(twinID uint) ([]models.MLTaskBinding, error)
	UpdateMLTaskBinding(binding *models.MLTaskBinding) error
	DeleteMLTaskBinding(id uint) error
	ActivateMLTaskBinding(id uint, active bool) error

	// ML model metadata operations
	CreateMLModelMetadata(metadata *models.MLModelMetadata) error
	GetMLModelMetadataByID(id uint) (*models.MLModelMetadata, error)
	GetMLModelMetadataByModelID(modelID string) (*models.MLModelMetadata, error)
	ListMLModelMetadata(offset, limit int) ([]models.MLModelMetadata, int64, error)
	UpdateMLModelMetadata(metadata *models.MLModelMetadata) error
	DeleteMLModelMetadata(id uint) error
}

// mlRepository implements MLRepository
type mlRepository struct {
	BaseRepository
}

// NewMLRepository creates a new ML repository
func NewMLRepository(db *gorm.DB) MLRepository {
	return &mlRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// CreateMLTask adds a new ML task to the database
func (r *mlRepository) CreateMLTask(task *models.MLTask) error {
	err := r.GetDB().Create(task).Error
	return r.handleError(err)
}

// GetMLTaskByID retrieves an ML task by ID
func (r *mlRepository) GetMLTaskByID(id uint) (*models.MLTask, error) {
	var task models.MLTask
	err := r.GetDB().Preload("Bindings").Where("id = ?", id).First(&task).Error
	if err != nil {
		return nil, r.handleError(err)
	}
	return &task, nil
}

// ListMLTasks retrieves a paginated list of ML tasks
func (r *mlRepository) ListMLTasks(offset, limit int) ([]models.MLTask, int64, error) {
	var tasks []models.MLTask
	var total int64

	// Get total count
	if err := r.GetDB().Model(&models.MLTask{}).Count(&total).Error; err != nil {
		return nil, 0, r.handleError(err)
	}

	// Get paginated tasks
	err := r.GetDB().Offset(offset).Limit(limit).Order("id asc").Find(&tasks).Error
	if err != nil {
		return nil, 0, r.handleError(err)
	}

	return tasks, total, nil
}

// UpdateMLTask updates an ML task's information
func (r *mlRepository) UpdateMLTask(task *models.MLTask) error {
	// Check if task exists
	var existingTask models.MLTask
	if err := r.GetDB().Where("id = ?", task.ID).First(&existingTask).Error; err != nil {
		return r.handleError(err)
	}

	// Update only allowed fields
	err := r.GetDB().Model(task).Updates(map[string]interface{}{
		"name":        task.Name,
		"description": task.Description,
		"type":        task.Type,
		"model_id":    task.ModelID,
		"version":     task.Version,
		"config_json": task.ConfigJSON,
	}).Error

	return r.handleError(err)
}

// DeleteMLTask soft-deletes an ML task
func (r *mlRepository) DeleteMLTask(id uint) error {
	// Delete in a transaction to also delete related bindings
	tx := r.GetDB().Begin()
	if tx.Error != nil {
		return r.handleError(tx.Error)
	}

	// Delete task bindings (hard delete)
	if err := tx.Where("task_id = ?", id).Delete(&models.MLTaskBinding{}).Error; err != nil {
		tx.Rollback()
		return r.handleError(err)
	}

	// Soft delete the task
	result := tx.Delete(&models.MLTask{}, id)
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

// ActivateMLTask activates or deactivates an ML task
func (r *mlRepository) ActivateMLTask(id uint, active bool) error {
	result := r.GetDB().Model(&models.MLTask{}).Where("id = ?", id).Update("active", active)
	if result.Error != nil {
		return r.handleError(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateMLTaskBinding adds a new ML task binding to the database
func (r *mlRepository) CreateMLTaskBinding(binding *models.MLTaskBinding) error {
	err := r.GetDB().Create(binding).Error
	return r.handleError(err)
}

// GetMLTaskBindingByID retrieves an ML task binding by ID
func (r *mlRepository) GetMLTaskBindingByID(id uint) (*models.MLTaskBinding, error) {
	var binding models.MLTaskBinding
	err := r.GetDB().Preload("Task").Preload("Twin").Where("id = ?", id).First(&binding).Error
	if err != nil {
		return nil, r.handleError(err)
	}
	return &binding, nil
}

// ListMLTaskBindingsByTaskID lists all bindings for a specific ML task
func (r *mlRepository) ListMLTaskBindingsByTaskID(taskID uint) ([]models.MLTaskBinding, error) {
	var bindings []models.MLTaskBinding
	err := r.GetDB().Preload("Twin").Where("task_id = ?", taskID).Find(&bindings).Error
	if err != nil {
		return nil, r.handleError(err)
	}
	return bindings, nil
}

// ListMLTaskBindingsByTwinID lists all ML task bindings for a specific twin
func (r *mlRepository) ListMLTaskBindingsByTwinID(twinID uint) ([]models.MLTaskBinding, error) {
	var bindings []models.MLTaskBinding
	err := r.GetDB().Preload("Task").Where("twin_id = ?", twinID).Find(&bindings).Error
	if err != nil {
		return nil, r.handleError(err)
	}
	return bindings, nil
}

// UpdateMLTaskBinding updates an ML task binding
func (r *mlRepository) UpdateMLTaskBinding(binding *models.MLTaskBinding) error {
	err := r.GetDB().Model(binding).Updates(map[string]interface{}{
		"input_mapping_json": binding.InputMappingJSON,
		"output_path_json":   binding.OutputPathJSON,
		"schedule_type":      binding.ScheduleType,
		"schedule_config":    binding.ScheduleConfig,
	}).Error
	return r.handleError(err)
}

// DeleteMLTaskBinding deletes an ML task binding
func (r *mlRepository) DeleteMLTaskBinding(id uint) error {
	result := r.GetDB().Delete(&models.MLTaskBinding{}, id)
	if result.Error != nil {
		return r.handleError(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ActivateMLTaskBinding activates or deactivates an ML task binding
func (r *mlRepository) ActivateMLTaskBinding(id uint, active bool) error {
	result := r.GetDB().Model(&models.MLTaskBinding{}).Where("id = ?", id).Update("active", active)
	if result.Error != nil {
		return r.handleError(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateMLModelMetadata adds new ML model metadata to the database
func (r *mlRepository) CreateMLModelMetadata(metadata *models.MLModelMetadata) error {
	// Check if model with the same model ID already exists
	var count int64
	if err := r.GetDB().Model(&models.MLModelMetadata{}).Where("model_id = ?", metadata.ModelID).Count(&count).Error; err != nil {
		return r.handleError(err)
	}

	if count > 0 {
		return ErrConflict
	}

	err := r.GetDB().Create(metadata).Error
	return r.handleError(err)
}

// GetMLModelMetadataByID retrieves ML model metadata by ID
func (r *mlRepository) GetMLModelMetadataByID(id uint) (*models.MLModelMetadata, error) {
	var metadata models.MLModelMetadata
	err := r.GetDB().Where("id = ?", id).First(&metadata).Error
	if err != nil {
		return nil, r.handleError(err)
	}
	return &metadata, nil
}

// GetMLModelMetadataByModelID retrieves ML model metadata by model ID
func (r *mlRepository) GetMLModelMetadataByModelID(modelID string) (*models.MLModelMetadata, error) {
	var metadata models.MLModelMetadata
	err := r.GetDB().Where("model_id = ?", modelID).First(&metadata).Error
	if err != nil {
		return nil, r.handleError(err)
	}
	return &metadata, nil
}

// ListMLModelMetadata retrieves a paginated list of ML model metadata
func (r *mlRepository) ListMLModelMetadata(offset, limit int) ([]models.MLModelMetadata, int64, error) {
	var metadata []models.MLModelMetadata
	var total int64

	// Get total count
	if err := r.GetDB().Model(&models.MLModelMetadata{}).Count(&total).Error; err != nil {
		return nil, 0, r.handleError(err)
	}

	// Get paginated metadata
	err := r.GetDB().Offset(offset).Limit(limit).Order("id asc").Find(&metadata).Error
	if err != nil {
		return nil, 0, r.handleError(err)
	}

	return metadata, total, nil
}

// UpdateMLModelMetadata updates ML model metadata
func (r *mlRepository) UpdateMLModelMetadata(metadata *models.MLModelMetadata) error {
	// Check if metadata exists
	var existingMetadata models.MLModelMetadata
	if err := r.GetDB().Where("id = ?", metadata.ID).First(&existingMetadata).Error; err != nil {
		return r.handleError(err)
	}

	// Check model ID uniqueness if it was changed
	if existingMetadata.ModelID != metadata.ModelID {
		var count int64
		if err := r.GetDB().Model(&models.MLModelMetadata{}).Where("model_id = ? AND id != ?", metadata.ModelID, metadata.ID).Count(&count).Error; err != nil {
			return r.handleError(err)
		}

		if count > 0 {
			return ErrConflict
		}
	}

	// Update metadata
	err := r.GetDB().Model(metadata).Updates(map[string]interface{}{
		"model_id":      metadata.ModelID,
		"name":          metadata.Name,
		"description":   metadata.Description,
		"type":          metadata.Type,
		"version":       metadata.Version,
		"input_schema":  metadata.InputSchema,
		"output_schema": metadata.OutputSchema,
	}).Error

	return r.handleError(err)
}

// DeleteMLModelMetadata deletes ML model metadata
func (r *mlRepository) DeleteMLModelMetadata(id uint) error {
	// Check if any ML tasks are using this model
	var count int64
	if err := r.GetDB().Model(&models.MLTask{}).Where("model_id = (SELECT model_id FROM ml_model_metadata WHERE id = ?)", id).Count(&count).Error; err != nil {
		return r.handleError(err)
	}

	if count > 0 {
		return ErrInvalidInput
	}

	result := r.GetDB().Delete(&models.MLModelMetadata{}, id)
	if result.Error != nil {
		return r.handleError(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
