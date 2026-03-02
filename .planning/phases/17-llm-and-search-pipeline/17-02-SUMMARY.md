---
phase: 17-llm-and-search-pipeline
plan: "02"
subsystem: api
tags: [crewai, litellm, anthropic, tavily, duckduckgo, python, sidecar]

# Dependency graph
requires:
  - phase: 17-llm-and-search-pipeline-01
    provides: Worker injects _llm_api_key, _llm_model, _tavily_api_key into Redis Streams payload
  - phase: 16-api-key-management
    provides: Encrypted API key storage and retrieval from database

provides:
  - Sidecar executor constructs crewai.LLM from injected credentials
  - Anthropic model max_tokens=4096 handled automatically
  - Tavily or DuckDuckGo search tool selected based on key availability
  - Credential keys stripped from settings before crew inputs
  - daily-news-digest crew uses imperative pattern with explicit llm= on all agents

affects: [18-briefing-result-display, testing, sidecar, daily-news-digest]

# Tech tracking
tech-stack:
  added: [crewai-tools>=0.55.0, tavily-python, langchain-community, duckduckgo-search>=4.1.0]
  patterns:
    - "Credential extraction pattern: _-prefixed keys read then stripped before crew kickoff"
    - "Imperative Agent/Task/Crew constructors instead of @CrewBase YAML pattern for dynamic llm= injection"
    - "Search tool selection: Tavily when key present, DuckDuckGo @tool wrapper as zero-cost fallback"
    - "Anthropic models require max_tokens in LLM constructor — added automatically when model starts with anthropic/"

key-files:
  created: []
  modified:
    - sidecar/executor.py
    - sidecar/pyproject.toml
    - plugins/daily-news-digest/crew/crew.py

key-decisions:
  - "Imperative crew pattern over @CrewBase — YAML config decorators cannot accept dynamic llm= at runtime; imperative constructors pass llm= directly to each Agent"
  - "DuckDuckGo wrapped with @tool decorator to match TavilySearchTool interface — CrewAI tools list accepts both native tools and @tool-wrapped functions"
  - "Credential stripping via _clean_settings filters all _-prefixed keys — prevents API keys from appearing in agent prompt templates as {_llm_api_key}"
  - "Anthropic max_tokens=4096 applied when model string starts with anthropic/ — required by Anthropic API, other providers use default"

patterns-established:
  - "Plugin create_crew signature: create_crew(settings: dict, llm=None, search_tool=None) -> Crew"
  - "All agents in a crew receive explicit llm= parameter — never rely on CrewAI default (avoids silent OPENAI_API_KEY requirement)"
  - "Only task-relevant agents receive tools= (researcher gets search, writer/reviewer do not)"

requirements-completed: [LLM-02, SRCH-01, SRCH-02, SRCH-03]

# Metrics
duration: 4min
completed: 2026-03-02
---

# Phase 17 Plan 02: LLM and Search Pipeline Summary

**crewai.LLM built from injected credentials with Anthropic max_tokens handling, Tavily/DuckDuckGo search tool selection, credential stripping, and imperative crew pattern with explicit llm= on all agents**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-02T21:11:41Z
- **Completed:** 2026-03-02T21:15:17Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Sidecar executor extracts `_llm_api_key` and `_llm_model` from injected settings to construct `crewai.LLM`, with automatic `max_tokens=4096` for Anthropic models
- Search tool selection: TavilySearchTool when `_tavily_api_key` present, DuckDuckGo `@tool` wrapper as zero-cost fallback
- Credential sanitization via `_clean_settings` strips all `_`-prefixed keys before passing to crew inputs, preventing API keys from appearing in agent prompt templates
- daily-news-digest crew rewritten from `@CrewBase` YAML-config pattern to imperative `Agent`/`Task`/`Crew` constructors — all three agents receive explicit `llm=llm`, researcher receives `tools=[search_tool]`

## Task Commits

Each task was committed atomically:

1. **Task 1: Add search tool dependencies and update executor with credential extraction** - `176c4a3` (feat)
2. **Task 2: Rewrite daily-news-digest crew.py with imperative pattern** - `61663c9` (feat)

**Plan metadata:** (pending docs commit)

## Files Created/Modified
- `sidecar/executor.py` - Added `_extract_llm`, `_extract_search_tool`, `_clean_settings` methods; updated `execute()` and `_load_crew()` to accept and pass llm/search_tool
- `sidecar/pyproject.toml` - Added crewai-tools, tavily-python, langchain-community, duckduckgo-search dependencies
- `plugins/daily-news-digest/crew/crew.py` - Replaced @CrewBase class with imperative `create_crew(settings, llm=None, search_tool=None)` factory

## Decisions Made
- **Imperative crew pattern over @CrewBase:** YAML config decorators cannot inject dynamic `llm=` at runtime; imperative constructors accept it directly. This is the research-recommended pattern (see 17-RESEARCH.md Pattern 4).
- **DuckDuckGo wrapped with @tool:** Needed to match TavilySearchTool's CrewAI tool interface so both can be passed in `tools=[search_tool]` without conditional code in the crew.
- **Credential stripping via `_clean_settings`:** Filters all `_`-prefixed keys so `{_llm_api_key}` cannot appear in agent prompt templates during CrewAI variable interpolation.
- **Anthropic max_tokens=4096:** Applied when model string starts with `anthropic/` — Anthropic API requires explicit max_tokens, other providers use defaults.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Self-Check: PASSED

All files found on disk. Both task commits verified in git log.

## Next Phase Readiness
- LLM and search pipeline complete — sidecar will now construct proper LLM instances and search tools from credentials injected by the Go worker
- Phase 18 (briefing result display) can proceed: the full pipeline from schedule trigger through AI generation to result publishing is now complete end-to-end
- Real CrewAI runs will require valid API keys in the user's settings; DuckDuckGo fallback handles no-Tavily-key case gracefully

---
*Phase: 17-llm-and-search-pipeline*
*Completed: 2026-03-02*
