package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Briefing status constants
const (
	BriefingStatusPending    = "pending"
	BriefingStatusProcessing = "processing"
	BriefingStatusCompleted  = "completed"
	BriefingStatusFailed     = "failed"
)

// Briefing represents a daily briefing with JSONB content and status lifecycle
type Briefing struct {
	gorm.Model
	UserID       uint           `gorm:"not null;index"`
	User         User           `gorm:"constraint:OnDelete:CASCADE;"`
	Content      datatypes.JSON `gorm:"type:jsonb"`
	Status       string         `gorm:"not null;default:'pending';index"`
	ErrorMessage string         `gorm:"column:error_message;type:text"`
	GeneratedAt  *time.Time
	ReadAt       *time.Time
}
