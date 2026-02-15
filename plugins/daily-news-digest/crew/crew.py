"""Daily News Digest CrewAI workflow.

Implements the researcher -> writer -> reviewer multi-agent pattern.
Loaded dynamically by the sidecar executor via the create_crew() factory.
"""
import os
from pathlib import Path
from crewai import Agent, Task, Crew, Process
from crewai.project import CrewBase, agent, task, crew


@CrewBase
class NewsDigestCrew:
    """Multi-agent crew for generating daily news digests."""

    agents_config = str(Path(__file__).parent / "config" / "agents.yaml")
    tasks_config = str(Path(__file__).parent / "config" / "tasks.yaml")

    @agent
    def researcher(self) -> Agent:
        return Agent(config=self.agents_config["researcher"], verbose=True)

    @agent
    def writer(self) -> Agent:
        return Agent(config=self.agents_config["writer"], verbose=True)

    @agent
    def reviewer(self) -> Agent:
        return Agent(config=self.agents_config["reviewer"], verbose=False)

    @task
    def research_task(self) -> Task:
        return Task(config=self.tasks_config["research_task"], agent=self.researcher())

    @task
    def write_task(self) -> Task:
        return Task(
            config=self.tasks_config["write_task"],
            agent=self.writer(),
            context=[self.research_task()],
        )

    @task
    def review_task(self) -> Task:
        return Task(
            config=self.tasks_config["review_task"],
            agent=self.reviewer(),
            context=[self.write_task()],
        )

    @crew
    def crew(self) -> Crew:
        return Crew(
            agents=self.agents,
            tasks=self.tasks,
            process=Process.sequential,
            verbose=True,
        )


def create_crew(settings: dict) -> Crew:
    """Factory function called by sidecar executor.

    This is the contract between plugin crew definitions and the sidecar:
    every plugin's crew/crew.py must export create_crew(settings) -> Crew.

    Args:
        settings: User plugin settings (topics, summary_length, etc.)

    Returns:
        Configured Crew instance ready for kickoff_async(inputs=settings).
    """
    return NewsDigestCrew().crew()
