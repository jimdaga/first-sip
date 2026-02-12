package models

import (
	"time"

	"github.com/jimdaga/first-sip/internal/crypto"
	"gorm.io/gorm"
)

var encryptor *crypto.TokenEncryptor

// InitEncryption initializes the token encryptor for the models package.
// Must be called before any database operations involving AuthIdentity.
func InitEncryption(encryptionKey string) error {
	var err error
	encryptor, err = crypto.NewTokenEncryptor(encryptionKey)
	return err
}

// AuthIdentity represents a user's OAuth identity with encrypted token storage
type AuthIdentity struct {
	gorm.Model
	UserID         uint   `gorm:"not null;index"`
	User           User   `gorm:"constraint:OnDelete:CASCADE;"`
	Provider       string `gorm:"not null"`                                                                   // e.g., "google"
	ProviderUserID string `gorm:"not null;uniqueIndex:idx_auth_identities_provider_user,where:deleted_at IS NULL"` // partial unique index
	AccessToken    string `gorm:"type:text"`                                                                  // stored encrypted
	RefreshToken   string `gorm:"type:text"`                                                                  // stored encrypted
	TokenExpiry    *time.Time
}

// BeforeSave encrypts tokens before saving to database.
// Always encrypts non-empty tokens (GCM produces different output each time due to random nonce).
func (a *AuthIdentity) BeforeSave(tx *gorm.DB) error {
	if encryptor == nil {
		// Allow operations without encryption (e.g., for testing or if encryption not initialized)
		return nil
	}

	// Encrypt access token if not empty
	if a.AccessToken != "" {
		encrypted, err := encryptor.Encrypt(a.AccessToken)
		if err != nil {
			return err
		}
		a.AccessToken = encrypted
	}

	// Encrypt refresh token if not empty
	if a.RefreshToken != "" {
		encrypted, err := encryptor.Encrypt(a.RefreshToken)
		if err != nil {
			return err
		}
		a.RefreshToken = encrypted
	}

	return nil
}

// AfterFind decrypts tokens after loading from database
func (a *AuthIdentity) AfterFind(tx *gorm.DB) error {
	if encryptor == nil {
		// Allow operations without encryption
		return nil
	}

	// Decrypt access token if not empty
	if a.AccessToken != "" {
		decrypted, err := encryptor.Decrypt(a.AccessToken)
		if err != nil {
			return err
		}
		a.AccessToken = decrypted
	}

	// Decrypt refresh token if not empty
	if a.RefreshToken != "" {
		decrypted, err := encryptor.Decrypt(a.RefreshToken)
		if err != nil {
			return err
		}
		a.RefreshToken = decrypted
	}

	return nil
}
