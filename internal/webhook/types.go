// Package webhook provides n8n webhook integration for briefing generation
package webhook

// BriefingContent represents the structured content returned by the n8n webhook
type BriefingContent struct {
	News    []NewsItem   `json:"news"`
	Weather WeatherInfo  `json:"weather"`
	Work    WorkSummary  `json:"work"`
}

// NewsItem represents a single news article in the briefing
type NewsItem struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	URL     string `json:"url"`
}

// WeatherInfo represents current weather conditions
type WeatherInfo struct {
	Location    string `json:"location"`
	Temperature int    `json:"temperature"`
	Condition   string `json:"condition"`
}

// WorkSummary represents upcoming work events and tasks
type WorkSummary struct {
	TodayEvents    []string `json:"today_events"`
	TomorrowTasks  []string `json:"tomorrow_tasks"`
}
