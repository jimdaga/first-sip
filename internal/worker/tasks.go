package worker

import (
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
)

// Task type constants
const (
	TaskGenerateBriefing = "briefing:generate"
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
// and retain for 24 hours after completion.
func EnqueueGenerateBriefing(briefingID uint) error {
	// Create task payload
	payload, err := json.Marshal(map[string]uint{
		"briefing_id": briefingID,
	})
	if err != nil {
		return err
	}

	// Create task with options
	task := asynq.NewTask(
		TaskGenerateBriefing,
		payload,
		asynq.MaxRetry(3),
		asynq.Timeout(5*time.Minute),
		asynq.Retention(24*time.Hour),
	)

	// Enqueue the task
	_, err = client.Enqueue(task)
	return err
}
