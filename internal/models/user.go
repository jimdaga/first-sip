package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents an application user with preferences and activity tracking
type User struct {
	gorm.Model
	Email                string     `gorm:"uniqueIndex:idx_users_email_not_deleted,where:deleted_at IS NULL;not null"`
	Name                 string     `gorm:"not null;default:''"`
	Timezone             string     `gorm:"not null;default:'UTC'"`
	PreferredBriefingTime string    `gorm:"not null;default:'06:00'"`
	Role                 string     `gorm:"not null;default:'user'"` // enum: 'user' or 'admin'
	LastLoginAt          *time.Time
	LastBriefingAt       *time.Time

	// Associations
	AuthIdentities []AuthIdentity `gorm:"constraint:OnDelete:CASCADE;"`
	Briefings      []Briefing     `gorm:"constraint:OnDelete:CASCADE;"`
}
