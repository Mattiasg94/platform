# agent — the pod harness

The Python component: the rented coding harness (Claude Agent SDK) baked into a
container image. The orchestrator launches this image with a repo bind-mounted
at `/workspace` and a task; the harness edits the workspace and exits. The
anti-lock-in seam is the pod's I/O contract, not an in-process interface
(ADR-0007).

## Why the image carries Node as well as Python

The Python `claude-agent-sdk` is **not** a standalone reimplementation of the
agent loop. It is a thin client: when you call `query(...)`, it spawns the
**Claude Code CLI** — a Node program (`@anthropic-ai/claude-code`) — as a
subprocess and talks to it. All the actual harness behavior (the turn loop,
tool execution, edit parsing, context management) lives in that CLI. The
TypeScript and Python SDKs are two bindings over the *same* Node engine; they
are not two independent engines. So "use the Python SDK" still means "run the
Node CLI underneath," and the image needs both runtimes: Node for the harness,
Python for our entrypoint and the client binding.

That is also *why* we bake it into an image instead of pip-installing into the
orchestrator: it drags a Node runtime and a global npm package along, which we
want quarantined inside the pod, not in the Go orchestrator's world.

## Build and run (walking skeleton)

From the repo root:

```sh
docker build -t agent-pod ./agent
docker run --rm \
  --user "$(id -u):$(id -g)" \
  -e HOME=/tmp \
  -v "$PWD/demo-project:/workspace" \
  --env-file .env \
  agent-pod
```

Then look at `demo-project/notes.md` — a new `agent ran at <timestamp>` line
should be appended. Run it again; a second line appears. That is the whole
proof for this slice.

Two run-time details the orchestrator will later encode in the container spec:

- **Non-root.** `--user` runs the harness as a non-root uid. The Claude Code
  CLI refuses `bypassPermissions` (its `--dangerously-skip-permissions`) under
  root, and matching the host uid also lets it write the bind-mounted workspace
  and leaves the files owned by you, not root. `HOME=/tmp` gives the CLI a
  writable place for its own state.
- **`.env` quoting.** `docker --env-file` does **not** strip surrounding
  quotes, so `ANTHROPIC_API_KEY` must be unquoted in `.env` or the key arrives
  with literal `"` characters and the API returns 401. (godotenv, which the Go
  orchestrator uses, *does* strip them — so keep the file unquoted to satisfy
  both.)
