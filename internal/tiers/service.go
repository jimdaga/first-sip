package tiers

import (
	"fmt"
	"time"

	cron "github.com/robfig/cron/v3"
	"github.com/jimdaga/first-sip/internal/models"
	"github.com/jimdaga/first-sip/internal/plugins"
	"gorm.io/gorm"
)

// cronParser parses standard 5-field cron expressions (Minute Hour Dom Month Dow).
var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// TierService provides account tier constraint checking for users.
type TierService struct {
	db *gorm.DB
}

// New creates a TierService backed by the given database connection.
func New(db *gorm.DB) *TierService {
	return &TierService{db: db}
}

// GetUserTier returns the AccountTier for the given user.
// If the user's AccountTierID is NULL (pre-migration users), falls back to the "free" tier.
func (s *TierService) GetUserTier(userID uint) (*models.AccountTier, error) {
	var user models.User
	if err := s.db.Select("account_tier_id").Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, fmt.Errorf("tiers: user %d not found: %w", userID, err)
	}

	if user.AccountTierID == nil {
		// Fall back to free tier for users who predate the migration
		return s.freeTier()
	}

	var tier models.AccountTier
	if err := s.db.First(&tier, *user.AccountTierID).Error; err != nil {
		return nil, fmt.Errorf("tiers: account tier %d not found: %w", *user.AccountTierID, err)
	}
	return &tier, nil
}

// CanEnablePlugin reports whether the user is allowed to enable one more plugin
// given their tier's MaxEnabledPlugins limit.
// Returns (allowed bool, tier *models.AccountTier, error).
func (s *TierService) CanEnablePlugin(userID uint) (bool, *models.AccountTier, error) {
	tier, err := s.GetUserTier(userID)
	if err != nil {
		return false, nil, err
	}

	// -1 means unlimited
	if tier.MaxEnabledPlugins < 0 {
		return true, tier, nil
	}

	var count int64
	if err := s.db.Model(&plugins.UserPluginConfig{}).
		Where("user_id = ? AND enabled = true AND deleted_at IS NULL", userID).
		Count(&count).Error; err != nil {
		return false, nil, fmt.Errorf("tiers: count enabled plugins for user %d: %w", userID, err)
	}

	return count < int64(tier.MaxEnabledPlugins), tier, nil
}

// CanUseFrequency reports whether the given cron expression satisfies the user's
// tier minimum frequency (MinFrequencyHours).
// Returns (allowed bool, tier *models.AccountTier, error).
func (s *TierService) CanUseFrequency(userID uint, cronExpr string) (bool, *models.AccountTier, error) {
	tier, err := s.GetUserTier(userID)
	if err != nil {
		return false, nil, err
	}

	// 0 or negative means no restriction
	if tier.MinFrequencyHours <= 0 {
		return true, tier, nil
	}

	schedule, err := cronParser.Parse(cronExpr)
	if err != nil {
		return false, tier, fmt.Errorf("tiers: invalid cron expression %q: %w", cronExpr, err)
	}

	// Compute the interval between two consecutive fires
	now := time.Now()
	next1 := schedule.Next(now)
	next2 := schedule.Next(next1)
	interval := next2.Sub(next1)

	minDuration := time.Duration(tier.MinFrequencyHours) * time.Hour
	return interval >= minDuration, tier, nil
}

// GetEnabledCount returns the number of currently enabled plugins for the user.
// Used for tier UI counter display.
func (s *TierService) GetEnabledCount(userID uint) (int, error) {
	var count int64
	if err := s.db.Model(&plugins.UserPluginConfig{}).
		Where("user_id = ? AND enabled = true AND deleted_at IS NULL", userID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("tiers: count enabled plugins for user %d: %w", userID, err)
	}
	return int(count), nil
}

// freeTier fetches the "free" tier from the database.
func (s *TierService) freeTier() (*models.AccountTier, error) {
	var tier models.AccountTier
	if err := s.db.Where("name = ?", "free").First(&tier).Error; err != nil {
		return nil, fmt.Errorf("tiers: free tier not found (run SeedAccountTiers): %w", err)
	}
	return &tier, nil
}
