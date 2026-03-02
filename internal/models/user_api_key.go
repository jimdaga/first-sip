package models

import (
	"gorm.io/gorm"
)

// UserAPIKey stores an encrypted API key for a user and provider
type UserAPIKey struct {
	gorm.Model
	UserID         uint   `gorm:"not null;index"`
	User           User   `gorm:"constraint:OnDelete:CASCADE;"`
	KeyType        string `gorm:"not null"`                                                                                          // "llm" or "tavily"
	Provider       string `gorm:"not null;default:''"`                                                                               // e.g., "openai", "anthropic", "groq", or "" for tavily
	EncryptedValue string `gorm:"type:text;not null"`                                                                                // stored encrypted
}

// BeforeSave encrypts the API key value before saving to database.
// Always encrypts non-empty values (GCM produces different output each time due to random nonce).
func (k *UserAPIKey) BeforeSave(tx *gorm.DB) error {
	if encryptor == nil {
		// Allow operations without encryption (e.g., for testing or if encryption not initialized)
		return nil
	}

	if k.EncryptedValue != "" {
		encrypted, err := encryptor.Encrypt(k.EncryptedValue)
		if err != nil {
			return err
		}
		k.EncryptedValue = encrypted
	}

	return nil
}

// AfterFind decrypts the API key value after loading from database
func (k *UserAPIKey) AfterFind(tx *gorm.DB) error {
	if encryptor == nil {
		// Allow operations without encryption
		return nil
	}

	if k.EncryptedValue != "" {
		decrypted, err := encryptor.Decrypt(k.EncryptedValue)
		if err != nil {
			return err
		}
		k.EncryptedValue = decrypted
	}

	return nil
}
