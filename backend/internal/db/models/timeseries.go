package models

import (
	"time"
)

// TimeseriesData represents a generic time-series data point
type TimeseriesData struct {
	Time      time.Time `gorm:"type:timestamptz;primaryKey;not null" json:"time"`
	TwinID    string    `gorm:"type:varchar(255);primaryKey;not null" json:"twin_id"`
	FeaturePath string  `gorm:"type:varchar(255);primaryKey;not null" json:"feature_path"`
	ValueType string    `gorm:"type:varchar(50);not null" json:"value_type"` // "number", "boolean", "string", "object"
	ValueNum  float64   `json:"value_num,omitempty"`
	ValueBool *bool     `json:"value_bool,omitempty"`
	ValueStr  string    `json:"value_str,omitempty"`
	ValueJSON string    `gorm:"type:jsonb" json:"value_json,omitempty"`
	Source    string    `gorm:"type:varchar(255)" json:"source"` // Event source (e.g., "ditto", "simulator")
}

// TableName overrides the table name for TimeseriesData
func (TimeseriesData) TableName() string {
	return "timeseries_data"
}

// AggregatedData represents aggregated time-series data
type AggregatedData struct {
	TimeInterval   time.Time `gorm:"type:timestamptz;primaryKey;not null" json:"time_interval"`
	TwinID         string    `gorm:"type:varchar(255);primaryKey;not null" json:"twin_id"`
	FeaturePath    string    `gorm:"type:varchar(255);primaryKey;not null" json:"feature_path"`
	IntervalType   string    `gorm:"type:varchar(20);primaryKey;not null" json:"interval_type"` // "minute", "hour", "day", "month"
	Min            float64   `json:"min"`
	Max            float64   `json:"max"`
	Avg            float64   `json:"avg"`
	Sum            float64   `json:"sum"`
	Count          int       `json:"count"`
	FirstTime      time.Time `gorm:"type:timestamptz" json:"first_time"`
	LastTime       time.Time `gorm:"type:timestamptz" json:"last_time"`
}

// TableName overrides the table name for AggregatedData
func (AggregatedData) TableName() string {
	return "aggregated_data"
}

// AlertData represents time-series alert data
type AlertData struct {
	Time        time.Time `gorm:"type:timestamptz;primaryKey;not null" json:"time"`
	AlertID     string    `gorm:"type:varchar(255);primaryKey;not null" json:"alert_id"`
	TwinID      string    `gorm:"type:varchar(255);not null" json:"twin_id"`
	FeaturePath string    `gorm:"type:varchar(255)" json:"feature_path,omitempty"`
	Severity    string    `gorm:"type:varchar(20);not null" json:"severity"` // "info", "warning", "error", "critical"
	Message     string    `json:"message"`
	ValueJSON   string    `gorm:"type:jsonb" json:"value_json,omitempty"`
	Source      string    `gorm:"type:varchar(255)" json:"source"` // Alert source (e.g., "ml", "rule", "manual")
	Acknowledged bool     `gorm:"default:false" json:"acknowledged"`
	AckBy       string    `json:"ack_by,omitempty"`
	AckTime     time.Time `gorm:"type:timestamptz" json:"ack_time,omitempty"`
}

// TableName overrides the table name for AlertData
func (AlertData) TableName() string {
	return "alert_data"
}

// MLPredictionData represents time-series ML prediction data
type MLPredictionData struct {
	Time       time.Time `gorm:"type:timestamptz;primaryKey;not null" json:"time"`
	TwinID     string    `gorm:"type:varchar(255);primaryKey;not null" json:"twin_id"`
	TaskID     string    `gorm:"type:varchar(255);primaryKey;not null" json:"task_id"`
	PredictionType string `gorm:"type:varchar(50);not null" json:"prediction_type"` // "anomaly", "classification", "regression"
	ScoreNum   float64   `json:"score_num,omitempty"`
	LabelStr   string    `json:"label_str,omitempty"`
	DetailsJSON string   `gorm:"type:jsonb" json:"details_json,omitempty"`
	ModelVersion string  `gorm:"type:varchar(100)" json:"model_version"`
}

// TableName overrides the table name for MLPredictionData
func (MLPredictionData) TableName() string {
	return "ml_prediction_data"
} 