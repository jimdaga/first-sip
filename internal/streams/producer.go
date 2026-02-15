package streams

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Publisher publishes plugin requests to Redis Streams
type Publisher struct {
	rdb *redis.Client
}

// NewPublisher creates a new Publisher instance
func NewPublisher(redisURL string) (*Publisher, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	return &Publisher{rdb: client}, nil
}

// PublishPluginRequest publishes a plugin request to the stream
func (p *Publisher) PublishPluginRequest(ctx context.Context, req PluginRequest) (string, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	result := p.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamPluginRequests,
		MaxLen: 10000,
		Approx: true,
		ID:     "*", // auto-generate ID
		Values: map[string]interface{}{
			"payload":        string(payload),
			"published_at":   time.Now().Unix(),
			"schema_version": SchemaVersionV1,
		},
	})

	if result.Err() != nil {
		return "", fmt.Errorf("failed to publish to stream: %w", result.Err())
	}

	return result.Val(), nil
}

// Close closes the Redis client connection
func (p *Publisher) Close() error {
	return p.rdb.Close()
}
