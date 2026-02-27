# Architecture Research: v1.2 Live AI Generation

**Domain:** Integrating per-user API keys and real AI generation into existing plugin platform
**Researched:** 2026-02-27
**Confidence:** HIGH

## Integration Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      User Browser                            │
│  Settings page: API key input, provider selection            │
│  Dashboard: Real briefing content with Markdown rendering    │
└──────────────────────────┬──────────────────────────────────┘
                           │ HTMX
┌──────────────────────────┴──────────────────────────────────┐
│                    Go Server (Gin)                            │
│                                                              │
│  NEW: /internal/apikeys/                                     │
│    ├── model.go      (UserAPIKey model, AES-256-GCM hooks)  │
│    ├── handler.go    (CRUD endpoints for key management)     │
│    └── service.go    (Key retrieval, decryption for runs)    │
│                                                              │
│  MODIFIED: /internal/streams/producer.go                     │
│    └── Include decrypted API keys in Redis Streams message   │
│                                                              │
│  MODIFIED: /internal/worker/plugin_task.go                   │
│    └── Fetch user's API keys before publishing to stream     │
│                                                              │
│  REMOVED: /internal/worker/briefing_task.go (N8N webhook)    │
│  REMOVED: /internal/worker/webhook_client.go                 │
│                                                              │
│  MODIFIED: /internal/templates/                              │
│    └── API key settings page, Markdown briefing rendering    │
└──────────────────────────┬──────────────────────────────────┘
                           │ Redis Streams
                           │ (plugin:requests now includes api_keys)
┌──────────────────────────┴──────────────────────────────────┐
│                  Python Sidecar (FastAPI)                     │
│                                                              │
│  MODIFIED: /sidecar/executor.py                              │
│    └── Pass LLM config + search API key to crew factory      │
│                                                              │
│  MODIFIED: /plugins/daily-news-digest/crew/crew.py           │
│    ├── Accept llm_config parameter                           │
│    ├── Create LLM instance with user's API key               │
│    ├── Configure TavilySearchTool or DuckDuckGoSearchRun     │
│    └── Assign LLM + tools to agents                          │
│                                                              │
│  NEW dependencies: tavily-python, duckduckgo-search          │
└─────────────────────────────────────────────────────────────┘
```

## Data Flow: API Key → Generation

```
1. User enters API key in settings UI
   → POST /api/apikeys
   → AES-256-GCM encrypt → store in user_api_keys table

2. Scheduler triggers plugin run
   → plugin_task.go fetches user's API keys from DB
   → Decrypt keys in Go memory
   → Publish to plugin:requests stream with keys in payload

3. Sidecar receives request
   → Extract api_keys from message
   → Pass to create_crew(settings, llm_config, search_key)
   → LLM class configured with user's key
   → TavilySearchTool configured with user's Tavily key
   → Crew executes with real API calls

4. Result flows back via plugin:results stream
   → handler.go stores structured output in PluginRun.output
   → Dashboard renders real content
```

## New Database Components

### user_api_keys table (Migration 000010)

```sql
CREATE TABLE user_api_keys (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    provider VARCHAR(50) NOT NULL,    -- 'openai', 'anthropic', 'groq', 'tavily'
    api_key TEXT NOT NULL,            -- AES-256-GCM encrypted
    model_name VARCHAR(100),          -- 'gpt-4o', 'claude-sonnet-4-20250514', etc.
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    UNIQUE(user_id, provider)
);
```

## Redis Streams Payload Change

**Current:**
```json
{
  "plugin_run_id": "uuid",
  "plugin_name": "daily-news-digest",
  "user_id": 123,
  "settings": {"topics": ["technology"], "summary_length": "standard"}
}
```

**v1.2:**
```json
{
  "plugin_run_id": "uuid",
  "plugin_name": "daily-news-digest",
  "user_id": 123,
  "settings": {"topics": ["technology"], "summary_length": "standard"},
  "llm_config": {
    "model": "openai/gpt-4o",
    "api_key": "sk-decrypted-key"
  },
  "search_config": {
    "provider": "tavily",
    "api_key": "tvly-decrypted-key"
  }
}
```

## Security Considerations

- API keys encrypted at rest in Postgres (AES-256-GCM, same pattern as OAuth tokens)
- Keys decrypted only in Go memory, briefly, before Redis publish
- Redis Streams messages contain plaintext keys — acceptable because:
  - Redis is local (same pod in K8s, localhost in dev)
  - Messages are ACKed and trimmed (maxlen ~10k)
  - Same trust boundary as current OAuth token handling
- Sidecar uses keys transiently (not stored to disk)

## Suggested Build Order

1. **API key model + encryption** — Foundation (reuse AuthIdentity pattern)
2. **API key management UI** — User-facing CRUD
3. **Redis Streams payload extension** — Wire keys through pipeline
4. **Sidecar crew factory update** — Accept and use LLM config
5. **Tavily/DuckDuckGo tool integration** — Search capability
6. **Real crew execution** — End-to-end generation
7. **Content rendering** — Display real Markdown output
8. **Legacy cleanup** — Remove N8N webhook path
