package auth

import (
	"log"

	"github.com/jimdaga/first-sip/internal/config"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/google"
)

// InitProviders initializes Goth OAuth providers
func InitProviders(cfg *config.Config) {
	if cfg.GoogleClientID == "" {
		log.Println("WARNING: GOOGLE_CLIENT_ID not set. OAuth login will not work until credentials are configured.")
		log.Println("See: Google Cloud Console -> APIs & Services -> Credentials -> OAuth 2.0 Client IDs")
		return
	}

	goth.UseProviders(
		google.New(
			cfg.GoogleClientID,
			cfg.GoogleClientSecret,
			cfg.GoogleCallbackURL,
			"email",
			"profile",
		),
	)

	log.Println("Goth providers initialized: google")
}
