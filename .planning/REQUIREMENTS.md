# Requirements: First Sip

**Defined:** 2026-02-27
**Core Value:** A user's configured briefing plugins run on schedule and their latest results appear automatically on a tile-based dashboard — no manual action needed to receive fresh, personalized briefings every day.

## v1.2 Requirements

Requirements for v1.2 Live AI Generation milestone. Each maps to roadmap phases.

### API Key Management

- [ ] **KEYS-01**: User can store an encrypted LLM provider API key (OpenAI, Anthropic, Groq, etc.)
- [ ] **KEYS-02**: User can store an encrypted Tavily search API key
- [ ] **KEYS-03**: User can view their stored keys with masked display (sk-...xxxx)
- [ ] **KEYS-04**: User can update or delete their stored API keys
- [ ] **KEYS-05**: User can select their preferred LLM provider and model

### LLM Configuration

- [ ] **LLM-01**: System passes user's LLM API key to CrewAI sidecar per run
- [ ] **LLM-02**: CrewAI crew uses provider-agnostic LLM via LiteLLM format (provider/model)
- [ ] **LLM-03**: User can override LLM model per plugin via plugin settings

### Web Search

- [ ] **SRCH-01**: CrewAI researcher agent uses Tavily search when user has Tavily key
- [ ] **SRCH-02**: CrewAI researcher agent falls back to DuckDuckGo when no Tavily key
- [ ] **SRCH-03**: Search queries incorporate user's topic preferences from plugin settings

### Content Generation

- [ ] **GEN-01**: Daily news digest generates real content via CrewAI with live LLM calls
- [ ] **GEN-02**: Generated content renders as formatted Markdown in briefing tiles
- [ ] **GEN-03**: Failed generation displays clear error message to user
- [ ] **GEN-04**: Generation works end-to-end: schedule trigger → API call → content display

### Legacy Cleanup

- [ ] **CLN-01**: N8N webhook client and briefing:generate task removed
- [ ] **CLN-02**: Webhook-related configuration and environment variables removed
- [ ] **CLN-03**: Existing briefing history data preserved (read-only)

## Future Requirements

Deferred to future milestones. Tracked but not in current roadmap.

### Enhanced Key Management

- **KEYS-06**: System validates API key on save with lightweight test call
- **KEYS-07**: System marks key as "needs attention" after N consecutive failures
- **KEYS-08**: User receives notification when API key needs rotation

### Generation Enhancements

- **GEN-05**: User sees generation progress stages (searching → writing → reviewing)
- **GEN-06**: System tracks and displays per-generation cost estimates

## Out of Scope

| Feature | Reason |
|---------|--------|
| API key rotation automation | Over-engineering for personal use |
| Cost tracking/billing per generation | Major scope, requires provider-specific pricing data |
| Search result caching | Premature optimization |
| Multiple search providers beyond Tavily/DDG | Two options sufficient for v1.2 |
| CrewAI memory/context across briefings | High complexity, deferred from v1.1 |
| Additional plugin types | Prove generation works first, add more in v1.3+ |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| KEYS-01 | — | Pending |
| KEYS-02 | — | Pending |
| KEYS-03 | — | Pending |
| KEYS-04 | — | Pending |
| KEYS-05 | — | Pending |
| LLM-01 | — | Pending |
| LLM-02 | — | Pending |
| LLM-03 | — | Pending |
| SRCH-01 | — | Pending |
| SRCH-02 | — | Pending |
| SRCH-03 | — | Pending |
| GEN-01 | — | Pending |
| GEN-02 | — | Pending |
| GEN-03 | — | Pending |
| GEN-04 | — | Pending |
| CLN-01 | — | Pending |
| CLN-02 | — | Pending |
| CLN-03 | — | Pending |

**Coverage:**
- v1.2 requirements: 18 total
- Mapped to phases: 0
- Unmapped: 18 ⚠️

---
*Requirements defined: 2026-02-27*
*Last updated: 2026-02-27 after initial definition*
