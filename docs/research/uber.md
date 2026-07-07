# Uber — Minions + MCP Gateway + GenAI Gateway + Agent Identity

The reference for the **control plane / governance** half. Uber's thesis: build
the governance layer first, then agents are cheap and swappable on top.

## RENTED / swappable (they do NOT lock to one)
- **The coding agents themselves** are interchangeable: Claude Code, Cursor, AND
  their own Minions all interface the *same* registry via a CLI (AIFX). The agent
  is a consumer surface, not the moat.
- External LLMs (OpenAI, Vertex) sit behind their own gateway alongside internal
  models — provider-agnostic by design.

## BUILT — the moat (all in-house, all infra not LLM)
- **MCP Gateway + Registry** — a *centralized control plane* between agents and
  10,000+ internal services. Auto-translates proto/thrift service defs into MCP
  tool descriptions (using an LLM to do the translation). Service owners control
  which tools are exposed. Governs every agent→tool connection. Runs ~60k agent
  tasks/week.
- **GenAI Gateway** — a Go service unifying external + internal LLMs behind one
  interface with auth, account mgmt, caching, observability. **This is exactly
  the `Brain` interface / provider-agnostic seam in VISION.** ~16M queries/month.
- **Agent Identity** (built 2025, published May 2026) — cryptographically
  attested tokens for *every agent hop* in a pipeline; traces every tool call
  back to the human who initiated it; enforces authz at each hop, not just entry.
- **Minions** — internal background coding-agent platform: monorepo access,
  optimized defaults. ~1,800 code changes/week, used by 95% of engineers. Submit
  via web UI / Slack / CLI → runs on Uber infra (not vendor cloud) → Slack ping
  with a ready GitHub PR minutes later.

## Security/permissions pattern (answers the "push with different permissions" Q)
- Auth on by default for sensitive data.
- Automated PII redaction on tool outputs.
- Continuous code scanning before production deploy.
- Write-operation guardrails on critical services.
- Different trust tiers for internal vs. third-party MCPs.

## REGRET
None published (roadmap-forward article only). Absence of regrets ≠ absence of
mistakes; treat as thin on this axis.

## Validates in VISION.md
- Provider-agnostic `Brain` behind an interface = literally their GenAI Gateway.
- The tool layer / permissions as a governed seam = their MCP Gateway.
- "Coding agent is one app, not the whole system" — Uber treats the agent as a
  swappable consumer surface on a shared platform. Strong external proof.
- Agent identity/permissions = a *later* trigger for VISION (single-user now),
  but this is what "scoped git push" grows into.

Sources:
- [How Uber runs 60,000 AI agent tasks/week with MCP (AAIF)](https://aaif.io/blog/how-uber-runs-60000-ai-agent-tasks-per-week-with-mcp/)
- [How Uber uses AI for development (Pragmatic Engineer)](https://newsletter.pragmaticengineer.com/p/how-uber-uses-ai-for-development)
- [Inside Uber's AI & MCP gateways (Codingscape)](https://codingscape.com/blog/inside-ubers-ai-mcp-gateways-blueprint-for-engineering-leaders)
- [How Uber built the enterprise AI security playbook (Speakeasy)](https://www.speakeasy.com/blog/uber-enterprise-ai-playbook)
