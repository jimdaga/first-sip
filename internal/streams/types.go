package streams

// Stream name constants
const (
	StreamPluginRequests = "plugin:requests"
	StreamPluginResults  = "plugin:results"
)

// Consumer group constants
const (
	GroupCrewAIWorkers = "crewai-workers" // Python side
	GroupGoWorkers     = "go-workers"     // Go side
)

// Schema version constant
const (
	SchemaVersionV1 = "v1"
)

// PluginRequest represents a plugin execution request message
type PluginRequest struct {
	PluginRunID string                 `json:"plugin_run_id"`
	PluginName  string                 `json:"plugin_name"`
	UserID      uint                   `json:"user_id"`
	Settings    map[string]interface{} `json:"settings"`
}

// PluginResult represents a plugin execution result message
type PluginResult struct {
	PluginRunID string `json:"plugin_run_id"`
	Status      string `json:"status"` // completed/failed
	Output      string `json:"output"` // the briefing content
	Error       string `json:"error"`  // error message if failed
}
