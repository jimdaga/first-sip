package dashboard

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"github.com/jimdaga/first-sip/internal/models"
	"github.com/jimdaga/first-sip/internal/plugins"
	"github.com/jimdaga/first-sip/internal/templates"
	"gorm.io/gorm"
)

// render is a package-local helper for rendering Templ components in Gin handlers.
func render(c *gin.Context, component templ.Component) {
	c.Header("Content-Type", "text/html")
	component.Render(c.Request.Context(), c.Writer)
}

// getAuthUser extracts the authenticated user from the Gin context (set by RequireAuth
// middleware) and looks up the full User record from the database.
func getAuthUser(c *gin.Context, db *gorm.DB) (*models.User, error) {
	emailVal, exists := c.Get("user_email")
	if !exists {
		return nil, fmt.Errorf("user_email not found in context")
	}
	emailStr, ok := emailVal.(string)
	if !ok || emailStr == "" {
		return nil, fmt.Errorf("user_email is empty")
	}
	var user models.User
	if err := db.Where("email = ?", emailStr).First(&user).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &user, nil
}

// DashboardHandler returns a Gin handler for GET /dashboard.
// It fetches enabled plugin tiles for the authenticated user and renders the
// dashboard page. For now it calls the existing DashboardPage template signature
// (name, email, latestBriefing) to keep compilation working until Plan 03 updates
// the template to accept []TileViewModel.
//
// TODO(11-03): Update this handler to call the new tile-aware template once Plan 03
// introduces templates.DashboardPage(greeting string, tiles []dashboard.TileViewModel).
func DashboardHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		nameVal, _ := c.Get("user_name")
		nameStr := ""
		if nameVal != nil {
			nameStr = nameVal.(string)
		}

		emailVal, _ := c.Get("user_email")
		emailStr := ""
		if emailVal != nil {
			emailStr = emailVal.(string)
		}

		// Query latest briefing for the user — keeps the existing template happy.
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
	}
}

// TileStatusHandler returns a Gin handler for GET /api/tiles/:pluginID.
// Used by HTMX polling to refresh a single tile's status.
//
// TODO(11-03): Render the actual tile Templ component once Plan 03 creates it.
// For now this is a working stub that returns 200 OK.
func TileStatusHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := getAuthUser(c, db)
		if err != nil {
			c.Status(http.StatusUnauthorized)
			return
		}

		pluginIDStr := c.Param("pluginID")
		pluginIDParsed, err := strconv.ParseUint(pluginIDStr, 10, 64)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		_, err = GetSingleTile(db, user.ID, uint(pluginIDParsed))
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		// TODO(11-03): render actual tile component once templates.TileCard exists.
		c.Status(http.StatusOK)
	}
}

// UpdateTileOrderHandler returns a Gin handler for POST /api/tiles/order.
// Persists drag-to-reorder display_order values for the authenticated user.
// Expects form values: plugin_id[] (ordered list of plugin IDs from SortableJS).
func UpdateTileOrderHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := getAuthUser(c, db)
		if err != nil {
			c.Status(http.StatusUnauthorized)
			return
		}

		pluginIDs := c.PostFormArray("plugin_id")
		for i, idStr := range pluginIDs {
			pluginID, err := strconv.ParseUint(idStr, 10, 64)
			if err != nil {
				continue // skip malformed IDs
			}
			order := i
			db.Model(&plugins.UserPluginConfig{}).
				Where("user_id = ? AND plugin_id = ? AND deleted_at IS NULL", user.ID, uint(pluginID)).
				Update("display_order", order)
		}

		// SortableJS uses hx-swap="none" — no body needed.
		c.Status(http.StatusOK)
	}
}
