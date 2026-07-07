# Ramp + Google Jules (sandbox pattern + two-LLM pattern)

Two smaller but sharp data points. Grouped because each contributes one clean
fact rather than a full platform story.

## Ramp — the rented-sandbox pattern
- Built a **background coding agent** on **Modal's** infrastructure.
- **Ephemeral sandboxed VM per task**, spun up then torn down. Agent gets a full
  realistic dev stack (Vite, Postgres, Temporal) **without ever touching
  production data**.
- They did NOT build isolation primitives — rented Modal (gVisor-based). Sandbox
  = commodity infra, integrated via SDK.
- **Validates:** VISION's "stronger sandbox isolation → rent it (swaps the walls,
  not your policy)"; the agent-runtime ≠ production-data boundary.

## Google Jules — validates the two-LLM planner/executor model
- Remote async coding agent (Gemini 3 Pro). Trigger from any GitHub event
  (issue/PR/schedule/dispatch) → opens cloud VM → clones repo → plans →
  implements → runs tests → opens PR.
- **Isolated Google-managed VM per task**, own checkout of the repo. Keeps
  **network access** (to install deps / run builds) — a deliberate choice vs.
  competitors who disable network for stricter isolation. (Isolation is a
  spectrum; you pick the point.)
- **Planner/executor/reviewer split:** a stronger model produces the plan, a
  faster model executes steps, a **separate review pass verifies** before the PR
  opens. (Gemini Pro plans; Flash does light subtasks for cost.)
  → **This is your exact "orchestrator LLM + coding agent" mental model, shipped
  by Google.** Strong validation that the two-LLM split is real, not invented.
- **Auto CI feedback loop (2026):** if CI fails on a Jules PR, Jules receives the
  error, analyzes, fixes, re-pushes — often no human. → validates "feedback loops
  = build early, highest-leverage."

## Validates in VISION.md
- Rent the sandbox substrate (Ramp/Modal).
- Two-LLM orchestrator+coder split is an industry pattern (Jules).
- Isolation strength is a dial set by a trigger (network on/off), not a fixed max.
- Agent feedback loops from real CI signals (Jules) = the reliability lever.

Sources:
- [Ramp on Modal — best microVM sandboxes 2026](https://modal.com/resources/best-microvm-sandboxes-ai-code-execution)
- [AI agent sandbox landscape (Firecrawl)](https://www.firecrawl.dev/blog/ai-agent-sandbox)
- [Jules — Google Labs](https://blog.google/innovation-and-ai/models-and-research/google-labs/jules/)
- [Jules async coding agent guide (Digital Applied)](https://www.digitalapplied.com/blog/google-jules-gemini-async-coding-agent-guide)
