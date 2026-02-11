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
		Env:                getEnvWithDefault("ENV", "development"),
		Port:               getEnvWithDefault("PORT", "8080"),
	}

	// Warn if using default session secret (insecure for production)
	if cfg.SessionSecret == "" {
		cfg.SessionSecret = "dev-secret-change-in-production-use-openssl-rand-hex-32"
		log.Println("WARNING: Using default SESSION_SECRET. Generate a secure secret with: openssl rand -hex 32")
	}

	return cfg
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
