package worker

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jimdaga/first-sip/internal/config"
)

// StartScheduler creates and starts an Asynq Scheduler for periodic tasks.
// Returns a stop function for graceful shutdown.
func StartScheduler(cfg *config.Config) (stop func(), err error) {
	redisOpt, err := asynq.ParseRedisURI(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Parse timezone from config
	location, err := time.LoadLocation(cfg.BriefingTimezone)
	if err != nil {
		slog.Warn("Invalid timezone, using UTC", "timezone", cfg.BriefingTimezone, "error", err)
		location = time.UTC
	}

	// Create logger for scheduler
	logger := NewLogger(cfg.LogLevel, cfg.LogFormat)

	scheduler := asynq.NewScheduler(
		redisOpt,
		&asynq.SchedulerOpts{
			Location: location,
			LogLevel: asynq.InfoLevel,
			Logger:   &asynqLoggerAdapter{logger: logger},
		},
	)

	// Register periodic briefing generation task
	task := asynq.NewTask(
		TaskScheduledBriefingGeneration,
		nil, // Empty payload - handler will query all users
		asynq.MaxRetry(3),
		asynq.Timeout(10*time.Minute), // Longer timeout for processing all users
		asynq.Retention(24*time.Hour),
		asynq.Unique(24*time.Hour), // Prevent duplicate if scheduler runs twice
	)

	entryID, err := scheduler.Register(cfg.BriefingSchedule, task)
	if err != nil {
		return nil, fmt.Errorf("failed to register briefing schedule: %w", err)
	}

	// Start scheduler (non-blocking)
	if err := scheduler.Start(); err != nil {
		return nil, fmt.Errorf("failed to start scheduler: %w", err)
	}

	slog.Info(
		"Scheduler started",
		"schedule", cfg.BriefingSchedule,
		"timezone", cfg.BriefingTimezone,
		"entry_id", entryID,
	)

	// Return shutdown function
	return func() { scheduler.Shutdown() }, nil
}
