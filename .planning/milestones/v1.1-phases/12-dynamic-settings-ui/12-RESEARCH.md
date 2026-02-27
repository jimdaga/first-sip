# Phase 12: Dynamic Settings UI - Research

**Researched:** 2026-02-22
**Domain:** Go settings handler, JSON Schema form generation, HTMX accordion UI, kaptinlin/jsonschema validation
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Page layout & plugin list**
- Collapsed accordion layout — each plugin is a row that expands to reveal settings form and status
- Collapsed row shows: plugin name, enable/disable toggle, and a colored status indicator dot (green = healthy, yellow = warning, red = errors)
- Enable/disable toggle works instantly via HTMX (no save button for toggle)
- Enabling a plugin with required settings auto-expands the accordion to show the settings form

**Form presentation**
- Help text shown via tooltip on hover/icon next to field labels — keeps forms compact
- Save button per plugin form within the expanded accordion

**Validation & error feedback**
- Validate on blur (individual fields) and on submit (full form) — both server-side via HTMX
- Inline errors only, no top-of-form error summary
- On successful save, the Save button briefly changes to "Saved ✓" then returns to normal
- Form preserves user's input on validation errors (re-render with submitted values, not defaults)

**Plugin status display**
- Detailed status section inside the expanded accordion (alongside settings form)
- Shows: last run time, next run time, and recent error list (last 3-5 errors with timestamps and messages)
- "Run Now" button in expanded view to manually trigger a plugin run

### Claude's Discretion

- Field label sourcing strategy (schema 'title' vs humanized key — pick best approach based on what schemas provide)
- Boolean field rendering (toggle switch vs checkbox — pick what fits glass design system)
- Enum field rendering (dropdown vs radio buttons — pick based on option count)

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| SET-01 | Settings page listing all available plugins with enable/disable toggle | Plugin registry + DB query for all plugins with UserPluginConfig join; HTMX POST toggle handler returning updated row fragment |
| SET-02 | Dynamic form generation from plugin's JSON Schema settings definition | kaptinlin/jsonschema `*Schema` parsed from file; iterate `schema.Properties` (a `*SchemaMap`); render fields by `schema.Type`, `schema.Enum`, `schema.Required` |
| SET-03 | kaptinlin/jsonschema validation with inline error display | `schema.Validate(map[string]any)` returns `*EvaluationResult`; use `result.DetailedErrors()` → `map[string]string` keyed by JSON pointer path for per-field error display |
| SET-04 | Form type coercion (HTML string inputs → JSON Schema types: integer, boolean) | HTML `<form>` always submits strings; coerce before calling `ValidateUserSettings`: `strconv.Atoi` for integer, string `"true"/"false"/"on"` → bool; enum/string fields pass through as-is |
| SET-05 | Form state preservation on validation errors (re-render with submitted values, not defaults) | Pass `submittedValues map[string]string` (raw form strings) into the Templ form component; form renders `value=` from submittedValues, falling back to saved settings, then schema defaults |
| SET-06 | Plugin status info on settings page (last run, next run, error count) | Query `plugin_runs` for this user+plugin; compute next run via existing `computeNextRun`; recent errors = last 3-5 failed runs with `error_message` and `completed_at` |
</phase_requirements>

---

## Summary

Phase 12 builds a settings page where each plugin appears as a collapsible accordion row. The primary technical challenge is the dynamic form generation: the handler reads each plugin's `settings.schema.json` file, parses it with kaptinlin/jsonschema v0.6.15, and the Templ component iterates over `schema.Properties` to render appropriate field types. No form fields are hardcoded — the schema drives everything.

The existing `ValidateUserSettings` function in `internal/plugins/validator.go` is the right foundation but returns only a single combined error string. For the settings UI it must be enhanced (or a new function written) to return per-field errors from `EvaluationResult.DetailedErrors()`. HTML forms always submit strings, so type coercion must happen before validation: integer fields need `strconv.Atoi`, boolean fields need string-to-bool conversion. This coercion must be applied consistently or validation will fail with spurious type errors.

The research flag in the roadmap — "extension field preservation (x-component, x-placeholder)" — is now resolved: kaptinlin/jsonschema v0.6.15 has a `PreserveExtra bool` field on `Compiler`. Setting `compiler.SetPreserveExtra(true)` causes unknown keywords (like `x-component`, `x-placeholder`) to be stored in `schema.Extra map[string]any`. Without this flag, extra fields are silently stripped. However, the current `daily-news-digest` schema does not use any `x-` extension fields, so the form generator should work fine from standard fields alone. Extension support can be added as a progressive enhancement.

**Primary recommendation:** Build `internal/settings/` package with handlers, viewmodel, and schema-to-form logic. Create `internal/templates/settings.templ` for the accordion page and form component. Wire `/settings` route in `main.go`. The settings form component receives a pre-built `[]FieldViewModel` slice (not the raw `*Schema`) so Templ stays free of Go-json dependencies.

---

## Standard Stack

### Core (already in project)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/kaptinlin/jsonschema` | v0.6.15 | Parse schema, validate settings | Already in `go.mod`; Google-backed JSON Schema Draft 2020-12 |
| `github.com/a-h/templ` | v0.3.977 | Server-side HTML templates | Project standard; compile-safe |
| `github.com/gin-gonic/gin` | v1.11.0 | HTTP handler wiring | Project standard |
| `gorm.io/gorm` | v1.31.1 | DB queries for UserPluginConfig and PluginRun | Project standard |
| HTMX 2.0 | CDN | Accordion expand, toggle, blur validation, form submit | Project standard |

### Supporting (already in project)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/robfig/cron/v3` | v3.0.1 | Compute next run time from cron expression | Already used in `dashboard/viewmodel.go`; reuse `computeNextRun` logic |
| `strconv` | stdlib | Integer coercion from form strings | For `type: integer` fields |
| `sort` | stdlib | Deterministic property order for form rendering | SchemaMap iteration order is not deterministic |

### No New Dependencies Needed

All required libraries are already in `go.mod`. No new packages to install.

---

## Architecture Patterns

### Recommended Project Structure

```
internal/settings/
├── handlers.go       # Gin handlers: GET /settings, POST /api/settings/:pluginID/toggle,
│                     # POST /api/settings/:pluginID/save, POST /api/settings/:pluginID/validate-field,
│                     # POST /api/settings/:pluginID/run-now
└── viewmodel.go      # PluginSettingsViewModel, FieldViewModel, StatusViewModel, schema parsing

internal/templates/
├── settings.templ        # SettingsPage, PluginAccordionRow, PluginSettingsForm
└── settings_templ.go     # Generated (make templ-generate)
```

### Pattern 1: Schema-to-FieldViewModel Conversion

**What:** Parse the JSON Schema file and convert to a flat `[]FieldViewModel` slice before passing to Templ. Templ never imports kaptinlin/jsonschema — it receives a clean view type.

**When to use:** Always. Keeps templates free of business logic and avoids import cycles.

```go
// Source: kaptinlin/jsonschema@v0.6.15/schema.go + compiler.go (direct inspection)

// FieldType classifies a schema property into a renderable form field type.
type FieldType string
const (
    FieldTypeText     FieldType = "text"     // string, no enum
    FieldTypeEnum     FieldType = "enum"     // string/integer with enum values
    FieldTypeInteger  FieldType = "integer"  // type: integer
    FieldTypeBoolean  FieldType = "boolean"  // type: boolean
    FieldTypeCheckboxGroup FieldType = "checkbox_group" // array of string enum items
)

// FieldViewModel is what Templ receives — no jsonschema types.
type FieldViewModel struct {
    Key         string    // JSON property key (e.g. "frequency")
    Label       string    // Display label: schema.Title if set, else humanize(key)
    Description string    // Help text for tooltip: schema.Description
    FieldType   FieldType // Determines which input to render
    Required    bool      // from schema.Required list
    Default     string    // schema.Default formatted as string
    EnumValues  []string  // for FieldTypeEnum and FieldTypeCheckboxGroup
    // Current value: set from saved settings or submitted form values
    CurrentValue string   // string form of saved/submitted value
    Error        string   // inline error message (empty = no error)
}

// schemaToFields converts a parsed *jsonschema.Schema to []FieldViewModel.
// sortedKeys ensures deterministic form field ordering.
func schemaToFields(schema *jsonschema.Schema, savedSettings map[string]any,
    submittedValues map[string]string, fieldErrors map[string]string) []FieldViewModel {

    if schema.Properties == nil {
        return nil
    }

    // Build required set
    requiredSet := make(map[string]bool)
    for _, r := range schema.Required {
        requiredSet[r] = true
    }

    // Sort keys for deterministic ordering
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

        // Value priority: submittedValues > savedSettings > default
        if v, ok := submittedValues[key]; ok {
            field.CurrentValue = v
        } else if v, ok := savedSettings[key]; ok {
            field.CurrentValue = fmt.Sprintf("%v", v)
        } else {
            field.CurrentValue = field.Default
        }

        // Populate enum values for dropdowns / checkbox groups
        if propSchema.Enum != nil {
            for _, e := range propSchema.Enum {
                field.EnumValues = append(field.EnumValues, fmt.Sprintf("%v", e))
            }
        }
        if propSchema.Items != nil && propSchema.Items.Enum != nil {
            // array of enum items (topics field pattern)
            for _, e := range propSchema.Items.Enum {
                field.EnumValues = append(field.EnumValues, fmt.Sprintf("%v", e))
            }
        }

        fields = append(fields, field)
    }
    return fields
}
```

### Pattern 2: Type Coercion Before Validation (SET-04)

**What:** HTML forms always submit strings. Coerce to the type the schema declares before calling `schema.Validate()`.

**When to use:** In the save handler, always. Build a typed `map[string]any` from raw `url.Values`.

```go
// Source: Direct analysis of kaptinlin/jsonschema@v0.6.15 type.go — validates Go types,
// not JSON strings. HTML form strings will fail type validation without coercion.

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
            // HTML checkboxes send "on" or are absent; toggles may send "true"/"false"
            val := false
            if len(rawVals) > 0 {
                raw := strings.ToLower(rawVals[0])
                val = raw == "true" || raw == "on" || raw == "1"
            }
            result[key] = val
        case "array":
            // Multi-select checkboxes: form sends multiple values for same key
            result[key] = rawVals // []string satisfies JSON Schema array validation
        default:
            // string, enum string
            if len(rawVals) > 0 {
                result[key] = rawVals[0]
            }
        }
    }
    return result, nil
}
```

### Pattern 3: Per-Field Error Extraction (SET-03)

**What:** Get a `map[string]string` keyed by field name from `EvaluationResult`.

**When to use:** In save handler and blur-validate handler.

```go
// Source: kaptinlin/jsonschema@v0.6.15/result.go - DetailedErrors() implementation

func validateAndGetFieldErrors(schema *jsonschema.Schema, typedValues map[string]any) map[string]string {
    result := schema.Validate(typedValues)
    if result.IsValid() {
        return nil
    }
    // DetailedErrors returns map[string]string keyed by JSON pointer path e.g. "/frequency"
    // strip leading "/" to match property keys: strings.TrimPrefix(path, "/")
    raw := result.DetailedErrors()
    errors := make(map[string]string, len(raw))
    for path, msg := range raw {
        key := strings.TrimPrefix(path, "/")
        errors[path] = msg       // keep original path for schemaToFields lookup ("/"+key)
        _ = key
    }
    return raw
}
```

### Pattern 4: Extension Field Preservation (Research Flag Resolution)

**What:** `compiler.SetPreserveExtra(true)` causes `x-component`, `x-placeholder`, or any non-standard keywords to be stored in `schema.Properties["fieldName"].Extra map[string]any`.

**When to use:** Only needed if `x-` extension fields are added to schema files in the future. Current `daily-news-digest` schema has no `x-` fields. Build the schema parser to read from `Extra` if present, but do not require it.

```go
// Source: kaptinlin/jsonschema@v0.6.15/compiler.go:39-41, schema.go:191-192

// To enable extension fields:
compiler := jsonschema.NewCompiler()
compiler.SetPreserveExtra(true)  // x-component, x-placeholder land in schema.Extra
schema, err := compiler.Compile(schemaData)

// Access in FieldViewModel builder:
if extra, ok := propSchema.Extra["x-placeholder"].(string); ok {
    field.Placeholder = extra
}
```

**Without `SetPreserveExtra(true)`** (the default): extension fields are silently stripped during `compiler.Compile()`. The current codebase uses `jsonschema.NewCompiler()` without this flag — extension fields are not preserved. This is fine for the current schema.

### Pattern 5: HTMX Accordion + Toggle Pattern

**What:** Enable/disable toggle fires immediately. Accordion expands on enable (if required fields exist) or on row click. Blur validation fires per-field HTMX requests.

**When to use:** Settings page interactions.

```html
<!-- Collapsed plugin row: clicking expands, toggle fires independently -->
<div class="settings-plugin-row" id="plugin-row-{id}">
    <!-- Toggle: hx-post fires instantly, swaps just the row -->
    <input type="checkbox"
           hx-post="/api/settings/{pluginID}/toggle"
           hx-target="#plugin-row-{id}"
           hx-swap="outerHTML"
           checked?={enabled}/>
    <span class="settings-status-dot settings-status-{healthColor}"></span>
    <span class="settings-plugin-name">{pluginName}</span>
    <!-- Expand arrow (client-side JS toggle of .settings-expanded class) -->
</div>

<!-- Expanded form with HTMX blur validation -->
<div class="settings-plugin-expanded" id="plugin-expanded-{id}">
    <!-- Per-field blur validation -->
    <input name="frequency"
           hx-post="/api/settings/{pluginID}/validate-field"
           hx-trigger="blur"
           hx-include="[name='frequency']"
           hx-target="#error-frequency-{pluginID}"
           hx-swap="innerHTML"/>
    <div id="error-frequency-{pluginID}" class="settings-field-error"></div>

    <!-- Full form save -->
    <button hx-post="/api/settings/{pluginID}/save"
            hx-include="closest form"
            hx-target="#plugin-expanded-{id}"
            hx-swap="outerHTML">Save</button>
</div>
```

**HTMX trigger note (verified):** `hx-trigger="blur"` fires an HTMX request when the field loses focus. The `changed` modifier (`hx-trigger="blur changed"`) only fires if the value was actually changed. Both work in HTMX 2.0.

### Pattern 6: "Saved ✓" Transient Confirmation

**What:** The save handler responds with a success fragment where the button shows "Saved ✓". A `hx-on::after-request` or CSS animation resets it after 2 seconds.

**Approach:** Server returns the full expanded accordion fragment with `data-saved="true"` attribute on the button. JavaScript detects this attribute and reverts the button text after a timeout:

```javascript
// In settings.templ <script> block
htmx.on('htmx:afterSwap', function(e) {
    var btn = e.target.querySelector('[data-saved="true"]');
    if (btn) {
        setTimeout(function() {
            btn.textContent = 'Save';
            btn.removeAttribute('data-saved');
        }, 2000);
    }
});
```

### Pattern 7: Schema File Path Resolution

**What:** `plugin.settings_schema_path` stored in DB is a filename relative to the plugin subdirectory (e.g., `"settings.schema.json"`). The full path is: `filepath.Join(cfg.PluginDir, plugin.Name, plugin.SettingsSchemaPath)`.

**When to use:** In any handler that needs to read or compile the schema.

```go
// Source: plugins/plugin.yaml: settings_schema_path: settings.schema.json
// plugins/loader.go:77: SettingsSchemaPath: meta.SettingsSchemaPath (stored as-is)
// config.go:44: PluginDir defaults to "./plugins"

schemaFullPath := filepath.Join(cfg.PluginDir, plugin.Name, plugin.SettingsSchemaPath)
schemaData, err := os.ReadFile(schemaFullPath)
```

The settings handler needs access to `cfg.PluginDir`. Pass it alongside `db` and `registry` in the handler constructor.

### Pattern 8: Existing Infrastructure to Reuse

**What:** Several functions from existing packages can be reused directly.

| Need | Existing Source | Function/Field |
|------|----------------|----------------|
| Next run time | `internal/dashboard/viewmodel.go` | `computeNextRun(cronExpr, timezone)` — consider moving to `internal/plugins` or duplicating |
| Cron validation | `internal/plugins/models.go` | `ValidateCronExpression(expr)` |
| Plugin DB model | `internal/plugins/models.go` | `Plugin`, `UserPluginConfig`, `PluginRun` structs |
| Plugin registry | `internal/plugins/registry.go` | `registry.List()`, `registry.Get(name)` |
| Auth user lookup | `internal/dashboard/handlers.go` | `getAuthUser(c, db)` — duplicate or move to shared `internal/auth` |
| Render helper | any handler package | `render(c, component templ.Component)` — local to each package by convention |

### Anti-Patterns to Avoid

- **Passing `*jsonschema.Schema` into Templ:** Templ would need to import kaptinlin/jsonschema; causes coupling. Always convert to `[]FieldViewModel` first.
- **Using `result.Errors` map directly for per-field display:** The `Errors` map on `EvaluationResult` is keyed by keyword (e.g., `"minLength"`, `"type"`), not field name. Use `result.DetailedErrors()` which returns paths like `"/frequency"`.
- **Iterating `*SchemaMap` without sorting:** Map iteration order in Go is random. Sort keys before building `[]FieldViewModel` to ensure deterministic form field order.
- **Storing schema path as absolute path in DB:** The `settings_schema_path` in DB is relative (`"settings.schema.json"`). Always join with `cfg.PluginDir + plugin.Name`.
- **Assuming all properties have a `type` field:** Some schemas use `anyOf`, `oneOf`, or reference types. The current `daily-news-digest` schema uses explicit `type` on every property — safe to assume for now, but guard against nil `propSchema.Type`.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSON Schema validation | Custom struct validation | `schema.Validate(map[string]any)` | Edge cases: required, pattern, enum, minItems all handled |
| Per-field error paths | String parsing of combined error | `result.DetailedErrors()` | Returns JSON pointer paths cleanly |
| Schema file parsing | Manual JSON unmarshaling | `compiler.Compile(schemaData)` | Handles `$ref`, defaults, nested schemas |
| Next run computation | Custom cron parser | `cronParser.Parse(expr).Next(now)` | Already in `dashboard/viewmodel.go` using `robfig/cron/v3` |

**Key insight:** The kaptinlin/jsonschema library handles all schema-driven validation edge cases. The custom work is only the form-generation layer (converting `*Schema` to `[]FieldViewModel`) and type coercion (HTML strings → Go types).

---

## Common Pitfalls

### Pitfall 1: Extension Fields Stripped by Default
**What goes wrong:** Adding `x-placeholder: "Enter topic..."` to a schema file and then accessing `propSchema.Extra["x-placeholder"]` returns nil.
**Why it happens:** `compiler.PreserveExtra` defaults to `false`; extra fields are stripped during `compiler.Compile()`.
**How to avoid:** Use `compiler.SetPreserveExtra(true)` when compiling schemas that contain `x-` extension fields. Document this in the schema loading function.
**Warning signs:** `propSchema.Extra` is always nil, even after adding `x-` fields to schema files.

### Pitfall 2: EvaluationResult.Errors Keyed by Keyword, Not Field Name
**What goes wrong:** Iterating `result.Errors` and using the key as a field name results in displaying errors under keys like `"minLength"` or `"type"` instead of `"frequency"` or `"preferred_time"`.
**Why it happens:** `result.Errors` at the top level holds errors keyed by JSON Schema keyword. Per-field errors are in `result.Details[i].Errors`.
**How to avoid:** Always use `result.DetailedErrors()` which recursively traverses `Details` and returns paths like `"/frequency"`. Strip the leading `/` to get the field key.
**Warning signs:** Errors appear under validation keyword names, not field names.

### Pitfall 3: Boolean Field Missing from Form on Validation Failure
**What goes wrong:** User unchecks a boolean toggle, submits the form, validation fails on another field, re-rendered form shows the boolean as checked (true) again.
**Why it happens:** HTML checkboxes that are unchecked are NOT submitted in the form body. `c.PostForm("enabled")` returns `""` for an unchecked checkbox, which the coercion layer may interpret as "not provided" and fall back to the saved value (true).
**How to avoid:** For boolean fields, use a hidden input with `value="false"` before the checkbox (`name="enabled"`). The checkbox's `value="true"` overrides it when checked. Alternatively, accept absence of the key as `false` in coercion. Document the convention.
**Warning signs:** Boolean fields reset to their saved/default value on validation error re-render.

### Pitfall 4: Array Field (topics) Multi-Value Form Submission
**What goes wrong:** User selects multiple topics checkboxes; handler calls `c.PostForm("topics")` and gets only the first value.
**Why it happens:** `c.PostForm()` returns only the first value for repeated form keys.
**How to avoid:** Use `c.PostFormArray("topics")` to get all values. In coercion, treat `type: array` fields by calling `rawForm["topics"]` (returns `[]string`).
**Warning signs:** Only one topic is saved when user selects multiple.

### Pitfall 5: Schema File Path Resolves to Wrong Location
**What goes wrong:** `os.ReadFile(plugin.SettingsSchemaPath)` fails because `SettingsSchemaPath` is `"settings.schema.json"` (relative) not an absolute path.
**Why it happens:** The path stored in DB is relative to the plugin directory, as set in `plugin.yaml`.
**How to avoid:** Always construct: `filepath.Join(cfg.PluginDir, plugin.Name, plugin.SettingsSchemaPath)`. Pass `cfg.PluginDir` into the settings handler constructor.
**Warning signs:** `no such file or directory` errors on settings page load.

### Pitfall 6: Determinism in Form Field Ordering
**What goes wrong:** Form fields appear in a different order on each page load, confusing users.
**Why it happens:** `*jsonschema.SchemaMap` is `map[string]*Schema`; Go map iteration is random.
**How to avoid:** Extract keys from `*schema.Properties`, sort them with `sort.Strings(keys)`, then build `[]FieldViewModel` in sorted order.
**Warning signs:** Field order changes between page loads.

---

## Code Examples

### Load and Compile Schema for a Plugin

```go
// Source: internal/plugins/validator.go (existing), kaptinlin/jsonschema compiler.go

func loadPluginSchema(pluginDir, pluginName, schemaRelPath string) (*jsonschema.Schema, error) {
    if schemaRelPath == "" {
        return nil, nil // plugin has no settings schema
    }
    fullPath := filepath.Join(pluginDir, pluginName, schemaRelPath)
    data, err := os.ReadFile(fullPath)
    if err != nil {
        return nil, fmt.Errorf("read schema %s: %w", fullPath, err)
    }
    compiler := jsonschema.NewCompiler()
    // SetPreserveExtra(true) if x- extension fields are needed in future
    schema, err := compiler.Compile(data)
    if err != nil {
        return nil, fmt.Errorf("compile schema %s: %w", fullPath, err)
    }
    return schema, nil
}
```

### Enhanced Validation Returning Per-Field Errors

```go
// Source: kaptinlin/jsonschema@v0.6.15/result.go DetailedErrors()

// ValidateSettingsWithFieldErrors validates settings and returns a map of
// field-path → error message. Returns nil map when valid.
func ValidateSettingsWithFieldErrors(schema *jsonschema.Schema, typedValues map[string]any) map[string]string {
    result := schema.Validate(typedValues)
    if result.IsValid() {
        return nil
    }
    return result.DetailedErrors() // keys: "/frequency", "/preferred_time", etc.
}
```

### Query Plugin Status for Settings Display (SET-06)

```go
// Source: plugin_runs DB table, internal/plugins/models.go PluginRun struct

type PluginStatusViewModel struct {
    LastRunAt    *time.Time
    NextRunAt    *time.Time
    RecentErrors []ErrorEntry
    HealthColor  string // "green", "yellow", "red"
}

type ErrorEntry struct {
    OccurredAt time.Time
    Message    string
}

func getPluginStatus(db *gorm.DB, userID, pluginID uint, cronExpr, timezone string) PluginStatusViewModel {
    // Latest run
    var latestRun plugins.PluginRun
    db.Where("user_id = ? AND plugin_id = ? AND deleted_at IS NULL", userID, pluginID).
        Order("created_at DESC").First(&latestRun)

    // Recent errors (last 5 failed runs)
    var failedRuns []plugins.PluginRun
    db.Where("user_id = ? AND plugin_id = ? AND status = 'failed' AND deleted_at IS NULL",
        userID, pluginID).
        Order("created_at DESC").Limit(5).Find(&failedRuns)

    errors := make([]ErrorEntry, 0, len(failedRuns))
    for _, r := range failedRuns {
        errors = append(errors, ErrorEntry{
            OccurredAt: r.CreatedAt,
            Message:    r.ErrorMessage,
        })
    }

    health := "green"
    if len(failedRuns) > 0 && latestRun.Status == "failed" {
        health = "red"
    } else if len(failedRuns) > 0 {
        health = "yellow"
    }

    return PluginStatusViewModel{
        LastRunAt:    latestRun.CompletedAt,
        NextRunAt:    computeNextRun(cronExpr, timezone),
        RecentErrors: errors,
        HealthColor:  health,
    }
}
```

### Label Humanization Strategy (Claude's Discretion)

The `daily-news-digest` schema uses `"description"` on every property but does NOT set `"title"` on individual properties (only on the root schema). The best label source is therefore:
1. `propSchema.Title` if non-nil and non-empty (future-proof)
2. Humanize the key: replace `_` with space, title-case: `"preferred_time"` → `"Preferred Time"`

```go
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
```

### Boolean Field Rendering (Claude's Discretion)

Use a CSS toggle switch styled with `glass-` design tokens. A toggle fits the glass aesthetic better than a plain checkbox. Implement with a `<label>` + hidden `<input type="checkbox">` + CSS `::before/::after` pseudo-elements. The hidden `value="false"` pattern prevents the checkbox-absent-means-unchanged pitfall.

```html
<!-- Boolean field pattern: hidden false + visible checkbox -->
<input type="hidden" name="{key}" value="false"/>
<label class="settings-toggle">
    <input type="checkbox" name="{key}" value="true" checked?={currentValue == "true"}/>
    <span class="settings-toggle-slider"></span>
</label>
```

### Enum Field Rendering (Claude's Discretion)

- **≤4 options:** Radio buttons — users can see all options at once
- **>4 options or array type (topics):** Checkbox group for `type: array`, dropdown (`<select>`) for `type: string`

The `daily-news-digest` schema has:
- `frequency`: 2 enum options → radio buttons
- `summary_length`: 3 enum options → radio buttons
- `topics`: array of up to 7 options → checkbox group

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `result.Errors` for per-field errors | `result.DetailedErrors()` → `map[string]string` | kaptinlin/jsonschema added `DetailedErrors` | Use `DetailedErrors()`; `result.Errors` is keyword-keyed not field-keyed |
| Hand-rolled form code per plugin | Schema-driven form generation | Phase 12 (new) | No hardcoded field definitions |

**Deprecated/outdated:**
- `ValidateUserSettings` in `internal/plugins/validator.go`: returns a single combined error string. It is fine for background job validation (worker rejects bad settings) but insufficient for the settings UI which needs per-field errors. The settings handler should call `schema.Validate()` directly and use `result.DetailedErrors()`.

---

## Open Questions

1. **Where does `computeNextRun` live?**
   - What we know: It is defined in `internal/dashboard/viewmodel.go` as an unexported function.
   - What's unclear: Should it be duplicated in `internal/settings/viewmodel.go` or moved to `internal/plugins`?
   - Recommendation: Duplicate it in `internal/settings/viewmodel.go` for now (same pattern as `getAuthUser` being duplicated). Moving it to `internal/plugins` in a future refactor is cleaner but out of scope for Phase 12.

2. **"Run Now" button: what does it trigger?**
   - What we know: `worker.EnqueueExecutePlugin(pluginID, userID, pluginName, settings)` exists in `internal/worker/tasks.go`. The button should call it.
   - What's unclear: Should "Run Now" require that settings are saved first? The user may have unsaved changes.
   - Recommendation: "Run Now" uses the currently-saved settings from DB (not any unsaved form state). Document this in the UI tooltip. The button fires `POST /api/settings/:pluginID/run-now` which reads `UserPluginConfig.Settings` from DB and enqueues the task.

3. **Schedule fields (cron_expression, timezone) in the settings form?**
   - What we know: `UserPluginConfig` has `CronExpression` and `Timezone` columns. These are not part of the JSON Schema (plugin-specific settings), but are user-level scheduling preferences.
   - What's unclear: Should the settings form include cron/timezone inputs separate from the schema-driven fields?
   - Recommendation: Yes, include a "Schedule" section in the expanded accordion with a cron expression text input and a timezone dropdown. Validate cron with `plugins.ValidateCronExpression()`. Keep this distinct from the plugin-specific settings form (saved to a different endpoint or the same `POST /api/settings/:pluginID/save` handling both).

---

## Sources

### Primary (HIGH confidence)

- `github.com/kaptinlin/jsonschema@v0.6.15` — direct source inspection: `compiler.go`, `schema.go`, `result.go`, `examples/error-handling/main.go`
- `/Users/jim/git/jimdaga/first-sip/internal/plugins/` — existing validator, registry, models (direct file read)
- `/Users/jim/git/jimdaga/first-sip/internal/dashboard/` — viewmodel, handlers (direct file read)
- `/Users/jim/git/jimdaga/first-sip/plugins/daily-news-digest/settings.schema.json` — actual schema file (direct read)
- `/Users/jim/git/jimdaga/first-sip/go.mod` — confirmed all required dependencies present
- HTMX 2.0 `hx-trigger="blur"` — verified via WebSearch against official HTMX docs

### Secondary (MEDIUM confidence)

- WebSearch: HTMX 2.0 form validation blur trigger patterns — confirmed `hx-trigger="blur"` and `hx-trigger="blur changed"` are valid HTMX 2.0 triggers

### Tertiary (LOW confidence)

None.

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries directly inspected in module cache and source files
- Architecture: HIGH — patterns derived from existing project conventions (dashboard pattern) and verified library API
- Pitfalls: HIGH — pitfalls derived from direct source inspection of kaptinlin/jsonschema and HTML form behavior facts
- Extension field preservation: HIGH — `PreserveExtra bool` field confirmed in `compiler.go:39-41`, `schema.go:275-279`

**Research date:** 2026-02-22
**Valid until:** 2026-03-22 (stable library versions, 30-day window)
