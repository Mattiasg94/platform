"""Agent pod entrypoint — run the rented coding harness on the mounted workspace.

Baked into the container image. The orchestrator launches this image with the
repo bind-mounted at /workspace and hands it a task as the single command-line
argument. The harness edits /workspace directly, then this script returns a
structured result on stdout — the pod's I/O contract (ADR-0007): task in,
{status, summary, diff} out.

Conventions that keep the contract clean:
- The harness's streaming messages go to *stderr* (human-visible trace).
- *stdout* carries nothing but the final JSON result, so the orchestrator can
  parse it without demuxing chatter.

Model is Haiku (cheapest) on purpose — the point is the plumbing.
"""

import json
import subprocess
import sys

import anyio
from claude_agent_sdk import ClaudeAgentOptions, query

WORKSPACE = "/workspace"
MODEL = "claude-haiku-4-5-20251001"


def workspace_diff() -> str:
    """Diff the workspace against HEAD, including newly created files.

    `git add -N` records intent-to-add so brand-new files show up in `git diff`
    without staging their content. Requires /workspace to be a
    git repo; the orchestrator/fixture guarantees that.
    """
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
        # Isolated, single-purpose container — let the harness act without
        # prompting on each edit.
        permission_mode="bypassPermissions",
        # The pod only edits files; it does not build or test. Verification runs
        # in the project's own environment (ADR-0008), so no Bash tool here.
        allowed_tools=["Read", "Write", "Edit"],
    )
    status = "success"
    summary = ""
    async for message in query(prompt=task, options=options):
        print(message, file=sys.stderr)  # trace, kept off stdout
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
    json.dump(result, sys.stdout)  # the ONLY thing on stdout: the result contract


if __name__ == "__main__":
    main()
