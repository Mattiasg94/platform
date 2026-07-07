# PRD: Orchestrator Coding Activities — SUPERSEDED

> **SUPERSEDED by ADR-0007 (Architecture B).** This epic planned a Go-side
> Read/Write/Bash "Activities" + adapter layer that piped tools into the sandbox
> (Architecture A). That path was not taken: under ADR-0007 the coding harness
> runs *inside* the pod and owns its own tools, so this layer was removed
> (`internal/tools`, `internal/activities`, `internal/session`). Kept for history
> only — not the active plan. The code is the source of truth.

## Business Goal & Objective

Give the orchestrator its first real coding capability — the ability to read, write, and execute code inside its sandbox — as the next milestone toward a self-hosted, provider-agnostic, fully autonomous coding agent. This milestone is explicitly built to *mimic* the shape of a Temporal-backed production system (Workflow / Activity separation, typed inputs/outputs) without yet introducing Temporal itself, so the later migration is a lift, not a rewrite.

## Problem Statement

Today the orchestrator can only hold a conversation and call one fake demonstration tool (`get_weather`). It has a working sandbox (start/destroy/orphan-sweep) and a working tool-call loop, but nothing inside the sandbox does real work. There is no way for the orchestrator to write code, read code back, or run a test — so it cannot yet act as a coding agent at all.

## Proposed Solution

Add three typed capabilities the model can call as tools, all operating inside the existing Docker sandbox against a fixed workspace root:

- **Read** — read a file's contents from the sandbox workspace.
- **Write** — create or overwrite a file's contents in the sandbox workspace (auto-creating missing parent directories).
- **Bash** — execute an arbitrary command (argv form) inside the sandbox workspace and return its stdout, stderr, and exit code.

These three give the model everything it needs to run the real coding loop: read existing code, write a fix, run the test suite, read the result, iterate.

The sandbox's base image switches from Python to a Go toolchain image so the workspace can build and test Go code. A small demo Go project (a deliberately buggy Fibonacci implementation with a failing test) is bind-mounted into the sandbox workspace as a stand-in for a real cloned repository, so the loop can be exercised end-to-end without building real repo-cloning yet.

Internally, the three capabilities are implemented as standalone, typed functions (an "Activities" layer) with no awareness of the model-provider's tool-calling format. A thin adapter layer translates the provider's loose tool-call arguments into these typed calls and back. The existing tool-calling loop (which asks the model for a reply, executes a tool if requested, and repeats) is recognized as already matching the shape of a durable workflow's orchestration logic — a pure decision loop with no direct I/O of its own — and is left as-is going into this milestone.

## User Stories

1. As an orchestrator, I want to read a file's contents from my sandbox workspace, so that I can see existing code before changing it.
2. As an orchestrator, I want to write a file's contents into my sandbox workspace, so that I can create new code or apply a fix.
3. As an orchestrator, I want missing parent directories to be created automatically when I write a file, so that I don't need a separate step to prepare the directory structure.
4. As an orchestrator, I want a file write to silently overwrite an existing file, so that iterating on a fix doesn't require a separate "delete first" step.
5. As an orchestrator, I want to execute an arbitrary command inside my sandbox workspace, so that I can run tests, builds, or any other tool without needing a bespoke tool per command type.
6. As an orchestrator, I want commands to be specified as an argument list rather than a single shell string, so that my commands aren't vulnerable to shell-quoting mistakes or injection.
7. As an orchestrator, I want to optionally specify a working directory for a command, so that I can run commands scoped to a subdirectory of a larger project.
8. As an orchestrator, I want file paths for read/write to be resolved relative to a fixed workspace root, so that I don't need to know or reason about host or container filesystem layout.
9. As an orchestrator, I want path traversal outside the workspace root to be rejected, so that I cannot accidentally (or via a bad instruction) write or read outside my intended sandbox area.
10. As an orchestrator, I want a command's non-zero exit code to be returned to me as normal result data (stdout, stderr, exit code), not as a tool failure, so that I can read and react to test failures the way a developer reads a failing test run.
11. As an orchestrator, I want a genuine tool-execution failure (e.g. the sandbox is unreachable) to be surfaced as a distinct error from a command's own failure, so that I can tell "my tool didn't run" apart from "the command I ran, failed."
12. As an orchestrator, I want a command execution to be bounded by a timeout, so that a hung process (e.g. an infinite loop or deadlocked test) cannot stall my run indefinitely.
13. As an orchestrator, I want a command's captured output to be capped in size with a clear truncation marker, so that a runaway amount of output cannot exhaust memory or flood my context.
14. As an orchestrator, I want my sandbox to run a Go toolchain image, so that I can build and test Go code.
15. As an orchestrator, I want a demo project already present in my sandbox workspace at startup, so that I have real code to read, fix, and test without needing repo-cloning built yet.
16. As an orchestrator, I want that demo project to contain a genuinely failing test, so that I can prove out a real read → fix → rerun-test → pass loop.
17. As a developer, I want the read/write/execute capabilities implemented as standalone functions with typed inputs and outputs (not loosely-typed maps), so that they can later be registered as Temporal Activities with little to no rewriting.
18. As a developer, I want these typed capability functions kept in their own package, separate from the provider-specific tool-declaration and argument-marshaling code, so that the "what the tool does" and "how a specific model provider calls it" concerns don't mix.
19. As a developer, I want the provider-facing tool adapter layer to be a thin translation (unmarshal arguments, call the typed function, marshal the result), so that switching or adding a model provider later doesn't require touching the underlying capability logic.
20. As a developer, I want the existing tool-calling loop to remain a pure decision loop with no direct I/O, so that it can later become a Temporal Workflow function largely unchanged.
21. As a developer, I want the sandbox workspace mount path to be a hardcoded constant for this milestone, so that I don't build configurability before there is a second use case that needs it.

## Technical Direction & Assumptions (Pre-Discovery)

- Base sandbox image switches to a Go toolchain image (Go 1.23 line) so `go build`/`go test` run natively without additional setup.
- A fixed workspace root inside the sandbox (conceptually `/workspace`) is bind-mounted from a demo project directory on the host — this simulates "repo present in sandbox" without implementing real repo cloning yet; the interface should not need to change when real cloning replaces the bind mount later.
- Three capabilities, each a standalone typed function ("Activity"), with no knowledge of any specific model provider's tool-call format:
  - **Read**: input is a path; output is file content; not-found is a genuine error.
  - **Write**: input is a path and content; creates missing parent directories; overwrites existing files without prompting; path resolution is relative to the workspace root with traversal rejected.
  - **Bash/Exec**: input is a command as an argument list (not a single shell string) plus an optional working directory (defaulting to the workspace root); output is stdout, stderr, and exit code. Bounded by a fixed timeout and a fixed output-size cap with a truncation marker on overflow.
- Tool-execution error semantics: a Go-level error return is reserved for genuine plumbing failure (sandbox unreachable, invalid path, exec API failure, file not found on read). A command's own non-zero exit code, or a failing test run, is not an error — it is ordinary result data the caller reads and reasons about, matching how a developer reads a failing terminal command.
- The typed capability functions live in their own package, separate from whatever package handles translating a specific model provider's loose tool-call arguments into these typed calls (and marshaling results back). This split exists specifically so the capability functions can later be registered as Temporal Activities without rewriting their signatures.
- The existing model-response/tool-execution loop (ask model → if tool call, execute it, feed result back, repeat) is treated as already matching the shape of a future Temporal Workflow — a pure orchestration loop with no direct I/O of its own, calling out through interfaces only. No structural change to this loop is planned in this milestone.
- No dedicated "Edit" (surgical find/replace) capability yet — whole-file Write is sufficient at current file sizes; deferred until working with files large enough that whole-file rewrite becomes wasteful or risky.
- No separate module/build-cache persistence mechanism — a single sandbox container persists for the lifetime of one orchestrator run, so any build cache accumulates naturally within that run; cross-run persistence is not needed yet.

## Out of Scope

- Temporal itself — not introduced this milestone; only the shape (typed Activities, I/O-free orchestration loop) is prepared for it.
- A dedicated "Edit" / surgical find-and-replace capability — deferred until whole-file rewrites become impractical.
- Real repository cloning into the sandbox — simulated via a bind-mounted demo project for now.
- Configurable/env-driven workspace mount path — hardcoded constant is sufficient until a second project/target exists.
- Cross-run build/module cache persistence — not needed while the sandbox is single-container, single-run.
- Output size cap or truncation on Read — only the Bash/Exec capability has an output cap in this milestone.
- Multi-sandbox or multi-tenant scenarios — unchanged from the prior sandbox milestone's scope, still a single sandbox per orchestrator run.

## Open Questions for Technical Discovery

- When real repo-cloning eventually replaces the bind-mounted demo project, does the workspace-root convention (fixed path, relative addressing, traversal guard) hold unchanged, or does a real repo's structure (multiple modules, nested dirs) require revisiting the "single fixed root" assumption?
- When Temporal is eventually introduced, will the Activities defined here need idempotency keys or additional retry-safety guarantees beyond "typed input/output," given that Write/Bash are not naturally idempotent (a retried Bash call could re-run a command with side effects)?
