package main

import (
	"log"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/jimdaga/first-sip/internal/auth"
	"github.com/jimdaga/first-sip/internal/config"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Create Gin router
	r := gin.Default()

	// Set up cookie-based session store
	store := cookie.NewStore([]byte(cfg.SessionSecret))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 30, // 30 days
		HttpOnly: true,
		Secure:   cfg.Env == "production",
		SameSite: http.SameSiteLaxMode, // Lax allows OAuth redirects, Strict would block them
	})

	// Register session middleware BEFORE routes
	r.Use(sessions.Sessions("first_sip_session", store))

	// Initialize Goth OAuth providers (after session middleware)
	auth.InitProviders(cfg)

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Public routes (no authentication required)
	r.GET("/login", func(c *gin.Context) {
		c.String(200, "Login page (coming in Plan 02)")
	})
	r.GET("/auth/google", auth.HandleLogin)
	r.GET("/auth/google/callback", auth.HandleCallback)

	// Protected routes (require authentication)
	protected := r.Group("/")
	protected.Use(auth.RequireAuth())
	{
		protected.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusFound, "/dashboard")
		})
		protected.GET("/dashboard", func(c *gin.Context) {
			c.String(200, "Dashboard (coming in Plan 02)")
		})
		protected.GET("/logout", auth.HandleLogout)
	}

	log.Printf("Starting server on :%s (env: %s)", cfg.Port, cfg.Env)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
