# PRD: Sandbox Process Lifecycle

## Business Goal & Objective

Move the orchestrator's sandbox from "runs a command and hopes it finishes" toward a process-lifecycle model an autonomous coding agent can actually rely on: commands that persist state like a real terminal, and results that always come back to the model as clean data. This is foundational — every future autonomous behavior runs *through* this layer, so getting its behavior policy right now prevents building higher features on a shaky base.

Framing rule carried from `docs/orchestrator/VISION.md`: **rent the substrate, build the policy.** Docker stays as the rented isolation substrate (the walls). Everything in this PRD is *policy* — how the agent behaves inside those walls — which is ours to build by hand because no rented substrate authors it for us, and it's the part that transfers to any future job.

**Scope update:** this PRD originally covered three problems. Problems 1 and 2 (timeout doesn't kill the process; no visibility into why a command is slow) are now **out of scope** for this project — see Out of Scope. Their fix (idle-timeout hang detection + process-group kill) was scoped in part around tee-ing live output to an interactive operator terminal, a feature that only made sense with the hand-rolled REPL loop (`main.go` + `internal/provider/`) that roadmap Step 0 removes (ADR-0006). Only problem 3 (persistent state) is this project's active scope; it's done (Issue 1).

## Problem Statement

**No persistent state across tool calls, and the model doesn't know it.** Each Bash/Read/Write call is a brand-new, independent process, so `cd` and exported environment variables don't carry over — but nothing tells the model this. It assumes ordinary persistent-shell behavior (from its training), runs `cd` then a command expecting the directory to stick, gets confused output, and wastes turns debugging an illusion.

## Proposed Solution

Adopt a **persistent-shell** model, while keeping file writes on their own separate path.

- **Persistent shell:** start *one* long-lived shell per task and type every command into its stdin, exactly like a human at a terminal. State persists because it's the same process throughout. Optimizes for fewer wasted model turns, since `cd`/env behave the way the model already expects.

## User Stories

1. As the orchestrator, I want a command's `cd` and exported env vars to persist to the next command, so that the model can work like it would at a real terminal.
2. As the model, I want the environment to behave the way I expect from training, so that I don't waste turns discovering that state didn't persist.
3. As the orchestrator, I want each command's exit code returned even though there's no per-command exec to inspect, so that pass/fail is never lost.
4. As the orchestrator, I want each command's output boundary detected reliably even when the command's own output resembles my boundary marker, so that I never cut output in the wrong place.
5. As the orchestrator, I want stdout and stderr kept distinguishable through the persistent shell, so that the model can tell error output from normal output.
6. As the orchestrator, I want file writes to work without competing with the shell's command-input channel, so that writing a file never corrupts the command stream.
7. As a developer, I want the sandbox behavior to stay behind the existing `Sandbox` interface, so that the isolation substrate (Docker today, E2B/Firecracker later) can be swapped without touching callers.

## Technical Direction & Assumptions (Pre-Discovery)

**Architecture shift.** Today = *exec-per-call*: every command is a new Docker exec, the command travels via the Docker API, stdin is rarely used, no state persists. Target = *persistent shell*: one Docker exec starts a single long-lived shell, and every subsequent command is written into that shell's stdin. Command delivery moves from the API channel to the stdin channel.

**Output boundaries (core mechanism).** One long-lived shell = one endless output stream, so there's no per-command EOF to signal "done" (exec-per-call got that for free). After each command, also write an `echo` of a **unique random marker**; read the stream until the marker appears — everything before it is this command's output. The marker must be random per-command so a command's own output can't accidentally collide with it and create a false boundary.

**Exit code.** No per-command exec to inspect anymore — capture the shell's last exit status and print it *on the marker line* alongside the done signal.

**stdout vs stderr.** `StdCopy` demuxed these for free in exec-per-call; a plain shell merges them. Wire the two streams so they stay distinguishable — a shell-setup-time decision that affects everything after.

**Write stays separate.** In a persistent shell, stdin is already the command-typing channel ("the keyboard"). Today's Write (`cat > file` with content piped via stdin) would make stdin *also* carry file-content payload, and the shell couldn't tell where content ends and the next command begins. Resolution: keep Write as its own separate mechanism (its own one-shot exec / direct file write), not routed through the persistent shell — a file write doesn't need the shell's `cd`/env state anyway.

## Decisions & Rationale

The reasoning trail behind the design above, so a future session doesn't re-litigate settled calls.

**Persistent shell over exec-per-call.**
- *Decision:* one long-lived shell per task, commands via stdin.
- *Why:* it's what the model already expects from training (`cd`/env persist), so it wastes the fewest model turns and removes the class of "state didn't stick" confusion.
- *Alternatives rejected:* (a) *keep exec-per-call and tell the model in the system prompt that state doesn't persist* — rejected because it burns model attention on a caveat and the model still slips back into terminal assumptions; fewer wasted turns matters more than implementation simplicity. (b) *replay the command history to reconstruct shell state after a restart* — rejected because side-effecting commands aren't deterministic (network, timestamps, races), so a replay can't rebuild the real shell state.

**Write kept on its own path, not routed through the persistent shell.**
- *Decision:* file writes stay a separate one-shot mechanism, not typed into the shell's stdin.
- *Why:* in a persistent shell, stdin is the command-typing channel; piping file *content* through it too (today's `cat > file`) means the shell can't tell where content ends and the next command begins. A file write doesn't need the shell's `cd`/env state anyway.
- *Alternative rejected:* *unify everything through the shell's stdin* — rejected for the boundary-corruption reason above.

**Load-bearing principles.**
- **Rent the substrate, build the policy** (from VISION.md): Docker is the rented isolation substrate; everything in this PRD is policy we author, because no rented substrate authors agent behavior for us — and it's the part that transfers to a job. This is why building it by hand is *not* wasted learning.
- **Non-zero exit is data, not a Go error** (existing `exec.go` contract): a command's own failing exit code is ordinary tool-result data, same as a failing test's exit code — never a Go-level error. The model reasons about it and picks the next step.

## Out of Scope

- **Idle-timeout hang detection and process-group kill** (originally Issue 2 of this project) — dropped. It was scoped in part around tee-ing live output to an interactive operator terminal, which only made sense with the hand-rolled REPL loop being removed in roadmap Step 0. If hang handling is needed again later, replan it against whatever the model-loop / durability layer (Temporal, roadmap P4) looks like by then, rather than reviving this design as-is.
- **A hung/orphaned process from a timed-out command** — a real gap left open by dropping the above; not solved by anything in this project currently. Revisit if it causes real pain.
- **Interactive prompts** ("Are you sure? [y/N]") that wait forever for input — unhandled for now, consistent with hang-handling being out of scope.
- **Output capping / truncation for huge output** — capping was deliberately removed earlier (hit-the-bug-then-fix); keep deferring, consistent with that call.
- **Stronger sandbox isolation** (gVisor/Firecracker/E2B) — substrate, not policy; gated on someone other than the owner submitting input (per VISION.md).
- **Full observability tooling** (Langfuse/Helicone) — gated on running unattended, per VISION.md.
- **Persisting shell state across a crash / durability** — a Temporal-shaped concern at the loop layer, explicitly not solved here (the sandbox layer can't restore a dead shell's cwd/env).

## Open Questions for Technical Discovery

- Mechanism for keeping stdout/stderr distinguishable through one shell (e.g. separate FDs / a second stream) — to be pinned during the boundary-scan step of TDD when the I/O signature is locked.
- Does the persistent shell live for the whole orchestrator run, or per-task with an explicit reset boundary?
