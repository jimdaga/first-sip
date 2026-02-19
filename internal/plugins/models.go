package plugins

import (
	"fmt"
	"time"

	cron "github.com/robfig/cron/v3"
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

// UserPluginConfig stores per-user per-plugin settings including optional scheduling
type UserPluginConfig struct {
	gorm.Model
	UserID         uint           `gorm:"not null;uniqueIndex:idx_user_plugin"`
	PluginID       uint           `gorm:"not null;uniqueIndex:idx_user_plugin"`
	Settings       datatypes.JSON `gorm:"type:jsonb"`
	Enabled        bool           `gorm:"default:false"`
	CronExpression string         `gorm:"column:cron_expression"` // nullable — empty means no schedule
	Timezone       string         `gorm:"column:timezone;not null;default:'UTC'"` // IANA timezone name
	User           models.User    `gorm:"constraint:OnDelete:CASCADE;"`
	Plugin         Plugin         `gorm:"constraint:OnDelete:CASCADE;"`
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

// cronParser is a shared parser for standard 5-field cron expressions.
var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// ValidateCronExpression parses a 5-field cron expression and returns an error
// if the expression is invalid. Returns nil if the expression is valid.
// Used at write time (settings UI) and evaluation time (scheduler).
func ValidateCronExpression(expr string) error {
	if expr == "" {
		return fmt.Errorf("cron expression must not be empty")
	}
	_, err := cronParser.Parse(expr)
	if err != nil {
		return fmt.Errorf("invalid cron expression %q: %w", expr, err)
	}
	return nil
}
