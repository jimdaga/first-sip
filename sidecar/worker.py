"""Redis Streams consumer for plugin execution requests."""

import asyncio
import logging
from typing import Any

import redis.asyncio as redis
from redis.exceptions import ResponseError

from models import (
    STREAM_PLUGIN_REQUESTS,
    STREAM_PLUGIN_RESULTS,
    GROUP_NAME,
    SCHEMA_VERSION,
    PluginRequest,
    PluginResult,
)
from executor import CrewExecutor


logger = logging.getLogger(__name__)


async def consume_plugin_requests(redis_client: redis.Redis, settings):
    """Main worker loop - consumes plugin requests from Redis Streams.

    Two-phase consumer pattern:
    1. Pending recovery - re-read unACKed messages from PEL
    2. New messages - read fresh messages from stream

    Args:
        redis_client: Redis async client
        settings: Application settings with crew_timeout_seconds, plugin_dir
    """
    # Create consumer group (idempotent - ignores BUSYGROUP error)
    try:
        await redis_client.xgroup_create(
            name=STREAM_PLUGIN_REQUESTS,
            groupname=GROUP_NAME,
            id='0',
            mkstream=True
        )
        logger.info(f"Created consumer group '{GROUP_NAME}' on stream '{STREAM_PLUGIN_REQUESTS}'")
    except ResponseError as e:
        if "BUSYGROUP" in str(e):
            logger.info(f"Consumer group '{GROUP_NAME}' already exists")
        else:
            raise

    logger.info(f"Starting worker loop (consumer: {settings.consumer_name})")

    try:
        while True:
            # Phase 1: Pending recovery - re-read unACKed messages from PEL
            try:
                pending_messages = await redis_client.xreadgroup(
                    groupname=GROUP_NAME,
                    consumername=settings.consumer_name,
                    streams={STREAM_PLUGIN_REQUESTS: '0'},
                    count=10,
                    block=100
                )

                if pending_messages:
                    logger.info(f"Found {len(pending_messages[0][1])} pending messages in PEL")
                    for stream_name, messages in pending_messages:
                        for msg_id, msg_data in messages:
                            await process_message(redis_client, msg_id, msg_data, settings)

            except Exception as e:
                logger.error(f"Error during pending recovery: {e}", exc_info=True)
                await asyncio.sleep(1)
                continue

            # Phase 2: New messages - read fresh messages from stream
            try:
                new_messages = await redis_client.xreadgroup(
                    groupname=GROUP_NAME,
                    consumername=settings.consumer_name,
                    streams={STREAM_PLUGIN_REQUESTS: '>'},
                    count=10,
                    block=5000
                )

                if new_messages:
                    for stream_name, messages in new_messages:
                        for msg_id, msg_data in messages:
                            await process_message(redis_client, msg_id, msg_data, settings)

            except Exception as e:
                logger.error(f"Error during new message processing: {e}", exc_info=True)
                await asyncio.sleep(1)
                continue

    except asyncio.CancelledError:
        logger.info("Worker loop cancelled, shutting down gracefully")
        raise


async def process_message(
    redis_client: redis.Redis,
    msg_id: str,
    msg_data: dict[str | bytes, Any],
    settings
):
    """Process a single plugin request message.

    Args:
        redis_client: Redis async client
        msg_id: Stream message ID
        msg_data: Message data (may have str or bytes keys)
        settings: Application settings
    """
    try:
        # Handle both str and bytes keys (redis-py behavior varies by decode_responses setting)
        payload_str = msg_data.get("payload") or msg_data.get(b"payload")
        if isinstance(payload_str, bytes):
            payload_str = payload_str.decode('utf-8')

        if not payload_str:
            logger.error(f"Message {msg_id} missing 'payload' field")
            await redis_client.xack(STREAM_PLUGIN_REQUESTS, GROUP_NAME, msg_id)
            return

        # Validate request payload
        try:
            request = PluginRequest.model_validate_json(payload_str)
        except Exception as e:
            logger.error(f"Invalid request payload in message {msg_id}: {e}")
            # ACK bad messages - don't retry
            await redis_client.xack(STREAM_PLUGIN_REQUESTS, GROUP_NAME, msg_id)
            return

        logger.info(f"Processing plugin_run_id={request.plugin_run_id} "
                   f"plugin={request.plugin_name}")

        # Execute CrewAI workflow
        executor = CrewExecutor(
            timeout_seconds=settings.crew_timeout_seconds,
            plugin_dir=settings.plugin_dir
        )
        result = await executor.execute(request)

        # Publish result to results stream
        await redis_client.xadd(
            STREAM_PLUGIN_RESULTS,
            {
                "payload": result.model_dump_json(),
                "schema_version": SCHEMA_VERSION
            }
        )

        # ACK request message (successful completion)
        await redis_client.xack(STREAM_PLUGIN_REQUESTS, GROUP_NAME, msg_id)

        logger.info(f"Completed plugin_run_id={request.plugin_run_id} "
                   f"status={result.status}")

    except Exception as e:
        # Don't ACK on processing error - message stays in PEL for retry
        logger.error(f"Error processing message {msg_id}: {e}", exc_info=True)
