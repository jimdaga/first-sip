# Phase 9: CrewAI Sidecar Integration - Research

**Researched:** 2026-02-14
**Domain:** Multi-agent AI workflows, Python-Go microservices communication, Redis Streams
**Confidence:** MEDIUM-HIGH

## Summary

Phase 9 requires integrating a FastAPI Python sidecar service to execute CrewAI multi-agent workflows, communicating with the existing Go application via Redis Streams. The research validates this architecture is production-ready for 2026.

CrewAI (v1.9.3, released Jan 30 2026) is actively maintained with Python 3.10-3.13 support and includes production-focused features like async execution (`kickoff_async`), state persistence (`@persist` decorator), and Flow-based workflow orchestration. The researcher → writer → reviewer pattern is a standard multi-agent design well-documented in the CrewAI ecosystem.

Redis Streams with consumer groups provide reliable at-least-once message delivery through `XREADGROUP` + `XACK` acknowledgment, with the existing Redis instance already running for Asynq. Both go-redis/v9 (Go) and redis-py (Python) have mature Stream support with consumer group operations.

FastAPI sidecar deployment follows established 2026 patterns: containerize with Docker, run Uvicorn with `--workers`, implement `/health` readiness/liveness endpoints, and scale independently via Kubernetes HPA. The timeout handling concern flagged in Phase 9 is real—a GitHub issue from Dec 2025 documents thread leaks in CrewAI's `_execute_with_timeout()`, requiring custom timeout wrapper using Python's `asyncio.timeout()` context manager.

**Primary recommendation:** Implement two-stream pattern (Go → `plugin:requests`, CrewAI → `plugin:results`), use consumer groups for reliability, wrap CrewAI execution with asyncio timeout, containerize FastAPI with multi-worker Uvicorn, deploy as separate K8s Deployment with HPA.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| CrewAI | 1.9.3+ | Multi-agent orchestration framework | Official production-ready framework, actively maintained (Jan 2026 release), Flow-based workflows |
| FastAPI | 0.115+ | Python async web framework | Industry standard for Python microservices (2026), async-native, auto-docs, type validation |
| redis-py | 6.1.0+ | Python Redis client | Official Python client, mature Stream support, consumer group operations |
| github.com/redis/go-redis/v9 | v9.7.0+ | Go Redis client | Official Go client, full Streams API, typed errors, OpenTelemetry support |
| Uvicorn | 0.32+ | ASGI server | Production ASGI server, built-in multi-worker support, FastAPI recommendation |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Pydantic | 2.x | Data validation | Always—FastAPI dependency, use for state management in Flows |
| asyncio | stdlib | Async execution, timeouts | Always—for `kickoff_async`, timeout wrappers, worker loop |
| slog | Go stdlib | Structured logging | Always—consistency with existing Go worker logging |
| OpenTelemetry | Latest | Tracing, metrics | Production deployments—go-redis has built-in support |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| CrewAI | LangGraph | More low-level control but steeper learning curve, less opinionated |
| Redis Streams | RabbitMQ | Feature-rich but new infrastructure, heavier operational overhead |
| FastAPI | Flask | Simpler but no async native, manual typing, slower performance |
| Uvicorn multi-worker | Gunicorn + Uvicorn workers | More complex orchestration, Uvicorn `--workers` simpler (2026 best practice) |

**Installation:**

**Python sidecar:**
```bash
# In Python sidecar directory (e.g., ./sidecar or ./crewai-executor)
uv pip install 'crewai>=1.9.3'
uv pip install 'fastapi>=0.115.0'
uv pip install 'uvicorn[standard]>=0.32.0'
uv pip install 'redis>=6.1.0'
```

**Go application:**
```bash
# go-redis/v9 (Redis Streams client)
go get github.com/redis/go-redis/v9
```

## Architecture Patterns

### Recommended Project Structure

```
first-sip/
├── cmd/
│   └── server/
│       └── main.go                    # Existing Go server
├── internal/
│   ├── worker/
│   │   ├── worker.go                  # Existing Asynq worker
│   │   └── crewai_publisher.go        # NEW: Publish to Redis Streams
│   ├── plugins/
│   │   ├── registry.go                # Existing plugin registry
│   │   └── executor.go                # NEW: CrewAI execution coordinator
│   └── streams/
│       ├── producer.go                # NEW: Redis Streams producer wrapper
│       └── consumer.go                # NEW: Redis Streams consumer wrapper
├── plugins/
│   └── daily-news-digest/
│       ├── plugin.yaml                # Existing metadata
│       ├── settings.schema.json       # Existing settings schema
│       └── crew/                      # CrewAI workflow definition
│           ├── main.py                # Crew entrypoint
│           ├── crew.py                # @CrewBase class
│           └── config/
│               ├── agents.yaml        # Agent definitions
│               └── tasks.yaml         # Task definitions
└── sidecar/                           # NEW: FastAPI Python service
    ├── Dockerfile
    ├── pyproject.toml                 # uv dependency management
    ├── main.py                        # FastAPI app with health endpoints
    ├── worker.py                      # Redis Streams consumer loop
    └── executor.py                    # CrewAI workflow executor with timeout
```

### Pattern 1: Two-Stream Communication

**What:** Separate Redis Streams for requests (Go → CrewAI) and results (CrewAI → Go)

**When to use:** Always for this architecture—decouples producers/consumers, simplifies error handling

**Example:**
```go
// Go producer (internal/streams/producer.go)
// Source: go-redis v9 documentation + research synthesis
package streams

import (
    "context"
    "encoding/json"
    "github.com/redis/go-redis/v9"
)

const (
    StreamPluginRequests = "plugin:requests"
    StreamPluginResults  = "plugin:results"
)

type PluginRequest struct {
    PluginRunID string                 `json:"plugin_run_id"`
    PluginName  string                 `json:"plugin_name"`
    UserID      uint                   `json:"user_id"`
    Settings    map[string]interface{} `json:"settings"`
}

func PublishPluginRequest(ctx context.Context, rdb *redis.Client, req PluginRequest) (string, error) {
    data, err := json.Marshal(req)
    if err != nil {
        return "", err
    }

    result := rdb.XAdd(ctx, &redis.XAddArgs{
        Stream: StreamPluginRequests,
        ID:     "*",  // Auto-generate
        Values: map[string]interface{}{
            "payload": string(data),
        },
    })

    return result.Result()
}
```

```python
# Python consumer (sidecar/worker.py)
# Source: redis-py Stream examples + research synthesis
import asyncio
import json
import redis.asyncio as redis
from typing import Optional

STREAM_PLUGIN_REQUESTS = "plugin:requests"
STREAM_PLUGIN_RESULTS = "plugin:results"
GROUP_NAME = "crewai-workers"
CONSUMER_NAME = "worker-01"  # Use hostname or unique ID in production

async def consume_plugin_requests(redis_client: redis.Redis):
    """Worker loop: consume plugin execution requests from Redis Stream."""

    # Create consumer group if not exists
    try:
        await redis_client.xgroup_create(
            STREAM_PLUGIN_REQUESTS,
            GROUP_NAME,
            id='0',  # Start from beginning
            mkstream=True
        )
    except redis.ResponseError as e:
        if "BUSYGROUP" not in str(e):
            raise

    while True:
        try:
            # Phase 1: Process pending messages (previously assigned but not ACKed)
            pending = await redis_client.xreadgroup(
                groupname=GROUP_NAME,
                consumername=CONSUMER_NAME,
                streams={STREAM_PLUGIN_REQUESTS: '0'},
                count=10,
                block=100  # 100ms timeout
            )

            if pending:
                for stream_name, messages in pending:
                    for msg_id, msg_data in messages:
                        await process_message(redis_client, msg_id, msg_data)

            # Phase 2: Read new messages
            messages = await redis_client.xreadgroup(
                groupname=GROUP_NAME,
                consumername=CONSUMER_NAME,
                streams={STREAM_PLUGIN_REQUESTS: '>'},  # New messages only
                count=10,
                block=5000  # 5 second timeout
            )

            if messages:
                for stream_name, msgs in messages:
                    for msg_id, msg_data in msgs:
                        await process_message(redis_client, msg_id, msg_data)

        except Exception as e:
            print(f"Error in consumer loop: {e}")
            await asyncio.sleep(1)
```

### Pattern 2: CrewAI Workflow with Timeout Wrapper

**What:** Wrap CrewAI `kickoff_async()` with `asyncio.timeout()` to prevent thread leaks

**When to use:** Always—CrewAI's built-in timeout has known issues (GitHub #4135)

**Example:**
```python
# sidecar/executor.py
# Source: Python asyncio docs + CrewAI issue #4135 research
import asyncio
from crewai import Crew
from typing import Dict, Any

class CrewExecutor:
    def __init__(self, timeout_seconds: int = 300):
        self.timeout_seconds = timeout_seconds

    async def execute_with_timeout(
        self,
        crew: Crew,
        inputs: Dict[str, Any]
    ) -> Dict[str, Any]:
        """
        Execute CrewAI workflow with robust timeout handling.

        CrewAI's built-in timeout has thread leak issues when tasks are
        already running. This wrapper uses asyncio.timeout() for proper
        cancellation.
        """
        try:
            async with asyncio.timeout(self.timeout_seconds):
                result = await crew.kickoff_async(inputs=inputs)
                return {
                    "status": "completed",
                    "output": result.raw if hasattr(result, 'raw') else str(result)
                }
        except asyncio.TimeoutError:
            return {
                "status": "failed",
                "error": f"Workflow exceeded {self.timeout_seconds}s timeout"
            }
        except Exception as e:
            return {
                "status": "failed",
                "error": str(e)
            }
```

### Pattern 3: FastAPI Health Endpoints for Kubernetes

**What:** Separate `/health/live` and `/health/ready` endpoints

**When to use:** Always for K8s deployments—liveness checks process health, readiness checks dependency availability

**Example:**
```python
# sidecar/main.py
# Source: FastAPI health check best practices (2026 research)
from fastapi import FastAPI, status
from fastapi.responses import JSONResponse
import redis.asyncio as redis

app = FastAPI(title="CrewAI Executor Sidecar")

# Global health state
redis_client: redis.Redis = None

@app.on_event("startup")
async def startup():
    global redis_client
    redis_client = redis.from_url("redis://localhost:6379")

@app.get("/health/live", status_code=200)
async def liveness():
    """
    Liveness probe: Is the process running?
    K8s will restart pod if this fails.
    """
    return {"status": "alive"}

@app.get("/health/ready", status_code=200)
async def readiness():
    """
    Readiness probe: Can the service handle traffic?
    K8s will remove from load balancer if this fails.
    """
    try:
        # Check Redis connection
        await redis_client.ping()
        return {"status": "ready", "redis": "connected"}
    except Exception as e:
        return JSONResponse(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            content={"status": "not_ready", "error": str(e)}
        )
```

### Pattern 4: Plugin-Specific Crew Loading

**What:** Load CrewAI agents/tasks from plugin directory YAML files at runtime

**When to use:** Always—enables per-plugin workflows without code changes

**Example:**
```python
# sidecar/executor.py (continued)
# Source: CrewAI YAML configuration patterns
from crewai import Agent, Task, Crew, Process
from crewai.project import CrewBase, agent, task, crew
from pathlib import Path
import yaml

@CrewBase
class NewsDigestCrew:
    """Dynamically loaded from plugins/daily-news-digest/crew/config/"""

    agents_config = 'config/agents.yaml'
    tasks_config = 'config/tasks.yaml'

    @agent
    def researcher(self) -> Agent:
        return Agent(
            config=self.agents_config['researcher'],
            verbose=True
        )

    @agent
    def writer(self) -> Agent:
        return Agent(
            config=self.agents_config['writer']
        )

    @agent
    def reviewer(self) -> Agent:
        return Agent(
            config=self.agents_config['reviewer']
        )

    @task
    def research_task(self) -> Task:
        return Task(
            config=self.tasks_config['research_task'],
            agent=self.researcher()
        )

    @task
    def write_task(self) -> Task:
        return Task(
            config=self.tasks_config['write_task'],
            agent=self.writer(),
            context=[self.research_task()]  # Sequential dependency
        )

    @task
    def review_task(self) -> Task:
        return Task(
            config=self.tasks_config['review_task'],
            agent=self.reviewer(),
            context=[self.write_task()]
        )

    @crew
    def crew(self) -> Crew:
        return Crew(
            agents=self.agents,
            tasks=self.tasks,
            process=Process.sequential,
            verbose=True
        )

# Dynamic plugin loading
def load_plugin_crew(plugin_name: str, plugin_dir: Path) -> Crew:
    """Load Crew from plugin directory."""
    crew_module = NewsDigestCrew()  # In production: dynamic import
    return crew_module.crew()
```

### Anti-Patterns to Avoid

- **Blocking XREADGROUP exhausting connection pool:** Use async Redis client (`redis.asyncio`) in Python, set reasonable block timeout (5s recommended), separate client for health checks
- **Hardcoding agent definitions in Python:** Always use YAML config files in plugin directories—enables plugin authors to modify workflows without touching sidecar code
- **Single consumer for all plugins:** Use plugin-specific consumer groups or route by plugin name—prevents head-of-line blocking
- **No timeout on CrewAI execution:** CrewAI's built-in timeout leaks threads—always wrap with `asyncio.timeout()`
- **Shared K8s Deployment for Go + Python:** Deploy as separate Deployments—Python needs different scaling characteristics (CPU-bound AI vs I/O-bound API)

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Multi-agent orchestration | Custom agent loop with LLM API calls | CrewAI Crew + Agents + Tasks | Context sharing, error recovery, state management, human-in-the-loop built-in |
| Redis Streams consumer groups | Custom message tracking, manual ACK/retry | `XREADGROUP` + `XACK` + `XPENDING` | Server-side tracking, at-least-once delivery, automatic PEL (pending entries list) |
| Async timeout with cleanup | ThreadPoolExecutor.shutdown() after timeout | `asyncio.timeout()` context manager | Proper task cancellation, no thread leaks (CrewAI issue #4135) |
| FastAPI multi-worker | Custom process forking, signal handling | Uvicorn `--workers` flag | Built-in graceful shutdown, worker management, preload optimization |
| LLM prompt chaining | Manual prompt construction, response parsing | CrewAI Tasks with `context=[prev_task]` | Automatic output passing, type validation, retry on failure |

**Key insight:** CrewAI handles the complexity of multi-agent coordination (context sharing between agents, sequential/hierarchical task execution, LLM retries, output parsing). Redis Streams handle the complexity of distributed messaging (consumer groups, pending message tracking, at-least-once delivery guarantees, backpressure via consumer count). Don't reimplement either.

## Common Pitfalls

### Pitfall 1: CrewAI Timeout Thread Leaks

**What goes wrong:** Using CrewAI's built-in `Agent._execute_with_timeout()` or relying on default timeout causes orphaned threads under load, eventually exhausting resources

**Why it happens:** When `future.cancel()` is called after task starts executing, it returns False and continues running. ThreadPoolExecutor shuts down but thread persists (GitHub issue #4135, Dec 2025)

**How to avoid:** Wrap `crew.kickoff_async()` with `asyncio.timeout()` context manager, handle TimeoutError explicitly, return structured error response

**Warning signs:** Thread count increasing over time, memory leaks in Python process, workers hanging after timeout, `ThreadPoolExecutor` warnings in logs

### Pitfall 2: Forgetting to ACK Messages After Processing

**What goes wrong:** Messages remain in Pending Entries List (PEL) indefinitely, get re-delivered on consumer restart, cause duplicate work

**Why it happens:** Exception during processing skips `XACK` call, developer assumes successful read = processed

**How to avoid:** Always `XACK` in try/finally block, or use two-phase pattern (process → publish result → ACK both streams atomically)

**Warning signs:** `XPENDING` shows growing backlog, same plugin_run_id processed multiple times, duplicate briefings created

### Pitfall 3: Blocking XREADGROUP Exhausting Connection Pool

**What goes wrong:** Every worker thread/coroutine holds a Redis connection waiting for messages, pool exhausted, health checks time out

**Why it happens:** Default redis-py connection pool size (max_connections) is limited, blocking `XREADGROUP` holds connection during block period

**How to avoid:** Use `redis.asyncio` with asyncio event loop (shares connections), set reasonable block timeout (5s not 60s), separate Redis client for health checks

**Warning signs:** Health endpoint timeouts, "connection pool exhausted" errors, workers idle but Redis connections maxed

### Pitfall 4: No Schema Validation on Stream Messages

**What goes wrong:** Go publishes message, Python can't parse, error not caught until processing, message stuck in PEL

**Why it happens:** Stream values are string maps, no type enforcement, version mismatches between Go publisher and Python consumer

**How to avoid:** Use Pydantic models in Python for parsing, version messages with `schema_version` field, validate early and NACK/DLQ invalid messages

**Warning signs:** JSON decode errors in consumer, "missing required field" errors, messages with old format stuck in PEL

### Pitfall 5: Single Kubernetes Deployment for Both Services

**What goes wrong:** Go and Python scaled together, Python CPU spikes force scaling Go instances, resource waste, slow rollouts

**Why it happens:** Sidecar pattern confusion—true sidecars share pod, but this is a separate service that communicates via Redis

**How to avoid:** Deploy as separate K8s Deployments with independent HPA (Horizontal Pod Autoscaler) targeting different metrics (Go: request rate, Python: CPU/GPU)

**Warning signs:** Over-provisioned Go pods during AI processing, under-provisioned Python during request spikes, deployment rollout affects both services

## Code Examples

Verified patterns from official sources and research:

### Go: Publishing Plugin Execution Request
```go
// Source: go-redis/v9 pkg.go.dev + Redis Streams documentation
package streams

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
)

// Publisher wraps Redis client for stream operations
type Publisher struct {
    rdb *redis.Client
}

func NewPublisher(redisURL string) (*Publisher, error) {
    opt, err := redis.ParseURL(redisURL)
    if err != nil {
        return nil, fmt.Errorf("invalid Redis URL: %w", err)
    }

    return &Publisher{
        rdb: redis.NewClient(opt),
    }, nil
}

// PublishPluginRequest adds execution request to stream, returns message ID
func (p *Publisher) PublishPluginRequest(ctx context.Context, req PluginRequest) (string, error) {
    payload, err := json.Marshal(req)
    if err != nil {
        return "", fmt.Errorf("failed to marshal request: %w", err)
    }

    msgID, err := p.rdb.XAdd(ctx, &redis.XAddArgs{
        Stream: StreamPluginRequests,
        MaxLen: 10000,  // Trim to last 10k messages (approximate)
        Approx: true,   // Approximate trimming for performance
        ID:     "*",    // Auto-generate timestamp-based ID
        Values: map[string]interface{}{
            "payload":        string(payload),
            "published_at":   time.Now().Unix(),
            "schema_version": "v1",
        },
    }).Result()

    if err != nil {
        return "", fmt.Errorf("XADD failed: %w", err)
    }

    return msgID, nil
}
```

### Go: Consuming Plugin Results
```go
// Source: go-redis/v9 consumer group pattern + research synthesis
package streams

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
)

type ResultConsumer struct {
    rdb          *redis.Client
    groupName    string
    consumerName string
}

func NewResultConsumer(redisURL, consumerName string) (*ResultConsumer, error) {
    opt, err := redis.ParseURL(redisURL)
    if err != nil {
        return nil, err
    }

    rdb := redis.NewClient(opt)

    // Create consumer group if not exists
    err = rdb.XGroupCreateMkStream(context.Background(),
        StreamPluginResults, "go-workers", "0").Err()
    if err != nil && !errors.Is(err, redis.Nil) {
        // Ignore BUSYGROUP error (group already exists)
        if err.Error() != "BUSYGROUP Consumer Group name already exists" {
            return nil, err
        }
    }

    return &ResultConsumer{
        rdb:          rdb,
        groupName:    "go-workers",
        consumerName: consumerName,
    }, nil
}

// ConsumeResults reads and processes plugin results
func (c *ResultConsumer) ConsumeResults(ctx context.Context, handler func(PluginResult) error) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        // Read from consumer group
        streams, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
            Group:    c.groupName,
            Consumer: c.consumerName,
            Streams:  []string{StreamPluginResults, ">"}, // ">" = only new messages
            Count:    10,
            Block:    5 * time.Second,
        }).Result()

        if err != nil {
            if errors.Is(err, redis.Nil) {
                // No messages, continue
                continue
            }
            return fmt.Errorf("XREADGROUP failed: %w", err)
        }

        // Process messages
        for _, stream := range streams {
            for _, msg := range stream.Messages {
                if err := c.processMessage(ctx, msg, handler); err != nil {
                    // Log error but continue processing other messages
                    fmt.Printf("Error processing message %s: %v\n", msg.ID, err)
                    continue
                }

                // ACK message
                if err := c.rdb.XAck(ctx, StreamPluginResults, c.groupName, msg.ID).Err(); err != nil {
                    fmt.Printf("Error ACKing message %s: %v\n", msg.ID, err)
                }
            }
        }
    }
}

func (c *ResultConsumer) processMessage(ctx context.Context, msg redis.XMessage, handler func(PluginResult) error) error {
    payloadStr, ok := msg.Values["payload"].(string)
    if !ok {
        return fmt.Errorf("missing payload field")
    }

    var result PluginResult
    if err := json.Unmarshal([]byte(payloadStr), &result); err != nil {
        return fmt.Errorf("invalid JSON payload: %w", err)
    }

    return handler(result)
}
```

### Python: Processing Messages with Timeout
```python
# Source: redis-py docs, asyncio timeout pattern, CrewAI research synthesis
import asyncio
import json
from typing import Dict, Any
import redis.asyncio as redis
from executor import CrewExecutor, load_plugin_crew

async def process_message(
    redis_client: redis.Redis,
    msg_id: str,
    msg_data: Dict[str, Any]
) -> None:
    """Process single plugin execution request with timeout."""
    try:
        # Parse message
        payload = json.loads(msg_data[b'payload'])
        plugin_run_id = payload['plugin_run_id']
        plugin_name = payload['plugin_name']
        user_settings = payload['settings']

        print(f"Processing {plugin_run_id} for plugin {plugin_name}")

        # Load plugin-specific Crew
        crew = load_plugin_crew(plugin_name)

        # Execute with timeout
        executor = CrewExecutor(timeout_seconds=300)  # 5 minutes
        result = await executor.execute_with_timeout(
            crew=crew,
            inputs=user_settings
        )

        # Publish result to results stream
        result_payload = {
            "plugin_run_id": plugin_run_id,
            "status": result["status"],
            "output": result.get("output"),
            "error": result.get("error"),
        }

        await redis_client.xadd(
            STREAM_PLUGIN_RESULTS,
            {"payload": json.dumps(result_payload)}
        )

        # ACK original request message
        await redis_client.xack(
            STREAM_PLUGIN_REQUESTS,
            GROUP_NAME,
            msg_id
        )

        print(f"Completed {plugin_run_id}: {result['status']}")

    except Exception as e:
        print(f"Error processing {msg_id}: {e}")
        # Don't ACK on error—message stays in PEL for retry/inspection
        # Consider XACK + publish to DLQ after N retries
```

### CrewAI: Researcher → Writer → Reviewer Flow
```yaml
# Source: CrewAI YAML configuration patterns
# plugins/daily-news-digest/crew/config/agents.yaml
researcher:
  role: "Senior News Research Analyst"
  goal: "Discover and analyze breaking news in {topics} to identify stories relevant to user interests"
  backstory: >
    You are an experienced news analyst with expertise in {topics}.
    You excel at finding high-quality sources, fact-checking claims,
    and identifying stories that matter to discerning readers.
  verbose: true

writer:
  role: "News Digest Writer"
  goal: "Transform research findings into concise, engaging news summaries"
  backstory: >
    You are a skilled writer who crafts compelling news digests.
    You distill complex topics into clear, actionable summaries
    that respect the reader's time while delivering full context.
  verbose: true

reviewer:
  role: "Editorial Quality Reviewer"
  goal: "Ensure news digest meets quality standards for accuracy, clarity, and relevance"
  backstory: >
    You are a meticulous editor who ensures every digest is
    factually accurate, well-structured, and free of bias.
    You catch errors that others miss.
  verbose: false
```

```yaml
# plugins/daily-news-digest/crew/config/tasks.yaml
research_task:
  description: >
    Research breaking news in these topics: {topics}.
    Find 3-5 high-quality stories from reputable sources.
    For each story, note: headline, source, key facts, why it matters.
  expected_output: >
    JSON array of stories with fields: headline, source_url,
    summary (2-3 sentences), relevance_score (1-10)

write_task:
  description: >
    Using the research findings, write a news digest.
    Format: Brief intro paragraph, then 3-5 story summaries.
    Each summary: headline, 2-3 sentence explanation, source credit.
    Tone: Professional but conversational. Max 500 words total.
  expected_output: >
    Markdown-formatted news digest ready for email delivery

review_task:
  description: >
    Review the news digest for:
    - Factual accuracy (check claims against research)
    - Clarity (no jargon, clear explanations)
    - Formatting (proper Markdown, consistent style)
    - Bias detection (neutral tone maintained)

    If issues found, return revised version. If acceptable, approve as-is.
  expected_output: >
    Final approved news digest in Markdown format, with quality score (1-10)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Gunicorn + Uvicorn workers | Uvicorn `--workers` native | 2025-2026 | Simpler deployment, one less dependency, same performance |
| LangChain for agents | CrewAI standalone | 2024-present | Lighter weight, no LangChain dependency, role-based focus |
| Manual process forking | asyncio native multi-worker | Python 3.11+ (2023) | Better resource efficiency, async-native timeouts |
| Redis Lists for queues | Redis Streams for messaging | Redis 5.0 (2018), adopted widely 2023+ | Consumer groups, message history, better observability |
| Thread-based Python workers | Async Python workers | FastAPI/asyncio ecosystem 2020+ | Higher concurrency, lower memory footprint |

**Deprecated/outdated:**
- **CrewAI + LangChain:** CrewAI is now "completely independent of LangChain" (official docs). Old tutorials showing LangChain imports are outdated.
- **Gunicorn wrapper for Uvicorn:** Modern Uvicorn (0.30+) has `--workers` built-in. The `uvicorn-gunicorn-fastapi-docker` image is no longer recommended (FastAPI docs 2026).
- **Python 3.9:** CrewAI requires Python >=3.10 <3.14. Python 3.9 reached end-of-life Oct 2025.
- **Redis protocol 2 as default:** go-redis defaults to protocol 3 in v9, but docs show `Protocol: 2` in examples for compatibility. Use protocol 3 for new deployments.

## Open Questions

1. **Plugin-specific Python dependencies (e.g., custom tools for news scraping)**
   - What we know: Each plugin's `crew/` directory could have `requirements.txt`, CrewAI supports custom tools
   - What's unclear: Should sidecar pre-install all plugin dependencies or dynamically install? Security implications of dynamic installs?
   - Recommendation: Start with sidecar Dockerfile installing all plugin deps (rebuild on plugin add). Explore dynamic install in Phase 10+ if needed.

2. **Handling plugins without CrewAI workflows (simple scripts)**
   - What we know: Some plugins might just run a Python script, not need multi-agent orchestration
   - What's unclear: Should sidecar support non-Crew execution? Separate executor types?
   - Recommendation: Phase 9 focuses on CrewAI-capable plugins only. Defer simple script execution to future phase.

3. **LLM API key management (per-user vs shared, rotation)**
   - What we know: CrewAI agents call LLM APIs (OpenAI, Anthropic, etc.), need API keys
   - What's unclear: User-provided keys vs First Sip managed keys? Key encryption in transit/storage?
   - Recommendation: Phase 9 uses shared API keys (env vars in sidecar). User-provided keys deferred to Phase 10+ when user settings UI exists.

4. **Redis Streams retention policy (MAXLEN, TTL)**
   - What we know: `XADD` supports `MAXLEN` for capping stream size, prevents unbounded growth
   - What's unclear: Optimal retention for plugin:requests (short, messages consumed quickly) vs plugin:results (longer, Go worker might be down)?
   - Recommendation: Start with `MAXLEN ~10000 APPROX` for both streams. Monitor with Redis INFO and adjust based on message volume.

5. **Dead Letter Queue (DLQ) for failed messages**
   - What we know: Messages that fail processing stay in PEL, can be claimed and retried
   - What's unclear: After N failed retries, how to move to DLQ? Separate stream or database table?
   - Recommendation: Phase 9 relies on PEL and manual inspection via `XPENDING`. Implement automated DLQ in Phase 10 when error patterns clear.

## Sources

### Primary (HIGH confidence)

**CrewAI:**
- [CrewAI PyPI](https://pypi.org/project/crewai/) - Version 1.9.3, Jan 30 2026, Python requirements
- [CrewAI Production Architecture](https://docs.crewai.com/en/concepts/production-architecture) - `kickoff_async`, `@persist`, error handling
- [CrewAI GitHub](https://github.com/crewAIInc/crewAI) - Installation, project structure, Python 3.10-3.14 requirement
- [CrewAI Issue #4135](https://github.com/crewAIInc/crewAI/issues/4135) - Timeout thread leak bug (Dec 2025)

**Redis Streams:**
- [Redis Streams Documentation](https://redis.io/docs/latest/develop/data-types/streams/) - Consumer groups, XREADGROUP, XACK patterns
- [XREADGROUP Command](https://redis.io/docs/latest/commands/xreadgroup/) - Syntax, `>` vs `0` for PEL
- [XACK Command](https://redis.io/docs/latest/commands/xack/) - Acknowledgment semantics
- [redis-py Stream Examples](https://redis.readthedocs.io/en/v6.1.0/examples/redis-stream-example.html) - Python consumer group code
- [go-redis Guide](https://redis.io/docs/latest/develop/clients/go/) - Installation, connection patterns
- [go-redis v9 pkg.go.dev](https://pkg.go.dev/github.com/redis/go-redis/v9) - XAddArgs, XReadGroupArgs, XAck documentation

**FastAPI:**
- [FastAPI Docker Documentation](https://fastapi.tiangolo.com/deployment/docker/) - Official deployment guide
- [FastAPI Health Check Patterns](https://www.index.dev/blog/how-to-implement-health-check-in-python) - Liveness vs readiness
- [Python asyncio timeout](https://docs.python.org/3/library/asyncio-task.html) - asyncio.timeout() context manager (3.11+)

### Secondary (MEDIUM confidence)

- [CrewAI YAML Configuration](https://codesignal.com/learn/courses/getting-started-with-crewai-agents-and-tasks/lessons/configuring-crewai-agents-and-tasks-with-yaml-files) - agents.yaml, tasks.yaml patterns
- [FastAPI Deployment Guide 2026](https://www.zestminds.com/blog/fastapi-deployment-guide/) - Uvicorn workers, production setup
- [Redis Streams Charles Leifer](https://charlesleifer.com/blog/redis-streams-with-python/) - Walrus library patterns (conceptual)
- [K8s FastAPI Scaling](https://agfianf.github.io/blog/2025/05/02/hands-on-auto-scaling-fastapi-with-kubernetes/) - HPA configuration
- [Redis Consumer Groups Scalability](https://oneuptime.com/blog/post/2026-02-09-redis-consumer-groups-scalable/view) - Two-phase pattern (pending + new)

### Tertiary (LOW confidence, flagged for validation)

- [CrewAI Multi-Agent Patterns](https://www.blog.brightcoding.dev/2026/02/13/crewai-the-revolutionary-multi-agent-framework) - General ecosystem overview, needs official docs verification
- [FastAPI Microservices 2025](https://talent500.com/blog/fastapi-microservices-python-api-design-patterns-2025/) - Design patterns, conceptual not code

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - CrewAI 1.9.3 confirmed via PyPI, go-redis/v9 confirmed via pkg.go.dev, FastAPI patterns verified via official docs
- Architecture: MEDIUM-HIGH - Redis Streams pattern verified via official Redis docs, two-stream pattern is synthesis but well-supported. Timeout wrapper pattern verified via Python docs + CrewAI issue tracker
- Pitfalls: MEDIUM - Thread leak confirmed via GitHub issue, other pitfalls synthesized from Redis docs + community patterns (need validation in implementation)

**Research date:** 2026-02-14
**Valid until:** 2026-03-31 (45 days—CrewAI is fast-moving, Python ecosystem stable)

**Next steps for planner:**
- Break Phase 9 into 3 plans: (1) Redis Streams infrastructure in Go, (2) FastAPI sidecar with health endpoints + worker loop, (3) CrewAI integration with example plugin
- Flag timeout wrapper as critical verification task (needs testing under load)
- Note Python dependency management strategy (all plugins in sidecar Dockerfile vs dynamic install) needs Phase 10 decision
