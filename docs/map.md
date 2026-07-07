# Orchestrator — Map (start here)

**What this doc is for:** the single entry point for a fresh session. It says what
this project is, how the docs relate, and where to look for what. It is
**pointers, not state** — deliberately so, which is why it doesn't rot. For the
*current* state of the project, the code is the source of truth; `ISSUES.md`
only matters if you're actively working the slice it tracks. **Do not trust
this doc for state.**

---

## What this project is (in one breath)

An **LLM orchestration platform** (Go, `orchestrator/`): a provider-agnostic
decision loop that, given a task, drives tools inside an isolated sandbox to do
real work unattended, with durability and guardrails. The **first application** on
it is a self-hosted autonomous **coding agent** (clone a repo, edit, run,
iterate).

The product — and the transferable skill — is the **platform skeleton**, not the
coding flavor. Swap the tool-pack and the same skeleton runs any agentic workflow
(business-ops automation, RAG). The coding agent is *one app*, never the whole
system.

**The north star is employability.** This is how I become hireable in AI
engineering. When two valid options compete, optimize for durable, transferable
platform skill over what merely ships or re-teaches a rentable commodity. The
working system is the vehicle; the career outcome is the point.

---

## The rent/build spine (the one fact that governs everything)

- **Rent** (don't hand-roll): the model, the inner agent loop (Claude Agent SDK),
  the sandbox/isolation, the durable-execution *engine* (Temporal).
- **Build** (the career core): the orchestration *policy* on top of Temporal, the
  separate verification runtime, feedback loops, the permissions seam,
  observability.

Everything rented sits behind an interface I own (`Brain`, `Sandbox`,
`ToolExecutor`) so it stays swappable. The evidence for every rent/build call is
in `references.md`; the ADRs themselves are in `adr/`.

---

## Where everything lives

| Doc | What it holds | When to read it |
|---|---|---|
| `working-method.md` | how we work — the build loop, planning, doc hygiene | before doing anything |
| `map.md` (this) | orientation + pointers | starting cold |
| `VISION.md` | the decision filter — rent/build rules, capability→trigger table, non-goals | before any build-vs-defer call |
| `references.md` | the external fact-base — how Spotify/Uber/Google/Ramp/Anthropic actually split rent vs build | when a rent/build call needs evidence |
| `roadmap.md` | the ordered prototype arc (P1, P2, ...) and why that order — riskiest-first | before starting the next prototype |
| `adr/` | ADRs — the append-only log of what was decided and why | to see recent/pivoted decisions |
| `research/` | raw source notes backing `references.md` (Anthropic SDK, Spotify/Uber/Ramp/Jules writeups) | when `references.md` needs its primary source |
| `epics/<NNNN-slug>/` | this project split into epics — each has its own `PRD.md`/`ISSUES.md` | **usually skip.** Open only the one epic you're actively working, only if you need the slice breakdown |
| the code (`orchestrator/`) | the true current state | for what actually exists right now |

**Rule of thumb:** *why* → VISION + references + ADRs. *What's true right now* →
the code, always. An epic's `ISSUES.md` is a bonus lens on the slice you're
currently building — not a file to open by habit. If a doc and the code
disagree, the code wins.

---

## Reading order for a fresh session

1. `CLAUDE.md` (auto-loads) → `working-method.md` → this map.
2. `VISION.md`, then `references.md`.
3. `roadmap.md` for the prototype arc — what's next and why.
4. The latest few ADRs in `adr/`.
5. Then **the code** for where things actually stand. Open the relevant
   epic's `ISSUES.md` too, but only if the task is that specific slice —
   otherwise skip it, the code already tells you what's true.
