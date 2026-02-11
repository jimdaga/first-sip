package auth

import (
	"log"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/jimdaga/first-sip/internal/config"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
)

// InitProviders initializes Goth OAuth providers
func InitProviders(cfg *config.Config) {
	// Configure Gothic's session store to match our app session settings.
	// Gothic uses its own gorilla/sessions store separate from gin-contrib/sessions.
	// The default has Secure=true which breaks localhost (plain HTTP).
	gothStore := sessions.NewCookieStore([]byte(cfg.SessionSecret))
	gothStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 30,
		HttpOnly: true,
		Secure:   cfg.Env == "production",
		SameSite: http.SameSiteLaxMode,
	}
	gothic.Store = gothStore

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
