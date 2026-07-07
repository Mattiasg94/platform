# Issues: Orchestrator Coding Activities — SUPERSEDED

> **SUPERSEDED by ADR-0007 (Architecture B).** These slices built the
> Architecture-A tool layer (typed Activities + provider adapter) that was later
> removed when the harness moved inside the pod. Unchecked issues here will not
> be built as written. Kept for history only.

## Issue 1: Switch sandbox image to Go toolchain

## What to build

Change the sandbox's base image from the Python image to a Go toolchain image so the workspace can build and test Go code.

## Acceptance criteria

- [x] Sandbox base image is a Go toolchain image (Go 1.23 line)
- [x] Manual test: sandbox starts successfully with the new image

## Blocked by

None — can start immediately.

## Completed
Completed: True

---

## Issue 2: Demo Go project with failing test

## What to build

A small standalone Go project to serve as the sandbox's workspace content for this milestone: a deliberately buggy Fibonacci implementation and a table-driven test that currently fails against it. This stands in for a real cloned repository until real repo-cloning exists.

## Acceptance criteria

- [x] Standalone Go module exists with a Fibonacci implementation and an entrypoint
- [x] A table-driven test exercises the implementation
- [x] Running the test suite currently fails (genuine bug present, not a placeholder)

## Blocked by

None — can start immediately.

## Completed
Completed: True

---

## Issue 3: Mount demo project into sandbox workspace

## What to build

Bind-mount the demo project's host directory into the sandbox container at a fixed workspace root path, at sandbox start time. The host path is a hardcoded constant for this milestone (no env/config-driven path yet). This establishes the fixed workspace-root convention that all subsequent tools address paths relative to.

## Acceptance criteria

- [ ] Sandbox `Start` mounts the demo project host directory into the container at the fixed workspace root
- [ ] Host mount path is a hardcoded constant, alongside the existing image constant
- [ ] Manual test: starting the sandbox and exec'ing `ls` (or equivalent) at the workspace root shows the demo project's files

## Blocked by

- Issue 1 (Go image must be in place for the mounted Go project to be usable)
- Issue 2 (demo project must exist on host before it can be mounted)

## Completed
Completed: False

---

## Issue 4: Read tool, end-to-end

## What to build

The first real coding tool: the model can call a read capability to get a file's contents back from the sandbox workspace. This issue also establishes the architecture all later tools follow — a typed, provider-agnostic "Activities" function (no awareness of any model provider's tool-call format) plus a thin adapter that translates the provider's loose tool-call arguments into the typed call and marshals the result back. File paths are resolved relative to the fixed workspace root; path traversal outside the root is rejected. A missing file is a genuine error, not a result to hand back as data.

## Acceptance criteria

- [ ] A typed Read function exists, taking a path and returning file content, independent of any model-provider SDK type
- [ ] A separate adapter layer exposes Read as a tool to the model (declaration + argument unmarshaling + result marshaling), calling the typed function
- [ ] Paths are resolved relative to the fixed workspace root; a path that attempts to traverse outside the root is rejected
- [ ] Reading a nonexistent file returns a genuine error, distinct from a successful read
- [ ] Manual test: ask the orchestrator to show the contents of a file in the demo project; it calls the tool and the real file content comes back

## Blocked by

- Issue 3 (workspace must be mounted and addressable before anything can be read from it)

## Completed
Completed: False

---

## Issue 5: Write tool, end-to-end

## What to build

A write capability the model can call to create or overwrite a file's contents in the sandbox workspace, following the same typed-Activity-plus-adapter pattern established by the Read tool. Missing parent directories are created automatically. An existing file is overwritten silently — no separate "does it already exist" step required. Path resolution and traversal rejection follow the same workspace-root convention as Read.

## Acceptance criteria

- [ ] A typed Write function exists, taking a path and content, independent of any model-provider SDK type
- [ ] Adapter layer exposes Write as a tool to the model, following the same pattern as Read
- [ ] Missing parent directories are created automatically before the write
- [ ] Writing to an existing file overwrites it silently
- [ ] Path traversal outside the workspace root is rejected, consistent with Read
- [ ] Manual test: ask the orchestrator to change a file's contents in the demo project; the file's content is genuinely changed inside the sandbox

## Blocked by

- Issue 4 (establishes the typed-Activity + adapter pattern this tool follows)

## Completed
Completed: False

---

## Issue 6: Bash tool, end-to-end

## What to build

An execute capability the model can call to run an arbitrary command inside the sandbox workspace, following the same typed-Activity-plus-adapter pattern. The command is specified as an argument list (not a single shell string), with an optional working directory that defaults to the workspace root. The result returned to the model is stdout, stderr, and exit code — a non-zero exit code (e.g. a failing test run) is ordinary result data, not a tool error. A genuine tool-execution failure (sandbox unreachable, exec API failure) is a distinct error, separate from the command's own outcome. Execution is bounded by a fixed timeout, and captured output is capped in size with a truncation marker if exceeded.

## Acceptance criteria

- [ ] A typed Exec function exists, taking a command as an argument list and an optional working directory (default: workspace root), independent of any model-provider SDK type
- [ ] Adapter layer exposes Exec as a tool to the model, following the same pattern as Read/Write
- [ ] The result returned includes stdout, stderr, and exit code as normal data; a non-zero exit code does not produce a Go-level error
- [ ] A genuine execution-plumbing failure (not the command's own outcome) does produce a distinct Go-level error
- [ ] Execution is bounded by a fixed timeout; a hung command is terminated and reported rather than stalling the run
- [ ] Captured stdout/stderr is capped at a fixed size, with a clear truncation marker if the cap is hit
- [ ] Manual test: ask the orchestrator to run the demo project's test suite; the real (failing) test output comes back as data

## Blocked by

- Issue 4 (establishes the typed-Activity + adapter pattern this tool follows)

## Completed
Completed: False

---

## Issue 7: Autonomous fix-the-bug demo (integration proof)

## What to build

No new capability — this is an end-to-end proof that Read, Write, and Bash work together as a real coding loop. Ask the orchestrator to fix the demo project's failing test with no further guidance, and let it run unaided: read the current code, understand the failure, write a fix, rerun the tests, and report success.

## Acceptance criteria

- [ ] Given only a high-level instruction ("fix the failing test"), the orchestrator reads the relevant file(s) without being told which ones
- [ ] The orchestrator writes a corrected implementation based on what it read
- [ ] The orchestrator reruns the test suite itself and observes the result
- [ ] The loop concludes with the test suite passing, reported back to the user, without manual intervention mid-loop

## Blocked by

- Issue 5 (Write tool)
- Issue 6 (Bash tool)

## Completed
Completed: False
