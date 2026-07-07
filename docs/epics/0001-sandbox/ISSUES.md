# Issues: Orchestrator Sandbox Layer

## Issue 1: Signal-safe root context (prefactor)

## What to build

Replace the bare `context.Background()` in `main.go` with a cancellable root context. Wire SIGINT and SIGTERM to cancel that context instead of relying on the OS to kill the process. Ensure the REPL loop exits cleanly when the context is cancelled. Remove `log.Fatalf` from any code path that will own a live sandbox — use error returns and context cancellation instead.

This is a pure refactor of existing code with no new features. It must land first so that all deferred sandbox cleanup calls are guaranteed to fire on every non-SIGKILL exit path.

## Acceptance criteria

- [ ] `main.go` creates a cancellable root context at startup
- [ ] SIGINT and SIGTERM cancel the root context (no `os.Exit` in the signal handler)
- [ ] The REPL loop exits cleanly when the context is cancelled
- [ ] `log.Fatalf` is removed from any code path that will later own a live sandbox; errors are returned or logged before a graceful exit via context cancellation
- [ ] Manual test: `Ctrl+C` exits the process cleanly without leaving the terminal in a broken state

## Blocked by

None — can start immediately.

## Completed
Completed: True

---

## Issue 2: Sandbox Start/Destroy with AutoRemove

## What to build

Add the Docker Go SDK and implement a thin sandbox layer the orchestrator can use to start and destroy an isolated container. The sandbox exposes a clean `Sandbox` interface (`Start`, `Destroy`) so the underlying runtime (Docker today, anything else later) is swappable without touching the orchestrator's call sites.

Every container is created with `AutoRemove: true` (Layer 1 zombie prevention) and tagged with a static orchestrator label (`ai.orchestrator=true`) so it can be identified later. The sandbox is started at orchestrator startup and destroyed via `defer sandbox.Destroy(ctx)` — which is safe because Issue 1 guarantees the defer chain fires. All lifecycle events (start, destroy, error) are logged.

The base image (`python:3.12-slim`) is a named constant in the sandbox struct, easy to change later.

## Acceptance criteria

- [ ] Docker Go SDK is added as a dependency (no shelling out to the Docker CLI)
- [ ] A `Sandbox` interface with `Start(ctx context.Context) error` and `Destroy(ctx context.Context) error` exists
- [ ] `DockerSandbox` implements the interface; the base image is a configurable field (hard-coded to `python:3.12-slim` for now)
- [ ] Every container is created with `HostConfig.AutoRemove: true`
- [ ] Every container is tagged with the label `ai.orchestrator=true`
- [ ] `main.go` starts the sandbox at startup and defers `sandbox.Destroy(ctx)`
- [ ] Sandbox errors surface up the call stack (no silent swallowing)
- [ ] Lifecycle events are logged: container started (with ID), container destroyed, any error
- [ ] Manual test: run the orchestrator → `docker ps` shows the container → quit via Ctrl+C → container is gone

## Blocked by

- Issue 1 (signal-safe context must be in place before sandbox is wired in)

## Completed
Completed: True

---

## Issue 3: Label-based orphan sweep at startup

## What to build

At orchestrator startup — before creating any new sandbox — query Docker for all running containers tagged with `ai.orchestrator=true` and remove them. This is Layer 3 zombie prevention: it recovers from the one scenario Layers 1 and 2 cannot handle (SIGKILL or a hard crash).

The sweep is idempotent: running it when no orphans exist is a no-op. Each orphan found is logged before removal. The static label (`ai.orchestrator=true`) is used rather than a per-instance label so that orphans from any previous orchestrator run on this host are caught, not just orphans from the current process.

## Acceptance criteria

- [ ] On startup, before `sandbox.Start()`, the orchestrator queries Docker for all containers with label `ai.orchestrator=true`
- [ ] Each matching container is force-removed; its ID is logged as "orphan removed"
- [ ] If no orphans are found, the sweep completes silently (no error, no log noise)
- [ ] The sweep is idempotent: running it multiple times in a row has no unintended side effects
- [ ] Manual test: start the orchestrator, `kill -9` its PID, verify the container is still running in `docker ps`, restart the orchestrator, verify the orphan is removed before the new container starts

## Blocked by

- Issue 2 (label must be applied to containers before the sweep has anything to target)

## Completed
Completed: True
