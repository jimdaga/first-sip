package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles communication with the n8n webhook for briefing generation
type Client struct {
	baseURL    string
	secret     string
	httpClient *http.Client
	stubMode   bool
}

// NewClient creates a new webhook client with the given configuration
func NewClient(baseURL, secret string, stubMode bool) *Client {
	return &Client{
		baseURL:    baseURL,
		secret:     secret,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		stubMode:   stubMode,
	}
}

// GenerateBriefing requests briefing content generation for the specified user
func (c *Client) GenerateBriefing(ctx context.Context, userID uint) (*BriefingContent, error) {
	if c.stubMode {
		// Return hardcoded mock data with simulated processing delay
		time.Sleep(2 * time.Second)
		return &BriefingContent{
			News: []NewsItem{
				{
					Title:   "Breaking: AI Breakthrough Announced",
					Summary: "Researchers have achieved a significant milestone in artificial intelligence development.",
					URL:     "https://example.com/ai-breakthrough",
				},
				{
					Title:   "Tech Giant Launches New Product Line",
					Summary: "Major technology company unveils innovative consumer devices at annual conference.",
					URL:     "https://example.com/new-product",
				},
			},
			Weather: WeatherInfo{
				Location:    "San Francisco",
				Temperature: 65,
				Condition:   "Partly Cloudy",
			},
			Work: WorkSummary{
				TodayEvents: []string{
					"10:00 AM - Team standup meeting",
					"2:00 PM - Client presentation",
					"4:00 PM - Code review session",
				},
				TomorrowTasks: []string{
					"Review Q1 roadmap document",
					"Update project dependencies",
					"Complete performance optimization tasks",
				},
			},
		}, nil
	}

	// Production mode: make actual HTTP request to n8n webhook
	reqBody := map[string]interface{}{
		"user_id": userID,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-N8N-SECRET", c.secret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	var content BriefingContent
	if err := json.NewDecoder(resp.Body).Decode(&content); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &content, nil
}
