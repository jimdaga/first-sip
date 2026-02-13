package briefings

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jimdaga/first-sip/internal/models"
	"github.com/jimdaga/first-sip/internal/templates"
	"github.com/jimdaga/first-sip/internal/worker"
	"gorm.io/gorm"
)

// CreateBriefingHandler creates a new briefing and enqueues generation task
func CreateBriefingHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user email from context (set by auth middleware)
		userEmail, exists := c.Get("user_email")
		if !exists {
			c.Status(http.StatusUnauthorized)
			return
		}

		// Look up user record by email to get the GORM uint ID
		var user models.User
		if err := db.Where("email = ?", userEmail.(string)).First(&user).Error; err != nil {
			c.Header("Content-Type", "text/html")
			c.String(http.StatusInternalServerError, `<div class="alert alert-error">Failed to lookup user</div>`)
			return
		}

		// Check if there's already a pending/processing briefing for this user
		var existing models.Briefing
		result := db.Where("user_id = ? AND status IN ?", user.ID, []string{models.BriefingStatusPending, models.BriefingStatusProcessing}).First(&existing)
		if result.Error == nil {
			// Found existing pending/processing briefing - return it instead of creating duplicate
			c.Header("Content-Type", "text/html")
			templates.BriefingCard(existing).Render(c.Request.Context(), c.Writer)
			return
		}

		// Create new briefing with pending status
		briefing := models.Briefing{
			UserID: user.ID,
			Status: models.BriefingStatusPending,
		}

		if err := db.Create(&briefing).Error; err != nil {
			c.Header("Content-Type", "text/html")
			c.String(http.StatusInternalServerError, `<div class="alert alert-error">Failed to create briefing</div>`)
			return
		}

		// Enqueue worker task
		if err := worker.EnqueueGenerateBriefing(briefing.ID); err != nil {
			// Update briefing to failed status
			db.Model(&briefing).Updates(map[string]interface{}{
				"status":        models.BriefingStatusFailed,
				"error_message": "Failed to enqueue generation task",
			})
			c.Header("Content-Type", "text/html")
			c.String(http.StatusInternalServerError, `<div class="alert alert-error">Failed to enqueue briefing generation</div>`)
			return
		}

		// Return HTML fragment with briefing card
		c.Header("Content-Type", "text/html")
		templates.BriefingCard(briefing).Render(c.Request.Context(), c.Writer)
	}
}

// GetBriefingStatusHandler returns the current status of a briefing
func GetBriefingStatusHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse briefing ID from URL parameter
		briefingID := c.Param("id")

		// Query briefing
		var briefing models.Briefing
		if err := db.First(&briefing, briefingID).Error; err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		// Return full briefing card (allows content to appear when completed)
		c.Header("Content-Type", "text/html")
		templates.BriefingCard(briefing).Render(c.Request.Context(), c.Writer)
	}
}

// MarkBriefingReadHandler marks a briefing as read and returns updated card HTML
func MarkBriefingReadHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse briefing ID from URL parameter
		briefingID := c.Param("id")

		// Query briefing
		var briefing models.Briefing
		if err := db.First(&briefing, briefingID).Error; err != nil {
			c.Header("Content-Type", "text/html")
			c.String(http.StatusNotFound, `<div class="alert alert-error">Briefing not found</div>`)
			return
		}

		// Only update if not already read (idempotent)
		if briefing.ReadAt == nil {
			now := time.Now()
			if err := db.Model(&briefing).Update("read_at", now).Error; err != nil {
				c.Header("Content-Type", "text/html")
				c.String(http.StatusInternalServerError, `<div class="alert alert-error">Failed to mark as read</div>`)
				return
			}
			// Update in-memory model for rendering
			briefing.ReadAt = &now
		}

		// Return updated briefing card HTML
		c.Header("Content-Type", "text/html")
		templates.BriefingCard(briefing).Render(c.Request.Context(), c.Writer)
	}
}

// GetHistoryHandler renders the full history page
func GetHistoryHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user email from context (set by auth middleware)
		userEmail, exists := c.Get("user_email")
		if !exists {
			c.Status(http.StatusUnauthorized)
			return
		}

		// Look up user record by email to get the GORM uint ID
		var user models.User
		if err := db.Where("email = ?", userEmail.(string)).First(&user).Error; err != nil {
			c.Header("Content-Type", "text/html")
			c.String(http.StatusInternalServerError, `<div class="content-error">Failed to lookup user</div>`)
			return
		}

		// Query briefings from last 30 days, completed and failed only
		thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
		var briefings []models.Briefing
		if err := db.Where("user_id = ? AND created_at >= ?", user.ID, thirtyDaysAgo).
			Where("status IN ?", []string{models.BriefingStatusCompleted, models.BriefingStatusFailed}).
			Order("created_at DESC").
			Limit(11). // Fetch pageSize+1 to detect hasMore
			Find(&briefings).Error; err != nil {
			c.Header("Content-Type", "text/html")
			c.String(http.StatusInternalServerError, `<div class="content-error">Failed to load briefings</div>`)
			return
		}

		// Detect if there are more results
		hasMore := len(briefings) > 10
		if hasMore {
			briefings = briefings[:10] // Trim to pageSize
		}

		// Get user name from context for page header
		name, _ := c.Get("user_name")
		nameStr := ""
		if name != nil {
			nameStr = name.(string)
		}

		// Render full history page
		c.Header("Content-Type", "text/html")
		templates.HistoryPage(nameStr, briefings, 0, hasMore).Render(c.Request.Context(), c.Writer)
	}
}

// GetHistoryPageHandler renders paginated history results (HTMX fragment)
func GetHistoryPageHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user email from context (set by auth middleware)
		userEmail, exists := c.Get("user_email")
		if !exists {
			c.Status(http.StatusUnauthorized)
			return
		}

		// Look up user record by email to get the GORM uint ID
		var user models.User
		if err := db.Where("email = ?", userEmail.(string)).First(&user).Error; err != nil {
			c.Header("Content-Type", "text/html")
			c.String(http.StatusInternalServerError, `<div class="content-error">Failed to lookup user</div>`)
			return
		}

		// Parse page query parameter
		page, _ := strconv.Atoi(c.DefaultQuery("page", "0"))
		if page < 0 {
			page = 0
		}

		// Query briefings from last 30 days with pagination
		thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
		var briefings []models.Briefing
		if err := db.Where("user_id = ? AND created_at >= ?", user.ID, thirtyDaysAgo).
			Where("status IN ?", []string{models.BriefingStatusCompleted, models.BriefingStatusFailed}).
			Order("created_at DESC").
			Offset(page * 10).
			Limit(11). // Fetch pageSize+1 to detect hasMore
			Find(&briefings).Error; err != nil {
			c.Header("Content-Type", "text/html")
			c.String(http.StatusInternalServerError, `<div class="content-error">Failed to load briefings</div>`)
			return
		}

		// Detect if there are more results
		hasMore := len(briefings) > 10
		if hasMore {
			briefings = briefings[:10] // Trim to pageSize
		}

		// Render history list fragment
		c.Header("Content-Type", "text/html")
		templates.HistoryList(briefings, page, hasMore).Render(c.Request.Context(), c.Writer)
	}
}

// MarkHistoryBriefingReadHandler marks a briefing as read and returns updated history card HTML
func MarkHistoryBriefingReadHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse briefing ID from URL parameter
		briefingID := c.Param("id")

		// Query briefing
		var briefing models.Briefing
		if err := db.First(&briefing, briefingID).Error; err != nil {
			c.Header("Content-Type", "text/html")
			c.String(http.StatusNotFound, `<div class="content-error">Briefing not found</div>`)
			return
		}

		// Only update if not already read (idempotent)
		if briefing.ReadAt == nil {
			now := time.Now()
			if err := db.Model(&briefing).Update("read_at", now).Error; err != nil {
				c.Header("Content-Type", "text/html")
				c.String(http.StatusInternalServerError, `<div class="content-error">Failed to mark as read</div>`)
				return
			}
			// Update in-memory model for rendering
			briefing.ReadAt = &now
		}

		// Return updated history briefing card HTML
		c.Header("Content-Type", "text/html")
		templates.HistoryBriefingCard(briefing).Render(c.Request.Context(), c.Writer)
	}
}
