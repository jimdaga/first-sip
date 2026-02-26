package dashboard

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
	"time"

	cron "github.com/robfig/cron/v3"
	"github.com/jimdaga/first-sip/internal/templates"
	"github.com/jimdaga/first-sip/internal/tiles"
	"gorm.io/gorm"
)

// cronParser is a standard 5-field cron expression parser (same options as in plugins/models.go).
var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// TileViewModel is an alias for tiles.TileViewModel.
// The type lives in internal/tiles to avoid an import cycle between
// internal/dashboard (query layer) and internal/templates (render layer).
type TileViewModel = tiles.TileViewModel

// OutputSection is a titled section within a plugin run's output.
type OutputSection struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// PluginRunOutput is the JSONB structure stored in plugin_runs.output.
type PluginRunOutput struct {
	Summary  string          `json:"summary"`
	Sections []OutputSection `json:"sections"`
	Content  string          `json:"content"` // legacy — old completed runs only
}

// configRow is an intermediate type for scanning the configs query result.
type configRow struct {
	PluginID       uint
	PluginName     string
	Icon           string
	TileSize       string
	DisplayOrder   *int
	CronExpression string
	Timezone       string
}

// latestRunRow is an intermediate type for scanning the latest runs query result.
type latestRunRow struct {
	PluginID    uint
	Status      string
	Output      []byte
	CompletedAt *time.Time
	CreatedAt   time.Time
}

// latestSuccessfulRunRow is an intermediate type for scanning the latest successful runs query.
type latestSuccessfulRunRow struct {
	PluginID uint
	Output   []byte
}

// getDashboardTiles fetches all enabled plugin configs for the user and assembles
// TileViewModels with the latest run data. Uses exactly three queries:
//  1. Enabled plugin configs joined with plugins table.
//  2. DISTINCT ON (plugin_id) latest plugin runs for this user (any status).
//  3. DISTINCT ON (plugin_id) latest successful (completed) plugin runs.
//
// No N+1 — last-successful lookup is map-based from the batch query.
func getDashboardTiles(db *gorm.DB, userID uint) ([]TileViewModel, error) {
	// --- Query 1: Enabled plugin configs ordered by display_order ---
	var configs []configRow
	err := db.Raw(`
		SELECT
			upc.plugin_id,
			p.name      AS plugin_name,
			p.icon,
			p.tile_size,
			upc.display_order,
			upc.cron_expression,
			upc.timezone
		FROM user_plugin_configs upc
		JOIN plugins p ON p.id = upc.plugin_id
		WHERE upc.user_id = ?
		  AND upc.enabled = true
		  AND upc.deleted_at IS NULL
		  AND p.deleted_at IS NULL
		ORDER BY upc.display_order ASC NULLS LAST, p.name ASC
	`, userID).Scan(&configs).Error
	if err != nil {
		return nil, fmt.Errorf("dashboard: query configs: %w", err)
	}

	if len(configs) == 0 {
		return []TileViewModel{}, nil
	}

	// --- Query 2: Latest run per plugin (any status) via DISTINCT ON ---
	var latestRuns []latestRunRow
	err = db.Raw(`
		SELECT DISTINCT ON (plugin_id)
			plugin_id,
			status,
			output,
			completed_at,
			created_at
		FROM plugin_runs
		WHERE user_id = ?
		  AND deleted_at IS NULL
		ORDER BY plugin_id, created_at DESC
	`, userID).Scan(&latestRuns).Error
	if err != nil {
		return nil, fmt.Errorf("dashboard: query latest runs: %w", err)
	}

	// Build O(1) lookup map for latest runs.
	latestRunMap := make(map[uint]latestRunRow, len(latestRuns))
	for _, r := range latestRuns {
		latestRunMap[r.PluginID] = r
	}

	// --- Query 3: Latest successful (completed) run per plugin via DISTINCT ON ---
	var latestSuccessRuns []latestSuccessfulRunRow
	err = db.Raw(`
		SELECT DISTINCT ON (plugin_id)
			plugin_id,
			output
		FROM plugin_runs
		WHERE user_id = ?
		  AND status = 'completed'
		  AND deleted_at IS NULL
		ORDER BY plugin_id, created_at DESC
	`, userID).Scan(&latestSuccessRuns).Error
	if err != nil {
		return nil, fmt.Errorf("dashboard: query latest successful runs: %w", err)
	}

	// Build O(1) lookup map for latest successful runs.
	latestSuccessMap := make(map[uint]latestSuccessfulRunRow, len(latestSuccessRuns))
	for _, r := range latestSuccessRuns {
		latestSuccessMap[r.PluginID] = r
	}

	// --- Assemble TileViewModels ---
	tiles := make([]TileViewModel, 0, len(configs))
	for _, cfg := range configs {
		tile := TileViewModel{
			PluginID:     cfg.PluginID,
			PluginName:   cfg.PluginName,
			DisplayName:  humanizePluginName(cfg.PluginName),
			PluginIcon:   cfg.Icon,
			TileSize:     cfg.TileSize,
			DisplayOrder: cfg.DisplayOrder,
			Enabled:      true,
		}

		if tile.TileSize == "" {
			tile.TileSize = "1x1"
		}

		// Populate from latest run (O(1) map lookup).
		if run, ok := latestRunMap[cfg.PluginID]; ok {
			tile.LatestRunStatus = run.Status
			tile.LatestRunAt = &run.CreatedAt
			tile.BriefingSummary = extractSummary(run.Output)
			tile.BriefingContent = extractContent(run.Output)

			if run.Status == "completed" {
				tile.HasContent = true
			}

			if run.Status == "failed" {
				tile.HasError = true
				// O(1) lookup for last successful run — no per-row query.
				if successRun, ok := latestSuccessMap[cfg.PluginID]; ok {
					tile.LastSuccessfulSummary = extractSummary(successRun.Output)
					tile.LastSuccessfulContent = extractContent(successRun.Output)
					if tile.LastSuccessfulContent != "" {
						tile.HasContent = true
					}
				}
			}
		}

		// Compute next scheduled run.
		tile.NextRunAt = computeNextRun(cfg.CronExpression, cfg.Timezone)

		// Format timing tooltip.
		tile.TimingTooltip = formatTimingTooltip(tile)

		tiles = append(tiles, tile)
	}

	return tiles, nil
}

// GetSingleTile fetches a single plugin's tile data for HTMX per-tile polling.
// Uses the same three-query batch logic as getDashboardTiles but filtered to one plugin.
func GetSingleTile(db *gorm.DB, userID, pluginID uint) (*TileViewModel, error) {
	// Query config for this specific plugin.
	var configs []configRow
	err := db.Raw(`
		SELECT
			upc.plugin_id,
			p.name      AS plugin_name,
			p.icon,
			p.tile_size,
			upc.display_order,
			upc.cron_expression,
			upc.timezone
		FROM user_plugin_configs upc
		JOIN plugins p ON p.id = upc.plugin_id
		WHERE upc.user_id = ?
		  AND upc.plugin_id = ?
		  AND upc.enabled = true
		  AND upc.deleted_at IS NULL
		  AND p.deleted_at IS NULL
	`, userID, pluginID).Scan(&configs).Error
	if err != nil {
		return nil, fmt.Errorf("dashboard: query single config: %w", err)
	}
	if len(configs) == 0 {
		return nil, nil
	}
	cfg := configs[0]

	// Latest run for this plugin.
	var latestRuns []latestRunRow
	err = db.Raw(`
		SELECT DISTINCT ON (plugin_id)
			plugin_id,
			status,
			output,
			completed_at,
			created_at
		FROM plugin_runs
		WHERE user_id = ?
		  AND plugin_id = ?
		  AND deleted_at IS NULL
		ORDER BY plugin_id, created_at DESC
	`, userID, pluginID).Scan(&latestRuns).Error
	if err != nil {
		return nil, fmt.Errorf("dashboard: query single latest run: %w", err)
	}

	// Latest successful run for this plugin.
	var latestSuccessRuns []latestSuccessfulRunRow
	err = db.Raw(`
		SELECT DISTINCT ON (plugin_id)
			plugin_id,
			output
		FROM plugin_runs
		WHERE user_id = ?
		  AND plugin_id = ?
		  AND status = 'completed'
		  AND deleted_at IS NULL
		ORDER BY plugin_id, created_at DESC
	`, userID, pluginID).Scan(&latestSuccessRuns).Error
	if err != nil {
		return nil, fmt.Errorf("dashboard: query single latest successful run: %w", err)
	}

	tile := &TileViewModel{
		PluginID:     cfg.PluginID,
		PluginName:   cfg.PluginName,
		DisplayName:  humanizePluginName(cfg.PluginName),
		PluginIcon:   cfg.Icon,
		TileSize:     cfg.TileSize,
		DisplayOrder: cfg.DisplayOrder,
		Enabled:      true,
	}
	if tile.TileSize == "" {
		tile.TileSize = "1x1"
	}

	if len(latestRuns) > 0 {
		run := latestRuns[0]
		tile.LatestRunStatus = run.Status
		tile.LatestRunAt = &run.CreatedAt
		tile.BriefingSummary = extractSummary(run.Output)
		tile.BriefingContent = extractContent(run.Output)

		if run.Status == "completed" {
			tile.HasContent = true
		}
		if run.Status == "failed" {
			tile.HasError = true
			if len(latestSuccessRuns) > 0 {
				tile.LastSuccessfulSummary = extractSummary(latestSuccessRuns[0].Output)
				tile.LastSuccessfulContent = extractContent(latestSuccessRuns[0].Output)
				if tile.LastSuccessfulContent != "" {
					tile.HasContent = true
				}
			}
		}
	}

	tile.NextRunAt = computeNextRun(cfg.CronExpression, cfg.Timezone)
	tile.TimingTooltip = formatTimingTooltip(*tile)

	return tile, nil
}

// computeNextRun parses a 5-field cron expression and returns the next scheduled
// time in the given IANA timezone. Returns nil if cronExpr is empty or invalid.
func computeNextRun(cronExpr, timezone string) *time.Time {
	if cronExpr == "" {
		return nil
	}
	schedule, err := cronParser.Parse(cronExpr)
	if err != nil {
		return nil
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil || loc == nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	next := schedule.Next(now)
	return &next
}

// extractSummary parses the JSONB output and returns the summary field.
// Falls back to the first 200 chars of content if summary is empty.
func extractSummary(output []byte) string {
	if len(output) == 0 {
		return ""
	}
	var out PluginRunOutput
	if err := json.Unmarshal(output, &out); err != nil {
		return ""
	}
	if out.Summary != "" {
		return out.Summary
	}
	// Fallback: first 200 chars of content.
	if len(out.Content) > 200 {
		return out.Content[:200]
	}
	return out.Content
}

// extractContent parses the JSONB output and returns HTML for tile display.
// New format: renders sections array as HTML (<h3> headings + <p> content).
// Legacy format: falls back to the Content string field for old completed runs.
// Malformed/unparseable output returns "" so old runs show as empty (no crash).
func extractContent(output []byte) string {
	if len(output) == 0 {
		return ""
	}
	var out PluginRunOutput
	if err := json.Unmarshal(output, &out); err != nil {
		return ""
	}
	// New format: build HTML from sections array.
	if len(out.Sections) > 0 {
		var sb strings.Builder
		for _, s := range out.Sections {
			if s.Title != "" && s.Title != "Briefing" {
				sb.WriteString("<h3>")
				sb.WriteString(template.HTMLEscapeString(s.Title))
				sb.WriteString("</h3>")
			}
			sb.WriteString("<p>")
			sb.WriteString(s.Content)
			sb.WriteString("</p>")
		}
		return sb.String()
	}
	// Legacy fallback: single content field (old completed runs).
	return out.Content
}

// timeAwareGreeting returns a time-appropriate greeting based on the user's timezone.
func timeAwareGreeting(name, timezone string) string {
	loc, err := time.LoadLocation(timezone)
	if err != nil || loc == nil {
		loc = time.UTC
	}
	hour := time.Now().In(loc).Hour()
	switch {
	case hour < 12:
		return "Good morning, " + name
	case hour < 17:
		return "Good afternoon, " + name
	default:
		return "Good evening, " + name
	}
}

// formatTimingTooltip formats the "Last run: X ago · Next: Y" tooltip string.
func formatTimingTooltip(tile TileViewModel) string {
	lastPart := "No runs yet"
	if tile.LatestRunAt != nil {
		lastPart = "Last run: " + formatRelativeTime(*tile.LatestRunAt)
	}

	nextPart := "No schedule"
	if tile.NextRunAt != nil {
		nextPart = "Next: " + formatRelativeTime(*tile.NextRunAt)
	}

	return lastPart + " \u00b7 " + nextPart
}

// formatRelativeTime returns a human-readable relative time string such as
// "2 hours ago" or "in 30 minutes".
func formatRelativeTime(t time.Time) string {
	diff := time.Until(t)
	abs := diff
	if abs < 0 {
		abs = -abs
	}

	switch {
	case abs < time.Minute:
		if diff >= 0 {
			return "just now"
		}
		return "just now"
	case abs < time.Hour:
		mins := int(abs.Minutes())
		if diff >= 0 {
			return fmt.Sprintf("in %d minute%s", mins, plural(mins))
		}
		return fmt.Sprintf("%d minute%s ago", mins, plural(mins))
	case abs < 24*time.Hour:
		hours := int(abs.Hours())
		if diff >= 0 {
			return fmt.Sprintf("in %d hour%s", hours, plural(hours))
		}
		return fmt.Sprintf("%d hour%s ago", hours, plural(hours))
	default:
		days := int(abs.Hours() / 24)
		if diff >= 0 {
			return fmt.Sprintf("in %d day%s", days, plural(days))
		}
		return fmt.Sprintf("%d day%s ago", days, plural(days))
	}
}

// plural returns "s" if n != 1, used for simple English pluralization.
func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// GetSidebarPlugins returns the list of enabled plugins for the sidebar sub-links.
// Uses a lightweight query that only fetches the plugin name and icon.
func GetSidebarPlugins(db *gorm.DB, userID uint) []templates.SidebarPlugin {
	type sidebarRow struct {
		Name string
		Icon string
	}
	var rows []sidebarRow
	db.Raw(`
		SELECT p.name, p.icon
		FROM user_plugin_configs upc
		JOIN plugins p ON p.id = upc.plugin_id
		WHERE upc.user_id = ?
		  AND upc.enabled = true
		  AND upc.deleted_at IS NULL
		  AND p.deleted_at IS NULL
		ORDER BY upc.display_order ASC NULLS LAST, p.name ASC
	`, userID).Scan(&rows)

	result := make([]templates.SidebarPlugin, 0, len(rows))
	for _, r := range rows {
		result = append(result, templates.SidebarPlugin{
			PluginName:  r.Name,
			DisplayName: humanizePluginName(r.Name),
			Icon:        r.Icon,
		})
	}
	return result
}

// humanizePluginName converts a kebab-case plugin name to title case.
// E.g. "daily-news-digest" → "Daily News Digest".
func humanizePluginName(name string) string {
	words := strings.Split(name, "-")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
