// Package settingsvm provides the view model types shared between the settings
// handler layer (internal/settings) and the Templ template layer (internal/templates).
// Keeping them in a dedicated package breaks the import cycle that would occur if
// templates imported settings directly (since settings/handlers.go imports templates).
package settingsvm

import "time"

// FieldType classifies a schema property into a renderable form field type.
type FieldType string

const (
	FieldTypeText          FieldType = "text"           // string, no enum
	FieldTypeEnum          FieldType = "enum"           // string/integer with enum values
	FieldTypeInteger       FieldType = "integer"        // type: integer, no enum
	FieldTypeBoolean       FieldType = "boolean"        // type: boolean
	FieldTypeCheckboxGroup FieldType = "checkbox_group" // type: array with enum items
	FieldTypeTagInput      FieldType = "tag_input"      // type: array with open-ended string tags
	FieldTypeTimeSelect    FieldType = "time_select"    // string with HH:MM time pattern — renders as select
)

// FieldViewModel is what Templ receives — no jsonschema types.
type FieldViewModel struct {
	Key          string    // JSON property key (e.g. "frequency")
	Label        string    // Display label: schema.Title if set, else humanize(key)
	Description  string    // Help text for tooltip: schema.Description
	FieldType    FieldType // Determines which input to render
	Required     bool      // from schema.Required list
	Default      string    // schema.Default formatted as string
	EnumValues   []string  // for FieldTypeEnum and FieldTypeCheckboxGroup
	CurrentValue string    // string form of saved/submitted value
	Error        string    // inline error message (empty = no error)
}

// ErrorEntry is a single plugin run error for display.
type ErrorEntry struct {
	OccurredAt time.Time
	Message    string
}

// PluginStatusViewModel holds computed plugin run status for display.
type PluginStatusViewModel struct {
	LastRunAt    *time.Time
	NextRunAt    *time.Time
	RecentErrors []ErrorEntry
	HealthColor  string // "green", "yellow", "red"
}

// PluginSettingsViewModel is the top-level view model for each plugin accordion row.
type PluginSettingsViewModel struct {
	PluginID          uint
	PluginName        string   // raw DB name (e.g. "daily-news-digest")
	DisplayName       string   // humanized display name (e.g. "Daily News Digest")
	Description       string
	Icon              string
	Enabled           bool
	Fields            []FieldViewModel // nil if plugin has no schema
	HasSchema         bool
	HasRequiredFields bool // for auto-expand on enable
	Status            *PluginStatusViewModel
	CronExpression    string // editable in form
	Timezone          string // editable in form
	SaveSuccess       bool   // set to true on successful save — drives "Saved ✓" in template
}
