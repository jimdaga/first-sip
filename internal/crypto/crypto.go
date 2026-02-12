package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// TokenEncryptor handles AES-256-GCM encryption and decryption of OAuth tokens
type TokenEncryptor struct {
	gcm cipher.AEAD
}

// NewTokenEncryptor creates a new TokenEncryptor with the provided base64-encoded key.
// The key must be exactly 32 bytes (AES-256) after base64 decoding.
func NewTokenEncryptor(base64Key string) (*TokenEncryptor, error) {
	if base64Key == "" {
		return nil, fmt.Errorf("encryption key is required")
	}

	// Decode base64 key
	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encryption key: %w", err)
	}

	// Validate key length for AES-256
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes for AES-256, got %d bytes", len(key))
	}

	// Create AES cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher block: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM mode: %w", err)
	}

	return &TokenEncryptor{gcm: gcm}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM and returns base64-encoded ciphertext with nonce prepended.
// Format: base64(nonce || ciphertext)
func (e *TokenEncryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil // Empty plaintext results in empty ciphertext
	}

	// Generate random nonce
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt plaintext
	ciphertext := e.gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Return base64-encoded result
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64-encoded ciphertext using AES-256-GCM and returns plaintext.
// Expects format: base64(nonce || ciphertext)
func (e *TokenEncryptor) Decrypt(base64Ciphertext string) (string, error) {
	if base64Ciphertext == "" {
		return "", nil // Empty ciphertext results in empty plaintext
	}

	// Decode base64
	ciphertext, err := base64.StdEncoding.DecodeString(base64Ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	// Check minimum length (nonce + at least 1 byte + auth tag)
	nonceSize := e.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := e.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}
