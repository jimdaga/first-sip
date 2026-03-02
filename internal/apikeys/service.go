package apikeys

import (
	"errors"
	"fmt"

	"github.com/jimdaga/first-sip/internal/models"
	"gorm.io/gorm"
)

// SaveKey upserts an API key for a user. If a key already exists for the given
// user/keyType/provider combination, it is updated. Otherwise a new record is created.
// The encryption is handled transparently by the UserAPIKey BeforeSave hook.
func SaveKey(db *gorm.DB, userID uint, keyType, provider, value string) error {
	var existing models.UserAPIKey
	result := db.Where("user_id = ? AND key_type = ? AND provider = ?", userID, keyType, provider).First(&existing)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return fmt.Errorf("looking up existing key: %w", result.Error)
		}
		// Not found — create new
		key := models.UserAPIKey{
			UserID:         userID,
			KeyType:        keyType,
			Provider:       provider,
			EncryptedValue: value,
		}
		if err := db.Create(&key).Error; err != nil {
			return fmt.Errorf("creating API key: %w", err)
		}
		return nil
	}

	// Found — update value
	existing.EncryptedValue = value
	if err := db.Save(&existing).Error; err != nil {
		return fmt.Errorf("updating API key: %w", err)
	}
	return nil
}

// DeleteKey soft-deletes a specific API key, scoped to the owning user to prevent
// deleting another user's key.
func DeleteKey(db *gorm.DB, userID uint, keyID uint) error {
	result := db.Where("id = ? AND user_id = ?", keyID, userID).Delete(&models.UserAPIKey{})
	if result.Error != nil {
		return fmt.Errorf("deleting API key: %w", result.Error)
	}
	return nil
}

// GetKeysForUser returns all active API keys for a user. The AfterFind hook on
// UserAPIKey automatically decrypts each key's EncryptedValue.
func GetKeysForUser(db *gorm.DB, userID uint) ([]models.UserAPIKey, error) {
	var keys []models.UserAPIKey
	if err := db.Where("user_id = ?", userID).Find(&keys).Error; err != nil {
		return nil, fmt.Errorf("fetching API keys: %w", err)
	}
	return keys, nil
}

// GetKeyByID returns a single API key by ID, scoped to the owning user.
func GetKeyByID(db *gorm.DB, userID uint, keyID uint) (*models.UserAPIKey, error) {
	var key models.UserAPIKey
	if err := db.Where("id = ? AND user_id = ?", keyID, userID).First(&key).Error; err != nil {
		return nil, fmt.Errorf("fetching API key: %w", err)
	}
	return &key, nil
}

// SaveLLMPreference updates the user's preferred LLM provider and model.
// Returns an error if the provider or model is not in the supported list.
func SaveLLMPreference(db *gorm.DB, userID uint, provider, model string) error {
	p := GetProviderByID(provider)
	if p == nil {
		return fmt.Errorf("unsupported LLM provider: %q", provider)
	}

	validModel := false
	for _, m := range p.Models {
		if m == model {
			validModel = true
			break
		}
	}
	if !validModel {
		return fmt.Errorf("unsupported model %q for provider %q", model, provider)
	}

	result := db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"llm_preferred_provider": provider,
		"llm_preferred_model":    model,
	})
	if result.Error != nil {
		return fmt.Errorf("saving LLM preference: %w", result.Error)
	}
	return nil
}

// MaskAPIKey returns a masked version of a plaintext API key for display.
// If the key is 7 characters or fewer, returns "***".
// Otherwise returns the first 3 chars + "..." + the last 4 chars.
func MaskAPIKey(plaintext string) string {
	if len(plaintext) <= 7 {
		return "***"
	}
	return plaintext[:3] + "..." + plaintext[len(plaintext)-4:]
}
