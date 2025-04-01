package models

import (
	"time"

	"gorm.io/gorm"
)

// ProjectRole represents a user's role within a project
type ProjectRole string

const (
	// ProjectRoleOwner has full access to the project
	ProjectRoleOwner ProjectRole = "owner"
	// ProjectRoleEditor can modify twins and data but can't delete the project
	ProjectRoleEditor ProjectRole = "editor"
	// ProjectRoleViewer can only view twins and data
	ProjectRoleViewer ProjectRole = "viewer"
)

// Project represents a collection of digital twins and resources
type Project struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `json:"description"`
	CreatedBy   uint      `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	
	// Relationships
	Members []ProjectMember `gorm:"foreignKey:ProjectID" json:"members,omitempty"`
	Twins   []Twin          `gorm:"foreignKey:ProjectID" json:"twins,omitempty"`
}

// ProjectMember represents a user's membership in a project
type ProjectMember struct {
	ID        uint        `gorm:"primarykey" json:"id"`
	ProjectID uint        `gorm:"not null" json:"project_id"`
	UserID    uint        `gorm:"not null" json:"user_id"`
	Role      ProjectRole `gorm:"type:varchar(20);default:'viewer'" json:"role"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`

	// Relationships
	User    User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Project Project `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
}

// HasPermission checks if the user has the required permission level for a project
func (p *ProjectMember) HasPermission(requiredRole ProjectRole) bool {
	switch p.Role {
	case ProjectRoleOwner:
		return true
	case ProjectRoleEditor:
		return requiredRole == ProjectRoleEditor || requiredRole == ProjectRoleViewer
	case ProjectRoleViewer:
		return requiredRole == ProjectRoleViewer
	default:
		return false
	}
}

// BeforeCreate hook for Project to set the created_by field if not set
func (p *Project) BeforeCreate(tx *gorm.DB) error {
	// Set created_by from the current user if not already set
	if p.CreatedBy == 0 {
		// Use GORM session value if available
		if userID, ok := tx.Statement.Context.Value("current_user_id").(uint); ok {
			p.CreatedBy = userID
		}
	}
	return nil
} 