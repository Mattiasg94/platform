# Project Context

## Layout

This repo is the **platform** monorepo — one project, two isolated components
that only ever talk through the pod's I/O contract (ADR-0007):

- `orchestrator/` — Go. The supervising brain: launches the pod, collects the
  result. Its own module; knows nothing of the agent's language.
- `agent/` — Python. The rented coding harness (Claude Agent SDK) baked into a
  container image. Edits the mounted workspace and exits.

`demo-project/` is the fixture repo the agent edits. `docs/` is shared.

## Docs

`docs/` is the canonical source of truth for the platform. It splits directly
into **epics**, each with its own `PRD.md`/`ISSUES.md` under
`docs/epics/<NNNN-slug>/` — see `docs/working-method.md` §3. For the full map and
reading order, start at `docs/map.md`, then `docs/working-method.md` §8.

## Working Method

Before moving the orchestrator forward, read `docs/working-method.md` — it
defines how we work: the plan→build→feedback loop, vertical slices, how much to
plan (reversibility calibration), when to write an ADR, doc structure, and the
rate-of-change rule that keeps docs from rotting.

## Writing AI Instructions

When writing an open-reasoning step for AI (skills, prompts, guidance docs): state the goal first, examples second, and never let the examples double as the stopping point. If examples are given, mark them explicitly as illustrative, not exhaustive, and instruct the AI to keep reasoning past them toward the stated goal.
