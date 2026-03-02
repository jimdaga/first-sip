# Phase 17: LLM and Search Pipeline - Research

**Researched:** 2026-03-02
**Domain:** CrewAI LiteLLM provider abstraction, Redis Streams payload extension, search tool integration (Tavily + DuckDuckGo)
**Confidence:** HIGH (architecture patterns verified against live codebase; LiteLLM/CrewAI patterns verified against official docs)

## Summary

Phase 17 wires the API keys stored in Phase 16 into the Redis Streams → CrewAI sidecar pipeline. The work splits cleanly into three concerns: (1) the Go side must look up the user's decrypted LLM key and Tavily key at dispatch time and inject them into the `PluginRequest` payload; (2) the Python sidecar `executor.py` must read those credentials from the payload and use them to build a `crewai.LLM` object via LiteLLM's `provider/model` string format; (3) the daily-news-digest crew must conditionally use `TavilySearchTool` (when a key is present) or a DuckDuckGo tool (when not), and apply topic preferences from plugin settings as agent inputs.

The critical design insight is that API keys must **never be logged** (STATE.md blocker note) and must travel through the existing JSON payload field `settings` under well-known keys (`_llm_api_key`, `_llm_model`, `_tavily_api_key`). The Go worker fetches decrypted values from `apikeys.GetKeysForUser`, adds them to the settings map at dispatch time, and the Python side strips them back out before passing user-visible settings to the crew as `inputs`.

The per-plugin LLM model override (LLM-03) is implemented as a convention: if the plugin's `UserPluginConfig.Settings` contains a `_llm_model` key, the sidecar uses it instead of the user-level default. This requires no schema migration — it's a soft convention in the settings map.

**Primary recommendation:** Extend `PluginRequest.Settings` (both Go struct and Python model) with private underscore-prefixed keys for credentials; build a `create_crew(settings, llm_config)` factory signature that accepts an explicit LLM object; use environment-variable injection for Tavily (`os.environ["TAVILY_API_KEY"]`) since `TavilySearchTool()` reads from env, not constructor args.

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| LLM-01 | System passes user's LLM API key to CrewAI sidecar per run | Go `handleExecutePlugin` fetches decrypted key via `apikeys.GetKeysForUser`, injects into `PluginRequest.Settings` as `_llm_api_key` + `_llm_model` |
| LLM-02 | CrewAI crew uses provider-agnostic LLM via LiteLLM format (provider/model) | `crewai.LLM(model="openai/gpt-4o", api_key="...")` is the verified constructor; CrewAI passes through to LiteLLM for all three supported providers (openai, anthropic, groq) |
| LLM-03 | User can override LLM model per plugin via plugin settings | Plugin `UserPluginConfig.Settings` JSON can carry a `_llm_model` key; sidecar checks for it before falling back to user-level default |
| SRCH-01 | CrewAI researcher agent uses Tavily search when user has Tavily key | `TavilySearchTool()` reads `TAVILY_API_KEY` from environment; inject key via `os.environ` before instantiating the tool |
| SRCH-02 | CrewAI researcher agent falls back to DuckDuckGo when no Tavily key | `DuckDuckGoSearchRun` from `langchain_community.tools` wraps cleanly as a CrewAI tool; install `duckduckgo-search>=4.1.0` |
| SRCH-03 | Search queries incorporate user's topic preferences from plugin settings | `{topics}` placeholder already exists in `agents.yaml` goal and `tasks.yaml`; crew receives `settings` as `inputs=` to `kickoff_async` |
</phase_requirements>

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `crewai` | >=1.9.3 (already in pyproject.toml) | Agent orchestration | Already pinned in sidecar; ships LiteLLM bundled |
| `crewai[tools]` or `crewai-tools` | matches crewai version | `TavilySearchTool` | Official CrewAI tooling package |
| `tavily-python` | latest compatible | TavilySearchTool dependency | Required by crewai-tools docs; install alongside crewai[tools] |
| `langchain-community` | latest compatible | `DuckDuckGoSearchRun` | Standard LangChain tool for DDG; CrewAI agents accept LangChain tools |
| `duckduckgo-search` | >=4.1.0 | DDG search backend | Required by langchain-community DuckDuckGo tool |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `crewai.LLM` class | (part of crewai) | Provider-agnostic LLM wrapper over LiteLLM | Use instead of raw openai/anthropic SDK clients; pass `model="provider/model"` + `api_key` |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `langchain_community.DuckDuckGoSearchRun` | Custom HTTP tool against DDG API | LangChain tool requires extra deps but zero custom code; custom tool avoids langchain dep but adds maintenance |
| Env-var injection for Tavily key | `TavilySearchTool(api_key=...)` constructor param | Docs only show env var pattern; no confirmed constructor `api_key` param in TavilySearchTool (source not verified); env injection is safe and confirmed |
| Extending `PluginRequest` with typed fields | Embedding in `settings` map | Typed fields require Go struct change + Python model change; settings map extension is zero-schema-change and already designed for flexibility |

**Installation (sidecar pyproject.toml additions):**
```toml
dependencies = [
    "crewai>=1.9.3",
    "crewai-tools>=0.55.0",      # adds TavilySearchTool
    "tavily-python",              # TavilySearchTool dependency
    "langchain-community",        # DuckDuckGoSearchRun
    "duckduckgo-search>=4.1.0",  # DDG search backend
    "fastapi>=0.115.0",
    "uvicorn[standard]>=0.32.0",
    "redis>=6.1.0",
]
```

---

## Architecture Patterns

### Pattern 1: Go-side key injection in `handleExecutePlugin`

**What:** Before publishing to Redis Streams, fetch the user's decrypted LLM key and Tavily key from the database and inject them into `PluginRequest.Settings`.

**When to use:** At the Asynq worker task handler level, after `CreatePluginRun` succeeds, before `PublishPluginRequest`.

**Current code location:** `internal/worker/worker.go` → `handleExecutePlugin`

**Pattern:**
```go
// Source: apikeys/service.go GetKeysForUser pattern (existing codebase)
keys, err := apikeys.GetKeysForUser(db, payload.UserID)
if err != nil {
    // log but don't fail - degrade gracefully
}

// Find the user's preferred LLM key
llmKey := findKey(keys, "llm", user.LLMPreferredProvider)
if llmKey != nil {
    settings["_llm_api_key"] = llmKey.EncryptedValue  // decrypted by AfterFind hook
    settings["_llm_model"] = user.LLMPreferredProvider + "/" + user.LLMPreferredModel
}

// Find Tavily key
tavilyKey := findKey(keys, "tavily", "tavily")
if tavilyKey != nil {
    settings["_tavily_api_key"] = tavilyKey.EncryptedValue
}
```

**Key constraint:** The worker needs access to `db` (already has it) AND the `models.User` (currently only has `UserID`). Need to fetch `User` record to get `LLMPreferredProvider` and `LLMPreferredModel`.

**Do NOT log** the `_llm_api_key` or `_tavily_api_key` values (STATE.md blocker).

### Pattern 2: CrewAI LLM constructor via LiteLLM provider/model string

**What:** Build a `crewai.LLM` object from the injected `_llm_model` and `_llm_api_key` values. Pass this LLM to each agent in the crew.

**Source:** [CrewAI LLM docs](https://docs.crewai.com/en/concepts/llms) — HIGH confidence

**Example:**
```python
from crewai import LLM

def build_llm(settings: dict) -> LLM | None:
    """Build LLM from settings payload. Returns None if no key configured."""
    api_key = settings.get("_llm_api_key")
    model = settings.get("_llm_model")  # e.g., "openai/gpt-4o"
    if not api_key or not model:
        return None
    return LLM(
        model=model,      # "provider/model" — LiteLLM routing
        api_key=api_key,
    )
```

**Provider/model string format (verified):**
- OpenAI: `"openai/gpt-4o"`, `"openai/gpt-4o-mini"`, `"openai/gpt-4-turbo"`, `"openai/gpt-3.5-turbo"`
- Anthropic: `"anthropic/claude-opus-4-5"`, `"anthropic/claude-sonnet-4-5"`, `"anthropic/claude-haiku-3-5"`
- Groq: `"groq/llama-3.3-70b-versatile"`, `"groq/llama-3.1-8b-instant"`, `"groq/mixtral-8x7b-32768"`

**Note on Groq:** CrewAI default is OpenAI. If no LLM is passed to agents, it requests `OPENAI_API_KEY`. Always pass an explicit `llm=` to every agent when using non-OpenAI models.

**Note on Anthropic:** Anthropic models require `max_tokens` to be set. Add `max_tokens=4096` to the LLM constructor for Anthropic.

### Pattern 3: `create_crew` factory signature extension

**What:** The existing `create_crew(settings: dict) -> Crew` factory in each plugin's `crew.py` needs to accept an LLM object (or build one from settings). The cleanest pattern is to extract credentials from settings inside `executor.py`, build the LLM there, and pass it as a second argument to `create_crew`.

**Why in executor:** `executor.py` is the sidecar's coordination layer. Putting LLM construction there keeps crew files focused on agent/task definitions.

**New factory signature:**
```python
def create_crew(settings: dict, llm=None) -> Crew:
    """
    Args:
        settings: User plugin settings (topics, summary_length, etc.)
        llm: crewai.LLM instance to use, or None to fall back to env-var default
    """
```

**Executor change:**
```python
# executor.py _load_crew
crew = module.create_crew(clean_settings, llm=llm_instance)
```

Where `clean_settings` strips `_llm_api_key`, `_tavily_api_key`, `_llm_model` before passing to the crew (so agent `inputs=` only get user-visible settings like `topics`, `summary_length`).

### Pattern 4: Per-agent LLM assignment in crew.py

**What:** Pass the `llm` parameter to every `Agent()` constructor.

**Source:** [CrewAI community verified](https://community.crewai.com/t/why-is-crewai-asking-for-openai-api-key-if-i-set-another-llm-provider-e-g-groq-to-the-crew/1163)

```python
# plugins/daily-news-digest/crew/crew.py
from crewai import Agent, Task, Crew, Process, LLM

def create_crew(settings: dict, llm=None) -> Crew:
    researcher = Agent(
        role="Senior News Research Analyst",
        goal="...",
        backstory="...",
        tools=[search_tool],   # Tavily or DDG
        llm=llm,               # explicit — never let it default to OpenAI
        verbose=True,
    )
    # ... writer, reviewer with same llm=llm
    return Crew(agents=[researcher, writer, reviewer], tasks=[...], process=Process.sequential)
```

**Important:** Do not use YAML-based `agents_config`/`tasks_config` approach if you need runtime-dynamic `llm=` assignments. The `@CrewBase` decorator pattern loads YAML at class instantiation and doesn't easily support dynamic LLM injection. Use the imperative constructor pattern instead.

### Pattern 5: Tavily + DuckDuckGo search tool selection

**What:** Select the right search tool based on whether `_tavily_api_key` was injected into settings.

**Source:** [TavilySearchTool docs](https://docs.crewai.com/en/tools/search-research/tavilysearchtool) + [community patterns for DDG](https://community.crewai.com/t/duckduckgo-cant-run/1770)

```python
import os
from crewai_tools import TavilySearchTool
from langchain_community.tools import DuckDuckGoSearchRun
from crewai.tools import tool

def build_search_tool(tavily_api_key: str | None):
    if tavily_api_key:
        # TavilySearchTool reads TAVILY_API_KEY from env
        os.environ["TAVILY_API_KEY"] = tavily_api_key
        return TavilySearchTool()
    else:
        # DuckDuckGo - free, no key needed
        ddg = DuckDuckGoSearchRun()

        @tool("Web Search")
        def web_search(query: str) -> str:
            """Search the web for information."""
            return ddg.run(query)

        return web_search
```

**Important for DuckDuckGo:** Direct use of `DuckDuckGoSearchRun` as a LangChain tool in CrewAI has had Pydantic compatibility issues in some versions. Wrapping it with `@tool` decorator avoids this. Alternatively use `DuckDuckGoSearchResults` from `langchain_community.tools`.

### Pattern 6: Per-plugin LLM model override (LLM-03)

**What:** If `UserPluginConfig.Settings` contains a `_llm_model` key (e.g., user set a different model for this specific plugin), the sidecar uses it instead of the user-level default.

**Implementation location:** Go side — `handleExecutePlugin` in `worker.go`.

**Logic:**
```go
// Check if plugin settings have a model override
pluginModelOverride, _ := settings["_llm_model"].(string)

// Default: use user-level preference
llmModel := user.LLMPreferredProvider + "/" + user.LLMPreferredModel

// Override with plugin-level if present
if pluginModelOverride != "" {
    llmModel = pluginModelOverride
}

settings["_llm_model"] = llmModel
```

**Note:** The settings schema (`settings.schema.json`) does NOT define `_llm_model` — it's a convention understood only by the sidecar. The plugin settings UI in phase settings page doesn't expose it yet (that's future work). For now, it enables the override mechanism without UI.

### Recommended Project Structure Changes

```
internal/
└── worker/
    └── worker.go          # handleExecutePlugin: add key injection logic
                           # Add fetchUserWithKeys() helper

sidecar/
├── executor.py            # Extract credentials, build LLM, build search tool
├── models.py              # No changes (Settings is dict[str, Any])
└── worker.py              # No changes

plugins/
└── daily-news-digest/
    └── crew/
        └── crew.py        # Rewrite: imperative pattern, accept llm=, use search_tool
```

### Anti-Patterns to Avoid

- **Do not log `_llm_api_key` or `_tavily_api_key`**: These travel plaintext in Redis Streams (same-host only). Never include them in slog output. Remove before logging settings.
- **Do not use `@CrewBase` decorator with dynamic LLM**: YAML-based agent configs with `@CrewBase` make runtime `llm=` injection awkward. Use the imperative pattern in `create_crew`.
- **Do not pass settings dict directly as crew inputs**: Strip underscore-prefixed credential keys before calling `crew.kickoff_async(inputs=clean_settings)`. Otherwise agents may attempt to use them as template interpolation variables.
- **Do not set OPENAI_API_KEY as env var for other providers**: CrewAI defaults to OpenAI only if no `llm=` is explicitly passed to agents. Always pass the LLM object explicitly.
- **Do not use bare `DuckDuckGoSearchRun` without `@tool` wrapper**: Pydantic validation issues reported in CrewAI community when passing LangChain tools directly without wrapping.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Provider-specific LLM API clients | Custom OpenAI/Anthropic/Groq clients | `crewai.LLM(model="provider/model", api_key=...)` | LiteLLM handles all three; retry, streaming, token counting built in |
| Web search scraper | Custom HTTP requests to DDG | `DuckDuckGoSearchRun` from `langchain_community` | Rate limiting, parsing, result formatting handled |
| Search result parsing | HTML parsing of Tavily response | `TavilySearchTool()` | Returns structured results optimized for LLM consumption |
| API key fetching and caching | Custom key cache | `apikeys.GetKeysForUser(db, userID)` | Already exists; AfterFind hook decrypts transparently |

**Key insight:** The entire LLM multi-provider abstraction is solved by LiteLLM's `provider/model` string convention. CrewAI's `LLM` class wraps this. Zero custom routing code needed.

---

## Common Pitfalls

### Pitfall 1: Missing `max_tokens` for Anthropic models
**What goes wrong:** Anthropic's API requires `max_tokens` to be set; CrewAI will raise `anthropic.BadRequestError` if omitted.
**Why it happens:** CrewAI's LLM class doesn't default it for Anthropic.
**How to avoid:** In `build_llm()`, detect Anthropic prefix and add `max_tokens=4096`.
**Warning signs:** `BadRequestError: max_tokens is required` in sidecar logs.

### Pitfall 2: TavilySearchTool import error
**What goes wrong:** `ImportError: cannot import name 'TavilySearchTool' from 'crewai_tools'` — reported against crewai-tools 0.55.0 but fixed in PR #400 (closed July 2025).
**Why it happens:** Tool wasn't exported from package `__init__.py`.
**How to avoid:** Use crewai-tools version after July 2025 fix. If encountered, use full path: `from crewai_tools.tools.tavily_search_tool.tavily_search_tool import TavilySearchTool`.
**Warning signs:** Import error at sidecar startup.

### Pitfall 3: CrewAI still requests OPENAI_API_KEY
**What goes wrong:** Even with Groq/Anthropic LLM passed to agents, some validation path requests `OPENAI_API_KEY`.
**Why it happens:** Some code paths in CrewAI (e.g., token counting, memory) fall back to OpenAI client.
**How to avoid:** Ensure `llm=` is passed to every `Agent()` constructor (not just one). Do NOT rely on global env var approach. If needed, set `OPENAI_API_KEY=dummy` to suppress the error when not using OpenAI.
**Warning signs:** `OPENAI_API_KEY environment variable not set` in sidecar logs.

### Pitfall 4: Settings dict credential leakage into agent inputs
**What goes wrong:** `_llm_api_key` gets passed to `crew.kickoff_async(inputs=settings)`, Jinja templating in tasks.yaml fails or agent sees the key value.
**Why it happens:** The entire settings dict is passed as inputs; underscore keys are not filtered.
**How to avoid:** In `executor.py`, strip all `_`-prefixed keys before building `clean_settings = {k: v for k, v in settings.items() if not k.startswith('_')}`.
**Warning signs:** Agent output contains API key substrings; template error about unknown variables.

### Pitfall 5: `handleExecutePlugin` needs `User` record but only has `UserID`
**What goes wrong:** The Asynq task payload has `user_id` but the handler needs `user.LLMPreferredProvider` and `user.LLMPreferredModel`.
**Why it happens:** The existing handler only stores `UserID` in the Asynq task payload.
**How to avoid:** In `handleExecutePlugin`, after unmarshaling payload, fetch `User` record: `db.First(&user, payload.UserID)`. This is already done implicitly by the scheduler (which preloads User). Minor DB read per dispatch — acceptable.
**Warning signs:** Empty `_llm_model` in Redis Streams payload.

### Pitfall 6: DuckDuckGo rate limiting in rapid succession
**What goes wrong:** Multiple plugin runs in quick succession may hit DDG's informal rate limits, returning empty results.
**Why it happens:** DDG has no official API; `duckduckgo-search` library respects a soft rate limit.
**How to avoid:** This is acceptable degradation; the crew will generate a briefing with less research data. No special handling needed in Phase 17.
**Warning signs:** Empty search results in agent output, researcher agent using only its training knowledge.

---

## Code Examples

### Example 1: Go — Key injection in handleExecutePlugin

```go
// Source: internal/worker/worker.go (proposed extension)
// After payload.UserID is known, before publishing to stream:

// Fetch user record for LLM preferences
var user models.User
if err := db.WithContext(ctx).First(&user, payload.UserID).Error; err != nil {
    logger.Warn("Failed to fetch user for LLM key injection",
        "user_id", payload.UserID, "error", err)
    // Continue without LLM keys — sidecar will fail gracefully
} else {
    // Fetch API keys (decryption via AfterFind hook)
    keys, err := apikeys.GetKeysForUser(db, payload.UserID)
    if err == nil {
        for _, key := range keys {
            if key.KeyType == "llm" && key.Provider == user.LLMPreferredProvider {
                settings["_llm_api_key"] = key.EncryptedValue  // already decrypted
                // Check plugin-level override first, then fall back to user default
                if _, hasOverride := settings["_llm_model"]; !hasOverride {
                    settings["_llm_model"] = user.LLMPreferredProvider + "/" + user.LLMPreferredModel
                }
            }
            if key.KeyType == "tavily" {
                settings["_tavily_api_key"] = key.EncryptedValue
            }
        }
    }
}
```

### Example 2: Python — Executor credential extraction + LLM construction

```python
# Source: sidecar/executor.py (proposed extension)
from crewai import LLM

def _extract_llm(self, settings: dict) -> LLM | None:
    """Build LLM from injected credentials. Returns None if no key."""
    api_key = settings.get("_llm_api_key")
    model = settings.get("_llm_model")  # e.g., "openai/gpt-4o"
    if not api_key or not model:
        return None

    kwargs = {"model": model, "api_key": api_key}
    # Anthropic requires max_tokens
    if model.startswith("anthropic/"):
        kwargs["max_tokens"] = 4096

    return LLM(**kwargs)

def _extract_search_tool(self, settings: dict):
    """Select search tool based on available credentials."""
    tavily_key = settings.get("_tavily_api_key")
    if tavily_key:
        os.environ["TAVILY_API_KEY"] = tavily_key
        return TavilySearchTool()
    else:
        ddg = DuckDuckGoSearchRun()
        @tool("Web Search")
        def web_search(query: str) -> str:
            """Search the web for current information."""
            return ddg.run(query)
        return web_search

def _clean_settings(self, settings: dict) -> dict:
    """Remove private underscore-prefixed keys before passing to crew inputs."""
    return {k: v for k, v in settings.items() if not k.startswith("_")}

async def execute(self, request: PluginRequest) -> PluginResult:
    llm = self._extract_llm(request.settings)
    search_tool = self._extract_search_tool(request.settings)
    clean_settings = self._clean_settings(request.settings)

    crew = self._load_crew(request.plugin_name, clean_settings, llm=llm, search_tool=search_tool)
    # ... rest of execution
```

### Example 3: Python — crew.py factory with dynamic LLM and search tool

```python
# Source: plugins/daily-news-digest/crew/crew.py (proposed rewrite)
from crewai import Agent, Task, Crew, Process

def create_crew(settings: dict, llm=None, search_tool=None) -> Crew:
    """
    Factory called by sidecar executor.
    settings: clean user settings (topics, summary_length, etc.) — no credentials.
    llm: crewai.LLM instance or None.
    search_tool: TavilySearchTool or web_search tool or None.
    """
    tools = [search_tool] if search_tool else []

    researcher = Agent(
        role="Senior News Research Analyst",
        goal="Discover and analyze breaking news in {topics}",
        backstory="You are an experienced news analyst...",
        tools=tools,
        llm=llm,        # always explicit — never default to OpenAI
        verbose=True,
    )
    writer = Agent(
        role="News Digest Writer",
        goal="Transform research findings into concise summaries",
        backstory="...",
        llm=llm,
        verbose=True,
    )
    reviewer = Agent(
        role="Editorial Quality Reviewer",
        goal="Ensure quality standards",
        backstory="...",
        llm=llm,
        verbose=False,
    )

    research_task = Task(
        description="Research breaking news in topics: {topics}. Find 3-5 stories.",
        expected_output="JSON array of stories with headline, source, summary",
        agent=researcher,
    )
    write_task = Task(
        description="Write a news digest using research. Max 500 words. Length: {summary_length}.",
        expected_output="Markdown-formatted news digest",
        agent=writer,
        context=[research_task],
    )
    review_task = Task(
        description="Review digest for accuracy, clarity, formatting.",
        expected_output="Final approved news digest in Markdown",
        agent=reviewer,
        context=[write_task],
    )

    return Crew(
        agents=[researcher, writer, reviewer],
        tasks=[research_task, write_task, review_task],
        process=Process.sequential,
        verbose=True,
    )
```

### Example 4: Go — Helper function to find key by type and provider

```go
// internal/worker/worker.go (helper)
func findAPIKey(keys []models.UserAPIKey, keyType, provider string) *models.UserAPIKey {
    for i := range keys {
        if keys[i].KeyType == keyType && keys[i].Provider == provider {
            return &keys[i]
        }
    }
    return nil
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| CrewAI needed `OPENAI_API_KEY` env var | `crewai.LLM(model="provider/model", api_key=...)` explicit constructor | CrewAI 0.60+ | No global env vars needed; per-call key injection works |
| `@CrewBase` with YAML config only | Imperative `Agent()`/`Task()` constructors | Always supported | Dynamic runtime config (llm, tools) easier without decorator |
| DuckDuckGo via `langchain_community` direct | Wrapped with `@tool` decorator | 2024-2025 | Avoids Pydantic compatibility issues with CrewAI >=0.76 |
| TavilySearchTool unavailable in crewai_tools | Available after PR #400 (July 2025) | July 2025 | Now importable from `crewai_tools` |

**Deprecated/outdated:**
- `@CrewBase` + YAML config for this use case: Still works for static setups, but incompatible with dynamic `llm=` injection without extra plumbing. Use imperative pattern instead.
- Groq without `groq/` prefix: Using just `"llama-3.3-70b-versatile"` without prefix causes `LLM Provider NOT provided` error via LiteLLM.

---

## Open Questions

1. **Does `TavilySearchTool` accept an `api_key` constructor parameter?**
   - What we know: Docs show only env var approach (`TAVILY_API_KEY`). Source not directly inspectable.
   - What's unclear: Whether `TavilySearchTool(api_key="...")` is supported.
   - Recommendation: Use env var injection (`os.environ["TAVILY_API_KEY"] = key`) — this is the documented pattern and is safe. Set and then unset after tool creation if isolation is needed.

2. **Thread safety of `os.environ` mutation in async sidecar**
   - What we know: The sidecar is asyncio-based; `executor.execute()` runs as a coroutine.
   - What's unclear: If two concurrent plugin runs both try to set `TAVILY_API_KEY` to different values simultaneously.
   - Recommendation: The sidecar processes one message at a time (blocking loop, one `executor.execute()` at a time per worker). In practice, no concurrent writes. If concurrency is added later, use a per-execution context instead.

3. **CrewAI version compatibility with `crewai-tools` after move to monorepo**
   - What we know: `crewai-tools` package moved to `https://github.com/crewAIInc/crewAI/tree/main/libs/crewai-tools`. The separate repo is deprecated.
   - What's unclear: Whether `pip install crewai-tools` still works or requires `pip install 'crewai[tools]'`.
   - Recommendation: Use `crewai[tools]` extras syntax — this is the forward-compatible form. Add `tavily-python` explicitly.

---

## Sources

### Primary (HIGH confidence)
- [CrewAI LLM Concepts Docs](https://docs.crewai.com/en/concepts/llms) — Full constructor signature, provider/model strings, api_key param, per-agent vs global config
- [CrewAI TavilySearchTool Docs](https://docs.crewai.com/en/tools/search-research/tavilysearchtool) — Installation, env var requirement, configuration options
- Existing codebase: `internal/worker/worker.go`, `internal/apikeys/service.go`, `sidecar/executor.py`, `sidecar/models.py`, `plugins/daily-news-digest/crew/crew.py` — architecture patterns, current payload structure

### Secondary (MEDIUM confidence)
- [CrewAI Community: Non-OpenAI LLM setup](https://community.crewai.com/t/why-is-crewai-asking-for-openai-api-key-if-i-set-another-llm-provider-e-g-groq-to-the-crew/1163) — Confirmed: must pass `llm=` to every agent explicitly
- [CrewAI GitHub Issue #3210](https://github.com/crewAIInc/crewAI/issues/3210) — TavilySearchTool import fix merged July 2025, now exported correctly
- [LiteLLM Groq docs](https://docs.litellm.ai/docs/providers/groq) — `groq/model-name` prefix format confirmed

### Tertiary (LOW confidence)
- [CrewAI Community: DuckDuckGo wrapper patterns](https://community.crewai.com/t/duckduckgo-cant-run/1770) — `@tool` wrapper recommended to avoid Pydantic issues (multiple community reports, not official docs)
- Anthropic `max_tokens` requirement — known from prior CrewAI issues, not explicitly documented in LLM constructor page

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — pyproject.toml already pins crewai>=1.9.3; additional deps confirmed from official docs
- Architecture: HIGH — existing codebase patterns are clear; LLM constructor verified against official docs
- Pitfalls: MEDIUM — most verified via official docs or GitHub issues; DDG `@tool` wrapper is community-confirmed but not official

**Research date:** 2026-03-02
**Valid until:** 2026-04-02 (CrewAI moves fast; re-verify if crewai version is bumped significantly)
