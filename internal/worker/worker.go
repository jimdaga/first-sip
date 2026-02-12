package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jimdaga/first-sip/internal/config"
	"github.com/jimdaga/first-sip/internal/models"
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

// Run starts the Asynq worker server and blocks until shutdown.
func Run(cfg *config.Config, db *gorm.DB, webhookClient *webhook.Client) error {
	// Parse Redis connection options
	redisOpt, err := asynq.ParseRedisURI(cfg.RedisURL)
	if err != nil {
		return fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Create structured logger
	logger := NewLogger(cfg.LogLevel, cfg.LogFormat)

	// Create Asynq server with configuration
	srv := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency:     5,
			ShutdownTimeout: 30 * time.Second,
			ErrorHandler:    asynq.ErrorHandlerFunc(makeErrorHandler(logger)),
			Logger:          &asynqLoggerAdapter{logger: logger},
		},
	)

	// Create task multiplexer
	mux := asynq.NewServeMux()

	// Register task handlers
	mux.HandleFunc(TaskGenerateBriefing, handleGenerateBriefing(logger, db, webhookClient))

	logger.Info("Worker starting", "concurrency", 5, "redis", cfg.RedisURL)

	// Run the server (blocks until shutdown signal)
	return srv.Run(mux)
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
