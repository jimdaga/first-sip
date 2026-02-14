package database

import (
	"log"

	"github.com/jimdaga/first-sip/internal/models"
	"github.com/jimdaga/first-sip/internal/plugins"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SeedDevData populates the database with development test data.
// Idempotent: skips if data already exists.
func SeedDevData(db *gorm.DB) error {
	// Check if seed data already exists
	var existingUser models.User
	result := db.Where("email = ?", "dev@firstsip.local").First(&existingUser)
	if result.Error == nil {
		log.Println("Seed data already exists, skipping")
		return nil
	}

	// Create test user
	user := models.User{
		Email:                 "dev@firstsip.local",
		Name:                  "Dev User",
		Timezone:              "America/Chicago",
		PreferredBriefingTime: "07:00",
		Role:                  "user",
	}

	if err := db.Create(&user).Error; err != nil {
		return err
	}

	// Create sample AuthIdentity for the test user
	identity := models.AuthIdentity{
		UserID:         user.ID,
		Provider:       "google",
		ProviderUserID: "dev-google-id-12345",
		AccessToken:    "dev-access-token-placeholder",
		RefreshToken:   "dev-refresh-token-placeholder",
	}

	if err := db.Create(&identity).Error; err != nil {
		return err
	}

	// Create sample completed briefing
	briefing := models.Briefing{
		UserID: user.ID,
		Status: models.BriefingStatusCompleted,
		Content: datatypes.JSON([]byte(`{
			"news": [
				{"title": "Go 1.23 Released", "summary": "Latest Go release includes improved standard library.", "source": "go.dev"},
				{"title": "Tech Industry Update", "summary": "Major developments in AI and cloud computing.", "source": "techcrunch.com"}
			],
			"weather": {
				"location": "Chicago, IL",
				"temperature": "72F",
				"condition": "Partly Cloudy",
				"forecast": "Clear skies expected through the week."
			},
			"work": [
				{"title": "Sprint Review", "summary": "Team demo at 2 PM today.", "source": "calendar"},
				{"title": "PR Review Pending", "summary": "3 pull requests awaiting review.", "source": "github"}
			]
		}`)),
	}

	if err := db.Create(&briefing).Error; err != nil {
		return err
	}

	// Create sample pending briefing
	pendingBriefing := models.Briefing{
		UserID: user.ID,
		Status: models.BriefingStatusPending,
	}

	if err := db.Create(&pendingBriefing).Error; err != nil {
		return err
	}

	// Create plugin configuration for dev user
	var plugin plugins.Plugin
	result = db.Where("name = ?", "daily-news-digest").First(&plugin)
	if result.Error == nil {
		// Plugin exists - create user configuration
		userPluginConfig := plugins.UserPluginConfig{
			UserID:   user.ID,
			PluginID: plugin.ID,
			Enabled:  true,
			Settings: datatypes.JSON([]byte(`{"frequency":"daily","preferred_time":"07:00","topics":["technology","business"]}`)),
		}
		db.Where("user_id = ? AND plugin_id = ?", user.ID, plugin.ID).FirstOrCreate(&userPluginConfig)
		log.Println("Seeded dev data: 1 user, 1 auth identity, 2 briefings, 1 plugin config")
	} else {
		// Plugin not found - log warning but don't fail
		log.Printf("Warning: daily-news-digest plugin not found, skipping plugin config seed: %v", result.Error)
		log.Println("Seeded dev data: 1 user, 1 auth identity, 2 briefings")
	}

	return nil
}
