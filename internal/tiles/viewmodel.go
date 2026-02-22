// Package tiles provides the TileViewModel type shared between the dashboard
// query layer (internal/dashboard) and the Templ template layer (internal/templates).
// Keeping it in a dedicated package breaks the import cycle that would occur if
// templates imported dashboard directly.
package tiles

import "time"

// TileViewModel holds all data needed to render a single dashboard tile.
type TileViewModel struct {
	PluginID     uint
	PluginName   string
	PluginIcon   string // emoji from plugin YAML
	TileSize     string // "1x1", "2x1", "2x2"
	DisplayOrder *int
	Enabled      bool

	// Latest run data (zero values if no runs yet)
	LatestRunStatus string     // pending/processing/completed/failed or ""
	LatestRunAt     *time.Time // created_at of the latest run
	NextRunAt       *time.Time
	BriefingSummary string // 2-3 line summary from PluginRun.Output JSON
	BriefingContent string // Full briefing content from PluginRun.Output JSON (for expand-in-place)
	HasContent      bool   // true if a completed run exists

	// Error overlay: show last successful content even if latest run failed
	LastSuccessfulSummary string
	LastSuccessfulContent string // Full content from last successful run
	HasError              bool   // true if latest run is failed

	// Formatted tooltip text
	TimingTooltip string
}
