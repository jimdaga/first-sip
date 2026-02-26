package dashboard

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

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

// formatDashboardDate returns a formatted date string for the dashboard header,
// e.g. "Sunday, February 22, 2026", using the user's IANA timezone.
func formatDashboardDate(timezone string) string {
	loc, err := time.LoadLocation(timezone)
	if err != nil || loc == nil {
		loc = time.UTC
	}
	return time.Now().In(loc).Format("Monday, January 2, 2006")
}

// DashboardHandler returns a Gin handler for GET /dashboard.
// It fetches enabled plugin tiles for the authenticated user and renders the
// tile-based dashboard page with a time-aware greeting and current date.
func DashboardHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := getAuthUser(c, db)
		if err != nil {
			// Fall back gracefully if DB lookup fails.
			nameVal, _ := c.Get("user_name")
			nameStr := ""
			if nameVal != nil {
				nameStr = nameVal.(string)
			}
			greeting := timeAwareGreeting(nameStr, "UTC")
			date := formatDashboardDate("UTC")
			render(c, templates.DashboardPage(greeting, date, []TileViewModel{}, false, nil))
			return
		}

		tiles, err := getDashboardTiles(db, user.ID)
		if err != nil {
			// On query error, render with no tiles rather than a 500 page.
			sidebarPlugins := GetSidebarPlugins(db, user.ID)
			greeting := timeAwareGreeting(user.Name, user.Timezone)
			date := formatDashboardDate(user.Timezone)
			render(c, templates.DashboardPage(greeting, date, []TileViewModel{}, false, sidebarPlugins))
			return
		}

		sidebarPlugins := GetSidebarPlugins(db, user.ID)
		greeting := timeAwareGreeting(user.Name, user.Timezone)
		date := formatDashboardDate(user.Timezone)
		render(c, templates.DashboardPage(greeting, date, tiles, len(tiles) > 0, sidebarPlugins))
	}
}

// TileStatusHandler returns a Gin handler for GET /api/tiles/:pluginID.
// Used by HTMX polling to refresh a single tile's HTML fragment (outerHTML swap).
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

		tile, err := GetSingleTile(db, user.ID, uint(pluginIDParsed))
		if err != nil || tile == nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		render(c, templates.TileCard(*tile))
	}
}

// UpdateTimezoneHandler returns a Gin handler for POST /api/user/timezone.
// Detects the browser timezone via JS and updates the user's timezone if still UTC.
func UpdateTimezoneHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := getAuthUser(c, db)
		if err != nil {
			c.Status(http.StatusUnauthorized)
			return
		}

		tz := c.PostForm("timezone")
		if tz == "" {
			c.Status(http.StatusBadRequest)
			return
		}

		// Only update if user still has the default UTC timezone.
		if user.Timezone != "" && user.Timezone != "UTC" {
			c.Status(http.StatusOK)
			return
		}

		// Validate the IANA timezone.
		if _, err := time.LoadLocation(tz); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		db.Model(user).Update("timezone", tz)
		c.Status(http.StatusOK)
	}
}

// PluginDetailHandler returns a Gin handler for GET /plugins/:pluginName.
// It renders the full-page plugin detail view with the latest briefing content.
func PluginDetailHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := getAuthUser(c, db)
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			return
		}

		pluginName := c.Param("pluginName")

		// Look up the plugin by name to get its ID.
		var pluginRow struct {
			ID uint
		}
		if err := db.Raw(`SELECT id FROM plugins WHERE name = ? AND deleted_at IS NULL`, pluginName).Scan(&pluginRow).Error; err != nil || pluginRow.ID == 0 {
			c.Redirect(http.StatusFound, "/dashboard")
			return
		}

		tile, err := GetSingleTile(db, user.ID, pluginRow.ID)
		if err != nil || tile == nil {
			c.Redirect(http.StatusFound, "/dashboard")
			return
		}

		sidebarPlugins := GetSidebarPlugins(db, user.ID)
		render(c, templates.PluginDetailPage(*tile, sidebarPlugins))
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
