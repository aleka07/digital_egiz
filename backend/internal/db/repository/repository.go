package repository

import (
	"errors"

	"gorm.io/gorm"
)

// Common repository errors
var (
	ErrNotFound     = errors.New("record not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrConflict     = errors.New("record already exists")
	ErrDatabase     = errors.New("database error")
	ErrUnauthorized = errors.New("unauthorized access")
)

// Repository defines the basic repository interface with common CRUD operations
type Repository interface {
	// GetDB returns the underlying database connection
	GetDB() *gorm.DB
}

// BaseRepository provides common functionality for repositories
type BaseRepository struct {
	db *gorm.DB
}

// NewBaseRepository creates a new base repository
func NewBaseRepository(db *gorm.DB) BaseRepository {
	return BaseRepository{db: db}
}

// GetDB returns the underlying database connection
func (r *BaseRepository) GetDB() *gorm.DB {
	return r.db
}

// handleError converts GORM errors to repository errors
func (r *BaseRepository) handleError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}

	return ErrDatabase
}
