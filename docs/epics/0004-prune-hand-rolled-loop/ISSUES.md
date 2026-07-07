# Issues: Prune the Hand-Rolled Loop (Roadmap Step 0)

One slice: clear the ground so a future epic can wire a rented SDK behind the `Brain` interface without fighting dead code.

---

## Issue 1 — Prune the hand-rolled loop

## What to build

Remove the hand-rolled, vendor-specific agent loop and everything wired directly to it, per ADR-0006 and ADR-0001. This is a prefactor: it doesn't add behavior, it clears the ground for whichever rented SDK a future epic wires in.

Remove: the hand-rolled Brain implementation and its vendor-specific turn loop; the interactive entry point wired directly to that vendor's client; the fake demo tool that only existed to prove the old loop worked; and any vendor-specific typing left on the tool declaration/dispatch layer, so that layer is vendor-neutral again. Vendor-specific configuration (API key, model name) tied only to the removed brain goes with it.

## Acceptance criteria

- [x] The hand-rolled Brain implementation and its turn-loop code no longer exist in the codebase
- [x] The interactive entry point no longer constructs or depends on the removed vendor's client
- [x] The fake demo tool is removed, including its dispatch entry
- [x] The tool declaration/dispatch layer for Read/Write/Bash no longer imports the removed vendor's SDK
- [x] The project still builds after removal, with no dead references to the removed code
- [x] Vendor-specific config values used only by the removed brain are removed from configuration loading

## Why this slice is cut here

Kept to pure removal, no behavior added, so it's cheap to verify (the project still builds and runs, just with no brain wired in yet). Choosing and wiring in whatever replaces the loop is deliberately not part of this epic — see the PRD's Out of Scope — so this stays a small, self-contained cleanup rather than growing into the next build.

## Blocked by

None — can start immediately.
