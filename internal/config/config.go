package config

import (
	"log"
	"os"
)

// Config holds application configuration loaded from environment variables
type Config struct {
	GoogleClientID     string
	GoogleClientSecret string
	GoogleCallbackURL  string
	SessionSecret      string
	DatabaseURL        string
	EncryptionKey      string
	Env                string
	Port               string
}

// Load reads configuration from environment variables
func Load() *Config {
	cfg := &Config{
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleCallbackURL:  os.Getenv("GOOGLE_CALLBACK_URL"),
		SessionSecret:      os.Getenv("SESSION_SECRET"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		EncryptionKey:      os.Getenv("ENCRYPTION_KEY"),
		Env:                getEnvWithDefault("ENV", "development"),
		Port:               getEnvWithDefault("PORT", "8080"),
	}

	// Warn if using default session secret (insecure for production)
	if cfg.SessionSecret == "" {
		cfg.SessionSecret = "dev-secret-change-in-production-use-openssl-rand-hex-32"
		log.Println("WARNING: Using default SESSION_SECRET. Generate a secure secret with: openssl rand -hex 32")
	}

	// Check for required database configuration
	if cfg.DatabaseURL == "" {
		if cfg.Env == "production" {
			log.Fatal("DATABASE_URL is required in production")
		}
		log.Println("WARNING: DATABASE_URL not set. Database features will be unavailable.")
	}

	// Check for required encryption configuration
	if cfg.EncryptionKey == "" {
		if cfg.Env == "production" {
			log.Fatal("ENCRYPTION_KEY is required in production")
		}
		log.Println("WARNING: ENCRYPTION_KEY not set. Token encryption will be unavailable.")
	}

	return cfg
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
