# 0006 — Continue on the existing codebase; prune the homegrown loop

**Status:** Accepted

**Context:** Issue #1 (sandbox process lifecycle) is partly done. It mixes two
things: real `Sandbox`-seam work (a plain container behind an interface) and a
hand-rolled slim agent loop I wrote before deciding to rent the inner loop. The
homegrown loop now contradicts ADR-0001. Question raised: start from a clean slate
or continue?

**Decision:** Continue on the existing codebase. Do a **surgical prune**, not a
restart: remove the hand-rolled agent loop / mini-SDK, keep the sandbox
process-lifecycle work behind the `Sandbox` interface. No cleanup-PRD ceremony —
the prune is a small slice done by reading the current code against ADR-0001 and
ADR-0004.

**Alternatives:** Delete everything and start fresh (rejected — throws away real
Sandbox learning for no gain; both are reversible, continuing is cheaper). Keep
the homegrown loop (rejected — it's the rented commodity per ADR-0001).

**Consequences:** After the prune, the next build is the walking skeleton
(roadmap P1): the rented SDK behind `Brain` issues one command into the existing
`Sandbox`, output comes back. Parts of issue #1 tied to the homegrown loop are
dropped; the sandbox parts carry forward.
