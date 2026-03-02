"""CrewAI workflow executor with timeout protection."""

import asyncio
import importlib.util
import json
import logging
import os
import re
from pathlib import Path
from typing import Any

from crewai import LLM
from crewai_tools import TavilySearchTool
from langchain_community.tools import DuckDuckGoSearchRun
from crewai.tools import tool

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

    def _extract_llm(self, settings: dict) -> "LLM | None":
        """Construct a crewai.LLM from injected credentials.

        Args:
            settings: Raw settings dict including credential keys.

        Returns:
            LLM instance or None if credentials are missing.
        """
        api_key = settings.get("_llm_api_key")
        model = settings.get("_llm_model")
        if not api_key or not model:
            return None
        kwargs: dict[str, Any] = {"model": model, "api_key": api_key}
        if model.startswith("anthropic/"):
            kwargs["max_tokens"] = 4096
        return LLM(**kwargs)

    def _extract_search_tool(self, settings: dict):
        """Select Tavily or DuckDuckGo search tool based on available credentials.

        Args:
            settings: Raw settings dict including credential keys.

        Returns:
            TavilySearchTool instance or DuckDuckGo @tool wrapper.
        """
        tavily_key = settings.get("_tavily_api_key")
        if tavily_key:
            os.environ["TAVILY_API_KEY"] = tavily_key
            return TavilySearchTool()

        # DuckDuckGo fallback — no API key required
        ddg = DuckDuckGoSearchRun()

        @tool("Web Search")
        def web_search(query: str) -> str:
            """Search the web for current information."""
            return ddg.run(query)

        return web_search

    def _clean_settings(self, settings: dict) -> dict:
        """Strip credential keys from settings before passing to crew inputs.

        Removes any key starting with '_' (e.g. _llm_api_key, _tavily_api_key,
        _llm_model) to prevent credential leakage into agent template variables.

        Args:
            settings: Raw settings dict.

        Returns:
            Dict with credential keys removed.
        """
        return {k: v for k, v in settings.items() if not k.startswith("_")}

    async def execute(self, request: PluginRequest) -> PluginResult:
        """Execute a plugin's CrewAI workflow.

        Args:
            request: Plugin execution request

        Returns:
            PluginResult with status, output, or error
        """
        # Extract credentials and build LLM + search tool
        llm = self._extract_llm(request.settings)
        search_tool = self._extract_search_tool(request.settings)
        clean_settings = self._clean_settings(request.settings)

        # Load crew for this plugin
        crew = self._load_crew(request.plugin_name, clean_settings, llm=llm, search_tool=search_tool)
        if crew is None:
            return PluginResult(
                plugin_run_id=request.plugin_run_id,
                status="failed",
                error=f"Plugin crew not found: {request.plugin_name}"
            )

        # Execute with timeout wrapper
        try:
            async with asyncio.timeout(self.timeout_seconds):
                result = await crew.kickoff_async(inputs=clean_settings)
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

    def _load_crew(self, plugin_name: str, settings: dict[str, Any], llm=None, search_tool=None):
        """Load a plugin's crew definition dynamically.

        Convention: Each plugin's crew/crew.py must export a
        create_crew(settings: dict, llm=None, search_tool=None) -> Crew factory function.

        Args:
            plugin_name: Name of plugin (subdirectory in plugin_dir)
            settings: Clean settings (credentials already stripped) to pass to create_crew
            llm: crewai.LLM instance or None
            search_tool: Search tool instance or None

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

            crew = module.create_crew(settings, llm=llm, search_tool=search_tool)
            logger.info(f"Loaded crew for plugin '{plugin_name}'")
            return crew

        except Exception as e:
            logger.error(f"Failed to load crew for plugin '{plugin_name}': {e}", exc_info=True)
            return None
