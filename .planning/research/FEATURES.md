# Feature Research: v1.2 Live AI Generation

**Domain:** Per-user API keys, LLM provider configuration, real content generation
**Researched:** 2026-02-27
**Confidence:** HIGH

## Feature Categories

### API Key Management

| Feature | Category | Complexity | Dependencies |
|---------|----------|------------|--------------|
| Store encrypted LLM API key per user | Table stakes | Low | Existing AES-256-GCM pattern |
| Store encrypted Tavily API key per user | Table stakes | Low | Same encryption pattern |
| API key management page in settings | Table stakes | Medium | Existing settings UI patterns |
| Key validation on save (test API call) | Differentiator | Medium | Sidecar or Go-side validation |
| Masked key display (sk-...xxxx) | Table stakes | Low | Frontend only |
| Multiple provider keys (OpenAI + Anthropic) | Table stakes | Low | Separate fields per provider type |
| LLM provider selection (which key to use) | Table stakes | Low | Dropdown/radio in settings |
| Key rotation reminder | Anti-feature | — | Over-engineering for personal use |

### LLM Provider Configuration

| Feature | Category | Complexity | Dependencies |
|---------|----------|------------|--------------|
| Provider-agnostic model selection via LiteLLM | Table stakes | Low | CrewAI native support |
| Model string input (e.g., openai/gpt-4o) | Table stakes | Low | Text field in settings |
| Per-plugin model override | Differentiator | Medium | Plugin settings schema extension |
| Temperature/parameter tuning | Differentiator | Low | Plugin settings schema |
| Cost tracking per generation | Anti-feature | — | Major scope, defer |

### Web Search Integration

| Feature | Category | Complexity | Dependencies |
|---------|----------|------------|--------------|
| Tavily search in researcher agent | Table stakes | Low | tavily-python + API key |
| DuckDuckGo fallback when no Tavily key | Table stakes | Low | duckduckgo-search (free) |
| Topic-based search queries from user settings | Table stakes | Low | Existing topics setting |
| Search result caching | Anti-feature | — | Premature optimization |

### Content Generation & Display

| Feature | Category | Complexity | Dependencies |
|---------|----------|------------|--------------|
| Real CrewAI crew execution with live LLM | Table stakes | Low | API key + sidecar already built |
| Structured output (summary + sections) | Table stakes | Low | Sidecar output wrapper exists |
| Markdown rendering in briefing tiles | Table stakes | Medium | Templ component for Markdown |
| Error display when generation fails | Table stakes | Low | Existing error states |
| Generation progress indication | Differentiator | Medium | Status polling exists, enhance |

### Legacy Cleanup

| Feature | Category | Complexity | Dependencies |
|---------|----------|------------|--------------|
| Remove N8N webhook client | Table stakes | Low | No dependencies |
| Remove briefing:generate Asynq task | Table stakes | Low | Replace with plugin:execute |
| Migrate existing briefings display | Table stakes | Medium | Dashboard already shows plugin runs |
| Remove webhook config/env vars | Table stakes | Low | Config cleanup |

## Recommended v1.2 Scope

**Include (table stakes + selected differentiators):**
- All table stakes features above
- Key validation on save
- Per-plugin model override (if simple to add via settings schema)

**Defer:**
- Cost tracking, key rotation reminders, search caching
- Temperature/parameter tuning (can add in settings schema later)
