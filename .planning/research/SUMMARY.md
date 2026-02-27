# Project Research Summary

**Project:** First Sip v1.2 Live AI Generation
**Domain:** Per-user API keys, provider-agnostic LLM, Tavily search, real CrewAI generation
**Researched:** 2026-02-27
**Confidence:** HIGH

## Executive Summary

The v1.2 milestone transforms First Sip from a platform with stubbed AI output into one that generates real content. The core challenge is bridging the per-user API key lifecycle (stored encrypted in Go, decrypted transiently, passed through Redis Streams) to the Python CrewAI sidecar where actual LLM and search API calls happen. The existing architecture handles 90% of this — AES-256-GCM encryption is proven, Redis Streams pipeline is built, CrewAI crews are defined. What's missing is the API key storage layer, the payload extension to carry keys, and the crew factory modifications to use them.

The biggest risk is execution timeouts: real multi-agent CrewAI runs with web search take 2-5 minutes vs instant stubs. The existing 10-minute Asynq timeout should cover this, but UX needs to communicate progress during longer waits. Secondary risk is API key lifecycle management — keys that work at save time may expire later, and users need clear feedback when generation fails.

## Stack Additions

**Python sidecar only — no new Go dependencies:**
- `litellm` — Already a CrewAI dependency. Provides `provider/model` format for 200+ LLM providers.
- `tavily-python` — Official Tavily SDK. Free tier: 1,000 credits/month (~33 daily briefings).
- `duckduckgo-search` — Free fallback via `langchain_community.tools.DuckDuckGoSearchRun`.

**Go side:** No new libraries. Reuse existing AES-256-GCM encryption pattern. New DB migration + model only.

## Feature Table Stakes

| Category | Table Stakes | Differentiators |
|----------|-------------|-----------------|
| API Key Management | Encrypted storage, masked display, per-provider keys, management UI | Validation on save |
| LLM Configuration | Provider-agnostic model selection, model string input | Per-plugin model override |
| Web Search | Tavily primary, DuckDuckGo fallback, topic-based queries | — |
| Content Generation | Real CrewAI execution, structured output, Markdown rendering, error display | Progress indication |
| Legacy Cleanup | Remove N8N webhook client, task handler, config vars | — |

## Architecture Changes

**New:** `user_api_keys` table with AES-256-GCM encrypted storage (Migration 000010)
**Modified:** Redis Streams payload adds `llm_config` and `search_config` objects
**Modified:** Sidecar `executor.py` and `create_crew()` accept LLM + search parameters
**Modified:** Dashboard renders real Markdown content from CrewAI output
**Removed:** N8N webhook client, `briefing:generate` task, webhook config vars

## Watch Out For

1. **CrewAI timeout** (HIGH) — Real runs take 2-5 minutes. Review Asynq + sidecar timeouts. Add progress UX.
2. **API keys in Redis Streams** (MEDIUM) — Plaintext in transit. Acceptable for same-host Redis. Don't log payloads.
3. **Expired keys at runtime** (MEDIUM) — Validate on save. Surface errors prominently. Consider "needs attention" flag.
4. **Webhook removal** (MEDIUM) — Keep old Briefing table read-only. Remove generation path only.
5. **DuckDuckGo reliability** (LOW) — Fallback only. Handle empty results gracefully.

## Suggested Build Order

1. API key model + encryption (foundation)
2. API key management UI (user-facing)
3. Redis Streams payload extension (wire keys through)
4. Sidecar crew factory + LLM config (accept keys)
5. Tavily/DuckDuckGo search tools (enable research)
6. End-to-end generation (prove it works)
7. Content rendering (display real output)
8. Legacy cleanup (remove webhook path)
