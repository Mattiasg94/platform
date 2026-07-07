# Orchestrator — Roadmap (the ordered arc)

**What this doc is for:** the riskiest-first order of prototypes and *why* that
order. Pointers, not state — for what's actually built, read the code. Revised as
planning sharpens; detail only the next slice or two (working-method §2).

## Where we are now

- Architecture decided: **B** — the rented coding harness runs *inside* the
  sandbox pod; the orchestrator supervises it as a bounded job (ADR-0007).
- Codebase pruned to its load-bearing core: a container-lifecycle `Sandbox`
  (create/mount/start/destroy/sweep) plus `main.go`. The Architecture-A Go tool
  layer and the persistent-shell machinery were removed — they served a path not
  taken (see the superseded `0002-activities` / `0003-process-lifecycle` epics).

## The arc (riskiest-first)

Each step is the thinnest slice that retires the next-biggest unknown. Only the
next one or two are detailed; later steps are direction, not commitment.

1. **Walking skeleton — the pod runs the harness (next).** A container image with
   the coding harness baked in; launch it on the mounted demo project with a
   hard-coded task; the harness edits the workspace and exits; collect the
   result. Retires the scariest unknown: *can we run the rented brain inside the
   box, unattended, and get a real edit back?* (ADR-0007.)
2. **Bounded-job lifecycle + result contract.** Shape the pod's I/O contract
   (repo in, task in, structured result out) — the one seam that ossifies, so
   worth getting clean.
3. **Durable supervision on Temporal.** Wrap the pod run as one bounded Temporal
   activity — launch, heartbeat, timeout, retry (ADR-0003; granularity per
   ADR-0007).
4. **Separate verification runtime.** Run the real tests in a clean place the
   agent can't touch, and gate the result on it (ADR-0005). Build early —
   highest reliability leverage.
5. **The orchestrator brain (business-ops, Go).** The provider-flexible outer
   loop: pull an issue / assemble context (RAG over the vision + project docs),
   decide what to dispatch, hand a task to the pod, consume the result, bring a
   human in at blockers.
6. **Feedback loops + observability.** Feed build/test/lint results back so the
   loop self-corrects; add tracing once it runs unattended.

For *why* any of this is rent-vs-build, see `VISION.md` + `adr/`; for facts about
how real teams split it, `references.md`.
