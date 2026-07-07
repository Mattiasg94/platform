# Spotify — Honk (background coding agent)

The richest public case study, and the regret goldmine. Full narrative also in
`../garbage.md`; this is the spine-organized version.

## RENTED
- **The inner agent loop** — abandoned their homegrown loop for the **Claude
  Agent SDK**. Their loop needed rigid step-by-step prompts; Claude Code did
  better with "describe the end state, leave room to figure out how." → the loop
  is a commodity; renting beats hand-rolling. (This is the exact mistake VISION
  is built to prevent.)
- Claude (the model) via the SDK. GCP for log storage, MLflow for traces.

## BUILT — the moat
- **Fleet Management / Fleetshift** (infra, NOT an LLM): identifies targets,
  schedules changes, tracks progress across thousands of repos. Honk sits in the
  middle doing edits; Fleetshift is the control plane. Predates the agent
  (since 2022).
- **Own harness** around the SDK, in **Kubernetes pods** for concurrency.
- **Verification loop** via local MCP tools (format/lint/build/test); agent loops
  on failures and only opens a PR after full verification passes.
- Integration with Backstage for component ownership; flow: Slack → Honk → Fleet
  Management → verification loop → automated PR.
- **Context engineering** (Honk Part 2) — what goes in the window, how the task
  is framed. Built by doing, not by adopting a tool.
- **Feedback loops** (Honk Part 3) — reliability came from strong build/test/lint
  feedback loops, NOT a smarter model or a stricter gate.

## REGRET — the "don't build this" / "we threw this away" list (highest value)
- **Agents cheated to force green builds** — commented out failing tests,
  downgraded Java versions. → you need a *separate* verification runtime the
  agent can't touch.
- **First verification was LLM-as-judge — too rigid, blocked valid changes, and
  was REMOVED** as models improved. Verification steps in prompts + real CI
  proved sufficient. → don't over-build rigid verification early; keep it light.
- **Prompt-eval: they fly "mostly by intuition"** — no structured prompt/model
  eval. → structured prompt-eval (promptfoo/Braintrust) is not where early effort
  pays. (Directly justifies VISION's "strong defer" on prompt-eval.)

## Their own conclusion
**"Infra is the moat."** The value was never the model or the loop — it was years
of platform infra (Fleet Management, Backstage, standardized builds, test suites)
around it. Can't replicate that scale solo, but the *shape* — durable
orchestration + separate verification + feedback loops — transfers.

## Results (credibility)
1,500+ merged PRs in 9 months; ~1,000 merged PRs / 10 days by Mar 2026;
60–90% time savings on migrations.

## Validates in VISION.md
- Rent the loop / build the platform (they lived both halves).
- Separate verification runtime + "keep it light."
- Feedback loops = "build early, highest-leverage."
- Prompt-eval = strong defer.

Sources:
- [Honk Part 1](https://engineering.atspotify.com/2025/11/spotifys-background-coding-agent-part-1)
- [Honk Part 2 — context engineering](https://engineering.atspotify.com/2025/11/context-engineering-background-coding-agents-part-2)
- [Honk Part 3 — feedback loops](https://engineering.atspotify.com/2025/12/feedback-loops-background-coding-agents-part-3)
- [Honk Part 4 — dataset migrations](https://engineering.atspotify.com/2026/4/background-coding-agents-dataset-migrations-honk-part-4)
- [InfoQ QCon writeup](https://www.infoq.com/news/2026/03/spotify-honk-rewrite/)
