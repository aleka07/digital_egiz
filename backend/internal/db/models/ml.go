package models

import (
	"time"

	"gorm.io/gorm"
)

// MLTaskType represents the type of ML task
type MLTaskType string

const (
	// MLTaskTypeAnomaly anomaly detection
	MLTaskTypeAnomaly MLTaskType = "anomaly_detection"
	// MLTaskTypePrediction predictive maintenance
	MLTaskTypePrediction MLTaskType = "predictive_maintenance"
	// MLTaskTypeClassification classification
	MLTaskTypeClassification MLTaskType = "classification"
)

// MLTask represents a machine learning task configuration
type MLTask struct {
	ID          uint        `gorm:"primarykey" json:"id"`
	Name        string      `gorm:"not null" json:"name"`
	Description string      `json:"description"`
	Type        MLTaskType  `gorm:"type:varchar(50);not null" json:"type"`
	ModelID     string      `gorm:"not null" json:"model_id"` // Identifier for the ML model
	Version     string      `gorm:"not null" json:"version"`
	Active      bool        `gorm:"default:true" json:"active"`
	ConfigJSON  string      `gorm:"type:jsonb" json:"config_json"` // Configuration parameters for the ML task
	CreatedBy   uint        `json:"created_by"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Bindings []MLTaskBinding `gorm:"foreignKey:TaskID" json:"bindings,omitempty"`
}

// MLTaskBinding represents the binding between a twin and an ML task
type MLTaskBinding struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	TaskID          uint      `gorm:"not null" json:"task_id"`
	TwinID          uint      `gorm:"not null" json:"twin_id"`
	InputMappingJSON string    `gorm:"type:jsonb" json:"input_mapping_json"` // Maps twin properties to ML inputs
	OutputPathJSON  string    `gorm:"type:jsonb" json:"output_path_json"`   // Maps ML outputs back to twin properties
	ScheduleType    string    `gorm:"not null;default:'event'" json:"schedule_type"` // "event", "interval", "cron"
	ScheduleConfig  string    `gorm:"type:jsonb" json:"schedule_config"`    // Configuration for the schedule
	Active          bool      `gorm:"default:true" json:"active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Relationships
	Task MLTask `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	Twin Twin   `gorm:"foreignKey:TwinID" json:"twin,omitempty"`
}

// MLModelMetadata represents metadata about ML models available in the system
type MLModelMetadata struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	ModelID      string    `gorm:"uniqueIndex;not null" json:"model_id"`
	Name         string    `gorm:"not null" json:"name"`
	Description  string    `json:"description"`
	Type         MLTaskType `gorm:"type:varchar(50);not null" json:"type"`
	Version      string    `gorm:"not null" json:"version"`
	InputSchema  string    `gorm:"type:jsonb" json:"input_schema"`  // JSON schema for inputs
	OutputSchema string    `gorm:"type:jsonb" json:"output_schema"` // JSON schema for outputs
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
} 