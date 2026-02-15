"""CrewAI workflow executor with timeout protection."""

import asyncio
import importlib.util
import logging
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
                output = result.raw if hasattr(result, 'raw') else str(result)
                return PluginResult(
                    plugin_run_id=request.plugin_run_id,
                    status="completed",
                    output=output
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
