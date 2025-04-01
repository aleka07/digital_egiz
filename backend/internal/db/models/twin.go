package models

import (
	"time"

	"gorm.io/gorm"
)

// TwinType represents a type of digital twin with specific features and metadata
type TwinType struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	Description string    `json:"description"`
	Version     string    `gorm:"not null" json:"version"`
	SchemaJSON  string    `gorm:"type:jsonb" json:"schema_json"`
	CreatedBy   uint      `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Twins []Twin `gorm:"foreignKey:TypeID" json:"twins,omitempty"`
}

// Twin represents an instance of a digital twin
type Twin struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	Name         string    `gorm:"not null" json:"name"`
	Description  string    `json:"description"`
	TypeID       uint      `gorm:"not null" json:"type_id"`
	ProjectID    uint      `gorm:"not null" json:"project_id"`
	DittoThingID string    `gorm:"uniqueIndex;not null" json:"ditto_thing_id"`
	MetadataJSON string    `gorm:"type:jsonb" json:"metadata_json"`
	Has3DModel   bool      `gorm:"default:false" json:"has_3d_model"`
	CreatedBy    uint      `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Type          TwinType       `gorm:"foreignKey:TypeID" json:"type,omitempty"`
	Project       Project        `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	DataBindings3D []DataBinding3D `gorm:"foreignKey:TwinID" json:"data_bindings_3d,omitempty"`
}

// DataBinding3D represents a binding between Ditto twin data and 3D model elements
type DataBinding3D struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	TwinID         uint      `gorm:"not null" json:"twin_id"`
	ObjectName     string    `gorm:"not null" json:"object_name"`
	DittoPath      string    `gorm:"not null" json:"ditto_path"`
	BindingType    string    `gorm:"not null" json:"binding_type"` // color, visibility, text, position, rotation, etc.
	BindingValueMap string   `gorm:"type:jsonb" json:"binding_value_map"` // JSON mapping of value ranges to visual properties
	Description    string    `json:"description"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// Relationships
	Twin Twin `gorm:"foreignKey:TwinID" json:"twin,omitempty"`
}

// TwinModel3D represents a 3D model for a digital twin
type TwinModel3D struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	TwinID      uint      `gorm:"uniqueIndex;not null" json:"twin_id"`
	ModelURL    string    `gorm:"not null" json:"model_url"`
	ModelFormat string    `gorm:"not null;default:'gltf'" json:"model_format"`
	VersionTag  string    `json:"version_tag"`
	CreatedBy   uint      `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relationships
	Twin Twin `gorm:"foreignKey:TwinID" json:"twin,omitempty"`
} 