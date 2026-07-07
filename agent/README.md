# agent — the pod harness

The Python component: the rented coding harness (Claude Agent SDK) baked into a
container image. The orchestrator launches this image with a repo bind-mounted
at `/workspace` and a task; the harness edits the workspace and the pod returns
`{status, summary, diff}`. The anti-lock-in seam is that I/O contract, not an
in-process interface (ADR-0007).

Run it with `make run` (orchestrator drives it) or `make agent-run` (pod alone,
for debugging). The commands live in the `Makefile`, not here, so they can't go
stale unread.

## Why the image carries Node as well as Python

The Python `claude-agent-sdk` is **not** a standalone reimplementation of the
agent loop. It is a thin client: `query(...)` spawns the **Claude Code CLI** —
a Node program (`@anthropic-ai/claude-code`) — as a subprocess and talks to it.
All the real harness behavior (turn loop, tool execution, edit parsing, context
management) lives in that CLI. The TypeScript and Python SDKs are two bindings
over the *same* Node engine, not two engines. So "use the Python SDK" still
runs the Node CLI underneath, and the image needs both runtimes.

That is also why we bake it into an image instead of pip-installing into the
orchestrator: it drags a Node runtime and a global npm package along, which we
keep quarantined inside the pod rather than in the Go orchestrator's world.
