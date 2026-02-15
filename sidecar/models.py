"""Pydantic models for Redis Streams message payloads."""

from typing import Any, Literal
from pydantic import BaseModel


# Stream constants - must match Go side exactly
STREAM_PLUGIN_REQUESTS = "plugin:requests"
STREAM_PLUGIN_RESULTS = "plugin:results"
GROUP_NAME = "crewai-workers"
SCHEMA_VERSION = "v1"


class PluginRequest(BaseModel):
    """Request message published by Go app to plugin:requests stream."""

    plugin_run_id: str
    plugin_name: str
    user_id: int
    settings: dict[str, Any]


class PluginResult(BaseModel):
    """Result message published by Python sidecar to plugin:results stream."""

    plugin_run_id: str
    status: Literal["completed", "failed"]
    output: str | None = None
    error: str | None = None
