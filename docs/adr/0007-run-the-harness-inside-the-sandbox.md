# 0007 — Run the rented harness inside the sandbox (Architecture B)

**Status:** Accepted

**Context:** ADR-0001 said rent the inner loop behind a `Brain` interface. Read
one way, that implies an in-process Go brain that pipes each tool call into the
sandbox from outside (call it Architecture A) — which forces custom tool
implementations to redirect the harness's built-in bash/read/write/edit across
the process boundary, fighting a harness designed to run its tools locally.
Spotify (`research/spotify-honk.md`) did the opposite: they ran the Claude Agent
SDK *inside* Kubernetes pods, where its built-in tools act on the pod's own
filesystem, no piping. That surfaced the fork — brain-outside-piping-in
(A) vs harness-inside-the-pod (B) — as a real decision, not an assumption.

**Decision:** Architecture B. The rented coding harness runs **inside** the
sandbox container, operating directly on the mounted workspace. The orchestrator
launches the pod with a task and collects the result as a bounded job — it does
**not** pipe individual tool calls. The seam that keeps this from hardening into
lock-in is the **pod's I/O contract** (repo in, task in, structured result out),
not an in-process `Brain` wrapper.

**Alternatives:** Architecture A — brain outside, pipe read/write/bash into the
sandbox (rejected: needs custom tools to redirect the harness's built-ins, fights
its local-execution design, no precedent; its only edge — per-tool-call
durability — isn't needed since coding tasks are naturally re-runnable). A
managed/hosted agent runtime (rejected per the platform goal — single-vendor,
commodity relative to what I'm building).

**Consequences:**
- The Architecture-A Go layer is removed: `internal/tools`, `internal/activities`,
  and `internal/session` served piping and an in-process brain and now serve no
  one — the harness owns conversation and tool execution inside the pod. The
  container-lifecycle sandbox is kept.
- Durability shifts from per-tool-call to per-session: Temporal supervises the
  pod as one bounded activity (launch, heartbeat, timeout, retry). This refines
  ADR-0003's granularity, not its decision.
- Portability cost, accepted with a trigger: the coding brain is now tied to the
  harness image it runs (the Claude Agent SDK). That's *one app's* brain behind
  the pod contract, not the platform — the orchestrator brain and the platform
  stay provider-flexible. Revisit only at a concrete coding-brain model-swap
  trigger.
- A Python/TS process re-enters, but only as a bounded job *inside* the pod —
  "launch, wait, collect," not a persistent sidecar RPC. The platform stays Go.

**Refines:** ADR-0001 — the rented loop runs inside the sandbox; the anti-lock-in
seam is the pod I/O contract, not an in-process `Brain` interface.
