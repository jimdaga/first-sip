"""Daily News Digest CrewAI workflow.

Implements the researcher -> writer -> reviewer multi-agent pattern.
Loaded dynamically by the sidecar executor via the create_crew() factory.
"""
from crewai import Agent, Task, Crew, Process


def create_crew(settings: dict, llm=None, search_tool=None) -> Crew:
    """Factory function called by sidecar executor.

    Args:
        settings: Clean user plugin settings (topics, summary_length, etc.) — no credentials.
        llm: crewai.LLM instance or None (falls back to CrewAI default).
        search_tool: TavilySearchTool or DuckDuckGo @tool wrapper, or None.

    Returns:
        Configured Crew instance ready for kickoff_async(inputs=settings).
    """
    tools = [search_tool] if search_tool else []

    researcher = Agent(
        role="Senior News Research Analyst",
        goal="Discover and analyze breaking news in {topics} to identify stories relevant to user interests",
        backstory=(
            "You are an experienced news analyst with expertise in {topics}. "
            "You excel at finding high-quality sources, fact-checking claims, "
            "and identifying stories that matter to discerning readers."
        ),
        tools=tools,
        llm=llm,
        verbose=True,
    )

    writer = Agent(
        role="News Digest Writer",
        goal="Transform research findings into concise, engaging news summaries",
        backstory=(
            "You are a skilled writer who crafts compelling news digests. "
            "You distill complex topics into clear, actionable summaries "
            "that respect the reader's time while delivering full context."
        ),
        llm=llm,
        verbose=True,
    )

    reviewer = Agent(
        role="Editorial Quality Reviewer",
        goal="Ensure news digest meets quality standards for accuracy, clarity, and relevance",
        backstory=(
            "You are a meticulous editor who ensures every digest is "
            "factually accurate, well-structured, and free of bias. "
            "You catch errors that others miss."
        ),
        llm=llm,
        verbose=False,
    )

    research_task = Task(
        description=(
            "Research breaking news in these topics: {topics}. "
            "Find 3-5 high-quality stories from reputable sources. "
            "For each story, note: headline, source, key facts, why it matters."
        ),
        expected_output=(
            "JSON array of stories with fields: headline, source_url, "
            "summary (2-3 sentences), relevance_score (1-10)"
        ),
        agent=researcher,
    )

    write_task = Task(
        description=(
            "Using the research findings, write a news digest. "
            "Format: Brief intro paragraph, then 3-5 story summaries. "
            "Each summary: headline, 2-3 sentence explanation, source credit. "
            "Tone: Professional but conversational. Max 500 words total. "
            "Summary length preference: {summary_length}."
        ),
        expected_output="Markdown-formatted news digest ready for display",
        agent=writer,
        context=[research_task],
    )

    review_task = Task(
        description=(
            "Review the news digest for: "
            "Factual accuracy (check claims against research), "
            "Clarity (no jargon, clear explanations), "
            "Formatting (proper Markdown, consistent style), "
            "Bias detection (neutral tone maintained). "
            "If issues found, return revised version. If acceptable, approve as-is."
        ),
        expected_output="Final approved news digest in Markdown format, with quality score (1-10)",
        agent=reviewer,
        context=[write_task],
    )

    return Crew(
        agents=[researcher, writer, reviewer],
        tasks=[research_task, write_task, review_task],
        process=Process.sequential,
        verbose=True,
    )
