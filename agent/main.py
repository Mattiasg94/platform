"""Agent pod entrypoint.

The pod is handed one thing — a run id — and fetches everything else from the
blackboard: the task and a tarball of the workspace come down from the bucket, the
result goes back up to it (ADR-0007). Nothing arrives on the command line and
nothing is scraped from stdout, because in Cloud Run there is no daemon holding the
other end of that pipe. stdout and stderr are logs, and only logs.
"""

import json
import os
import subprocess
import sys
import tarfile
import tempfile

import anyio
from claude_agent_sdk import ClaudeAgentOptions, query
from google.cloud import storage

WORKSPACE = "/workspace"
MODEL = "claude-haiku-4-5-20251001"


def blackboard() -> tuple[storage.Bucket, str]:
    bucket = os.environ.get("RUNS_BUCKET", "")
    run_id = os.environ.get("RUN_ID", "")
    if not bucket or not run_id:
        raise SystemExit("RUNS_BUCKET and RUN_ID must both be set")
    return storage.Client().bucket(bucket), run_id


def fetch_task(bucket: storage.Bucket, run_id: str) -> str:
    body = bucket.blob(f"runs/{run_id}/input.json").download_as_bytes()
    task = json.loads(body).get("task", "").strip()
    if not task:
        raise SystemExit(f"run {run_id} carries no task")
    return task


def safe_members(tar: tarfile.TarFile):
    """Only regular files and directories, and only inside the workspace.

    tarfile's own `filter="data"` would do this, but it needs Python 3.12 and the
    base image is on 3.11. Checking it here also keeps the guarantee ours: an
    entry named "../.ssh/authorized_keys" writes outside the tree, and we simply
    refuse rather than trusting whoever built the archive.
    """
    for member in tar.getmembers():
        if not (member.isfile() or member.isdir()):
            continue
        path = os.path.normpath(member.name)
        if os.path.isabs(path) or path.startswith(".."):
            raise SystemExit(f"refusing tar entry outside the workspace: {member.name}")
        yield member


def fetch_workspace(bucket: storage.Bucket, run_id: str) -> None:
    # The tarball carries .git, so the harness lands in a real repository and
    # `git diff` still means what it meant when this tree was bind-mounted.
    os.makedirs(WORKSPACE, exist_ok=True)
    with tempfile.NamedTemporaryFile(suffix=".tar") as archive:
        bucket.blob(f"runs/{run_id}/workspace.tar").download_to_file(archive)
        archive.flush()
        with tarfile.open(archive.name) as tar:
            tar.extractall(WORKSPACE, members=safe_members(tar))  # noqa: S202


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
    bucket, run_id = blackboard()
    print(f"run {run_id}: starting", file=sys.stderr)

    task = fetch_task(bucket, run_id)
    fetch_workspace(bucket, run_id)

    try:
        result = anyio.run(run_harness, task)
        result["diff"] = workspace_diff()
    except Exception as err:  # noqa: BLE001
        # A crash still owes the orchestrator a result; silence would look like a
        # pod that never ran.
        result = {"status": "error", "summary": f"agent crashed: {err}", "diff": ""}

    bucket.blob(f"runs/{run_id}/result.json").upload_from_string(
        json.dumps(result), content_type="application/json"
    )
    print(f"run {run_id}: {result['status']}", file=sys.stderr)


if __name__ == "__main__":
    main()
