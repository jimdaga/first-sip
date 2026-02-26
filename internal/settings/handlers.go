package settings

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"github.com/jimdaga/first-sip/internal/dashboard"
	"github.com/jimdaga/first-sip/internal/models"
	"github.com/jimdaga/first-sip/internal/plugins"
	"github.com/jimdaga/first-sip/internal/settingsvm"
	"github.com/jimdaga/first-sip/internal/templates"
	"github.com/jimdaga/first-sip/internal/tiers"
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

// SettingsHubPageHandler returns a Gin handler for GET /settings.
// Renders the settings hub page with tile grid linking to sub-pages.
func SettingsHubPageHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var sidebarPlugins []templates.SidebarPlugin
		if user, err := getAuthUser(c, db); err == nil {
			sidebarPlugins = dashboard.GetSidebarPlugins(db, user.ID)
		}
		render(c, templates.SettingsHubPage(sidebarPlugins))
	}
}

// PluginSettingsPageHandler returns a Gin handler for GET /settings/plugins.
// Renders the plugin settings page with all plugins and their current state.
func PluginSettingsPageHandler(db *gorm.DB, pluginDir string, tierService *tiers.TierService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := getAuthUser(c, db)
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			return
		}

		tierInfo, err := BuildTierInfo(db, tierService, user.ID)
		if err != nil {
			slog.Warn("settings: failed to build tier info", "user_id", user.ID, "error", err)
			// Use default free tier info — don't block page render
		}

		sidebarPlugins := dashboard.GetSidebarPlugins(db, user.ID)

		viewModels, err := BuildPluginSettingsViewModels(db, user.ID, pluginDir, tierInfo)
		if err != nil {
			// Render with empty list rather than 500.
			render(c, templates.PluginSettingsPage(settingsvm.SettingsPageViewModel{
				Plugins:  []PluginSettingsViewModel{},
				TierInfo: tierInfo,
			}, sidebarPlugins))
			return
		}

		render(c, templates.PluginSettingsPage(settingsvm.SettingsPageViewModel{
			Plugins:  viewModels,
			TierInfo: tierInfo,
		}, sidebarPlugins))
	}
}

// TogglePluginHandler returns a Gin handler for POST /api/settings/:pluginID/toggle.
// Flips the enabled state of a plugin and returns the updated accordion row HTML fragment.
// For free users, blocks enabling a 4th plugin and returns the row with IsDisabledByTier=true.
func TogglePluginHandler(db *gorm.DB, pluginDir string, tierService *tiers.TierService) gin.HandlerFunc {
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

		// Determine the target enabled state after toggle.
		isNewRecord := result.Error != nil
		targetEnabled := true // first toggle = enable
		if !isNewRecord {
			targetEnabled = !config.Enabled
		}

		// If enabling, check tier limit.
		if targetEnabled {
			canEnable, _, err := tierService.CanEnablePlugin(user.ID)
			if err != nil {
				slog.Warn("settings: tier check failed", "user_id", user.ID, "error", err)
				// Fail open — allow the action if we can't check the tier
			} else if !canEnable {
				// Tier limit reached — re-render the row with disabled state.
				vm, buildErr := BuildSinglePluginSettingsViewModel(db, user.ID, pluginID, pluginDir, nil, nil, false)
				if buildErr != nil {
					c.Status(http.StatusInternalServerError)
					return
				}
				vm.IsDisabledByTier = true

				// Build current TierInfo for OOB counter update.
				tierInfo, _ := BuildTierInfo(db, tierService, user.ID)

				c.Header("Content-Type", "text/html")
				// Render the accordion row + OOB tier counter together.
				templates.PluginAccordionRow(*vm).Render(c.Request.Context(), c.Writer)
				templates.TierPluginCounter(tierInfo).Render(c.Request.Context(), c.Writer)
				return
			}
		}

		if isNewRecord {
			// Not found — create a new one with enabled=true (first toggle = enable).
			config = plugins.UserPluginConfig{
				UserID:   user.ID,
				PluginID: pluginID,
				Enabled:  true,
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

		// Re-build TierInfo after the toggle for OOB counter update.
		tierInfo, _ := BuildTierInfo(db, tierService, user.ID)

		// Re-build and render the updated accordion row fragment.
		vm, err := BuildSinglePluginSettingsViewModel(db, user.ID, pluginID, pluginDir, nil, nil, false)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		// Set IsFreeUser from tierInfo (BuildSinglePluginSettingsViewModel doesn't have it).
		vm.IsFreeUser = tierInfo.TierName == "free"

		c.Header("Content-Type", "text/html")
		// Render the accordion row + OOB tier counter together.
		templates.PluginAccordionRow(*vm).Render(c.Request.Context(), c.Writer)
		templates.TierPluginCounter(tierInfo).Render(c.Request.Context(), c.Writer)
	}
}

// SaveSettingsHandler returns a Gin handler for POST /api/settings/:pluginID/save.
// Validates and saves plugin settings and schedule for the authenticated user.
// Rejects cron expressions faster than the user's tier minimum frequency.
func SaveSettingsHandler(db *gorm.DB, pluginDir string, tierService *tiers.TierService) gin.HandlerFunc {
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

		// Check frequency tier limit before standard cron validation.
		if cronExpression != "" {
			canUse, tier, freqErr := tierService.CanUseFrequency(user.ID, cronExpression)
			if freqErr == nil && !canUse && tier != nil {
				// Frequency too fast for this tier — re-render with error.
				vm, buildErr := BuildSinglePluginSettingsViewModel(db, user.ID, pluginID, pluginDir, nil, nil, false)
				if buildErr != nil {
					c.Status(http.StatusInternalServerError)
					return
				}
				vm.ForceExpanded = true
				vm.FrequencyError = fmt.Sprintf(
					"Schedules faster than once daily require Pro. Your tier allows minimum %dh intervals.",
					tier.MinFrequencyHours,
				)
				vm.IsFreeUser = tier.Name == "free"
				render(c, templates.PluginAccordionRow(*vm))
				return
			}
		}

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
				}
			}
			config.Settings = settingsJSON
			if cronExpression != "" {
				config.CronExpression = cronExpression
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
				}
			}
			if cronExpression != "" {
				config.CronExpression = cronExpression
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

// AccountSettingsPageHandler returns a Gin handler for GET /settings/account.
// Renders the account settings page with the user's timezone picker.
func AccountSettingsPageHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := getAuthUser(c, db)
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			return
		}

		sidebarPlugins := dashboard.GetSidebarPlugins(db, user.ID)
		render(c, templates.AccountSettingsPage(user.Name, user.Timezone, sidebarPlugins))
	}
}

// SaveTimezoneHandler returns a Gin handler for POST /api/user/settings/timezone.
// Validates and saves the user's account-level timezone preference.
func SaveTimezoneHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := getAuthUser(c, db)
		if err != nil {
			c.Status(http.StatusUnauthorized)
			return
		}

		timezone := c.PostForm("timezone")
		if timezone == "" {
			c.Header("Content-Type", "text/html")
			c.String(http.StatusBadRequest, `<span class="settings-field-error">Timezone is required.</span>`)
			return
		}

		// Validate the IANA timezone.
		if _, err := time.LoadLocation(timezone); err != nil {
			c.Header("Content-Type", "text/html")
			c.String(http.StatusBadRequest, `<span class="settings-field-error">Invalid timezone.</span>`)
			return
		}

		// Save to DB.
		if err := db.Model(user).Update("timezone", timezone).Error; err != nil {
			slog.Error("settings: failed to save user timezone", "user_id", user.ID, "error", err)
			c.Header("Content-Type", "text/html")
			c.String(http.StatusInternalServerError, `<span class="settings-field-error">Failed to save timezone.</span>`)
			return
		}

		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, `<span class="settings-save-success">Timezone saved</span>`)
	}
}

// ProNotifyHandler returns a Gin handler for POST /api/pro/notify.
// Logs the submitted email address and returns a thank-you HTML fragment.
// No DB persistence needed for scaffolding phase.
func ProNotifyHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		email := c.PostForm("email")
		slog.Info("pro notify: email interest captured", "email", email)
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, `<p class="pro-thank-you">Thanks! We'll notify you when Pro launches.</p>`)
	}
}
