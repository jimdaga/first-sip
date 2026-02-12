package worker

import (
	"log/slog"
	"os"
	"strings"
)

// NewLogger creates a structured logger for the worker with configurable level and format.
// level: "debug", "info", "warn", "error" (defaults to info if invalid)
// format: "json" for JSON output, anything else for human-readable text
func NewLogger(level, format string) *slog.Logger {
	// Parse log level
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	// Create handler options
	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	// Create handler based on format
	var handler slog.Handler
	if strings.ToLower(format) == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
