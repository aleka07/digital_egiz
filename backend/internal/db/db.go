package db

import (
	"fmt"
	"time"

	"github.com/digital-egiz/backend/internal/config"
	"github.com/digital-egiz/backend/internal/utils"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database is a wrapper around gorm.DB with additional functionality
type Database struct {
	*gorm.DB
	logger *utils.Logger
}

// NewDatabase creates a new database connection
func NewDatabase(cfg *config.DatabaseConfig, log *utils.Logger) (*Database, error) {
	dbLogger := log.Named("db")

	// Create DSN string for PostgreSQL
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	// Configure GORM logger based on our application logger
	gormLogLevel := logger.Silent
	if log.Core().Enabled(zap.DebugLevel) {
		gormLogLevel = logger.Info
	}

	gormLogger := logger.New(
		&gormLogAdapter{logger: dbLogger},
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  gormLogLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	// Connect to database
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set connection pool settings
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Return the database wrapper
	return &Database{
		DB:     gormDB,
		logger: dbLogger,
	}, nil
}

// EnableTimescaleDB enables TimescaleDB extensions and features
func (db *Database) EnableTimescaleDB() error {
	// Enable TimescaleDB extension if not already enabled
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE").Error; err != nil {
		return fmt.Errorf("failed to enable TimescaleDB extension: %w", err)
	}

	db.logger.Info("TimescaleDB extension enabled")
	return nil
}

// Close closes the database connection
func (db *Database) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	return sqlDB.Close()
}

// gormLogAdapter adapts our zap logger to GORM's logger interface
type gormLogAdapter struct {
	logger *utils.Logger
}

func (l *gormLogAdapter) Printf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.logger.Debug(msg)
} 