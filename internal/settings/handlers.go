package settings

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"github.com/jimdaga/first-sip/internal/models"
	"github.com/jimdaga/first-sip/internal/plugins"
	"github.com/jimdaga/first-sip/internal/templates"
	"github.com/jimdaga/first-sip/internal/worker"
	"gorm.io/gorm"
)

// render is a package-local helper for rendering Templ components in Gin handlers.
// Duplicated from dashboard/handlers.go to avoid import cycle.
func render(c *gin.Context, component templ.Component) {
	c.Header("Content-Type", "text/html")
	component.Render(c.Request.Context(), c.Writer)
}

// getAuthUser extracts the authenticated user from the Gin context (set by RequireAuth
// middleware) and looks up the full User record from the database.
// Duplicated from dashboard/handlers.go to avoid import cycle.
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

// parsePluginID extracts and parses the :pluginID URL param from the Gin context.
func parsePluginID(c *gin.Context) (uint, error) {
	pluginIDStr := c.Param("pluginID")
	pluginIDParsed, err := strconv.ParseUint(pluginIDStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid plugin ID %q: %w", pluginIDStr, err)
	}
	return uint(pluginIDParsed), nil
}

// SettingsPageHandler returns a Gin handler for GET /settings.
// Renders the settings page with all plugins and their current state.
func SettingsPageHandler(db *gorm.DB, pluginDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := getAuthUser(c, db)
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			return
		}

		viewModels, err := BuildPluginSettingsViewModels(db, user.ID, pluginDir)
		if err != nil {
			// Render with empty list rather than 500.
			render(c, templates.SettingsPage([]PluginSettingsViewModel{}))
			return
		}

		render(c, templates.SettingsPage(viewModels))
	}
}

// TogglePluginHandler returns a Gin handler for POST /api/settings/:pluginID/toggle.
// Flips the enabled state of a plugin and returns the updated accordion row HTML fragment.
func TogglePluginHandler(db *gorm.DB, pluginDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := getAuthUser(c, db)
		if err != nil {
			c.Status(http.StatusUnauthorized)
			return
		}

		pluginID, err := parsePluginID(c)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		// Find or create UserPluginConfig for this user + plugin.
		var config plugins.UserPluginConfig
		result := db.Where("user_id = ? AND plugin_id = ? AND deleted_at IS NULL", user.ID, pluginID).
			First(&config)

		if result.Error != nil {
			// Not found — create a new one with enabled=true (first toggle = enable).
			config = plugins.UserPluginConfig{
				UserID:   user.ID,
				PluginID: pluginID,
				Enabled:  true,
				Timezone: "UTC",
			}
			if err := db.Create(&config).Error; err != nil {
				c.Status(http.StatusInternalServerError)
				return
			}
		} else {
			// Toggle the existing enabled state.
			config.Enabled = !config.Enabled
			if err := db.Save(&config).Error; err != nil {
				c.Status(http.StatusInternalServerError)
				return
			}
		}

		// If enabling and plugin has required fields, trigger auto-expand.
		if config.Enabled {
			// Check if the plugin's schema has required fields.
			var plugin plugins.Plugin
			if err := db.First(&plugin, pluginID).Error; err == nil {
				schema, err := loadPluginSchema(pluginDir, plugin.Name, plugin.SettingsSchemaPath)
				if err == nil && schema != nil && len(schema.Required) > 0 {
					c.Header("HX-Trigger", "settings-auto-expand")
				}
			}
		}

		// Re-build and render the updated accordion row fragment.
		vm, err := BuildSinglePluginSettingsViewModel(db, user.ID, pluginID, pluginDir, nil, nil, false)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		render(c, templates.PluginAccordionRow(*vm))
	}
}

// SaveSettingsHandler returns a Gin handler for POST /api/settings/:pluginID/save.
// Validates and saves plugin settings and schedule for the authenticated user.
func SaveSettingsHandler(db *gorm.DB, pluginDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := getAuthUser(c, db)
		if err != nil {
			c.Status(http.StatusUnauthorized)
			return
		}

		pluginID, err := parsePluginID(c)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		// Parse raw form values.
		if err := c.Request.ParseForm(); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		rawForm := c.Request.PostForm

		// Extract schedule fields (separate from plugin-specific settings).
		cronExpression := rawForm.Get("cron_expression")
		timezone := rawForm.Get("timezone")

		// Validate cron expression if provided.
		var cronErr string
		if cronExpression != "" {
			if err := plugins.ValidateCronExpression(cronExpression); err != nil {
				cronErr = err.Error()
			}
		}

		// Load plugin to get schema path.
		var plugin plugins.Plugin
		if err := db.First(&plugin, pluginID).Error; err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		// Load schema for plugin-specific field validation.
		schema, err := loadPluginSchema(pluginDir, plugin.Name, plugin.SettingsSchemaPath)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		// Build submitted values map (raw strings from form, excluding schedule fields).
		submittedValues := make(map[string]string)
		if schema != nil && schema.Properties != nil {
			for key := range *schema.Properties {
				if vals, ok := rawForm[key]; ok && len(vals) > 0 {
					submittedValues[key] = vals[0]
				}
			}
		}

		var fieldErrors map[string]string

		// Validate plugin-specific settings against schema.
		if schema != nil {
			typedValues, coerceErr := coerceFormValues(rawForm, schema)
			if coerceErr != nil {
				// Coercion error — treat as a validation failure.
				fieldErrors = map[string]string{"": coerceErr.Error()}
			} else {
				fieldErrors = validateAndGetFieldErrors(schema, typedValues)
			}

			// Add cron error to field errors if present.
			if cronErr != "" {
				if fieldErrors == nil {
					fieldErrors = make(map[string]string)
				}
				fieldErrors["/cron_expression"] = cronErr
			}

			if len(fieldErrors) > 0 {
				// Validation failed: re-render with submitted values and errors, keep expanded.
				vm, err := BuildSinglePluginSettingsViewModel(db, user.ID, pluginID, pluginDir, submittedValues, fieldErrors, false)
				if err != nil {
					c.Status(http.StatusInternalServerError)
					return
				}
				vm.ForceExpanded = true
				vm.CronError = cronErr
				render(c, templates.PluginAccordionRow(*vm))
				return
			}

			// All valid — marshal typedValues to JSON and save.
			typedValues, _ = coerceFormValues(rawForm, schema)
			settingsJSON, err := json.Marshal(typedValues)
			if err != nil {
				c.Status(http.StatusInternalServerError)
				return
			}

			// Find or create UserPluginConfig and save.
			var config plugins.UserPluginConfig
			result := db.Where("user_id = ? AND plugin_id = ? AND deleted_at IS NULL", user.ID, pluginID).First(&config)
			if result.Error != nil {
				config = plugins.UserPluginConfig{
					UserID:   user.ID,
					PluginID: pluginID,
					Enabled:  false,
					Timezone: "UTC",
				}
			}
			config.Settings = settingsJSON
			if cronExpression != "" {
				config.CronExpression = cronExpression
			}
			if timezone != "" {
				config.Timezone = timezone
			}

			if config.ID == 0 {
				db.Create(&config)
			} else {
				db.Save(&config)
			}
		} else {
			// No schema — only update schedule fields.
			if cronErr != "" {
				fieldErrors = map[string]string{"/cron_expression": cronErr}
				vm, err := BuildSinglePluginSettingsViewModel(db, user.ID, pluginID, pluginDir, submittedValues, fieldErrors, false)
				if err != nil {
					c.Status(http.StatusInternalServerError)
					return
				}
				vm.ForceExpanded = true
				vm.CronError = cronErr
				render(c, templates.PluginAccordionRow(*vm))
				return
			}

			var config plugins.UserPluginConfig
			result := db.Where("user_id = ? AND plugin_id = ? AND deleted_at IS NULL", user.ID, pluginID).First(&config)
			if result.Error != nil {
				config = plugins.UserPluginConfig{
					UserID:   user.ID,
					PluginID: pluginID,
					Enabled:  false,
					Timezone: "UTC",
				}
			}
			if cronExpression != "" {
				config.CronExpression = cronExpression
			}
			if timezone != "" {
				config.Timezone = timezone
			}
			if config.ID == 0 {
				db.Create(&config)
			} else {
				db.Save(&config)
			}
		}

		// Success: re-render with SaveSuccess=true for "Saved ✓" feedback, keep expanded.
		vm, err := BuildSinglePluginSettingsViewModel(db, user.ID, pluginID, pluginDir, nil, nil, true)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		vm.ForceExpanded = true

		render(c, templates.PluginAccordionRow(*vm))
	}
}

// ValidateFieldHandler returns a Gin handler for POST /api/settings/:pluginID/validate-field.
// Validates a single field on blur and returns an inline error HTML fragment or empty string.
func ValidateFieldHandler(db *gorm.DB, pluginDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, err := getAuthUser(c, db)
		if err != nil {
			c.Status(http.StatusUnauthorized)
			return
		}

		pluginID, err := parsePluginID(c)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		// Get the field name and value from the form.
		fieldName := c.PostForm("field_name")
		if fieldName == "" {
			c.Status(http.StatusBadRequest)
			return
		}
		fieldValue := c.PostForm(fieldName)

		// Load plugin schema.
		var plugin plugins.Plugin
		if err := db.First(&plugin, pluginID).Error; err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		schema, err := loadPluginSchema(pluginDir, plugin.Name, plugin.SettingsSchemaPath)
		if err != nil || schema == nil {
			// No schema or error — no validation to perform.
			c.Header("Content-Type", "text/html")
			c.String(http.StatusOK, "")
			return
		}

		errorMsg := validateSingleField(schema, fieldName, fieldValue)

		c.Header("Content-Type", "text/html")
		if errorMsg != "" {
			c.String(http.StatusOK, `<span class="settings-field-error">%s</span>`, errorMsg)
		} else {
			c.String(http.StatusOK, "")
		}
	}
}

// RunNowHandler returns a Gin handler for POST /api/settings/:pluginID/run-now.
// Enqueues a plugin execution task using saved settings from DB.
func RunNowHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := getAuthUser(c, db)
		if err != nil {
			c.Status(http.StatusUnauthorized)
			return
		}

		pluginID, err := parsePluginID(c)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		// Load UserPluginConfig — must exist and be enabled.
		var config plugins.UserPluginConfig
		if err := db.Where("user_id = ? AND plugin_id = ? AND deleted_at IS NULL", user.ID, pluginID).
			First(&config).Error; err != nil {
			c.String(http.StatusNotFound, "Plugin not configured")
			return
		}
		if !config.Enabled {
			c.String(http.StatusBadRequest, "Plugin is not enabled")
			return
		}

		// Load plugin name from plugins table.
		var plugin plugins.Plugin
		if err := db.First(&plugin, pluginID).Error; err != nil {
			c.String(http.StatusNotFound, "Plugin not found")
			return
		}

		// Unmarshal saved settings.
		var settingsMap map[string]interface{}
		if len(config.Settings) > 0 {
			if err := json.Unmarshal(config.Settings, &settingsMap); err != nil {
				settingsMap = nil
			}
		}

		// Enqueue the plugin execution task.
		if err := worker.EnqueueExecutePlugin(pluginID, user.ID, plugin.Name, settingsMap); err != nil {
			c.Header("Content-Type", "text/html")
			c.String(http.StatusInternalServerError, `<svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="8" x2="12" y2="12"></line><line x1="12" y1="16" x2="12.01" y2="16"></line></svg> Failed`)
			return
		}

		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, `<svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg> Triggered`)
	}
}
