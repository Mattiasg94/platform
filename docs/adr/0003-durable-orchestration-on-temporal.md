# 0003 — Build durable orchestration ON Temporal, don't hand-roll it

**Status:** Accepted

**Context:** The platform needs durable, resumable execution — crash recovery,
retries, long-running lifecycle, human-approval waits, compensation on mid-run
failure. An in-process Go loop would teach me little and reinvent a solved
substrate. Temporal is the industry-standard durable-execution engine; the SDK
vendors are wiring *into* it (official Temporal integrations with Google ADK and
the OpenAI Agents SDK).

**Decision:** Rent Temporal's replay engine as substrate; **build my workflows on
top of it.** "Build Temporal now" means build the durable orchestration *policy*
on Temporal — never reimplement Temporal.

**Alternatives:** In-process Go loop / hand-rolled durability (rejected —
reinvents a substrate nobody rebuilds, and the durability *policy* on top is the
part that's actually the hire-able skill). A different engine (deferred — behind
the orchestration seam, swappable if ever needed).

**Consequences:** This is a *build-to-learn* seam: the orchestration policy is the
crown-jewel build and the interim must be minimal-but-real, never a throwaway
in-process loop. A trivial scaffold task just to give Temporal something to run is
fine, but gets zero learning effort.
