package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/a-h/templ"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/jimdaga/first-sip/internal/auth"
	"github.com/jimdaga/first-sip/internal/briefings"
	"github.com/jimdaga/first-sip/internal/config"
	"github.com/jimdaga/first-sip/internal/database"
	"github.com/jimdaga/first-sip/internal/models"
	"github.com/jimdaga/first-sip/internal/plugins"
	"github.com/jimdaga/first-sip/internal/templates"
	"github.com/jimdaga/first-sip/internal/webhook"
	"github.com/jimdaga/first-sip/internal/worker"
	"gorm.io/gorm"
)

// render is a helper function to render Templ components in Gin handlers
func render(c *gin.Context, component templ.Component) {
	c.Header("Content-Type", "text/html")
	component.Render(c.Request.Context(), c.Writer)
}

func main() {
	// Parse command-line flags
	workerMode := flag.Bool("worker", false, "Run in worker mode")
	flag.Parse()

	// Load configuration
	cfg := config.Load()

	// Initialize encryption (must be before any model operations)
	if cfg.EncryptionKey != "" {
		if err := models.InitEncryption(cfg.EncryptionKey); err != nil {
			log.Fatalf("Failed to initialize encryption: %v", err)
		}
	}

	// Create webhook client
	webhookClient := webhook.NewClient(cfg.N8NWebhookURL, cfg.N8NWebhookSecret, cfg.N8NStubMode)

	// Initialize Asynq client (runs in BOTH modes so server can enqueue tasks)
	if cfg.RedisURL != "" {
		if err := worker.InitClient(cfg.RedisURL); err != nil {
			log.Fatalf("Failed to initialize Asynq client: %v", err)
		}
		defer worker.CloseClient()
	}

	// Initialize database connection
	var db *gorm.DB
	if cfg.DatabaseURL != "" {
		var err error
		db, err = database.Init(cfg.DatabaseURL)
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

	// Initialize plugin registry
	var pluginRegistry *plugins.Registry
	if db != nil {
		var err error
		pluginRegistry, err = plugins.InitPlugins(db, cfg.PluginDir)
		if err != nil {
			log.Printf("Warning: plugin initialization failed: %v", err)
		} else {
			log.Printf("Plugin registry loaded: %d plugin(s)", pluginRegistry.Count())
		}
	}

	// Mode branching: run as worker or web server
	if *workerMode {
		log.Println("Starting in WORKER mode")
		// Start scheduler in worker mode
		stopScheduler, err := worker.StartScheduler(cfg)
		if err != nil {
			log.Fatalf("Failed to start scheduler: %v", err)
		}
		defer stopScheduler()

		if err := worker.Run(cfg, db, webhookClient); err != nil {
			log.Fatalf("Worker failed: %v", err)
		}
		return
	}

	// Start embedded worker in development mode (non-blocking)
	var stopWorker func()
	var stopScheduler func()
	if cfg.Env == "development" && cfg.RedisURL != "" {
		log.Println("Starting embedded worker for development")
		var err error
		stopWorker, err = worker.Start(cfg, db, webhookClient)
		if err != nil {
			log.Fatalf("Failed to start embedded worker: %v", err)
		}

		// Start embedded scheduler
		log.Println("Starting embedded scheduler for development")
		stopScheduler, err = worker.StartScheduler(cfg)
		if err != nil {
			log.Fatalf("Failed to start embedded scheduler: %v", err)
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

	// Serve static files
	r.Static("/static", "./static")

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
	r.GET("/auth/google/callback", auth.HandleCallback(db))

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

			// Query latest briefing for the user
			var latestBriefing models.Briefing
			var latestBriefingPtr *models.Briefing
			if db != nil && emailStr != "" {
				var user models.User
				if err := db.Where("email = ?", emailStr).First(&user).Error; err == nil {
					result := db.Where("user_id = ?", user.ID).Order("created_at DESC").First(&latestBriefing)
					if result.Error == nil {
						latestBriefingPtr = &latestBriefing
					}
				}
			}

			render(c, templates.DashboardPage(nameStr, emailStr, latestBriefingPtr))
		})
		protected.GET("/logout", auth.HandleLogout)

		// Briefing API routes
		protected.POST("/api/briefings", briefings.CreateBriefingHandler(db))
		protected.GET("/api/briefings/:id/status", briefings.GetBriefingStatusHandler(db))
		protected.POST("/api/briefings/:id/read", briefings.MarkBriefingReadHandler(db))

		// History routes
		protected.GET("/history", briefings.GetHistoryHandler(db))
		protected.GET("/api/history", briefings.GetHistoryPageHandler(db))
		protected.POST("/api/history/briefings/:id/read", briefings.MarkHistoryBriefingReadHandler(db))
	}

	// Create HTTP server for graceful shutdown
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	// Listen for interrupt signals
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start HTTP server in background
	go func() {
		log.Printf("Starting server on :%s (env: %s)", cfg.Port, cfg.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Block until signal
	<-ctx.Done()
	log.Println("Shutting down...")

	// Shut down HTTP server
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Shut down embedded scheduler
	if stopScheduler != nil {
		stopScheduler()
	}

	// Shut down embedded worker
	if stopWorker != nil {
		stopWorker()
	}

	log.Println("Server stopped")
}
