"""Agent pod entrypoint — run the rented coding harness on the mounted workspace.

Baked into the container image. On start it hands the Claude Agent SDK one
hard-coded task and lets the harness edit /workspace directly, then exits.
This is roadmap step 1 (ADR-0007): prove the rented coding loop runs *inside*
the sandbox, unattended, and produces a real edit we can see on the host.

The workspace is bind-mounted at /workspace; the API key arrives as
ANTHROPIC_API_KEY in the environment. Model is Haiku (cheapest) on purpose —
the point is the plumbing, not the reasoning.
"""

from datetime import datetime, timezone

import anyio
from claude_agent_sdk import ClaudeAgentOptions, query

WORKSPACE = "/workspace"
MODEL = "claude-haiku-4-5-20251001"


def build_task() -> str:
    """One deterministic, repeatable edit: append a timestamped line.

    The timestamp is computed here rather than asked of the model, so the run
    proves 'the harness can edit the workspace' without also depending on the
    model correctly fetching the time. Repeated runs stack visible lines.
    """
    stamp = datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M:%SZ")
    return (
        f"Append exactly one new line reading 'agent ran at {stamp}' to the "
        f"end of notes.md in the current directory. Create notes.md if it does "
        f"not exist. Change nothing else, and do not reformat existing lines."
    )


async def run() -> None:
    options = ClaudeAgentOptions(
        model=MODEL,
        cwd=WORKSPACE,
        # Isolated, single-purpose container — let the harness act without
        # prompting for permission on each edit.
        permission_mode="bypassPermissions",
        allowed_tools=["Read", "Write", "Edit"],
    )
    async for message in query(prompt=build_task(), options=options):
        # Stream the harness's messages to stdout so the orchestrator (and we,
        # by hand for now) can see what it did. The structured result contract
        # comes in roadmap step 2; for the skeleton, printing is enough.
        print(message)


if __name__ == "__main__":
    anyio.run(run)
