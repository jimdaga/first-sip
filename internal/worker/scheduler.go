package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/hibiken/asynq"
	cron "github.com/robfig/cron/v3"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/jimdaga/first-sip/internal/config"
	"github.com/jimdaga/first-sip/internal/plugins"
)

// lastRunHashKey is the Redis hash key that stores per-user-plugin last-run timestamps.
const lastRunHashKey = "scheduler:last_run"

// StartPerMinuteScheduler creates and starts an Asynq Scheduler that fires once per minute.
// Each tick enqueues a TaskPerMinuteScheduler task which then queries the DB and dispatches
// any plugin executions that are due according to their cron schedules.
// Returns a stop function for graceful shutdown.
func StartPerMinuteScheduler(cfg *config.Config) (stop func(), err error) {
	redisOpt, err := asynq.ParseRedisURI(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	logger := NewLogger(cfg.LogLevel, cfg.LogFormat)

	scheduler := asynq.NewScheduler(
		redisOpt,
		&asynq.SchedulerOpts{
			Location: time.UTC, // Scheduler itself is UTC; per-user timezone is applied in isDue
			LogLevel: asynq.InfoLevel,
			Logger:   &asynqLoggerAdapter{logger: logger},
		},
	)

	// Fire once per minute; the handler queries the DB and enqueues what is due.
	task := asynq.NewTask(
		TaskPerMinuteScheduler,
		nil,
		asynq.Queue("critical"),       // Process before default-queue plugin:execute tasks
		asynq.MaxRetry(0),             // Don't retry — next minute catches up
		asynq.Timeout(50*time.Second), // Leave 10 s headroom before the next tick
		asynq.Unique(55*time.Second),  // Prevent duplicate tasks if scheduler restarts
	)

	entryID, err := scheduler.Register("* * * * *", task)
	if err != nil {
		return nil, fmt.Errorf("failed to register per-minute scheduler task: %w", err)
	}

	if err := scheduler.Start(); err != nil {
		return nil, fmt.Errorf("failed to start per-minute scheduler: %w", err)
	}

	slog.Info(
		"Per-minute scheduler started",
		"entry_id", entryID,
		"task_type", TaskPerMinuteScheduler,
	)

	return func() { scheduler.Shutdown() }, nil
}

// handlePerMinuteScheduler returns an Asynq handler that queries all enabled
// user-plugin configs with a cron_expression and enqueues plugin:execute for
// any pair whose schedule is currently due.
func handlePerMinuteScheduler(logger *slog.Logger, db *gorm.DB, rdb *redis.Client) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, task *asynq.Task) error {
		var configs []plugins.UserPluginConfig
		if err := db.WithContext(ctx).
			Preload("Plugin").
			Preload("User").
			Where("enabled = ? AND cron_expression IS NOT NULL AND cron_expression != ''", true).
			Find(&configs).Error; err != nil {
			return fmt.Errorf("failed to query scheduled user plugin configs: %w", err)
		}

		enqueued := 0
		skipped := 0
		errored := 0

		for _, cfg := range configs {
			// Determine effective timezone: prefer the per-config value, fall back to User.Timezone if present.
			// (User model does not yet have a Timezone field; this is prepared for Phase 12.)
			effectiveTimezone := cfg.Timezone
			if effectiveTimezone == "" {
				effectiveTimezone = "UTC"
			}

			lastRun := getLastRun(ctx, rdb, cfg.UserID, cfg.PluginID)

			due, err := isDue(cfg.CronExpression, effectiveTimezone, lastRun)
			if err != nil {
				logger.Warn(
					"Failed to evaluate cron expression — skipping",
					"user_id", cfg.UserID,
					"plugin_id", cfg.PluginID,
					"cron_expression", cfg.CronExpression,
					"error", err,
				)
				errored++
				continue
			}
			if !due {
				skipped++
				continue
			}

			// Unmarshal settings for the enqueue payload
			var settings map[string]interface{}
			if len(cfg.Settings) > 0 {
				if err := json.Unmarshal(cfg.Settings, &settings); err != nil {
					logger.Warn(
						"Failed to unmarshal plugin settings — using empty map",
						"user_id", cfg.UserID,
						"plugin_id", cfg.PluginID,
						"error", err,
					)
					settings = map[string]interface{}{}
				}
			} else {
				settings = map[string]interface{}{}
			}

			if err := EnqueueExecutePlugin(cfg.PluginID, cfg.UserID, cfg.Plugin.Name, settings); err != nil {
				logger.Error(
					"Failed to enqueue plugin execution",
					"user_id", cfg.UserID,
					"plugin_id", cfg.PluginID,
					"plugin_name", cfg.Plugin.Name,
					"error", err,
				)
				errored++
				continue
			}

			setLastRun(ctx, rdb, cfg.UserID, cfg.PluginID, time.Now())
			logger.Info(
				"Enqueued scheduled plugin execution",
				"user_id", cfg.UserID,
				"plugin_id", cfg.PluginID,
				"plugin_name", cfg.Plugin.Name,
				"cron_expression", cfg.CronExpression,
				"timezone", effectiveTimezone,
			)
			enqueued++
		}

		logger.Info(
			"Per-minute scheduler tick complete",
			"total_configs", len(configs),
			"enqueued", enqueued,
			"skipped", skipped,
			"errored", errored,
		)

		return nil
	}
}

// isDue reports whether a plugin execution is due at the current moment.
//
// Rules:
//   - Returns (false, nil) if cronExpr is empty.
//   - Falls back to "UTC" if timezone is empty or invalid.
//   - CRITICAL: if lastRunAt is zero (cold cache), treats it as one minute ago to
//     prevent mass-firing all scheduled jobs on first startup.
//   - Returns true when schedule.Next(lastRunAt) is not after time.Now() — i.e. the
//     next scheduled time has already passed.
func isDue(cronExpr string, timezone string, lastRunAt time.Time) (bool, error) {
	if cronExpr == "" {
		return false, nil
	}

	if timezone == "" {
		timezone = "UTC"
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		slog.Warn("Invalid timezone, falling back to UTC", "timezone", timezone, "error", err)
		loc = time.UTC
	}

	// Prepend CRON_TZ directive so the robfig parser evaluates in the user's local timezone.
	exprWithTZ := "CRON_TZ=" + loc.String() + " " + cronExpr

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(exprWithTZ)
	if err != nil {
		return false, fmt.Errorf("failed to parse cron expression: %w", err)
	}

	// Cold-cache protection: treat a zero lastRunAt as "one minute ago" so we
	// do not fire every scheduled job simultaneously on the first scheduler tick
	// after a cold start (Redis empty / new deployment).
	if lastRunAt.IsZero() {
		lastRunAt = time.Now().Add(-time.Minute)
	}

	nextRun := schedule.Next(lastRunAt)
	return !nextRun.After(time.Now()), nil
}

// fieldKey returns the Redis hash field key for a user-plugin pair.
func fieldKey(userID, pluginID uint) string {
	return fmt.Sprintf("%d:%d", userID, pluginID)
}

// getLastRun retrieves the last-run timestamp for a user-plugin pair from the Redis
// hash. Returns the zero time on cache miss or parse error.
func getLastRun(ctx context.Context, rdb *redis.Client, userID, pluginID uint) time.Time {
	val, err := rdb.HGet(ctx, lastRunHashKey, fieldKey(userID, pluginID)).Result()
	if err != nil {
		// Miss (redis.Nil) or connection error — both treated as zero time (cold cache)
		return time.Time{}
	}
	ts, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(ts, 0)
}

// setLastRun stores the last-run timestamp for a user-plugin pair in the Redis hash.
func setLastRun(ctx context.Context, rdb *redis.Client, userID, pluginID uint, t time.Time) {
	if err := rdb.HSet(ctx, lastRunHashKey, fieldKey(userID, pluginID), t.Unix()).Err(); err != nil {
		slog.Warn(
			"Failed to update last-run cache",
			"user_id", userID,
			"plugin_id", pluginID,
			"error", err,
		)
	}
}

// newSchedulerRedisClient creates a Redis client from a redis:// or rediss:// URL.
// Used by the per-minute scheduler handler to obtain its own Redis connection for
// the last-run cache (separate from the Asynq internal connection).
func newSchedulerRedisClient(redisURL string) (*redis.Client, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL for scheduler cache: %w", err)
	}
	return redis.NewClient(opt), nil
}
