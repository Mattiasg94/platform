# 0002 — First app on the platform is the coding agent

**Status:** Accepted

**Context:** The platform can host many agentic apps (coding, business-ops
automation, RAG). Which to build first is close to a one-way door — it shapes the
early tool-pack and the demo. Re-shaken deliberately rather than assumed, because
I'd jumped to "build a coding agent" without weighing it.

**Decision:** Build the coding agent first — it's the superset. It has the
cleanest, hardest feedback loop (real tests, real pass/fail), which stresses the
verification and orchestration I most need to prove. It's the reversible choice:
the same platform skeleton carries the other apps by swapping the tool-pack.

**Alternatives:** Start with business-ops automation or RAG (rejected as *first* —
weaker/softer verification signal to prove the platform; but they remain the
planned later apps, which is the whole point of keeping the skeleton generic).

**Consequences:** Three guardrails ride with this: (1) rent/minimize sandbox
isolation; (2) later bolt on one business-ops demo to show range for target jobs;
(3) let data ride opportunistically via observability. The coding agent stays
*one app behind the `ToolExecutor` seam* — not the whole system.
