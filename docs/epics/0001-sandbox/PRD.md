# PRD: Orchestrator Sandbox Layer

## Business Goal & Objective

Build the foundational sandbox primitive for a self-hosted, provider-agnostic AI orchestrator. The sandbox is the key building block that all autonomous AI behavior depends on — without it, the orchestrator cannot execute code, iterate on output, or operate without human intervention. This milestone establishes the base that every future capability (code execution, repo cloning, autonomous loops) will be built on top of.

## Problem Statement

An AI orchestrator that can only generate text is not autonomous — it needs a place to *act*. Today there is no sandboxed execution environment under orchestrator control. Every AI-driven task requires manual setup, and there is no programmatic way to start, use, or clean up an isolated environment. Without this primitive, building an autonomous coding agent is impossible.

## Proposed Solution

A Docker-based sandbox layer, controlled entirely by the orchestrator via the Docker Go SDK. The orchestrator can programmatically start a container (the sandbox) and destroy it when done. The implementation includes a three-layer zombie container prevention strategy to ensure that no orphaned containers are left running, regardless of how the orchestrator exits — clean shutdown, crash, or hard kill.

## User Stories

1. As an orchestrator, I want to start a sandbox container, so that I have an isolated environment to perform autonomous tasks in.
2. As an orchestrator, I want to destroy a sandbox container after my task is complete, so that I do not leave unused resources running.
3. As an orchestrator, I want the container to be automatically removed when it stops, so that Docker does not accumulate dead containers over time.
4. As an orchestrator, I want to receive a signal (SIGINT/SIGTERM) and cleanly shut down the sandbox before exiting, so that no containers are left running after a graceful shutdown.
5. As an orchestrator, I want all shutdown logic to be triggered via context cancellation (not os.Exit), so that Go's defer-based cleanup is guaranteed to run on every exit path.
6. As an orchestrator, I want every container I create to be tagged with an identifying label, so that orphaned containers from previous runs can be detected.
7. As an orchestrator, I want to scan for and destroy any labeled orphan containers at startup, so that a previous crash or hard kill does not leave zombie containers behind indefinitely.
8. As an orchestrator, I want the base image used for the sandbox to be configurable (even if hard-coded initially), so that I can run different types of workloads.
9. As a developer, I want the sandbox to expose a clean interface (Start, Destroy), so that I can swap the underlying implementation (Docker → Firecracker, etc.) without changing the orchestrator's call sites.
10. As a developer, I want the sandbox lifecycle to be tied to a context.Context, so that cancellation propagates correctly through the entire orchestrator run path.
11. As a developer, I want sandbox errors to surface clearly up the call stack, so that the orchestrator can make informed decisions about retry or abort.
12. As a developer, I want the orphan sweep at startup to be idempotent, so that running it multiple times produces no unintended side effects.
13. As a developer, I want the sandbox to log its lifecycle events (start, destroy, orphan found), so that I can observe and debug orchestrator runs.

## Technical Direction & Assumptions (Pre-Discovery)

- **Docker Go SDK** (`github.com/docker/docker/client`) for all container operations — no shelling out to the Docker CLI.
- **Base image:** `python:3.12-slim`, hard-coded for this milestone. Image name should be a parameter in the sandbox struct so it is easy to change later.
- **Container-level isolation** (Linux namespaces + cgroups) is sufficient. VM-level isolation is not required at this stage.
- **AutoRemove (`HostConfig.AutoRemove: true`)** set on every container at creation time — Layer 1 of zombie prevention.
- **Context cancellation pattern** — a root `context.Context` is created at program startup and passed through the entire orchestrator run path. Signal handlers (SIGINT, SIGTERM) cancel this context rather than calling `os.Exit()`. This guarantees that all `defer sandbox.Destroy()` calls fire on every non-SIGKILL exit path. `log.Fatal()` and `os.Exit()` are banned from any code path that owns a live sandbox.
- **Label-based orphan sweep** — every container is created with a label (e.g., `ai.orchestrator=true` or a more specific instance label). On orchestrator startup, before creating any new container, the program queries Docker for all running containers matching that label and removes them. This is Layer 3 and recovers from SIGKILL and hard crashes.
- **Sandbox interface:** a Go interface or struct with at minimum `Start(ctx context.Context) error` and `Destroy(ctx context.Context) error`. The implementation detail (Docker) sits behind this boundary.
- **Three-layer zombie prevention summary:**
  - Layer 1 — `AutoRemove: true`: container self-removes when stopped
  - Layer 2 — context cancellation + defer: unwinds call stack cleanly, fires all deferred cleanups
  - Layer 3 — label + orphan sweep on startup: recovers from the one case nothing else can handle (SIGKILL / hard crash)

## Out of Scope

- Managed sandbox services (e.g., E2B) — intentionally avoided to maintain full control.
- VM-level isolation (e.g., Firecracker) — valid future direction but not needed now.
- Executing code inside the container — next milestone, not this one.
- Cloning repositories into the container — future milestone.
- Networking configuration, volume mounts, or resource limits — future milestones.
- Multi-tenant or multi-sandbox scenarios — single sandbox per orchestrator run for now.
- Custom image builds — base image is pulled as-is; no Dockerfile authoring at this stage.

## Open Questions for Technical Discovery

- Should the orphan sweep label be per-instance (e.g., include a process ID or a UUID generated at startup) or a static label shared by all orchestrator instances? A per-instance label would not catch orphans from a *different* previous instance; a static label would catch all orchestrator-owned containers but could interfere if two orchestrator instances run simultaneously on the same host.
