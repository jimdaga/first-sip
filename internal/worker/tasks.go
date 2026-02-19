package worker

import (
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/hibiken/asynq"
)

// Task type constants
const (
	TaskGenerateBriefing   = "briefing:generate"
	TaskExecutePlugin      = "plugin:execute"
	TaskPerMinuteScheduler = "scheduler:per_minute"
)

// Package-level Asynq client (singleton)
var client *asynq.Client

// InitClient initializes the global Asynq client for task enqueueing.
// Must be called before any EnqueueX functions.
func InitClient(redisURL string) error {
	opt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		return err
	}

	client = asynq.NewClient(opt)
	return nil
}

// CloseClient closes the Asynq client connection gracefully.
func CloseClient() error {
	if client != nil {
		return client.Close()
	}
	return nil
}

// EnqueueGenerateBriefing enqueues a briefing generation task for the given briefing ID.
// The task will be processed by the worker with a 5-minute timeout, retry up to 3 times,
// and retain for 24 hours after completion. Duplicate tasks within 1 hour are prevented.
func EnqueueGenerateBriefing(briefingID uint) error {
	// Create task payload
	payload, err := json.Marshal(map[string]uint{
		"briefing_id": briefingID,
	})
	if err != nil {
		return err
	}

	// Create task with options including Unique to prevent duplicates
	task := asynq.NewTask(
		TaskGenerateBriefing,
		payload,
		asynq.MaxRetry(3),
		asynq.Timeout(5*time.Minute),
		asynq.Retention(24*time.Hour),
		asynq.Unique(1*time.Hour), // Prevent duplicate generation within 1 hour
	)

	// Enqueue the task
	_, err = client.Enqueue(task)
	if err != nil {
		// Check if duplicate task error (not a failure condition)
		if errors.Is(err, asynq.ErrDuplicateTask) {
			log.Printf("Briefing %d already queued (duplicate), skipping", briefingID)
			return nil // Not an error - task already enqueued
		}
		return err
	}
	return nil
}

// EnqueueExecutePlugin enqueues a plugin execution task.
// Uses a 10-minute timeout (CrewAI workflows are long-running), retries up to 2 times,
// retains for 24 hours, and prevents duplicate executions within 30 minutes.
func EnqueueExecutePlugin(pluginID uint, userID uint, pluginName string, settings map[string]interface{}) error {
	payload, err := json.Marshal(map[string]interface{}{
		"plugin_id":   pluginID,
		"user_id":     userID,
		"plugin_name": pluginName,
		"settings":    settings,
	})
	if err != nil {
		return err
	}

	task := asynq.NewTask(
		TaskExecutePlugin,
		payload,
		asynq.MaxRetry(2),
		asynq.Timeout(10*time.Minute),
		asynq.Retention(24*time.Hour),
		asynq.Unique(30*time.Minute), // Prevent duplicate plugin executions
	)

	_, err = client.Enqueue(task)
	if err != nil {
		if errors.Is(err, asynq.ErrDuplicateTask) {
			log.Printf("Plugin %s (user %d) already queued (duplicate), skipping", pluginName, userID)
			return nil // Not an error - task already enqueued
		}
		return err
	}
	return nil
}
