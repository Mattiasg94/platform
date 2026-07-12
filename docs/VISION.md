# Orchestrator — Vision (decision filter)

**What this doc is for:** a tight filter to hold every new feature against
*before* building it, so each step builds toward a system — and a skill set —
worth having, instead of a locally-reasonable step that adds up to the wrong
thing. It's meant to be injected into a session and stay short: every line
here should change a build-vs-defer decision. It holds **no** implementation
state, task lists, or time-ordering — those go stale and live in the roadmap,
PRDs/ISSUES, or git history.

Companion docs: `references.md` (the external fact-base — how real teams
build these systems, i.e. what industry treats as rent-vs-build; lean on it to
answer this doc's questions with facts, not guesses), `garbage.md` (the full
reasoning/evidence trimmed out of here, kept so nothing's lost), and the
roadmap (ordered execution plan — the "what next," which this doc deliberately
does not carry). Living document — revise as the goal sharpens.

## The goal

An **LLM orchestration platform** for autonomous, multi-step agentic
workflows: a provider-agnostic decision loop that, given a task, drives tools
inside an isolated sandbox to do real work unattended, with durability and
guardrails. The **first application** on it is a self-hosted autonomous
**coding agent** (clone/read a repo, edit, run, iterate).

But the product — and the transferable skill — is the **platform skeleton**,
not the coding flavor. Swap the tool set and the same skeleton runs any
agentic workflow (business-ops automation, RAG, document processing). So keep
the coding agent as *one app*, never the whole system. Built single-user
first, but production-shaped so "works for me" → "hosted for others" is a
series of swap-ins, not a rewrite.

## Primary lens: employability

Above all, this project is how you become **employable in AI engineering and
future-proof your career.** That is the primary goal and takes precedence
when priorities genuinely conflict. The working system is the vehicle; the
career outcome is the point. In practice the two rarely fight — the hire-able
skill *is* the real-system skill — so when choosing between two valid
options, optimize for **durable, transferable skill** (backend/platform
fundamentals that survive model churn) over what merely ships or what
re-teaches a commodity you could rent.

Operationally, this lens makes **learning a trigger in its own right.** A
capability may be built before the system feels any pain when the skill it
teaches is a career target. Scope it as a deliberate learning module and
build it minimal-but-real — production-shaped, done properly — because a
dirty hack teaches nothing and here the learning *is* the deliverable. The
system being small bounds **scope, never craft**: "one user, one box" is a
reason to build a small version well, not a rough version fast.

Two other things stay true and bound scope: it has exactly one user (you), so
don't pay multi-tenant/externally-exposed costs yet; and it's built with
production-shaped seams so later hardening is additive, not a redesign.

## Decision principles

**Gate complexity on a concrete trigger, not a vibe.** "Production-grade"
isn't a checklist — each capability is justified by a concrete trigger, and
**two kinds of trigger count**: a specific pain that starts hurting without
it, or a named skill target (the employability lens above — learning is a
trigger in its own right). No trigger of either kind → correctly out of
scope, not neglected. For any capability not in the table below, ask the
same question: what concrete, currently-true pain — or skill target —
justifies this *now*? None → defer it and write down what would trigger it.

**Rent the substrate, build the policy.** Substrate = the black-box
foundation nobody rebuilds even at expert companies (Docker/Firecracker
isolation; Temporal's replay engine). Rent it; learn its interface, not its
guts. Policy = how *your agent behaves* on top (hang detection, kill
strategy, what's returned to the model) — the substrate never authors this,
so build it yourself, once. Test: *"is this a black box even at companies
that specialize in it?"* Yes → rent. Behavior on top → build. (Building a
substrate once to *understand* it, then swapping in the rented version behind
an interface, is legitimate learning — just don't ship the hand-roll as if it
were the point.)

**Rent the commodity loop, build the platform.** A third category: the
opinionated commodity layer — the inner agent loop (parsing edits to files,
context/token management, the turn loop). You *could* build it, but it churns
every model release and off-the-shelf now beats a hand-roll → **rent it**
(Claude Agent SDK or similar). Per ADR-0007 the rented coding loop runs
*inside* the sandbox pod, so its anti-lock-in seam is the pod's I/O contract
(repo in, task in, result out), not an in-process wrapper. What you **build**
is the platform around it:
durable orchestration, a separate verification runtime, feedback loops,
deployment, observability. That platform is the career core — it's what
transfers. (Full Spotify/Honk evidence → `garbage.md`.)

**Build the skeleton generic.** The platform skeleton (Brain loop + pluggable
tools + sandbox + orchestration + verification + observability) is
application-agnostic. Keep the tool layer pluggable so the coding agent is
one app, not the whole system — the generic skeleton is what makes the work
transfer to any LLM application.

**Build behind swappable interfaces.** Every substrate/commodity you rent
sits behind a seam you own — the `Sandbox` interface, a `Brain` for the
orchestrator's own decision loop, and the pod's I/O contract for the rented
coding loop (ADR-0007). Build the concrete thing you need now behind the seam,
so the deferred/harder/rented version slots in later without touching callers.

**Never hand-roll a throwaway for a build-to-learn seam.** Classify each seam
first: is its *target* a rented commodity, or the transferable skill itself?
The build-to-learn set is **open, and the call is Mattias's**: any seam
becomes build-to-learn the moment its skill is a career target (durable
orchestration on Temporal, the verification runtime, feedback loops are
examples, not the closed set — the telemetry/data layer joined the same way).
For a **build-to-learn** seam, the interim must be a *minimal-but-real
version of the target* — never a hack you'll delete, because the building is
the point and a throwaway teaches none of the hire-able skill. Interim stubs
are only for **rent** seams (sandbox, inner loop, observability backend), where
building the real thing teaches nothing and you swap in the rented version at
its trigger. One exception: a scaffold that merely *unblocks* a build-to-learn
seam (e.g. one trivial task so Temporal has something to orchestrate) is fine —
but invest zero learning effort in it and don't mistake it for the real thing.

## Capability → trigger table

Illustrative, not exhaustive. When a new need shows up, add a row with the
same shape (capability → trigger) rather than treating the doc as closed.

| Capability | Example tool(s) | Build now, or gated on... |
|---|---|---|
| Durable, resumable workflow execution | Temporal | **Build now — your workflows *on* Temporal**, not a reimplementation of it. Temporal's replay engine is rented substrate (nobody rebuilds it); the durable orchestration *policy* on top is the crown-jewel build — that's what platform teams hire for. Pays off once runs get long or multi-step; loop is already shaped for it. |
| The inner agent loop (file-edit, context/token mgmt, turn loop) | Claude Agent SDK, OpenAI Agents SDK | **Rent now** — commodity that churns each model release; runs inside the pod, kept swappable by the pod's I/O contract (ADR-0007). |
| Adversarial verification (separate runtime; the agent can't verify or cheat on its own work) | your own CI + git diff, isolated runtime | Build when the agent acts on its own output unattended (opens PRs, self-verifies). Keep it **light**. |
| Agent feedback loops (build/test/lint results fed back so it self-corrects) | your own wiring + real CI signals | **Build early** — highest-leverage reliability lever, entirely yours to build. |
| Stronger sandbox isolation (beyond a plain container) | Docker Sandboxes (Firecracker microVM), gVisor, E2B | Gate on: someone other than you can submit input. Swaps the walls, not your policy. |
| LLM tracing / observability | Langfuse, Helicone, OpenTelemetry | Gate on: the agent runs unattended. Career caveat: build a *taste* pre-trigger — job posts expect you to speak to it. |
| System telemetry pipeline (structured logs → collector → store → dashboards) | slog + Vector/OTel Collector, Loki, Grafana | **Build now as a learning module** (skill-target trigger). Rent the collector and backends (substrate); build the instrumentation — structured JSON logs to stdout only, correlation IDs, dashboards. App never knows where logs go. Kafka/Elasticsearch stay out: their trigger is fleet-scale volume. |
| Business-events pipeline (append-only events table → views → warehouse → dbt) | Postgres now; BigQuery + dbt later | **Events table + SQL views now** (build-to-learn — the data layer is a skill target). Separate pipeline from telemetry: low-volume, loss-intolerant, Postgres is the system of record. Warehouse + dbt gate on: enough run history to analyze. *Copy* events out, never move-and-delete. |
| Prompt/eval regression testing | promptfoo, Braintrust | **Strong defer** — gate on repeatedly breaking things you already fixed while iterating on prompts/models. |
| Prompt versioning/management | (bundled into the tracing tool) | Don't adopt separately — rides along with whichever tracing tool is adopted above. |
| Local model serving | Ollama, LM Studio | Opportunistic only — dev-loop iteration; never load-bearing (local models weaker at reliable tool-calling). |

## Explicit non-goals for now

Not "never" — "no concrete trigger yet, don't build preemptively":
- Multi-tenant support / per-user isolation or auth.
- Managed hosting/deployment infra *as a product* (deploying your own single
  instance to learn hosting is fine, and belongs on the roadmap).
- Cost optimization beyond "don't be wasteful."
- A dedicated prompt-management product separate from tracing.
- Heavy data engineering (Kafka/streaming, big-data scale) as a forced
  feature. Light, opportunistic DE via data-shaped observability (structured
  events → warehouse → dbt → metrics over agent runs) is welcome; the heavy
  end is a separate later module, only if hybrid data roles stay a target.

## How to use this doc when designing a feature

1. Does the feature serve the goal (the platform, or its first app), or solve
   a problem the goal doesn't have?
2. Is it a rent-or-build call? Apply the three rent/build principles above.
3. If it's a "production-grade" capability, is its trigger true *today*, or
   is it being added because it *feels* like what a real system would have?
4. Keep the seam it plugs into swappable (`Sandbox`, `Brain`, or the pod's I/O contract per ADR-0007).

## How to advise on decisions (for an AI reading this doc)

The system-design skill being trained here is *choosing between paths with
understanding* — so when a build decision is on the table, lay out the
realistic option landscape and its trade-offs before recommending, then
recommend one path and say why. A single pre-collapsed answer defeats the
point of the exercise.

Never argue "the system is too small to justify doing this properly" —
smallness bounds scope, not craft (see the employability lens). Suggesting a
prototype, a thinner first slice, or an iterative path is welcome; steering
toward a throwaway hack on a build-to-learn seam is not, because the
learning is the deliverable and a hack teaches nothing. When you'd defer
something, name the missing trigger (pain or skill target) rather than
appealing to simplicity.
