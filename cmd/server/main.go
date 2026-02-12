package main

import (
	"log"
	"net/http"

	"github.com/a-h/templ"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/jimdaga/first-sip/internal/auth"
	"github.com/jimdaga/first-sip/internal/config"
	"github.com/jimdaga/first-sip/internal/database"
	"github.com/jimdaga/first-sip/internal/models"
	"github.com/jimdaga/first-sip/internal/templates"
)

// render is a helper function to render Templ components in Gin handlers
func render(c *gin.Context, component templ.Component) {
	c.Header("Content-Type", "text/html")
	component.Render(c.Request.Context(), c.Writer)
}

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize encryption (must be before any model operations)
	if cfg.EncryptionKey != "" {
		if err := models.InitEncryption(cfg.EncryptionKey); err != nil {
			log.Fatalf("Failed to initialize encryption: %v", err)
		}
	}

	// Initialize database connection
	if cfg.DatabaseURL != "" {
		db, err := database.Init(cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer database.Close(db)

		// Run migrations
		if err := database.RunMigrations(db); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}

		// Seed development data
		if cfg.Env != "production" {
			if err := database.SeedDevData(db); err != nil {
				log.Printf("Warning: seed data failed: %v", err)
			}
		}
	}

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

	// Root route - redirect based on auth status
	r.GET("/", func(c *gin.Context) {
		session := sessions.Default(c)
		if session.Get("user_id") != nil {
			c.Redirect(http.StatusFound, "/dashboard")
		} else {
			c.Redirect(http.StatusFound, "/login")
		}
	})

	// Public routes (no authentication required)
	r.GET("/login", func(c *gin.Context) {
		errorMsg := ""
		if c.Query("error") == "auth_failed" {
			errorMsg = "Authentication failed. Please try again."
		} else if c.Query("error") == "session_failed" {
			errorMsg = "Session error. Please try again."
		}
		render(c, templates.LoginPage(errorMsg))
	})
	r.GET("/auth/google", auth.HandleLogin)
	r.GET("/auth/google/callback", auth.HandleCallback)

	// Protected routes (require authentication)
	protected := r.Group("/")
	protected.Use(auth.RequireAuth())
	{
		protected.GET("/dashboard", func(c *gin.Context) {
			// Extract user info from context (set by RequireAuth middleware)
			name, _ := c.Get("user_name")
			email, _ := c.Get("user_email")

			nameStr := ""
			emailStr := ""
			if name != nil {
				nameStr = name.(string)
			}
			if email != nil {
				emailStr = email.(string)
			}

			render(c, templates.DashboardPage(nameStr, emailStr))
		})
		protected.GET("/logout", auth.HandleLogout)
	}

	log.Printf("Starting server on :%s (env: %s)", cfg.Port, cfg.Env)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
