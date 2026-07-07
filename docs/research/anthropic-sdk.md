# Anthropic — Claude Agent SDK (the commodity loop, and its exact seams)

**Why this one matters most:** it is the single clearest public statement of where
the rent/build line sits, because it's a vendor telling you precisely what they
give you vs. what you still have to build. This is the fact-base that would have
stopped the "build my own SDK" mistake.

## RENTED — what the SDK ships (do NOT build these)
The inner agent loop and everything commodity around it:
- The agent loop itself: *gather context → act → verify → repeat*.
- Built-in tools: bash, read, write, web_search; MCP integration.
- Tool-use protocol: parallel calls, dynamic discovery, strict schema enforcement.
- Streaming (SSE, sync/async, text iterators).
- Prompt caching (~90% input-cost reduction).
- A permission system routing tool calls through safety checks.
- Context compaction at a configurable token threshold.
- Subagent spawning / multi-agent (launched April 2026) — isolated context
  windows per subagent, only results returned to the orchestrator.

Renamed Claude Code SDK → **Claude Agent SDK** specifically to signal it's for
any agent, not just coding. It runs Claude Code as a subprocess; it's a *process
that runs/observes/acts/iterates*, not a request/response API.

## BUILT — what the SDK explicitly leaves to you (this IS the platform / the moat)
The vendor's own list of what's NOT included — i.e. the transferable career core:
- **Orchestration:** per-agent permission scoping, structured agent handoffs,
  multi-agent coordination beyond subagents-as-tools.
- **Operations:** distributed tracing / observability, **durable execution &
  checkpoint recovery**, monitoring/metrics/logging, circuit breakers, cost
  controls, emergency shutdown.
- **Security:** prompt-injection defenses, least-privilege access, PII controls,
  audit logging.
- **Deployment:** strategies for long-running agents surviving code changes.
- **Evaluation:** testing/validation pipelines.
- **Context/memory:** compaction lifecycle hooks, graceful degradation at limits,
  cross-session state persistence.

Their estimate to make this production-ready: **2,200–4,500 engineer-hours.** That
number *is* the platform. That's what companies hire for.

## REGRET / anti-pattern
Implicit: building the loop yourself is the mistake — the SDK now beats a
hand-roll and churns every model release. (Spotify is the explicit version of
this regret; see spotify-honk.md.)

## Validates in VISION.md
- "Rent the commodity loop, build the platform" — this is the literal seam list.
- Every VISION "build now" item (Temporal/durable exec, verification, feedback
  loops, observability) appears verbatim on the SDK's "you build this" list.

Sources:
- [Anthropic Agent SDK: What It Ships vs. What You Build (Augment)](https://www.augmentcode.com/guides/anthropic-agent-sdk-what-ships-vs-what-you-build)
- [Agent SDK overview — Claude Code Docs](https://code.claude.com/docs/en/agent-sdk/overview)
- [Building agents with the Claude Agent SDK (Anthropic)](https://www.anthropic.com/engineering/building-agents-with-the-claude-agent-sdk)
