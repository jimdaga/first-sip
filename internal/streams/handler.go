package streams

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/jimdaga/first-sip/internal/plugins"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// HandlePluginResult returns a handler function that updates PluginRun records
// based on stream results
func HandlePluginResult(db *gorm.DB) func(PluginResult) error {
	return func(result PluginResult) error {
		var pluginRun plugins.PluginRun

		// Find PluginRun by PluginRunID field (not GORM ID)
		if err := db.Where("plugin_run_id = ?", result.PluginRunID).First(&pluginRun).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("plugin run not found: %s", result.PluginRunID)
			}
			return fmt.Errorf("failed to find plugin run: %w", err)
		}

		// Update based on status
		now := time.Now()
		updates := map[string]interface{}{
			"completed_at": now,
		}

		if result.Status == "completed" {
			updates["status"] = plugins.PluginRunStatusCompleted
			updates["output"] = datatypes.JSON(result.Output)

			slog.Info("Plugin run completed",
				"plugin_run_id", result.PluginRunID,
				"status", "completed",
			)
		} else if result.Status == "failed" {
			updates["status"] = plugins.PluginRunStatusFailed
			updates["error_message"] = result.Error

			slog.Error("Plugin run failed",
				"plugin_run_id", result.PluginRunID,
				"status", "failed",
				"error", result.Error,
			)
		} else {
			return fmt.Errorf("unknown status: %s", result.Status)
		}

		// Apply updates
		if err := db.Model(&pluginRun).Updates(updates).Error; err != nil {
			return fmt.Errorf("failed to update plugin run: %w", err)
		}

		return nil
	}
}
