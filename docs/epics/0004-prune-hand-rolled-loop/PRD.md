# PRD: Prune the Hand-Rolled Loop (Roadmap Step 0)

## Business Goal & Objective

Clear a known contradiction out of the codebase before building anything else on top of it. A hand-rolled, vendor-specific agent loop exists from before the decision to rent the inner loop was made — leaving it in place would mean the next epic (wiring a rented SDK behind the `Brain` interface) has to fight or route around dead, contradicting code instead of landing in a clean seam.

## Problem Statement

A hand-rolled turn loop, tied directly to one model vendor's SDK, was written before the decision to rent the inner loop instead of building it. It now contradicts that decision. It also drags a fake demo tool and vendor-specific typing into otherwise vendor-neutral code, and nothing currently depends on any of it in a way that's worth preserving.

## Proposed Solution

Remove the hand-rolled loop and everything wired directly to it: the vendor-specific `Brain` implementation and its turn-management code, the interactive entry point that constructed that vendor's client, the fake demo tool that only existed to prove the old loop worked, and the vendor-specific typing left on the tool declaration/dispatch layer. Remove vendor-specific configuration that only the removed loop used. Nothing is added or replaced — this is subtraction only, done so the next epic starts from a clean, vendor-neutral seam.

## User Stories

1. As a developer, I want the hand-rolled, vendor-specific agent loop removed, so that it no longer contradicts the decision to rent the inner loop.
2. As a developer, I want the tool declaration/dispatch layer free of vendor-specific typing, so that it's ready to serve whichever rented SDK comes next without carrying the old vendor's assumptions.
3. As a developer, I want the fake demo tool removed, so that dead scaffolding doesn't linger now that real tools exist.
4. As a developer, I want the project to still build and run (sandbox lifecycle intact) after the removal, so that the prune is verifiably safe rather than just assumed safe.
5. As a developer, I want vendor-specific configuration removed along with the code that used it, so that no dead configuration surface is left behind.

## Technical Direction & Assumptions (Pre-Discovery)

- This is a pure removal — no new abstractions, no interim stub standing in for what's removed.
- Whatever isn't touched by the removal (the sandbox/isolation layer, the typed tool-execution layer) is assumed correct as-is and is out of scope for this epic.
- The program should remain runnable after the prune even with no model loop wired in — a reduced but working state, not a broken intermediate.

## Decisions & Rationale

**Continue on the existing codebase; do a surgical prune, not a restart.**
- *Decision:* remove the hand-rolled loop and its direct dependents; keep everything else as-is.
- *Why:* the surrounding code (sandbox/isolation, typed tool execution) is real, working infrastructure unrelated to the loop mistake — throwing it away to "start clean" would waste it for no benefit.
- *Alternatives rejected:* deleting everything and starting from a clean slate — rejected because it throws away real, unrelated infrastructure work for no gain; both options are reversible, and continuing is cheaper. Keeping the hand-rolled loop as-is — rejected because it's the exact commodity the rent decision already ruled out building.

**No separate ceremony beyond this record.**
- *Decision:* this epic is scoped to removal only — it does not plan, scope, or begin whatever replaces the loop.
- *Why:* the removal is mechanical once the contradicting code is identified; deciding what replaces it is a separate, larger decision (which SDK, how it's wired) that deserves its own epic rather than being rushed alongside a cleanup.

**Load-bearing principle:** rent the commodity loop, don't hand-roll it — carried over from the earlier decision to rent the inner agent loop. This epic exists only to make the codebase consistent with that decision, nothing more.

## Out of Scope

- Choosing or wiring in whatever replaces the removed loop (which SDK, how it's integrated) — a separate future epic.
- Any new capability, tool, or behavior — this epic only removes.
- Changes to the sandbox/isolation layer or the typed tool-execution layer — untouched, assumed correct.

## Open Questions for Technical Discovery

- None specific to the removal itself. What replaces the loop, and how, is deferred entirely to the epic that takes it on.
