"""FastAPI sidecar service for CrewAI plugin execution."""

import asyncio
import logging
import os
import socket
from contextlib import asynccontextmanager

import redis.asyncio as redis
from fastapi import FastAPI
from fastapi.responses import JSONResponse
import uvicorn

from worker import consume_plugin_requests


# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class Settings:
    """Environment-based configuration."""

    def __init__(self):
        self.redis_url = os.getenv("REDIS_URL", "redis://localhost:6379")
        self.plugin_dir = os.getenv("PLUGIN_DIR", "../plugins")
        self.crew_timeout_seconds = int(os.getenv("CREW_TIMEOUT_SECONDS", "300"))
        self.consumer_name = socket.gethostname()

        logger.info(f"Settings loaded: redis_url={self.redis_url}, "
                   f"plugin_dir={self.plugin_dir}, "
                   f"crew_timeout_seconds={self.crew_timeout_seconds}, "
                   f"consumer_name={self.consumer_name}")


settings = Settings()


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Startup and shutdown lifecycle."""
    # Startup: create Redis clients, start worker task
    logger.info("Starting sidecar service...")

    # Worker Redis client for stream consumption
    app.state.redis = redis.from_url(settings.redis_url, decode_responses=True)

    # Separate Redis client for health checks (prevents blocking)
    app.state.health_redis = redis.from_url(settings.redis_url, decode_responses=True)

    # Start background worker task
    app.state.worker_task = asyncio.create_task(
        consume_plugin_requests(app.state.redis, settings)
    )

    logger.info("Sidecar service started")
    yield

    # Shutdown: cancel worker, close Redis connections
    logger.info("Shutting down sidecar service...")
    app.state.worker_task.cancel()
    try:
        await app.state.worker_task
    except asyncio.CancelledError:
        pass

    await app.state.redis.aclose()
    await app.state.health_redis.aclose()
    logger.info("Sidecar service stopped")


app = FastAPI(
    title="First Sip CrewAI Sidecar",
    version="0.1.0",
    lifespan=lifespan
)


@app.get("/health/live")
async def health_live():
    """Liveness probe - always returns 200 if process is running."""
    return {"status": "alive"}


@app.get("/health/ready")
async def health_ready():
    """Readiness probe - checks Redis connectivity."""
    try:
        # Use separate health check client to avoid blocking worker
        await app.state.health_redis.ping()
        return {"status": "ready", "redis": "connected"}
    except Exception as e:
        logger.error(f"Readiness check failed: {e}")
        return JSONResponse(
            status_code=503,
            content={"status": "not_ready", "error": str(e)}
        )


def run():
    """Entry point for uvicorn server."""
    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=8000,
        log_level="info"
    )


if __name__ == "__main__":
    run()
