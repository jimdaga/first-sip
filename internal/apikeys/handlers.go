package apikeys

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"github.com/jimdaga/first-sip/internal/dashboard"
	"github.com/jimdaga/first-sip/internal/models"
	"github.com/jimdaga/first-sip/internal/templates"
	"gorm.io/gorm"
)

// render is a package-local helper for rendering Templ components in Gin handlers.
// Duplicated from settings/handlers.go to avoid import cycles.
func render(c *gin.Context, component templ.Component) {
	c.Header("Content-Type", "text/html")
	component.Render(c.Request.Context(), c.Writer)
}

// getAuthUser extracts the authenticated user from the Gin context and fetches
// the full User record from the database.
// Duplicated from settings/handlers.go to avoid import cycles.
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

// PageHandler returns a Gin handler for GET /settings/api-keys.
// Renders the full API Keys settings page with stored keys, masked values,
// provider dropdown, and LLM preference selects.
func PageHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := getAuthUser(c, db)
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			return
		}

		vm := BuildViewModel(db, user)
		sidebarPlugins := dashboard.GetSidebarPlugins(db, user.ID)
		render(c, templates.APIKeysSettingsPage(vm, sidebarPlugins))
	}
}

// SaveKeyHandler returns a Gin handler for POST /api/user/api-keys.
// Validates and saves (upserts) a new API key for the authenticated user.
// On success returns the refreshed #api-keys-section fragment for HTMX swap.
func SaveKeyHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := getAuthUser(c, db)
		if err != nil {
			c.Status(http.StatusUnauthorized)
			return
		}

		keyType := c.PostForm("key_type")
		provider := c.PostForm("provider")
		apiKey := c.PostForm("api_key")

		// Validate API key value.
		if apiKey == "" {
			render(c, templates.APIKeysErrorAlert("API key cannot be empty."))
			return
		}

		// For LLM keys, validate provider is supported.
		if keyType == "llm" {
			if GetProviderByID(provider) == nil {
				render(c, templates.APIKeysErrorAlert("Unsupported LLM provider."))
				return
			}
		}

		// For Tavily keys, set provider to "tavily".
		if keyType == "tavily" {
			provider = "tavily"
		}

		if err := SaveKey(db, user.ID, keyType, provider, apiKey); err != nil {
			slog.Error("apikeys: failed to save key", "user_id", user.ID, "error", err)
			render(c, templates.APIKeysErrorAlert("Failed to save API key. Please try again."))
			return
		}

		// Rebuild and return the refreshed section fragment.
		vm := BuildViewModel(db, user)
		render(c, templates.APIKeysSection(vm))
	}
}

// DeleteKeyHandler returns a Gin handler for POST /api/user/api-keys/:id/delete.
// Soft-deletes the specified API key (scoped to the authenticated user).
// On success returns the refreshed #api-keys-section fragment for HTMX swap.
func DeleteKeyHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := getAuthUser(c, db)
		if err != nil {
			c.Status(http.StatusUnauthorized)
			return
		}

		idStr := c.Param("id")
		idParsed, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		keyID := uint(idParsed)

		if err := DeleteKey(db, user.ID, keyID); err != nil {
			slog.Error("apikeys: failed to delete key", "user_id", user.ID, "key_id", keyID, "error", err)
			c.Status(http.StatusInternalServerError)
			return
		}

		vm := BuildViewModel(db, user)
		render(c, templates.APIKeysSection(vm))
	}
}

// SaveLLMPreferenceHandler returns a Gin handler for POST /api/user/llm-preference.
// Validates and saves the user's preferred LLM provider and model.
// On success returns the refreshed preference section fragment for HTMX swap.
func SaveLLMPreferenceHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := getAuthUser(c, db)
		if err != nil {
			c.Status(http.StatusUnauthorized)
			return
		}

		provider := c.PostForm("provider")
		model := c.PostForm("model")

		if err := SaveLLMPreference(db, user.ID, provider, model); err != nil {
			slog.Warn("apikeys: failed to save LLM preference", "user_id", user.ID, "error", err)
			render(c, templates.LLMPreferenceErrorAlert("Invalid provider or model selection."))
			return
		}

		// Refresh user record to reflect saved preferences.
		user.LLMPreferredProvider = provider
		user.LLMPreferredModel = model
		vm := BuildViewModel(db, user)
		render(c, templates.LLMPreferenceSection(vm))
	}
}
