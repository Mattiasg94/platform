# Cross-cutting landscape — the rent/build line, industry-wide

Not one company — the *consensus* across the field on two layers VISION treats as
load-bearing. This is where the "facts, not opinion" live.

## Sandbox isolation — RENTED, universally
The industry consensus in 2026: isolation is **commodity infra you rent via SDK,
not build.** Even teams at massive scale don't manage isolation primitives.

Three approaches, chosen by trigger:
- **Firecracker microVMs** (separate kernel, hardware-level). Used by E2B, Vercel;
  AWS Lambda internally. Choose for: ephemeral workloads needing the strongest
  security boundary.
- **gVisor** (user-space kernel intercepting syscalls). Used by **Modal** (hence
  **Ramp**), with custom syscall filtering. Choose for: massive concurrent scale
  + GPU. Modal runs 100k+ concurrent sandboxes; Quora pushed 1,000 sandboxes/sec.
- **Hardened containers** (e.g. Sysbox). Used by Daytona. Middle ground.

Key line: *"Your container is not a sandbox."* A plain Docker container is the
*weakest* point on the spectrum — fine single-user/trusted, not fine once
untrusted input arrives. Matches VISION's trigger: plain container now, rent a
microVM/gVisor sandbox when someone other than you can submit input.
- E2B: 94% of Fortune 100 use it. Sandboxes = a bought service, not a build.

## Durable orchestration — BUILD (on a RENTED engine)
The production standard for reliable agents is a **two-layer** architecture:
- **Intelligence layer** — the framework/SDK/loop (LangGraph, CrewAI, AutoGen,
  Claude Agent SDK). Solves prompting/chaining/coordination. This is the rented
  commodity.
- **Durability layer — Temporal.** The outer orchestration: durable execution,
  fault tolerance, automatic retry, state persistence, long-running lifecycle,
  cross-service coordination, human-approval waits, compensating transactions.

The explicit gap: frameworks "**don't solve the infrastructure problem**: crash
recovery, durable state, long-running approvals, compensation when things fail
halfway." Temporal does. This is precisely the SDK's "you build this: durable
execution & checkpoint recovery" line (see anthropic-sdk.md).

Signal it's the standard, not a fad:
- At Replay 2026, Temporal shipped integrations with **Google ADK** and the
  **OpenAI Agents SDK** — the SDK vendors are wiring themselves *into* Temporal.
- Gartner: 40% of enterprise apps embed task-specific agents by end of 2026
  (up from <5% in 2025) → durability stops being optional.

**Nuance for VISION:** Temporal itself is a rented *substrate* (nobody rebuilds
its replay engine). What you *build* is the orchestration *policy/workflows* on
top — the durable coding-agent workflow. So "build Temporal now" in VISION means
"build your orchestration ON Temporal now," not "reimplement Temporal." Worth
making that precise in references.md.

## Observability / tracing — RENT the backend, BUILD the instrumentation seam
The one row VISION half-covered. Industry-standard split:
- **BUILD the seam once: instrument to OpenTelemetry GenAI conventions.** OTel is
  the emerging standard (v1.41 defines agent/workflow/tool/model spans + latency &
  token metrics). Instrumenting to OTel = portability: multiple backends
  (Langfuse, Phoenix, LangSmith, Laminar) ingest the same spans, so you never
  vendor-lock the trace format. This is the seam you own.
- **RENT the backend.** Pick by deployment model first, features second:
  - **Self-host (Langfuse, Arize Phoenix)** — data stays in your infra, cost
    control. Langfuse: Apache-2.0, OTel-native, deepest open-source eval
    primitives. **Best fit for VISION** (single-user, self-hosted,
    provider-agnostic) — it's the "build a taste" tool without a SaaS bill.
  - **Managed (LangSmith, Braintrust)** — backend+UI+scorers out of the box;
    LangSmith is LangChain/LangGraph-native, weak if stack-agnostic.
  - **Proxy gateway (Helicone/Portkey)** — 5-min wire-level cost logging, but
    weakest agent-trajectory visibility.
- Common production shape: gateway (cost/routing) + eval tool (quality) side by
  side, tied together by OTel export.

**Validates in VISION:** observability row = rent backend / build OTel seam.
The career caveat ("build a taste pre-trigger, job posts expect it") → stand up
self-hosted Langfuse behind an OTel seam early; cheap, and it's the thing you
speak to in interviews. Nobody builds the tracing backend by hand.

## The one-line synthesis (for VISION's fact-base)
Across every company: **rent the model, the inner loop, the sandbox, and the
durable-execution engine. Build the orchestration policy, the verification
runtime, the feedback loops, the permissions/governance, and the observability.**
The built column is identical across coding agents (Spotify, Uber, Jules) AND
non-coding platforms (Uber's gateways) — proving the platform skeleton is
general, not coding-specific. That generality is the transferable career skill.

Sources:
- [Best microVM sandboxes for AI code execution 2026 (Modal)](https://modal.com/resources/best-microvm-sandboxes-ai-code-execution)
- [Your container is not a sandbox: microVM isolation 2026](https://emirb.github.io/blog/microvm-2026/)
- [AI workflow orchestration: durable agent pipelines with LangGraph + Temporal](https://aiworkflowlab.dev/article/ai-workflow-orchestration-in-production-building-durable-agent-pipelines-with-langgraph-and-temporal)
- [Agentic AI workflows: why orchestration with Temporal is key (IntuitionLabs)](https://intuitionlabs.ai/articles/agentic-ai-temporal-orchestration)
- [Temporal — durable execution](https://temporal.io/)
- [Helicone vs Langfuse vs LangSmith: LLM observability 2026 (Particula)](https://particula.tech/blog/helicone-vs-langfuse-vs-langsmith-llm-observability)
- [AI agent observability 2026: tracing & monitoring stack (Digital Applied)](https://www.digitalapplied.com/blog/ai-agent-observability-2026-tracing-monitoring-stack-guide)
- [Best LLM observability tools 2026 (Firecrawl)](https://www.firecrawl.dev/blog/best-llm-observability-tools)
