package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jimdaga/first-sip/internal/config"
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
func Run(cfg *config.Config) error {
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
	mux.HandleFunc(TaskGenerateBriefing, handleGenerateBriefing(logger))

	logger.Info("Worker starting", "concurrency", 5, "redis", cfg.RedisURL)

	// Run the server (blocks until shutdown signal)
	return srv.Run(mux)
}

// handleGenerateBriefing is a placeholder handler for the briefing:generate task.
// Phase 4 will implement the actual briefing generation logic.
func handleGenerateBriefing(logger *slog.Logger) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, task *asynq.Task) error {
		// Unmarshal the payload
		var payload struct {
			BriefingID uint `json:"briefing_id"`
		}
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload: %w", err)
		}

		logger.Info(
			"Processing briefing:generate task (placeholder handler)",
			"briefing_id", payload.BriefingID,
			"task_type", task.Type(),
		)

		// Placeholder - Phase 4 will implement actual briefing generation
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
