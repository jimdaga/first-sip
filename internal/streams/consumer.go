package streams

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ResultConsumer consumes plugin results from Redis Streams
type ResultConsumer struct {
	rdb          *redis.Client
	groupName    string
	consumerName string
}

// NewResultConsumer creates a new ResultConsumer instance
func NewResultConsumer(redisURL, consumerName string) (*ResultConsumer, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	// Read timeout must exceed the XReadGroup Block duration (5s)
	// to avoid spurious i/o timeout errors on idle streams.
	opts.ReadTimeout = 10 * time.Second

	client := redis.NewClient(opts)

	// Create consumer group on plugin:results stream
	// Start ID "0" means read from beginning if group is new
	err = client.XGroupCreateMkStream(context.Background(), StreamPluginResults, GroupGoWorkers, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}
	// Ignore BUSYGROUP error - group already exists

	return &ResultConsumer{
		rdb:          client,
		groupName:    GroupGoWorkers,
		consumerName: consumerName,
	}, nil
}

// ConsumeResults runs a blocking loop consuming results from the stream
func (c *ResultConsumer) ConsumeResults(ctx context.Context, handler func(PluginResult) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Read from stream with consumer group
		streams, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.groupName,
			Consumer: c.consumerName,
			Streams:  []string{StreamPluginResults, ">"},
			Count:    10,
			Block:    5000, // 5 seconds
		}).Result()

		if err == redis.Nil {
			// No messages available, continue loop
			continue
		}

		if err != nil {
			// Blocking reads return a timeout when no messages arrive
			// within the Block duration — this is normal, not an error.
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				continue
			}
			slog.Error("Failed to read from stream", "error", err)
			continue
		}

		// Process messages
		for _, stream := range streams {
			for _, message := range stream.Messages {
				// Extract payload from message values
				payloadStr, ok := message.Values["payload"].(string)
				if !ok {
					slog.Error("Invalid message payload", "message_id", message.ID)
					continue
				}

				// Unmarshal result
				var result PluginResult
				if err := json.Unmarshal([]byte(payloadStr), &result); err != nil {
					slog.Error("Failed to unmarshal result", "error", err, "message_id", message.ID)
					continue
				}

				// Call handler
				if err := handler(result); err != nil {
					slog.Error("Handler failed", "error", err, "plugin_run_id", result.PluginRunID)
					// Message stays in PEL for retry, don't ACK
					continue
				}

				// ACK successful processing
				if err := c.rdb.XAck(ctx, StreamPluginResults, c.groupName, message.ID).Err(); err != nil {
					slog.Error("Failed to ACK message", "error", err, "message_id", message.ID)
				}
			}
		}
	}
}

// Close closes the Redis client connection
func (c *ResultConsumer) Close() error {
	return c.rdb.Close()
}

// StartResultConsumer is a convenience function that starts the result consumer
// in a background goroutine and returns a stop function
func StartResultConsumer(redisURL string, db *gorm.DB) (stop func(), err error) {
	consumer, err := NewResultConsumer(redisURL, "go-worker-1")
	if err != nil {
		return nil, fmt.Errorf("failed to create result consumer: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Start consumer in background goroutine
	go func() {
		if err := consumer.ConsumeResults(ctx, HandlePluginResult(db)); err != nil {
			if err != context.Canceled {
				slog.Error("Result consumer stopped with error", "error", err)
			}
		}
	}()

	slog.Info("Result consumer started")

	// Return stop function that cancels context and closes consumer
	return func() {
		cancel()
		consumer.Close()
	}, nil
}
