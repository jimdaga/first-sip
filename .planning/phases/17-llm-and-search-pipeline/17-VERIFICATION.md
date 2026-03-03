---
phase: 17-llm-and-search-pipeline
verified: 2026-03-03T15:54:04Z
status: passed
score: 6/6 must-haves verified
re_verification:
  previous_status: gaps_found
  previous_score: 5/6
  gaps_closed:
    - "User can override the LLM model per plugin via plugin settings (LLM-03)"
  gaps_remaining: []
  regressions: []
human_verification:
  - test: "End-to-end LLM call with real API key"
    expected: "LLM constructor receives correct model and api_key; sidecar completes a real CrewAI run without API key errors"
    why_human: "Cannot test live API key flows programmatically in static verification"
  - test: "Tavily search invocation"
    expected: "TavilySearchTool is used; researcher task output references real web results"
    why_human: "Requires live Tavily credentials and network access"
  - test: "DuckDuckGo fallback invocation"
    expected: "DuckDuckGoSearchRun is used; no Tavily-related error; researcher returns web results"
    why_human: "Requires live execution to confirm the @tool wrapper integrates correctly with CrewAI tool dispatch"
  - test: "Per-plugin LLM override UI rendering"
    expected: "_llm_model field renders as a <select> dropdown in the daily-news-digest plugin settings panel with all 11 options"
    why_human: "Rendering depends on live templ compilation and browser rendering"
  - test: "Per-plugin LLM override save and use"
    expected: "Selecting openai/gpt-4o in plugin settings, saving, and triggering a run causes the sidecar to use that model instead of the account default"
    why_human: "Requires live form submission, DB write, worker execution, and sidecar verification"
---

# Phase 17: LLM and Search Pipeline Verification Report

**Phase Goal:** API keys flow through Redis Streams to the CrewAI sidecar, enabling live LLM calls and web search
**Verified:** 2026-03-03T15:54:04Z
**Status:** passed
**Re-verification:** Yes — after LLM-03 gap closure (Plan 03, commit 461a0c5)

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Redis Streams job payload carries the user's LLM API key and provider/model string | VERIFIED | `worker.go:246` calls `apikeys.GetKeysForUser`; lines 254-259 inject `_llm_api_key` and `_llm_model` into `payload.Settings`; line 258 guard preserves per-plugin override if set |
| 2 | CrewAI sidecar accepts LLM config and initializes the crew with the specified provider/model via LiteLLM | VERIFIED | `executor.py:40-56` `_extract_llm` builds `crewai.LLM(model=model, api_key=api_key)` with Anthropic `max_tokens=4096` guard; `_load_crew` passes `llm=` to `module.create_crew`; all three agents in `crew.py` receive `llm=llm` |
| 3 | User can override the LLM model per plugin via plugin settings | VERIFIED | `settings.schema.json:36-41` defines `_llm_model` as optional string enum with 11 values; `viewmodel.go:614-616` maps it to `FieldTypeEnum`; `renderField` in `settings.templ:639-644` renders it as `<select>` (11 > 4 threshold); `SaveSettingsHandler` in `handlers.go:284-327` persists all schema fields to `UserPluginConfig.Settings` JSONB; `worker.go:258` guard reads it back |
| 4 | When user has a Tavily key, the researcher agent searches via Tavily | VERIFIED | `executor.py:67-70` checks `_tavily_api_key`, sets `TAVILY_API_KEY` env var, returns `TavilySearchTool()`; passed as `search_tool` to crew; researcher receives `tools=[search_tool]` at `crew.py:30` |
| 5 | When user has no Tavily key, the researcher agent falls back to DuckDuckGo | VERIFIED | `executor.py:73-80` creates `DuckDuckGoSearchRun()` wrapped with `@tool("Web Search")` decorator as fallback; same `tools=[search_tool]` path in crew |
| 6 | Search queries use the user's topic preferences from plugin settings | VERIFIED | `_clean_settings` strips `_`-prefixed credential keys at `executor.py:82-94`; clean settings passed to `crew.kickoff_async(inputs=clean_settings)` at line 122; `{topics}` placeholder in researcher goal (`crew.py:24`), backstory (`crew.py:26-27`), and research task description (`crew.py:61`) |

**Score:** 6/6 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `plugins/daily-news-digest/settings.schema.json` | `_llm_model` field with enum of all supported provider/model combos | VERIFIED | Lines 36-41: type string, 11 enum values (`""` + 10 LiteLLM combos), default `""`, NOT in `required` array |
| `.planning/REQUIREMENTS.md` | All six Phase 17 requirements checked | VERIFIED | All six `[x] **LLM-01**` through `[x] **SRCH-03**` confirmed; traceability table shows "Done" for all |
| `internal/worker/worker.go` | Key injection logic in `handleExecutePlugin` | VERIFIED (regression) | `findAPIKey` at line 200; injection block at lines 231-268; `_llm_api_key`, `_llm_model`, `_tavily_api_key` all injected |
| `sidecar/executor.py` | LLM construction, search tool selection, settings sanitization | VERIFIED (regression) | `_extract_llm` (line 40), `_extract_search_tool` (line 58), `_clean_settings` (line 82) all present and called in `execute()` |
| `sidecar/pyproject.toml` | Python dependencies for search tools | VERIFIED (regression) | `crewai-tools>=0.55.0`, `tavily-python`, `langchain-community`, `duckduckgo-search>=4.1.0` |
| `plugins/daily-news-digest/crew/crew.py` | Imperative crew factory with `llm=` and `search_tool=` parameters | VERIFIED (regression) | `def create_crew(settings, llm=None, search_tool=None)` at line 9; `llm=llm` on all three agents |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `plugins/daily-news-digest/settings.schema.json` | `internal/settings/viewmodel.go` | `loadPluginSchema` → `schemaToFields` → `fieldTypeFromSchema` detects enum, returns `FieldTypeEnum` | WIRED | `viewmodel.go:353-426` iterates all schema properties including `_llm_model`; `fieldTypeFromSchema` at line 614-616 matches `len(s.Enum) > 0` for string type |
| `internal/settings/handlers.go` | `internal/worker/worker.go` | `SaveSettingsHandler` iterates `schema.Properties` (line 284), calls `coerceFormValues`, marshals to JSON, persists to `UserPluginConfig.Settings`; worker reads `payload.Settings["_llm_model"]` at line 258 | WIRED | `handlers.go:284-327` iterates schema including `_llm_model` key; coercion treats it as `string` (default case line 486-490); marshaled JSON includes it in JSONB |
| `internal/worker/worker.go` | `internal/apikeys/service.go` | `apikeys.GetKeysForUser(db, payload.UserID)` | WIRED (regression) | Line 246 confirmed |
| `sidecar/executor.py` | `plugins/daily-news-digest/crew/crew.py` | `module.create_crew(settings, llm=llm, search_tool=search_tool)` | WIRED (regression) | Line 253 confirmed |
| `plugins/daily-news-digest/crew/crew.py` | `crewai.Agent` | `llm=llm` parameter on every Agent constructor | WIRED (regression) | Lines 22-57 confirmed |

### Requirements Coverage

| Requirement | Description | Status | Blocking Issue |
|-------------|-------------|--------|----------------|
| LLM-01 | System passes user's LLM API key to CrewAI sidecar per run | SATISFIED | — |
| LLM-02 | CrewAI crew uses provider-agnostic LLM via LiteLLM format (provider/model) | SATISFIED | — |
| LLM-03 | User can override LLM model per plugin via plugin settings | SATISFIED | Gap closed: `_llm_model` field in schema, schema-driven dropdown rendered, save handler persists, worker reads |
| SRCH-01 | CrewAI researcher agent uses Tavily search when user has Tavily key | SATISFIED | — |
| SRCH-02 | CrewAI researcher agent falls back to DuckDuckGo when no Tavily key | SATISFIED | — |
| SRCH-03 | Search queries incorporate user's topic preferences from plugin settings | SATISFIED | — |

All six requirements confirmed checked (`[x]`) in REQUIREMENTS.md. Traceability table updated.

### Anti-Patterns Found

| File | Pattern | Severity | Impact |
|------|---------|----------|--------|
| `internal/worker/worker.go:255` | Comment "AfterFind hook has already decrypted" on `EncryptedValue` field — field name is misleading | Info | No functional impact; naming inherited from model |

No TODO/FIXME/placeholder comments in any phase files. No empty implementations or stub returns. No blocker anti-patterns.

### Human Verification Required

#### 1. Per-plugin LLM override UI rendering

**Test:** Open the daily-news-digest plugin settings panel in a browser.
**Expected:** The `_llm_model` field renders as a `<select>` dropdown with 11 options: an empty first option ("") and all 10 provider/model combos in LiteLLM format.
**Why human:** Rendering requires live templ compilation and browser rendering.

#### 2. Per-plugin LLM override save and use

**Test:** Select `openai/gpt-4o` in the plugin settings dropdown, save settings, trigger a run.
**Expected:** The sidecar uses `openai/gpt-4o` instead of the user's account-level default model.
**Why human:** Requires live form submission, DB write, worker execution, and sidecar log inspection.

#### 3. End-to-end LLM call with real API key

**Test:** Configure a real OpenAI or Anthropic API key in user settings, trigger a daily-news-digest run, observe sidecar logs.
**Expected:** LLM constructor receives correct `model` and `api_key`; sidecar completes a real CrewAI run without API key errors.
**Why human:** Cannot test live API key flows programmatically in static verification.

#### 4. Tavily search invocation

**Test:** Set a real Tavily API key in user settings, trigger a run, observe sidecar logs.
**Expected:** `TavilySearchTool` is used; researcher task output references real web results.
**Why human:** Requires live Tavily credentials and network access.

#### 5. DuckDuckGo fallback invocation

**Test:** Remove the Tavily key from user settings, trigger a run.
**Expected:** `DuckDuckGoSearchRun` is used; no Tavily-related error; researcher returns web results.
**Why human:** Requires live execution to confirm the `@tool` wrapper integrates correctly with CrewAI tool dispatch.

### Gap Closure Summary

The single gap from the initial verification (LLM-03) has been closed by adding a `_llm_model` optional enum field to `plugins/daily-news-digest/settings.schema.json` (commit `461a0c5`). The field has:
- 11 enum values: empty string (use account default) + 10 LiteLLM provider/model combos
- Default `""` matching the worker guard condition `!ok || existing == ""`
- Not in the `required` array — optional override

The schema-driven pipeline handles the new field with zero code changes:
1. `schemaToFields` in `viewmodel.go` converts it to `FieldTypeEnum` (11 values > 4 threshold)
2. `renderField` in `settings.templ` renders it as a `<select>` dropdown
3. `SaveSettingsHandler` in `handlers.go` coerces and persists it to `UserPluginConfig.Settings` JSONB
4. `worker.go:258` reads it back and uses it as the model override, falling back to the account default when empty

All six Phase 17 requirements (LLM-01 through SRCH-03) and all five Phase 16 requirements (KEYS-01 through KEYS-05) are now marked complete in REQUIREMENTS.md.

---

_Verified: 2026-03-03T15:54:04Z_
_Verifier: Claude (gsd-verifier)_
