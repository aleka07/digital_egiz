package db

import (
	"fmt"
	"time"

	"github.com/digital-egiz/backend/internal/config"
	"github.com/digital-egiz/backend/internal/db/models"
	"github.com/digital-egiz/backend/internal/utils"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database wraps a GORM DB connection with additional functionality
type Database struct {
	*gorm.DB
	logger *utils.Logger
	config *config.DatabaseConfig
}

// NewDatabase creates a new database connection
func NewDatabase(cfg *config.DatabaseConfig, log *utils.Logger) (*Database, error) {
	dbLogger := log.Named("database")

	// Configure GORM logger
	gormLogger := logger.New(
		&logAdapter{logger: dbLogger},
		logger.Config{
			SlowThreshold:             1 * time.Second,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	// Configure GORM
	gormConfig := &gorm.Config{
		Logger:                 gormLogger,
		PrepareStmt:            true,
		SkipDefaultTransaction: true,
	}

	// Connect to database
	dbLogger.Info("Connecting to database",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("dbname", cfg.DBName),
		zap.String("user", cfg.User),
	)

	dsn := cfg.GetDSN()
	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Create database wrapper
	database := &Database{
		DB:     db,
		logger: dbLogger,
		config: cfg,
	}

	// Verify connection
	if err := database.VerifyConnection(); err != nil {
		return nil, err
	}

	return database, nil
}

// VerifyConnection checks if the database connection is working
func (db *Database) VerifyConnection() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB instance: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	db.logger.Info("Successfully connected to database")
	return nil
}

// AutoMigrate runs auto migration for the given models
func (db *Database) AutoMigrate() error {
	db.logger.Info("Running auto migrations")

	// Register PostgreSQL extension for UUID generation
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";").Error; err != nil {
		return fmt.Errorf("failed to create UUID extension: %w", err)
	}

	// Register TimescaleDB extension if not already enabled
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;").Error; err != nil {
		db.logger.Warn("Failed to create TimescaleDB extension, time-series optimization disabled", zap.Error(err))
	}

	// Auto migrate models
	if err := db.DB.AutoMigrate(
		&models.User{},
		&models.Project{},
		&models.TwinType{},
		&models.Twin{},
		&models.TwinModel3D{},
		&models.DataBinding3D{},
		&models.TimeseriesData{},
		&models.AggregatedData{},
		&models.AlertData{},
		&models.MLPredictionData{},
		&models.MLTask{},
		&models.MLTaskBinding{},
		&models.MLModelMetadata{},
	); err != nil {
		return fmt.Errorf("failed to auto migrate models: %w", err)
	}

	// Create hypertables for time-series data
	if err := db.CreateHypertables(); err != nil {
		db.logger.Warn("Failed to create hypertables", zap.Error(err))
	}

	return nil
}

// CreateHypertables creates TimescaleDB hypertables for time-series data
func (db *Database) CreateHypertables() error {
	// Check if TimescaleDB extension is enabled
	var extensionExists bool
	if err := db.DB.Raw("SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'timescaledb');").Scan(&extensionExists).Error; err != nil {
		return fmt.Errorf("failed to check TimescaleDB extension: %w", err)
	}

	if !extensionExists {
		return fmt.Errorf("TimescaleDB extension not installed")
	}

	// Create hypertable for time-series data
	tables := []string{
		"timeseries_data",
		"aggregated_data",
		"alert_data",
		"ml_prediction_data",
	}

	for _, table := range tables {
		var hypertableExists bool
		if err := db.DB.Raw("SELECT EXISTS(SELECT 1 FROM timescaledb_information.hypertables WHERE hypertable_name = ?);", table).Scan(&hypertableExists).Error; err != nil {
			return fmt.Errorf("failed to check if hypertable exists for %s: %w", table, err)
		}

		if !hypertableExists {
			// For TimescaleDB, table must already exist before converting to hypertable
			timeCol := "time"
			if table == "aggregated_data" {
				timeCol = "time_interval"
			}

			if err := db.DB.Exec(fmt.Sprintf("SELECT create_hypertable('%s', '%s');", table, timeCol)).Error; err != nil {
				return fmt.Errorf("failed to create hypertable for %s: %w", table, err)
			}
			db.logger.Info(fmt.Sprintf("Created hypertable for %s", table))
		}
	}

	return nil
}

// Close closes the database connection
func (db *Database) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB instance: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	db.logger.Info("Database connection closed")
	return nil
}

// logAdapter adapts our logger to GORM's logger interface
type logAdapter struct {
	logger *utils.Logger
}

// Printf implements GORM's logger interface
func (l *logAdapter) Printf(format string, v ...interface{}) {
	l.logger.Info(fmt.Sprintf(format, v...))
}
