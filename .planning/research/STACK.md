# Stack Additions for v1.2 Live AI Generation

**Domain:** Per-user API keys, LiteLLM providers, Tavily search, real CrewAI generation
**Researched:** 2026-02-27
**Confidence:** HIGH

## Context: What We Already Have (DO NOT Add)

**Validated v1.0 + v1.1 Stack:**
- Go 1.24, Gin, Templ, HTMX 2.0, GORM (PostgreSQL), Asynq (Redis), Goth
- CrewAI sidecar with Redis Streams (producer.go, consumer.go, handler.go)
- AES-256-GCM encryption (AuthIdentity model, GORM hooks)
- Plugin framework with YAML metadata, JSON Schema settings, database registry
- Per-user per-plugin scheduling with timezone-aware cron

## New Stack Additions

### Python (Sidecar)

| Library | Version | Purpose | Rationale |
|---------|---------|---------|-----------|
| `litellm` | latest | Provider-agnostic LLM interface | CrewAI uses LiteLLM natively for 200+ model providers. Already a CrewAI dependency. |
| `tavily-python` | latest | Web search API client | Official Tavily SDK. CrewAI has `TavilySearchTool` in `crewai_tools`. Free tier: 1,000 credits/month. |
| `duckduckgo-search` | latest | Free web search fallback | Used via `DuckDuckGoSearchRun` from `langchain_community.tools`. No API key needed. |

### Go (Server)

| Library | Version | Purpose | Rationale |
|---------|---------|---------|-----------|
| (none new) | — | — | Existing AES-256-GCM encryption and GORM patterns are sufficient for API key storage. New DB migration + model fields only. |

### CrewAI LLM Configuration

CrewAI's `LLM` class accepts `model`, `api_key`, `base_url`, and `temperature` parameters. Each agent can have a different LLM. The model string follows LiteLLM format: `provider/model-name` (e.g., `openai/gpt-4o`, `anthropic/claude-sonnet-4-20250514`, `groq/llama-3.1-70b`).

```python
from crewai import LLM
llm = LLM(
    model="openai/gpt-4o",
    api_key="user-provided-key",
    temperature=0.7
)
```

### What NOT to Add

- **No vault/KMS** — AES-256-GCM with GORM hooks is already proven and sufficient for this scale
- **No LiteLLM proxy server** — Direct library usage is simpler; proxy is for multi-service setups
- **No additional search APIs** — Tavily + DuckDuckGo covers the use case; no Serper/Google needed
- **No Redis sessions** — Flagged for revisit but not part of this milestone's scope

## Integration Points

1. **API key encryption** — Reuse existing `InitEncryption()` and AES-256-GCM pattern from `internal/models/auth_identity.go`
2. **Redis Streams payload** — Extend request message to include user's API keys (encrypted in DB, decrypted before publish)
3. **CrewAI crew factory** — Modify `create_crew(settings)` to accept `llm_config` and `search_api_key` parameters
4. **Plugin settings schema** — Add API key fields to JSON Schema (marked as sensitive/secret type)

## Tavily Free Tier Details

- 1,000 credits/month (no rollover)
- Basic search: 1 credit per search
- Advanced search: 2 credits per search
- Sufficient for: ~33 daily briefings/month (assuming ~30 searches per briefing)
- Pay-as-you-go: $0.008 per credit if exceeded
