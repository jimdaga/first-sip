"""CrewAI workflow executor with timeout protection."""

import asyncio
import importlib.util
import json
import logging
import re
from pathlib import Path
from typing import Any

from models import PluginRequest, PluginResult


logger = logging.getLogger(__name__)


class CrewExecutor:
    """Executes CrewAI workflows with timeout protection.

    Dynamically loads plugin crew definitions and wraps execution
    with asyncio.timeout to prevent thread leaks from long-running workflows.
    """

    def __init__(self, timeout_seconds: int = 300, plugin_dir: str = "../plugins"):
        """Initialize executor.

        Args:
            timeout_seconds: Maximum execution time per workflow
            plugin_dir: Base directory containing plugin subdirectories
        """
        self.timeout_seconds = timeout_seconds
        self.plugin_dir = plugin_dir

    async def execute(self, request: PluginRequest) -> PluginResult:
        """Execute a plugin's CrewAI workflow.

        Args:
            request: Plugin execution request

        Returns:
            PluginResult with status, output, or error
        """
        # Load crew for this plugin
        crew = self._load_crew(request.plugin_name, request.settings)
        if crew is None:
            return PluginResult(
                plugin_run_id=request.plugin_run_id,
                status="failed",
                error=f"Plugin crew not found: {request.plugin_name}"
            )

        # Execute with timeout wrapper
        try:
            async with asyncio.timeout(self.timeout_seconds):
                result = await crew.kickoff_async(inputs=request.settings)
                # Extract output (CrewAI result may have .raw attribute)
                raw_output = result.raw if hasattr(result, 'raw') else str(result)

                # Wrap as structured JSON before publishing to Redis Stream
                structured = self._wrap_output(raw_output)

                return PluginResult(
                    plugin_run_id=request.plugin_run_id,
                    status="completed",
                    output=structured  # valid JSON string
                )
        except asyncio.TimeoutError:
            logger.warning(
                f"Workflow timeout for plugin_run_id={request.plugin_run_id} "
                f"after {self.timeout_seconds}s"
            )
            return PluginResult(
                plugin_run_id=request.plugin_run_id,
                status="failed",
                error=f"Workflow exceeded {self.timeout_seconds}s timeout"
            )
        except Exception as e:
            logger.error(
                f"Workflow execution failed for plugin_run_id={request.plugin_run_id}: {e}",
                exc_info=True
            )
            return PluginResult(
                plugin_run_id=request.plugin_run_id,
                status="failed",
                error=str(e)
            )

    def _wrap_output(self, raw: str) -> str:
        """Wrap raw CrewAI output as structured JSON string.

        Args:
            raw: Raw Markdown string from CrewAI workflow

        Returns:
            JSON string with {"summary": "...", "sections": [...]}
        """
        if not raw:
            return json.dumps({
                "summary": "Briefing generated",
                "sections": [{"title": "Content", "content": ""}]
            })
        summary = self._extract_summary(raw)
        sections = self._build_sections(raw)
        return json.dumps({"summary": summary, "sections": sections})

    def _extract_summary(self, raw: str) -> str:
        """Extract first non-heading paragraph as summary (<=300 chars).

        Args:
            raw: Raw Markdown string

        Returns:
            Summary string truncated to 300 characters
        """
        paragraphs = [p.strip() for p in raw.split('\n\n') if p.strip()]
        for p in paragraphs:
            if not p.startswith('#'):
                return p[:300]
        return raw[:300]

    def _build_sections(self, raw: str) -> list:
        """Split Markdown output on ## headings into sections.

        Args:
            raw: Raw Markdown string

        Returns:
            List of {"title": str, "content": str} dicts
        """
        parts = re.split(r'^##\s+', raw, flags=re.MULTILINE)
        if len(parts) <= 1:
            return [{"title": "Briefing", "content": raw.strip()}]
        sections = []
        for part in parts[1:]:  # skip content before first heading
            lines = part.split('\n', 1)
            title = lines[0].strip()
            content = lines[1].strip() if len(lines) > 1 else ""
            sections.append({"title": title, "content": content})
        return sections

    def _load_crew(self, plugin_name: str, settings: dict[str, Any]):
        """Load a plugin's crew definition dynamically.

        Convention: Each plugin's crew/crew.py must export a
        create_crew(settings: dict) -> Crew factory function.

        Args:
            plugin_name: Name of plugin (subdirectory in plugin_dir)
            settings: Settings to pass to create_crew factory

        Returns:
            Crew instance or None if not found/failed to load
        """
        crew_dir = Path(self.plugin_dir) / plugin_name / "crew"
        crew_file = crew_dir / "crew.py"

        # Check if crew directory and file exist
        if not crew_dir.exists():
            logger.warning(f"Plugin '{plugin_name}' has no crew/ directory")
            return None

        if not crew_file.exists():
            logger.warning(f"Plugin '{plugin_name}' has no crew/crew.py file")
            return None

        # Dynamic import
        try:
            spec = importlib.util.spec_from_file_location(
                f"{plugin_name}_crew",
                crew_file
            )
            if spec is None or spec.loader is None:
                logger.error(f"Failed to create module spec for {crew_file}")
                return None

            module = importlib.util.module_from_spec(spec)
            spec.loader.exec_module(module)

            # Call create_crew factory function
            if not hasattr(module, 'create_crew'):
                logger.error(f"Plugin '{plugin_name}' crew.py missing create_crew() function")
                return None

            crew = module.create_crew(settings)
            logger.info(f"Loaded crew for plugin '{plugin_name}'")
            return crew

        except Exception as e:
            logger.error(f"Failed to load crew for plugin '{plugin_name}': {e}", exc_info=True)
            return None
