package repository

import (
	"errors"

	"github.com/digital-egiz/backend/internal/db/models"
	"gorm.io/gorm"
)

// UserRepository defines operations for managing users
type UserRepository interface {
	Repository
	Create(user *models.User) error
	GetByID(id uint) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	List(offset, limit int) ([]models.User, int64, error)
	Update(user *models.User) error
	Delete(id uint) error
	ChangePassword(id uint, newPassword string) error
	UpdateLastLogin(id uint) error
}

// userRepository implements UserRepository
type userRepository struct {
	BaseRepository
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Create adds a new user to the database
func (r *userRepository) Create(user *models.User) error {
	// Check if user with the same email already exists
	var count int64
	if err := r.GetDB().Model(&models.User{}).Where("email = ?", user.Email).Count(&count).Error; err != nil {
		return r.handleError(err)
	}

	if count > 0 {
		return ErrConflict
	}

	err := r.GetDB().Create(user).Error
	return r.handleError(err)
}

// GetByID retrieves a user by ID
func (r *userRepository) GetByID(id uint) (*models.User, error) {
	var user models.User
	err := r.GetDB().Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, r.handleError(err)
	}
	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *userRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.GetDB().Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, r.handleError(err)
	}
	return &user, nil
}

// List retrieves a paginated list of users
func (r *userRepository) List(offset, limit int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	// Get total count
	if err := r.GetDB().Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, r.handleError(err)
	}

	// Get paginated users
	err := r.GetDB().Offset(offset).Limit(limit).Order("id asc").Find(&users).Error
	if err != nil {
		return nil, 0, r.handleError(err)
	}

	return users, total, nil
}

// Update updates a user's information
func (r *userRepository) Update(user *models.User) error {
	// Check if user exists
	var existingUser models.User
	if err := r.GetDB().Where("id = ?", user.ID).First(&existingUser).Error; err != nil {
		return r.handleError(err)
	}

	// Check email uniqueness if it was changed
	if existingUser.Email != user.Email {
		var count int64
		if err := r.GetDB().Model(&models.User{}).Where("email = ? AND id != ?", user.Email, user.ID).Count(&count).Error; err != nil {
			return r.handleError(err)
		}

		if count > 0 {
			return ErrConflict
		}
	}

	// Update user but don't modify password field
	err := r.GetDB().Model(user).Omit("password").Updates(map[string]interface{}{
		"email":      user.Email,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
		"role":       user.Role,
		"active":     user.Active,
	}).Error

	return r.handleError(err)
}

// Delete soft-deletes a user
func (r *userRepository) Delete(id uint) error {
	result := r.GetDB().Delete(&models.User{}, id)
	if result.Error != nil {
		return r.handleError(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ChangePassword updates a user's password
func (r *userRepository) ChangePassword(id uint, newPassword string) error {
	var user models.User
	if err := r.GetDB().Where("id = ?", id).First(&user).Error; err != nil {
		return r.handleError(err)
	}

	if err := user.UpdatePassword(newPassword); err != nil {
		return errors.New("password hashing failed")
	}

	err := r.GetDB().Model(&user).Update("password", user.Password).Error
	return r.handleError(err)
}

// UpdateLastLogin updates the last login timestamp for a user
func (r *userRepository) UpdateLastLogin(id uint) error {
	err := r.GetDB().Model(&models.User{}).Where("id = ?", id).
		UpdateColumn("last_login", gorm.Expr("NOW()")).Error
	return r.handleError(err)
}
