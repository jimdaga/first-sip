package models

import "gorm.io/gorm"

// AccountTier defines the limits and permissions for a user's subscription tier.
// Free tier: max 3 enabled plugins, minimum 24-hour cron interval.
// Pro tier: unlimited plugins (MaxEnabledPlugins = -1), minimum 2-hour interval.
type AccountTier struct {
	gorm.Model
	Name              string `gorm:"uniqueIndex;not null"`
	MaxEnabledPlugins int    `gorm:"not null"` // -1 means unlimited
	MinFrequencyHours int    `gorm:"not null"` // minimum cron interval in hours
}
