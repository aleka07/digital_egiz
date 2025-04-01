package services

import (
	"errors"
	"time"

	"github.com/digital-egiz/backend/internal/db"
	"github.com/digital-egiz/backend/internal/db/models"
	"github.com/digital-egiz/backend/internal/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// UserService handles user-related business logic
type UserService struct {
	db     *db.Database
	logger *utils.Logger
}

// NewUserService creates a new user service
func NewUserService(db *db.Database, logger *utils.Logger) *UserService {
	return &UserService{
		db:     db,
		logger: logger.Named("user_service"),
	}
}

// Authenticate verifies user credentials and returns the user
func (s *UserService) Authenticate(email, password string) (*models.User, error) {
	var user models.User

	// Find user by email
	result := s.db.Where("email = ? AND active = ?", email, true).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid credentials")
		}
		s.logger.Error("Database error during authentication", zap.Error(result.Error))
		return nil, errors.New("database error")
	}

	// Verify password
	if !user.CheckPassword(password) {
		return nil, errors.New("invalid credentials")
	}

	return &user, nil
}

// Create adds a new user
func (s *UserService) Create(user *models.User) error {
	// Check if email already exists
	var count int64
	if err := s.db.Model(&models.User{}).Where("email = ?", user.Email).Count(&count).Error; err != nil {
		s.logger.Error("Database error checking email uniqueness", zap.Error(err))
		return errors.New("database error")
	}

	if count > 0 {
		return errors.New("email already exists")
	}

	// Create user
	if err := s.db.Create(user).Error; err != nil {
		s.logger.Error("Database error creating user", zap.Error(err))
		return errors.New("failed to create user")
	}

	return nil
}

// GetByID retrieves a user by ID
func (s *UserService) GetByID(id uint) (*models.User, error) {
	var user models.User

	result := s.db.First(&user, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		s.logger.Error("Database error getting user by ID", zap.Uint("id", id), zap.Error(result.Error))
		return nil, errors.New("database error")
	}

	return &user, nil
}

// UpdateLastLogin updates the last login time for a user
func (s *UserService) UpdateLastLogin(id uint) error {
	result := s.db.Model(&models.User{}).Where("id = ?", id).Update("last_login", time.Now())
	if result.Error != nil {
		s.logger.Error("Database error updating last login", zap.Uint("id", id), zap.Error(result.Error))
		return errors.New("database error")
	}
	return nil
}

// List returns a paginated list of users
func (s *UserService) List(page, pageSize int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	// Count total users
	if err := s.db.Model(&models.User{}).Count(&total).Error; err != nil {
		s.logger.Error("Database error counting users", zap.Error(err))
		return nil, 0, errors.New("database error")
	}

	// Apply pagination
	offset := (page - 1) * pageSize
	result := s.db.Offset(offset).Limit(pageSize).Find(&users)
	if result.Error != nil {
		s.logger.Error("Database error listing users", zap.Error(result.Error))
		return nil, 0, errors.New("database error")
	}

	return users, total, nil
}

// Update updates a user's information
func (s *UserService) Update(user *models.User) error {
	result := s.db.Model(user).Updates(map[string]interface{}{
		"first_name": user.FirstName,
		"last_name":  user.LastName,
		"role":       user.Role,
		"active":     user.Active,
		"updated_at": time.Now(),
	})
	if result.Error != nil {
		s.logger.Error("Database error updating user", zap.Uint("id", user.ID), zap.Error(result.Error))
		return errors.New("database error")
	}
	return nil
}

// ChangePassword updates a user's password
func (s *UserService) ChangePassword(id uint, currentPassword, newPassword string) error {
	// Get user
	user, err := s.GetByID(id)
	if err != nil {
		return err
	}

	// Verify current password
	if !user.CheckPassword(currentPassword) {
		return errors.New("current password is incorrect")
	}

	// Update password
	if err := user.UpdatePassword(newPassword); err != nil {
		s.logger.Error("Error hashing new password", zap.Error(err))
		return errors.New("failed to update password")
	}

	// Save to database
	result := s.db.Model(user).Update("password", user.Password)
	if result.Error != nil {
		s.logger.Error("Database error updating password", zap.Uint("id", id), zap.Error(result.Error))
		return errors.New("database error")
	}

	return nil
}

// Delete soft-deletes a user
func (s *UserService) Delete(id uint) error {
	result := s.db.Delete(&models.User{}, id)
	if result.Error != nil {
		s.logger.Error("Database error deleting user", zap.Uint("id", id), zap.Error(result.Error))
		return errors.New("database error")
	}
	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}
	return nil
} 