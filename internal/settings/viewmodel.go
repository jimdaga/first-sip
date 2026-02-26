package settings

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kaptinlin/jsonschema"
	cron "github.com/robfig/cron/v3"
	"github.com/jimdaga/first-sip/internal/plugins"
	"github.com/jimdaga/first-sip/internal/settingsvm"
	"github.com/jimdaga/first-sip/internal/tiers"
	"gorm.io/gorm"
)

// Re-export settingsvm types as aliases so callers in this package can use short names.
// Templates import settingsvm directly; settings/handlers.go uses these aliases.
type FieldType = settingsvm.FieldType
type FieldViewModel = settingsvm.FieldViewModel
type ErrorEntry = settingsvm.ErrorEntry
type PluginStatusViewModel = settingsvm.PluginStatusViewModel
type PluginSettingsViewModel = settingsvm.PluginSettingsViewModel

const (
	FieldTypeText          = settingsvm.FieldTypeText
	FieldTypeEnum          = settingsvm.FieldTypeEnum
	FieldTypeInteger       = settingsvm.FieldTypeInteger
	FieldTypeBoolean       = settingsvm.FieldTypeBoolean
	FieldTypeCheckboxGroup = settingsvm.FieldTypeCheckboxGroup
	FieldTypeTagInput      = settingsvm.FieldTypeTagInput
	FieldTypeTimeSelect    = settingsvm.FieldTypeTimeSelect
)

// cronParser is a standard 5-field cron expression parser.
// Duplicated from dashboard/viewmodel.go to avoid import cycle.
var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// pluginRow is an intermediate scan type for BuildPluginSettingsViewModels.
type pluginRow struct {
	PluginID           uint
	PluginName         string
	Description        string
	Icon               string
	SettingsSchemaPath string
	Enabled            bool
	Settings           []byte
	CronExpression     string
	Timezone           string // sourced from users.timezone via JOIN
}

// BuildTierInfo builds a TierInfo for the given user by querying the TierService.
// Returns default TierInfo (free tier, 0 enabled) on any error to avoid blocking the page.
func BuildTierInfo(db *gorm.DB, tierService *tiers.TierService, userID uint) (settingsvm.TierInfo, error) {
	tier, err := tierService.GetUserTier(userID)
	if err != nil {
		return settingsvm.TierInfo{
			TierName:          "free",
			MaxEnabledPlugins: 3,
			EnabledCount:      0,
			AtPluginLimit:     false,
			MinFrequencyHours: 24,
			UpgradeURL:        "/pro",
		}, fmt.Errorf("settings: get user tier: %w", err)
	}

	count, err := tierService.GetEnabledCount(userID)
	if err != nil {
		count = 0 // non-fatal
	}

	atLimit := tier.MaxEnabledPlugins >= 0 && count >= tier.MaxEnabledPlugins

	return settingsvm.TierInfo{
		TierName:          tier.Name,
		MaxEnabledPlugins: tier.MaxEnabledPlugins,
		EnabledCount:      count,
		AtPluginLimit:     atLimit,
		MinFrequencyHours: tier.MinFrequencyHours,
		UpgradeURL:        "/pro",
	}, nil
}

// BuildPluginSettingsViewModels queries all plugins with user config and assembles
// PluginSettingsViewModel slice for the settings page.
// tierInfo is used to set IsDisabledByTier and IsFreeUser on each viewmodel.
func BuildPluginSettingsViewModels(db *gorm.DB, userID uint, pluginDir string, tierInfo settingsvm.TierInfo) ([]PluginSettingsViewModel, error) {
	var rows []pluginRow
	err := db.Raw(`
		SELECT
			p.id         AS plugin_id,
			p.name       AS plugin_name,
			p.description,
			p.icon,
			p.settings_schema_path,
			COALESCE(upc.enabled, false)           AS enabled,
			upc.settings,
			COALESCE(upc.cron_expression, '')      AS cron_expression,
			COALESCE(u.timezone, 'UTC')            AS timezone
		FROM plugins p
		LEFT JOIN user_plugin_configs upc
			ON p.id = upc.plugin_id
			AND upc.user_id = ?
			AND upc.deleted_at IS NULL
		LEFT JOIN users u ON u.id = ?
		WHERE p.deleted_at IS NULL
		ORDER BY p.name ASC
	`, userID, userID).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("settings: query plugins: %w", err)
	}

	vms := make([]PluginSettingsViewModel, 0, len(rows))
	for _, row := range rows {
		// Parse saved settings JSON.
		var savedSettings map[string]any
		if len(row.Settings) > 0 {
			if err := json.Unmarshal(row.Settings, &savedSettings); err != nil {
				savedSettings = nil
			}
		}

		// Load schema for this plugin.
		schema, err := loadPluginSchema(pluginDir, row.PluginName, row.SettingsSchemaPath)
		if err != nil {
			// Non-fatal: log and continue without schema.
			schema = nil
		}

		// Convert schema to field view models.
		var fields []FieldViewModel
		hasRequired := false
		if schema != nil {
			fields = schemaToFields(schema, savedSettings, nil, nil)
			// Check if any fields are required.
			for _, f := range fields {
				if f.Required {
					hasRequired = true
					break
				}
			}
		}

		// Compute plugin status.
		status := getPluginStatus(db, userID, row.PluginID, row.CronExpression, row.Timezone)

		vm := PluginSettingsViewModel{
			PluginID:          row.PluginID,
			PluginName:        row.PluginName,
			DisplayName:       humanizePluginName(row.PluginName),
			Description:       row.Description,
			Icon:              row.Icon,
			Enabled:           row.Enabled,
			Fields:            fields,
			HasSchema:         schema != nil,
			HasRequiredFields: hasRequired,
			Status:            status,
			CronExpression:    row.CronExpression,
			IsFreeUser:        tierInfo.TierName == "free",
		}
		// Disable the toggle for non-enabled plugins when user is at the plugin limit.
		if tierInfo.AtPluginLimit && !row.Enabled {
			vm.IsDisabledByTier = true
		}
		vms = append(vms, vm)
	}

	return vms, nil
}

// BuildSinglePluginSettingsViewModel builds a PluginSettingsViewModel for a single plugin.
// Used by handlers that need to re-render one accordion row.
func BuildSinglePluginSettingsViewModel(
	db *gorm.DB,
	userID, pluginID uint,
	pluginDir string,
	submittedValues map[string]string,
	fieldErrors map[string]string,
	saveSuccess bool,
) (*PluginSettingsViewModel, error) {
	var rows []pluginRow
	err := db.Raw(`
		SELECT
			p.id         AS plugin_id,
			p.name       AS plugin_name,
			p.description,
			p.icon,
			p.settings_schema_path,
			COALESCE(upc.enabled, false)           AS enabled,
			upc.settings,
			COALESCE(upc.cron_expression, '')      AS cron_expression,
			COALESCE(u.timezone, 'UTC')            AS timezone
		FROM plugins p
		LEFT JOIN user_plugin_configs upc
			ON p.id = upc.plugin_id
			AND upc.user_id = ?
			AND upc.deleted_at IS NULL
		LEFT JOIN users u ON u.id = ?
		WHERE p.deleted_at IS NULL
		  AND p.id = ?
	`, userID, userID, pluginID).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("settings: query single plugin: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("settings: plugin %d not found", pluginID)
	}
	row := rows[0]

	// Parse saved settings JSON.
	var savedSettings map[string]any
	if len(row.Settings) > 0 {
		if err := json.Unmarshal(row.Settings, &savedSettings); err != nil {
			savedSettings = nil
		}
	}

	// Load schema.
	schema, err := loadPluginSchema(pluginDir, row.PluginName, row.SettingsSchemaPath)
	if err != nil {
		schema = nil
	}

	var fields []FieldViewModel
	hasRequired := false
	if schema != nil {
		fields = schemaToFields(schema, savedSettings, submittedValues, fieldErrors)
		for _, f := range fields {
			if f.Required {
				hasRequired = true
				break
			}
		}
	}

	status := getPluginStatus(db, userID, row.PluginID, row.CronExpression, row.Timezone)

	vm := &PluginSettingsViewModel{
		PluginID:          row.PluginID,
		PluginName:        row.PluginName,
		DisplayName:       humanizePluginName(row.PluginName),
		Description:       row.Description,
		Icon:              row.Icon,
		Enabled:           row.Enabled,
		Fields:            fields,
		HasSchema:         schema != nil,
		HasRequiredFields: hasRequired,
		Status:            status,
		CronExpression:    row.CronExpression,
		SaveSuccess:       saveSuccess,
	}
	return vm, nil
}

// getPluginStatus queries plugin runs to compute status for a single plugin.
// Returns nil if there are no runs yet.
func getPluginStatus(db *gorm.DB, userID, pluginID uint, cronExpr, timezone string) *PluginStatusViewModel {
	// Latest run (any status).
	var latestRun plugins.PluginRun
	latestResult := db.Where("user_id = ? AND plugin_id = ? AND deleted_at IS NULL", userID, pluginID).
		Order("created_at DESC").First(&latestRun)

	// No runs at all.
	if latestResult.Error != nil {
		return nil
	}

	// Recent failed runs (last 5).
	var failedRuns []plugins.PluginRun
	db.Where("user_id = ? AND plugin_id = ? AND status = ? AND deleted_at IS NULL",
		userID, pluginID, plugins.PluginRunStatusFailed).
		Order("created_at DESC").Limit(5).Find(&failedRuns)

	errors := make([]ErrorEntry, 0, len(failedRuns))
	for _, r := range failedRuns {
		errors = append(errors, ErrorEntry{
			OccurredAt: r.CreatedAt,
			Message:    r.ErrorMessage,
		})
	}

	// Compute health color.
	health := "green"
	if latestRun.Status == plugins.PluginRunStatusFailed {
		health = "red"
	} else if len(failedRuns) > 0 {
		health = "yellow"
	}

	nextRun := computeNextRun(cronExpr, timezone)

	// Use CompletedAt if set, otherwise fall back to CreatedAt (run exists but hasn't completed yet).
	lastRunAt := latestRun.CompletedAt
	if lastRunAt == nil {
		lastRunAt = &latestRun.CreatedAt
	}

	return &PluginStatusViewModel{
		LastRunAt:    lastRunAt,
		NextRunAt:    nextRun,
		RecentErrors: errors,
		HealthColor:  health,
	}
}

// computeNextRun parses a 5-field cron expression and returns the next scheduled
// time in the given IANA timezone. Returns nil if cronExpr is empty or invalid.
// Duplicated from dashboard/viewmodel.go to avoid import cycle.
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

// loadPluginSchema reads and compiles the JSON Schema for a plugin.
// Returns nil, nil if schemaRelPath is empty (plugin has no schema).
func loadPluginSchema(pluginDir, pluginName, schemaRelPath string) (*jsonschema.Schema, error) {
	if schemaRelPath == "" {
		return nil, nil
	}
	fullPath := filepath.Join(pluginDir, pluginName, schemaRelPath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("read schema %s: %w", fullPath, err)
	}
	compiler := jsonschema.NewCompiler()
	// SetPreserveExtra(true) enables x- extension fields to land in schema.Extra.
	// Not required for current schemas but future-proofs extension field support.
	compiler.SetPreserveExtra(true)
	schema, err := compiler.Compile(data)
	if err != nil {
		return nil, fmt.Errorf("compile schema %s: %w", fullPath, err)
	}
	return schema, nil
}

// schemaToFields converts a compiled *jsonschema.Schema into a flat []FieldViewModel slice.
// Keys are sorted for deterministic ordering (Pitfall 6).
// Value priority: submittedValues > savedSettings > schema default.
func schemaToFields(
	schema *jsonschema.Schema,
	savedSettings map[string]any,
	submittedValues map[string]string,
	fieldErrors map[string]string,
) []FieldViewModel {
	if schema.Properties == nil {
		return nil
	}

	// Build required set from schema.Required.
	requiredSet := make(map[string]bool, len(schema.Required))
	for _, r := range schema.Required {
		requiredSet[r] = true
	}

	// Sort keys for deterministic form ordering.
	keys := make([]string, 0, len(*schema.Properties))
	for k := range *schema.Properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fields := make([]FieldViewModel, 0, len(keys))
	for _, key := range keys {
		propSchema := (*schema.Properties)[key]

		field := FieldViewModel{
			Key:         key,
			Label:       labelFromSchema(key, propSchema),
			Description: descriptionFromSchema(propSchema),
			Required:    requiredSet[key],
			FieldType:   fieldTypeFromSchema(propSchema),
			Default:     defaultAsString(propSchema.Default),
			Error:       fieldErrors["/"+key], // DetailedErrors uses JSON pointer paths
		}

		// Populate enum values for dropdowns / checkbox groups.
		if len(propSchema.Enum) > 0 {
			for _, e := range propSchema.Enum {
				field.EnumValues = append(field.EnumValues, fmt.Sprintf("%v", e))
			}
		}
		// Array type with enum items (e.g. topics field).
		if propSchema.Items != nil && len(propSchema.Items.Enum) > 0 {
			for _, e := range propSchema.Items.Enum {
				field.EnumValues = append(field.EnumValues, fmt.Sprintf("%v", e))
			}
		}

		// Value priority: submittedValues > savedSettings > default.
		if submittedValues != nil {
			if v, ok := submittedValues[key]; ok {
				field.CurrentValue = v
				fields = append(fields, field)
				continue
			}
		}
		if savedSettings != nil {
			if v, ok := savedSettings[key]; ok {
				field.CurrentValue = anyToString(v)
				fields = append(fields, field)
				continue
			}
		}
		field.CurrentValue = field.Default
		fields = append(fields, field)
	}

	return fields
}

// coerceFormValues converts raw HTML form values to typed Go values based on the schema.
// HTML forms always submit strings; this coerces to the JSON Schema declared types.
func coerceFormValues(rawForm url.Values, schema *jsonschema.Schema) (map[string]any, error) {
	if schema.Properties == nil {
		return map[string]any{}, nil
	}

	result := make(map[string]any)
	for key, propSchema := range *schema.Properties {
		rawVals := rawForm[key]

		typeStr := ""
		if len(propSchema.Type) > 0 {
			typeStr = propSchema.Type[0]
		}

		switch typeStr {
		case "integer":
			if len(rawVals) > 0 && rawVals[0] != "" {
				v, err := strconv.Atoi(rawVals[0])
				if err != nil {
					return nil, fmt.Errorf("field %s: not a valid integer", key)
				}
				result[key] = v
			}
		case "boolean":
			// HTML checkboxes send "on" when checked and are absent when unchecked.
			// Use hidden input value="false" + checkbox value="true" pattern (Pitfall 3).
			val := false
			if len(rawVals) > 0 {
				raw := strings.ToLower(rawVals[0])
				val = raw == "true" || raw == "on" || raw == "1"
			}
			result[key] = val
		case "array":
			// Tag input: hidden field sends a single JSON array string e.g. '["tech","science"]'.
			// Legacy multi-checkbox: multiple values sent for same key.
			if len(rawVals) == 1 {
				val := strings.TrimSpace(rawVals[0])
				if strings.HasPrefix(val, "[") {
					// Parse JSON array from tag input hidden field.
					var arr []string
					if jsonErr := json.Unmarshal([]byte(val), &arr); jsonErr == nil {
						result[key] = arr
					} else if rawVals != nil {
						result[key] = rawVals
					} else {
						result[key] = []string{}
					}
				} else {
					result[key] = rawVals
				}
			} else if rawVals != nil {
				result[key] = rawVals // multiple checkbox values (legacy)
			} else {
				result[key] = []string{}
			}
		default:
			// string, enum string
			if len(rawVals) > 0 {
				result[key] = rawVals[0]
			}
		}
	}
	return result, nil
}

// validateAndGetFieldErrors validates typed settings against the schema and returns
// a map of JSON pointer path → error message. Returns nil when valid.
func validateAndGetFieldErrors(schema *jsonschema.Schema, typedValues map[string]any) map[string]string {
	result := schema.Validate(typedValues)
	if result.IsValid() {
		return nil
	}
	// DetailedErrors returns map[string]string keyed by JSON pointer paths like "/frequency".
	return result.DetailedErrors()
}

// validateSingleField validates a single field value against the schema.
// Returns an error message string, or "" if valid.
func validateSingleField(schema *jsonschema.Schema, fieldKey string, rawValue string) string {
	if schema.Properties == nil {
		return ""
	}

	propSchema, ok := (*schema.Properties)[fieldKey]
	if !ok {
		return ""
	}

	// Build a minimal coerced map with just this one field.
	minimalForm := url.Values{}

	typeStr := ""
	if len(propSchema.Type) > 0 {
		typeStr = propSchema.Type[0]
	}

	switch typeStr {
	case "array":
		// For array fields, treat rawValue as a single item.
		minimalForm[fieldKey] = []string{rawValue}
	default:
		minimalForm[fieldKey] = []string{rawValue}
	}

	typedValues, err := coerceFormValues(minimalForm, schema)
	if err != nil {
		return err.Error()
	}

	// Validate the full schema against the minimal map.
	// Only return the error for this specific field.
	result := schema.Validate(typedValues)
	if result.IsValid() {
		return ""
	}

	detailed := result.DetailedErrors()
	if msg, ok := detailed["/"+fieldKey]; ok {
		return msg
	}
	return ""
}

// humanizePluginName converts a kebab-case plugin name to a title-case display name.
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

// labelFromSchema returns the display label for a field.
// Uses schema.Title if non-empty, otherwise humanizes the key.
func labelFromSchema(key string, s *jsonschema.Schema) string {
	if s.Title != nil && *s.Title != "" {
		return *s.Title
	}
	// Humanize: "preferred_time" -> "Preferred Time"
	words := strings.Split(key, "_")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// descriptionFromSchema returns the description for a field, or "" if unset.
func descriptionFromSchema(s *jsonschema.Schema) string {
	if s.Description != nil {
		return *s.Description
	}
	return ""
}

// timePatternRe matches HH:MM 24-hour time pattern schemas (e.g. "^([0-1][0-9]|2[0-3]):[0-5][0-9]$").
// These are rendered as a <select> with half-hour time slots instead of a free text input.
const timePatternSubstr = ":[0-5][0-9]$"

// fieldTypeFromSchema determines the FieldType for a schema property.
func fieldTypeFromSchema(s *jsonschema.Schema) FieldType {
	typeStr := ""
	if len(s.Type) > 0 {
		typeStr = s.Type[0]
	}

	switch typeStr {
	case "boolean":
		return FieldTypeBoolean
	case "integer":
		// Integer with enum → use enum dropdown.
		if len(s.Enum) > 0 {
			return FieldTypeEnum
		}
		return FieldTypeInteger
	case "array":
		// All array fields use a tag input — whether or not they have predefined enum items.
		// Predefined enums appear as hints; validation enforces constraints at save time.
		return FieldTypeTagInput
	default:
		// string or unknown
		if len(s.Enum) > 0 {
			return FieldTypeEnum
		}
		// String with a time pattern → time select.
		if s.Pattern != nil && strings.Contains(*s.Pattern, timePatternSubstr) {
			return FieldTypeTimeSelect
		}
		return FieldTypeText
	}
}

// defaultAsString formats a schema default value as a string.
func defaultAsString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case bool:
		return strconv.FormatBool(val)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return strconv.Itoa(val)
	case []any:
		// Array defaults: serialize as JSON.
		b, err := json.Marshal(val)
		if err != nil {
			return ""
		}
		return string(b)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// anyToString converts any value from savedSettings to a string for display.
func anyToString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case bool:
		return strconv.FormatBool(val)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return strconv.Itoa(val)
	case []any:
		// Array values from JSON: serialize as joined string or JSON.
		b, err := json.Marshal(val)
		if err != nil {
			return ""
		}
		return string(b)
	default:
		return fmt.Sprintf("%v", v)
	}
}
