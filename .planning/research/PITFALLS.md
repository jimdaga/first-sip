# Pitfalls Research: v1.2 Live AI Generation

**Domain:** Adding per-user API keys + real AI generation to existing Go+Python plugin platform
**Researched:** 2026-02-27
**Confidence:** HIGH

## Critical Pitfalls

### Pitfall 1: API Keys in Redis Streams Messages

**What goes wrong:**
User's plaintext API keys sit in Redis Streams until consumed and ACKed. If Redis is compromised or messages back up, keys are exposed. If sidecar crashes mid-processing, unACKed messages with keys persist in the Pending Entry List indefinitely.

**Why it happens:**
Redis Streams are persistent by design. The existing architecture passes settings (non-sensitive) through streams, but API keys are a different threat class.

**Prevention:**
- Accept the risk for same-host Redis (localhost/same pod). This is the same trust boundary as the current session cookies stored in Redis.
- Set aggressive message TTL/trimming (maxlen already at 10k).
- Ensure PEL cleanup on sidecar restart (existing pending recovery logic handles this).
- Do NOT log message payloads that contain keys.
- **Phase to address:** Early — when extending Redis Streams payload.

### Pitfall 2: CrewAI Execution Timeout with Real LLMs

**What goes wrong:**
Real LLM calls take 10-60 seconds per agent step. A 3-agent crew (researcher → writer → reviewer) with web search can take 2-5 minutes. Current Asynq task timeout may be too aggressive. User sees "failed" when it just took longer than expected.

**Why it happens:**
Stub/mock generation was instant. Real multi-step CrewAI with search + LLM calls is orders of magnitude slower. The sidecar has a 300s default timeout, but Asynq task timeout and HTMX polling UX weren't designed for multi-minute waits.

**Prevention:**
- Review and increase Asynq plugin task timeout (currently 10 minutes — should be sufficient).
- Sidecar CREW_TIMEOUT_SECONDS may need increase for complex crews with search.
- Add intermediate status updates (processing → "searching" → "writing" → "reviewing").
- **Phase to address:** During sidecar crew factory update and end-to-end testing.

### Pitfall 3: Invalid or Expired API Keys at Runtime

**What goes wrong:**
User saves an API key that's valid at save time. Key expires, hits rate limit, or is revoked weeks later. Scheduled briefing silently fails. User doesn't know why their briefings stopped working.

**Why it happens:**
API key validation at save time doesn't guarantee future validity. Keys have external lifecycle (billing, rotation, provider changes). No feedback loop to user when keys fail.

**Prevention:**
- Validate key on save (make a lightweight API call).
- Store and surface error messages from failed runs prominently.
- Consider: mark user's key as "needs attention" after N consecutive failures.
- **Phase to address:** API key management (validation) and content rendering (error display).

### Pitfall 4: DuckDuckGo Search Reliability

**What goes wrong:**
DuckDuckGo's unofficial API (via `duckduckgo-search` library) is rate-limited and occasionally returns empty results or gets blocked. Briefing generation fails or produces content without any real news data.

**Why it happens:**
Unlike Tavily (official API with SLA), DuckDuckGo search is a scraping-based library with no reliability guarantees. It's a free fallback, not a primary tool.

**Prevention:**
- Use Tavily as primary, DuckDuckGo only as fallback.
- Handle empty search results gracefully in crew — agent should note "unable to find recent news" rather than crash.
- Don't promise DuckDuckGo quality parity with Tavily.
- **Phase to address:** Search tool integration phase.

### Pitfall 5: N8N Webhook Removal Breaks Existing Briefings

**What goes wrong:**
Removing the N8N webhook path deletes the `briefing:generate` task type. Existing briefing records in the database reference this task type. Old briefing content (from webhook stub) may display differently than new plugin-generated content.

**Why it happens:**
Two separate content models: `Briefing` (v1.0, webhook-based) and `PluginRun` (v1.1+, CrewAI-based). Dashboard may still query/display old briefings. Removing webhook code path without migrating display logic leaves orphaned UI.

**Prevention:**
- Keep old Briefing model/table for historical data (read-only).
- Remove only the generation path (webhook client, task handler).
- Ensure dashboard exclusively uses PluginRun for new content.
- Don't delete old briefing data — just stop generating new ones.
- **Phase to address:** Legacy cleanup (last phase, after new path proven).

### Pitfall 6: LiteLLM Model String Validation

**What goes wrong:**
User enters an invalid model string (e.g., "gpt4" instead of "openai/gpt-4o"). CrewAI/LiteLLM fails at runtime with cryptic error. User doesn't know what went wrong.

**Why it happens:**
LiteLLM model strings follow a `provider/model` convention but accept freeform text. No validation until actual API call. Different providers have different model name formats.

**Prevention:**
- Provide a curated dropdown of common models (openai/gpt-4o, anthropic/claude-sonnet-4-20250514, groq/llama-3.1-70b).
- Allow freeform input for advanced users.
- Validate model string format (must contain `/` separator for non-OpenAI providers).
- Surface clear error messages from failed generation attempts.
- **Phase to address:** API key management UI.

## Integration Pitfalls

### Pitfall 7: Encryption Key Sharing Between Models

**What goes wrong:**
New `UserAPIKey` model uses same AES-256-GCM encryption as `AuthIdentity`. If encryption implementation is copy-pasted rather than shared, any bug fix needs to be applied in two places.

**Prevention:**
- Extract encryption helpers into shared package (or reuse existing `internal/models` encryption functions).
- Both models use the same `ENCRYPTION_KEY` environment variable.
- **Phase to address:** API key model creation (first phase).

## Summary

| # | Pitfall | Severity | Phase |
|---|---------|----------|-------|
| 1 | API keys in Redis Streams | Medium | Streams payload extension |
| 2 | CrewAI timeout with real LLMs | High | Sidecar + E2E testing |
| 3 | Invalid/expired API keys | Medium | Key management + error display |
| 4 | DuckDuckGo reliability | Low | Search tool integration |
| 5 | Webhook removal breaks history | Medium | Legacy cleanup (last) |
| 6 | LiteLLM model string validation | Low | Key management UI |
| 7 | Encryption code duplication | Low | API key model (first) |
