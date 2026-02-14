package plugins

import (
	"time"

	"github.com/jimdaga/first-sip/internal/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// PluginRun status constants
const (
	PluginRunStatusPending    = "pending"
	PluginRunStatusProcessing = "processing"
	PluginRunStatusCompleted  = "completed"
	PluginRunStatusFailed     = "failed"
)

// Plugin represents a discovered plugin with its metadata
type Plugin struct {
	gorm.Model
	Name               string         `gorm:"uniqueIndex;not null"`
	Description        string         `gorm:"type:text"`
	Owner              string
	Version            string         `gorm:"not null"`
	SchemaVersion      string         `gorm:"column:schema_version;not null;default:'v1'"`
	Capabilities       datatypes.JSON `gorm:"type:jsonb"`
	DefaultConfig      datatypes.JSON `gorm:"type:jsonb;column:default_config"`
	SettingsSchemaPath string         `gorm:"column:settings_schema_path"`
	Enabled            bool           `gorm:"default:true"`
}

// UserPluginConfig stores per-user per-plugin settings
type UserPluginConfig struct {
	gorm.Model
	UserID   uint           `gorm:"not null;uniqueIndex:idx_user_plugin"`
	PluginID uint           `gorm:"not null;uniqueIndex:idx_user_plugin"`
	Settings datatypes.JSON `gorm:"type:jsonb"`
	Enabled  bool           `gorm:"default:false"`
	User     models.User    `gorm:"constraint:OnDelete:CASCADE;"`
	Plugin   Plugin         `gorm:"constraint:OnDelete:CASCADE;"`
}

// PluginRun tracks a single plugin execution
type PluginRun struct {
	gorm.Model
	PluginRunID  string         `gorm:"uniqueIndex;not null"`
	UserID       uint           `gorm:"not null;index"`
	PluginID     uint           `gorm:"not null;index"`
	Status       string         `gorm:"not null;default:'pending';index"`
	Input        datatypes.JSON `gorm:"type:jsonb"`
	Output       datatypes.JSON `gorm:"type:jsonb"`
	ErrorMessage string         `gorm:"column:error_message;type:text"`
	StartedAt    *time.Time     `gorm:"column:started_at"`
	CompletedAt  *time.Time     `gorm:"column:completed_at"`
	User         models.User    `gorm:"constraint:OnDelete:CASCADE;"`
	Plugin       Plugin         `gorm:"constraint:OnDelete:CASCADE;"`
}
