package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// TwinType represents a type of digital twin with a specific schema
type TwinType struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	Name        string         `gorm:"uniqueIndex;not null" json:"name"`
	Description string         `json:"description"`
	Version     string         `gorm:"not null" json:"version"`
	SchemaJSON  JSON           `gorm:"column:schema_json" json:"schema_json"`
	CreatedBy   uint           `json:"created_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Twins []Twin `gorm:"foreignKey:TypeID" json:"twins,omitempty"`
}

// Twin represents a digital twin instance
type Twin struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	Name        string         `gorm:"not null" json:"name"`
	Description string         `json:"description"`
	DittoID     string         `gorm:"uniqueIndex;not null" json:"ditto_id"`
	TypeID      uint           `gorm:"not null" json:"type_id"`
	ProjectID   uint           `gorm:"not null" json:"project_id"`
	ModelURL    string         `json:"model_url"`
	Metadata    JSON           `json:"metadata"`
	CreatedBy   uint           `json:"created_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Type    TwinType `gorm:"foreignKey:TypeID" json:"type,omitempty"`
	Project Project  `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
}

// ModelBinding represents a binding between a 3D model part and a twin feature
type ModelBinding struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	TwinID      uint           `gorm:"not null" json:"twin_id"`
	PartID      string         `gorm:"not null" json:"part_id"`
	FeaturePath string         `gorm:"not null" json:"feature_path"`
	BindingType string         `gorm:"not null" json:"binding_type"`
	Properties  JSON           `json:"properties"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Twin Twin `gorm:"foreignKey:TwinID" json:"twin,omitempty"`
}

// JSON is a wrapper for json.RawMessage with methods to implement the Scanner and Valuer interfaces
type JSON json.RawMessage

// Value returns the JSON value to be stored in the database
func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return string(j), nil
}

// Scan scans a JSON value from the database
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = JSON("null")
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("invalid scan source for JSON")
	}

	*j = JSON(bytes)
	return nil
}

// MarshalJSON returns the JSON encoding of j
func (j JSON) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON sets *j to a copy of data
func (j *JSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("JSON: UnmarshalJSON on nil pointer")
	}
	*j = JSON(data)
	return nil
}

// DataBinding3D represents a binding between Ditto twin data and 3D model elements
type DataBinding3D struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	TwinID          uint      `gorm:"not null" json:"twin_id"`
	ObjectName      string    `gorm:"not null" json:"object_name"`
	DittoPath       string    `gorm:"not null" json:"ditto_path"`
	BindingType     string    `gorm:"not null" json:"binding_type"`        // color, visibility, text, position, rotation, etc.
	BindingValueMap string    `gorm:"type:jsonb" json:"binding_value_map"` // JSON mapping of value ranges to visual properties
	Description     string    `json:"description"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

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
