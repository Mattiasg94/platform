# Orchestrator — References (how the industry builds these systems)

**What this doc is for:** the external fact-base under `VISION.md`. VISION makes
rent-vs-build decisions; this doc is the *evidence* those decisions rest on —
what real teams (Spotify, Uber, Google, Ramp, Anthropic) actually treat as
commodity to rent vs. differentiator to build. When VISION asks "is this worth
building myself?", the answer should be traceable to a fact here, not to a guess.
This is the doc that would have stopped the "build my own agent SDK" mistake.

It holds **facts, not plans** — no roadmap, no task order. Detailed per-company
notes live in `research/`; this is the consolidation. Living document — add a row
when new evidence lands.

## The one fact that governs everything

Every company studied — coding agents *and* non-coding LLM platforms — splits the
system the same way. The split is unanimous, which is what makes it a fact rather
than an opinion:

| Layer | Verdict | Who proves it |
|---|---|---|
| The model | **Rent** | everyone |
| The inner agent loop (file-edit, tool loop, context mgmt) | **Rent** (SDK) | Spotify (killed their own), Anthropic SDK, Uber |
| The sandbox / isolation | **Rent** | Ramp/Modal, E2B, Vercel, Jules |
| The durable-execution *engine* | **Rent** (Temporal) | industry standard |
| Orchestration *policy* (the durable workflow) | **Build** | Spotify, Uber |
| Verification runtime (separate, agent can't touch) | **Build** | Spotify, Jules, Uber |
| Feedback loops (real CI signals fed back) | **Build** | Spotify, Jules |
| Permissions / governance seam | **Build** | Uber |
| Observability backend | **Rent**; **build** the OTel seam | industry standard |

**The built column is identical across coding and non-coding platforms.** That's
the load-bearing fact: the platform skeleton is general, not coding-specific —
which is exactly why the skill transfers to any LLM application, and why VISION
keeps the coding agent as "one app, not the whole system."

The rented column is the mistake column: hand-rolling anything in it is the
"reinventing the wheel" trap.

---

## Layer-by-layer: the evidence

### 1. Inner agent loop — RENT (this is the sharpest fact)

Anthropic's own Claude Agent SDK docs list what ships vs. what you build. What
**ships** (rent it): the agent loop, built-in tools (bash/read/write/search),
tool-use protocol, streaming, prompt caching, a permission system, context
compaction, subagent spawning.

What the SDK **explicitly says you must build** — this *is* the platform: durable
execution & checkpoint recovery, distributed tracing/observability, per-agent
permission scoping, structured handoffs, security (prompt-injection defenses,
least-privilege, PII, audit), deployment strategy, evaluation pipelines. Their
estimate to close that gap: **2,200–4,500 engineer-hours.** That number is the
career core.

**Spotify is the lived version:** they abandoned their homegrown loop for the
SDK. Their loop needed rigid step-by-step prompts; Claude did better with
"describe the end state, leave room to figure out how." The loop churns every
model release — renting beats hand-rolling.

### 2. Sandbox / isolation — RENT

Industry consensus 2026: isolation is commodity infra rented via SDK, even at
massive scale. Nobody manages isolation primitives by hand.
- **Firecracker microVMs** (own kernel, hardware-level) — E2B, Vercel, AWS
  Lambda. For the strongest boundary.
- **gVisor** (user-space kernel) — Modal, hence **Ramp**. For massive concurrency
  (100k+ sandboxes; Quora hit 1,000/sec).
- **Hardened containers** (Sysbox) — Daytona. Middle ground.

*"Your container is not a sandbox."* A plain Docker container is the weakest point
on the spectrum — fine while trusted/single-user, not fine once untrusted input
arrives. Matches VISION's trigger exactly: plain container now, rent a
microVM/gVisor sandbox when someone other than you can submit input. **Ramp's
pattern:** ephemeral VM per task, full realistic dev stack (Vite/Postgres/
Temporal), never touches production data.

### 3. Durable orchestration — BUILD, on a RENTED engine

The production standard is a two-layer architecture:
- **Intelligence layer** (rent): the framework/SDK/loop — solves
  prompting/chaining.
- **Durability layer** (rent the engine, build the workflows): **Temporal** —
  durable execution, retries, state persistence, long-running lifecycle,
  human-approval waits, compensation on mid-run failure.

Frameworks "don't solve the infrastructure problem: crash recovery, durable
state, long-running approvals, compensation halfway through." Temporal does —
it's the same gap the SDK lists as "you build this."

**Precision that matters for VISION:** Temporal is a rented *substrate* (nobody
rebuilds its replay engine). You build the orchestration **policy on top** — the
durable coding-agent workflow. "Build Temporal now" means "build your workflows
*on* Temporal now," never "reimplement Temporal." Signal it's the standard, not a
fad: Temporal shipped official integrations with Google ADK and the OpenAI Agents
SDK — the SDK vendors are wiring *into* it.

### 4. Verification runtime — BUILD, keep it separate, keep it light

- **Spotify:** agents *cheated* to force green builds — commented out failing
  tests, downgraded Java. → verification must be a separate runtime the agent
  can't touch.
- **Spotify (the "keep it light" fact):** their first verification was
  LLM-as-judge — too rigid, blocked valid changes, and was **removed** as models
  improved. Verification steps in prompts + real CI proved enough. → don't
  over-build rigid verification early.
- **Google Jules:** a separate review pass verifies before the PR opens.
- **Uber:** continuous code-scanning before production deploy.

### 5. Feedback loops — BUILD early (highest-leverage reliability lever)

- **Spotify (Honk Part 3):** predictability came from strong build/test/lint
  feedback loops — *not* a smarter model or a stricter gate.
- **Google Jules (2026):** if CI fails on a Jules PR, it auto-receives the error,
  fixes, re-pushes — often no human. The loop from real CI signal is the
  reliability mechanism.

### 6. Provider-agnostic Brain + permissions — BUILD the seam

- **Uber GenAI Gateway:** a Go service unifying external (OpenAI, Vertex) and
  internal models behind one interface with auth, caching, observability — ~16M
  queries/month. **This is literally the `Brain` interface in VISION, in
  production.**
- **Uber MCP Gateway:** centralized control plane between agents and 10,000+
  services; governs every agent→tool connection. The tool/permission seam.
- **Uber Agent Identity:** per-hop cryptographically attested tokens, every tool
  call traced to the initiating human. The grown-up form of "scoped git push" — a
  *later* trigger for VISION (single-user now), but this is what it grows into.

### 7. Two-LLM orchestrator + coder — a real, shipped pattern

**Google Jules ships your exact mental model:** a stronger model produces the
plan, a faster model executes the steps, a separate review pass verifies before
opening the PR. Your "orchestrator LLM + coding agent" split is not invented —
it's the shipped industry pattern. (Uber, too, treats the coding agent — Claude
Code / Cursor / their own Minions — as a *swappable consumer surface* on a shared
platform, not the moat.)

### 8. Observability — RENT the backend, BUILD the OTel seam

- **Build once:** instrument to **OpenTelemetry GenAI conventions** (v1.41:
  agent/workflow/tool/model spans + latency/token metrics). OTel = portability;
  Langfuse, Phoenix, LangSmith all ingest the same spans, so the trace format
  never vendor-locks.
- **Rent the backend**, by deployment model: **self-host Langfuse** (Apache-2.0,
  OTel-native, deepest OSS evals) — best fit for VISION's single-user,
  self-hosted, provider-agnostic shape; managed (LangSmith/Braintrust) for speed;
  proxy (Helicone/Portkey) for wire-level cost logging.
- Career caveat (from VISION): stand up self-hosted Langfuse behind an OTel seam
  early. Cheap, and it's the thing you speak to in interviews.

---

## Regrets / "what they threw away" (highest-value, but honestly thin)

The single most useful column for mistake-prevention — and it leans almost
entirely on **Spotify**, because the others publish roadmaps, not post-mortems.
Treat this as one strong witness, not a consensus:
- Killed their homegrown agent loop → rent the SDK.
- Killed LLM-as-judge verification (too rigid) → keep verification light, lean on
  real CI.
- Agents cheated on their own verification → separate, untouchable verification
  runtime.
- Fly "by intuition" on prompt-eval, no structured tooling → structured
  prompt-eval (promptfoo/Braintrust) is a strong-defer; even Spotify skips it.

Spotify's own one-line conclusion: **"infra is the moat."** The value was never
the model or the loop — it was years of platform infrastructure around it. You
can't match that scale solo, but the *shape* — durable orchestration + separate
verification + feedback loops — is exactly what transfers.

---

## How to use this doc

1. Weighing a build? Find its layer above. If the verdict is **Rent**, building
   it by hand is the reinvent-the-wheel trap — rent it behind an interface you
   own.
2. Verdict **Build**? It's platform/differentiator work — the transferable core.
   Do it.
3. New capability not listed? Ask the same question this doc answers for every
   row: *does industry treat this as commodity or moat?* Add a row with the
   evidence.
4. Pair with VISION: this doc is the *facts*; VISION is the *decision* those
   facts justify. If they ever disagree, the fact wins — update VISION.

*Sources: see per-company notes in `research/` (each file carries its own source
links).*
