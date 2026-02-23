---
status: diagnosed
trigger: "When saving settings with invalid values (e.g., bad cron expression), no validation error is shown to the user. The save appears to succeed silently or the form re-renders without any visible error messages."
created: 2026-02-23T00:00:00Z
updated: 2026-02-23T00:00:00Z
---

## Current Focus

hypothesis: CONFIRMED — two independent root causes found
test: full static trace of error flow from handler to template
expecting: n/a — diagnosis complete
next_action: return diagnosis

## Symptoms

expected: When user submits a bad cron expression (e.g. "not-a-cron"), a validation error should appear under the Cron Expression input field
actual: Save appears to succeed silently OR re-renders without visible error messages
errors: none thrown — silent failure
reproduction: Enter bad cron expression (e.g. "bad-cron"), submit form
started: unknown — present in current codebase

## Eliminated

- hypothesis: cronErr is never set
  evidence: handlers.go:170 — ValidateCronExpression is called and cronErr is set on error
  timestamp: 2026-02-23

- hypothesis: cronErr is never added to fieldErrors
  evidence: handlers.go:212-217 — cronErr IS added to fieldErrors["/cron_expression"] when schema != nil
  timestamp: 2026-02-23

- hypothesis: fieldErrors is never passed to BuildSinglePluginSettingsViewModel
  evidence: handlers.go:221 — fieldErrors IS passed on validation failure path
  timestamp: 2026-02-23

- hypothesis: schemaToFields never looks up the cron error
  evidence: viewmodel.go:351 — schemaToFields only iterates over schema.Properties keys; "cron_expression" is NOT a schema property, so fieldErrors["/cron_expression"] is NEVER read by this loop. But this is ROOT CAUSE #1, not eliminated.
  timestamp: 2026-02-23

## Evidence

- timestamp: 2026-02-23
  checked: handlers.go:167-173
  found: cronErr is correctly set when cron expression is invalid
  implication: the error value exists in memory

- timestamp: 2026-02-23
  checked: handlers.go:199-217
  found: |
    The code calls validateAndGetFieldErrors first (schema validation only),
    then conditionally appends cronErr to fieldErrors["/cron_expression"].
    The validation block at line 219 checks len(fieldErrors) > 0.
  implication: IF schema validation passes AND cronErr is set, the block at 219 WILL trigger

- timestamp: 2026-02-23
  checked: settings.schema.json:37
  found: schema has "required": ["frequency", "preferred_time", "topics"] — all three fields ARE required
  implication: if user submits valid schema field values alongside a bad cron, validateAndGetFieldErrors returns nil, then cronErr gets added; len(fieldErrors)==1; the error path IS taken. So far so good.

- timestamp: 2026-02-23
  checked: viewmodel.go:317-387 (schemaToFields)
  found: |
    schemaToFields iterates only over schema.Properties keys:
      for _, key := range keys { ... field.Error = fieldErrors["/"+key] }
    The keys come from schema.Properties which contains: frequency, preferred_time, topics, summary_length.
    "cron_expression" is NOT in schema.Properties — it is a schedule/UI field, not a JSON schema property.
    Therefore fieldErrors["/cron_expression"] is NEVER assigned to any FieldViewModel.Error.
  implication: ROOT CAUSE #1 — the cron error is passed in but never surfaced to any field in the Fields slice

- timestamp: 2026-02-23
  checked: settings.templ:326
  found: |
    The cron error div IS in the template:
      <div id={ fmt.Sprintf("error-cron_expression-%d", plugin.PluginID) } class="settings-field-error"></div>
    But it is ALWAYS rendered empty — there is no conditional check on a CronError field in PluginSettingsViewModel.
    PluginSettingsViewModel (settingsvm.go) has no CronError field at all.
  implication: ROOT CAUSE #2 — the template has a placeholder div for the cron error but the view model has no field to populate it

- timestamp: 2026-02-23
  checked: settingsvm/settingsvm.go:49-65 (PluginSettingsViewModel struct)
  found: no CronError string field exists in the struct
  implication: confirms ROOT CAUSE #2 — even if the handler wanted to pass a cron error to the template, there is no vehicle to do so

- timestamp: 2026-02-23
  checked: handlers.go:221 — BuildSinglePluginSettingsViewModel call signature
  found: fieldErrors is passed in, but schemaToFields only consumes entries keyed to schema properties. The "/cron_expression" entry in fieldErrors is effectively dropped on the floor.
  implication: confirms the full chain — the error is computed, stored in fieldErrors, passed to the VM builder, but then silently discarded inside schemaToFields

## Resolution

root_cause: |
  Two tightly coupled failures combine to produce the silent bug:

  ROOT CAUSE #1 (primary): schemaToFields() in viewmodel.go only iterates over
  schema.Properties keys. The key "cron_expression" is not a JSON schema property
  — it is a schedule/UI concern. Therefore fieldErrors["/cron_expression"] is
  never read inside the loop, and no FieldViewModel.Error is ever populated with
  the cron validation error. The error is computed and placed in fieldErrors but
  then silently discarded.

  ROOT CAUSE #2 (structural): PluginSettingsViewModel (settingsvm.go) has no
  CronError field. The template (settings.templ:326) has a static empty div
  for the cron error:
    <div id="error-cron_expression-{id}" class="settings-field-error"></div>
  but there is no data path connecting the server-side cronErr string to that div
  during a full form re-render. The div is only reachable via the inline
  hx-validate-field HTMX call (which fires on blur of individual fields), not
  during the whole-form save re-render.

fix: |
  (not applied — diagnose-only mode)

  To fix both root causes together:
  1. Add a CronError string field to PluginSettingsViewModel in settingsvm/settingsvm.go
  2. In handlers.go SaveSettingsHandler, populate vm.CronError = cronErr before
     calling render on the validation-failure path
  3. In settings.templ PluginSettingsForm, change the static empty cron error div to
     conditionally render the error:
       if plugin.CronError != "" {
         <div ... class="settings-field-error">{ plugin.CronError }</div>
       } else {
         <div ... class="settings-field-error"></div>
       }
  4. In BuildSinglePluginSettingsViewModel, accept and thread through a cronError
     string parameter (or extract it from fieldErrors["/cron_expression"] before
     passing fieldErrors to schemaToFields)

verification: not performed — diagnose-only mode
files_changed: []
