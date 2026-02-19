package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jimdaga/first-sip/internal/config"
	"github.com/jimdaga/first-sip/internal/models"
	"github.com/jimdaga/first-sip/internal/plugins"
	"github.com/jimdaga/first-sip/internal/streams"
	"github.com/jimdaga/first-sip/internal/webhook"
	"gorm.io/gorm"
)

// asynqLoggerAdapter wraps slog.Logger to implement asynq.Logger interface
type asynqLoggerAdapter struct {
	logger *slog.Logger
}

// Implement asynq.Logger interface methods
func (a *asynqLoggerAdapter) Debug(args ...interface{}) {
	a.logger.Debug(fmt.Sprint(args...))
}

func (a *asynqLoggerAdapter) Info(args ...interface{}) {
	a.logger.Info(fmt.Sprint(args...))
}

func (a *asynqLoggerAdapter) Warn(args ...interface{}) {
	a.logger.Warn(fmt.Sprint(args...))
}

func (a *asynqLoggerAdapter) Error(args ...interface{}) {
	a.logger.Error(fmt.Sprint(args...))
}

func (a *asynqLoggerAdapter) Fatal(args ...interface{}) {
	a.logger.Error(fmt.Sprint(args...))
	panic(fmt.Sprint(args...))
}

// Run starts the Asynq worker server and blocks until shutdown signal.
// Use this for standalone worker mode.
func Run(cfg *config.Config, db *gorm.DB, webhookClient *webhook.Client, publisher *streams.Publisher) error {
	srv, mux, err := newServer(cfg, db, webhookClient, publisher)
	if err != nil {
		return err
	}

	// Note: Scheduler is started separately in main.go worker mode
	// and deferred there for shutdown coordination.
	// Run blocks and handles its own signal interception
	return srv.Run(mux)
}

// Start starts the Asynq worker in non-blocking mode and returns a stop function.
// Use this for embedded mode so the caller can coordinate shutdown.
func Start(cfg *config.Config, db *gorm.DB, webhookClient *webhook.Client, publisher *streams.Publisher) (stop func(), err error) {
	srv, mux, err := newServer(cfg, db, webhookClient, publisher)
	if err != nil {
		return nil, err
	}
	if err := srv.Start(mux); err != nil {
		return nil, fmt.Errorf("failed to start worker: %w", err)
	}
	return func() { srv.Shutdown() }, nil
}

func newServer(cfg *config.Config, db *gorm.DB, webhookClient *webhook.Client, publisher *streams.Publisher) (*asynq.Server, *asynq.ServeMux, error) {
	redisOpt, err := asynq.ParseRedisURI(cfg.RedisURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	logger := NewLogger(cfg.LogLevel, cfg.LogFormat)

	srv := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency:     5,
			ShutdownTimeout: 30 * time.Second,
			ErrorHandler:    asynq.ErrorHandlerFunc(makeErrorHandler(logger)),
			Logger:          &asynqLoggerAdapter{logger: logger},
		},
	)

	// Create a dedicated Redis client for the per-minute scheduler's last-run cache.
	// This is separate from the Asynq internal connection.
	rdb, err := newSchedulerRedisClient(cfg.RedisURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create scheduler Redis client: %w", err)
	}

	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskGenerateBriefing, handleGenerateBriefing(logger, db, webhookClient))
	mux.HandleFunc(TaskExecutePlugin, handleExecutePlugin(logger, db, publisher))
	mux.HandleFunc(TaskPerMinuteScheduler, handlePerMinuteScheduler(logger, db, rdb))

	logger.Info("Worker starting", "concurrency", 5, "redis", cfg.RedisURL)
	return srv, mux, nil
}

// handleGenerateBriefing processes briefing generation tasks by calling the webhook
// client and updating the database with the generated content.
func handleGenerateBriefing(logger *slog.Logger, db *gorm.DB, webhookClient *webhook.Client) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, task *asynq.Task) error {
		// Unmarshal the payload
		var payload struct {
			BriefingID uint `json:"briefing_id"`
		}
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			// Invalid payload - don't retry
			return fmt.Errorf("invalid payload: %w", asynq.SkipRetry)
		}

		// Fetch briefing from database
		var briefing models.Briefing
		if err := db.WithContext(ctx).First(&briefing, payload.BriefingID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Record not found - don't retry
				logger.Error("Briefing not found", "briefing_id", payload.BriefingID)
				return fmt.Errorf("briefing not found: %w", asynq.SkipRetry)
			}
			// Database error - retryable
			return fmt.Errorf("failed to fetch briefing: %w", err)
		}

		logger.Info(
			"Processing briefing:generate task",
			"briefing_id", payload.BriefingID,
			"user_id", briefing.UserID,
		)

		// Update status to processing
		db.Model(&briefing).Update("status", models.BriefingStatusProcessing)

		// Call webhook client to generate briefing content
		content, err := webhookClient.GenerateBriefing(ctx, briefing.UserID)
		if err != nil {
			// Update briefing to failed status
			db.Model(&briefing).Updates(map[string]interface{}{
				"status":        models.BriefingStatusFailed,
				"error_message": err.Error(),
			})
			logger.Error(
				"Webhook generation failed",
				"briefing_id", payload.BriefingID,
				"error", err.Error(),
			)
			return fmt.Errorf("webhook generation failed: %w", err)
		}

		// Marshal content to JSON
		jsonBytes, err := json.Marshal(content)
		if err != nil {
			// Failed to marshal - update briefing and don't retry
			db.Model(&briefing).Updates(map[string]interface{}{
				"status":        models.BriefingStatusFailed,
				"error_message": "Failed to marshal content",
			})
			return fmt.Errorf("failed to marshal content: %w", asynq.SkipRetry)
		}

		// Update briefing with completed status and content
		now := time.Now()
		if err := db.Model(&briefing).Updates(map[string]interface{}{
			"status":        models.BriefingStatusCompleted,
			"content":       jsonBytes,
			"generated_at":  now,
			"error_message": "",
		}).Error; err != nil {
			return fmt.Errorf("failed to update briefing: %w", err)
		}

		logger.Info(
			"Briefing generation completed",
			"briefing_id", payload.BriefingID,
		)

		return nil
	}
}

// handleExecutePlugin processes plugin execution tasks by creating a PluginRun record
// and publishing the request to the Redis Stream for the CrewAI sidecar to consume.
func handleExecutePlugin(logger *slog.Logger, db *gorm.DB, publisher *streams.Publisher) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, task *asynq.Task) error {
		// Unmarshal the payload
		var payload struct {
			PluginID   uint                   `json:"plugin_id"`
			UserID     uint                   `json:"user_id"`
			PluginName string                 `json:"plugin_name"`
			Settings   map[string]interface{} `json:"settings"`
		}
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			return fmt.Errorf("invalid payload: %w", asynq.SkipRetry)
		}

		logger.Info(
			"Processing plugin:execute task",
			"plugin_name", payload.PluginName,
			"plugin_id", payload.PluginID,
			"user_id", payload.UserID,
		)

		// Generate a unique plugin_run_id for external tracking
		pluginRunID := uuid.New().String()

		// Marshal settings to JSON for storage
		settingsJSON, err := json.Marshal(payload.Settings)
		if err != nil {
			return fmt.Errorf("failed to marshal settings: %w", asynq.SkipRetry)
		}

		// Create PluginRun record with pending status
		now := time.Now()
		pluginRun := plugins.PluginRun{
			PluginRunID: pluginRunID,
			UserID:      payload.UserID,
			PluginID:    payload.PluginID,
			Status:      plugins.PluginRunStatusPending,
			Input:       settingsJSON,
			StartedAt:   &now,
		}
		if err := db.WithContext(ctx).Create(&pluginRun).Error; err != nil {
			return fmt.Errorf("failed to create plugin run record: %w", err)
		}

		logger.Info("Created PluginRun record", "plugin_run_id", pluginRunID, "db_id", pluginRun.ID)

		// Graceful degradation: if publisher is not configured, fail the run
		if publisher == nil {
			logger.Warn("Streams publisher not configured — cannot publish plugin request",
				"plugin_run_id", pluginRunID)
			db.Model(&pluginRun).Updates(map[string]interface{}{
				"status":        plugins.PluginRunStatusFailed,
				"error_message": "streams publisher not configured",
			})
			return fmt.Errorf("streams publisher not configured: %w", asynq.SkipRetry)
		}

		// Build and publish the plugin request to Redis Stream
		req := streams.PluginRequest{
			PluginRunID: pluginRunID,
			PluginName:  payload.PluginName,
			UserID:      payload.UserID,
			Settings:    payload.Settings,
		}
		msgID, err := publisher.PublishPluginRequest(ctx, req)
		if err != nil {
			logger.Error("Failed to publish plugin request to stream",
				"plugin_run_id", pluginRunID,
				"error", err.Error(),
			)
			db.Model(&pluginRun).Updates(map[string]interface{}{
				"status":        plugins.PluginRunStatusFailed,
				"error_message": err.Error(),
			})
			// Return error (retryable — stream may be temporarily unavailable)
			return fmt.Errorf("failed to publish to stream: %w", err)
		}

		// Update PluginRun to processing status
		db.Model(&pluginRun).Update("status", plugins.PluginRunStatusProcessing)

		logger.Info(
			"Plugin request published to stream",
			"plugin_run_id", pluginRunID,
			"stream_msg_id", msgID,
			"plugin_name", payload.PluginName,
		)

		return nil
	}
}

// makeErrorHandler creates an error handler function with logger closure.
func makeErrorHandler(logger *slog.Logger) func(context.Context, *asynq.Task, error) {
	return func(ctx context.Context, task *asynq.Task, err error) {
		retried, _ := asynq.GetRetryCount(ctx)
		maxRetry, _ := asynq.GetMaxRetry(ctx)

		logger.Error(
			"Task execution failed",
			"task_type", task.Type(),
			"error", err.Error(),
			"retry_count", retried,
			"max_retry", maxRetry,
		)

		// Check if this is the final failure (task will move to dead letter queue)
		if retried >= maxRetry {
			logger.Error(
				"Task moved to dead letter queue (all retries exhausted)",
				"task_type", task.Type(),
				"payload", string(task.Payload()),
			)
		}
	}
}
