# Issues: Sandbox Process Lifecycle — SUPERSEDED

> **SUPERSEDED by ADR-0007 (Architecture B).** The persistent-shell Exec this
> slice built was removed when the harness moved inside the pod. Kept for history
> only.

Vertical slices for the persistent-shell sandbox.

**Scope note:** this project originally had two issues — persistent-shell Exec,
then idle-timeout hang detection + process-group kill. Issue 2 is dropped: it
was scoped in part around tee-ing live output to an interactive operator
terminal, which only made sense with the hand-rolled REPL loop (`main.go` +
`internal/provider/`) that Step 0 of the roadmap removes (ADR-0006). Only
Issue 1 remains, and it's done.

---

# Issue 1 — Persistent-shell Exec with correct output

## What to build

Move the sandbox's arbitrary-command execution from exec-per-call to a single long-lived shell per sandbox. On sandbox start, launch one shell and keep it running. Every `Exec` call writes its command into that shell's stdin instead of creating a fresh Docker exec, so shell state (`cd`, exported env vars) persists from one command to the next — the way the model already expects a terminal to behave.

Because one shell produces one endless output stream, each command must carve out its own output: after the command, emit a unique per-command marker carrying the command's exit code, and read the stream up to that marker to know the command is done and how it finished. stdout and stderr must stay distinguishable through the shell so the model can tell error output from normal output.

Scope this to `Exec` only. File read and write stay on their existing separate exec-per-call path and are **not** routed through the shell — that is what keeps file content off the shell's command-input channel and prevents a write from corrupting the command stream. The change stays behind the existing sandbox seam so callers (activities, tools, the model path) are untouched.

## Acceptance criteria

- [x] One shell is started when the sandbox starts and lives for the sandbox's lifetime.
- [x] `Exec` sends its command through the shell's stdin; no new Docker exec is created per command.
- [x] `cd` in one command is visible to the next command (state persists) — demoable with `cd <subdir>` then `pwd`.
- [x] An exported env var in one command is visible to the next command.
- [x] Each command's output is bounded correctly using a per-command marker, even if the command's own output resembles the marker (marker is unique/random per command).
- [x] Each command's exit code is captured and returned, with non-zero returned as ordinary data (not a Go error).
- [x] stdout and stderr remain distinguishable in the returned result.
- [x] Read and Write still work and are verifiably not routed through the shell (writing content containing marker-like bytes does not corrupt the command stream).
- [x] Change is behind the existing sandbox interface; activities/tools callers are unchanged.

## Why this slice is cut here

Scoped to `Exec` only, deliberately. Read and Write stay on their existing exec-per-call path — folding them into the shell would route file content through the shell's command-input channel and corrupt the command stream, so "Write stays separate" needs no issue of its own; it falls out of scoping this one to Exec. The output-boundary marker, exit-code capture, and stdout/stderr separation are bundled here rather than split into their own issues because a persistent shell that returns merged or unbounded output ships wrong data — this slice is only demoable once all three are in place, so they form one coherent milestone.

## Blocked by

None - can start immediately.
