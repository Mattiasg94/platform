"""Agent pod entrypoint: run the harness on /workspace, then print the JSON result
(status, summary, diff) to stdout. Trace goes to stderr so stdout carries only that
JSON — the pod's I/O contract (ADR-0007)."""

import json
import subprocess
import sys

import anyio
from claude_agent_sdk import ClaudeAgentOptions, query

WORKSPACE = "/workspace"
MODEL = "claude-haiku-4-5-20251001"


def workspace_diff() -> str:
    # `git add -N` makes new files show in `git diff` without staging their content.
    subprocess.run(["git", "-C", WORKSPACE, "add", "-N", "."], check=False)
    done = subprocess.run(
        ["git", "-C", WORKSPACE, "diff"],
        capture_output=True,
        text=True,
        check=False,
    )
    return done.stdout


async def run_harness(task: str) -> dict:
    options = ClaudeAgentOptions(
        model=MODEL,
        cwd=WORKSPACE,
        permission_mode="bypassPermissions",
        # Bash lets the agent run `make test` and iterate — its own untrusted check.
        allowed_tools=["Read", "Write", "Edit", "Bash"],
    )
    status = "success"
    summary = ""
    async for message in query(prompt=task, options=options):
        print(message, file=sys.stderr)
        if type(message).__name__ == "ResultMessage":
            summary = getattr(message, "result", "") or ""
            status = "error" if getattr(message, "is_error", False) else "success"
    return {"status": status, "summary": summary}


def main() -> None:
    if len(sys.argv) < 2 or not sys.argv[1].strip():
        json.dump({"status": "error", "summary": "no task provided", "diff": ""}, sys.stdout)
        sys.exit(1)

    task = sys.argv[1]
    result = anyio.run(run_harness, task)
    result["diff"] = workspace_diff()
    json.dump(result, sys.stdout)  # stdout carries only the result contract


if __name__ == "__main__":
    main()
