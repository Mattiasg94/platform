# Working Method — how we run a project

**What this doc is for:** the *how*, not the *what*. It tells any AI session (and
me) how to move a project forward — the build loop, how to plan, when to write
things down, and how to keep the docs from rotting. It is pure method: it holds
**no** project-specific facts, so it never goes stale. Referenced from
`CLAUDE.md` so it loads into every session. Goals live in `VISION.md`/`PRD.md`;
live state lives in code, `ISSUES.md`, and the decision log — never here.

The goal of everything below: **make real progress by building thin, learning
from what runs, and writing down only what won't rot.** The practices are the
means; if one ever fights that goal, the goal wins.

---

## 1. The build loop (the core method)

Planning is **not a phase you finish before building.** It's one turn of a loop
you ride the whole project:

```
plan a thin slice → build it for real → run it, get real feedback
      ↑                                              │
      └──────── update the plan / log a decision ←───┘
```

- **Plan a thin slice.** Not the whole system — the next *one* vertical slice
  (see §2). Enough to start, no more. If you're planning slice 5 while slice 1
  isn't built, stop.
- **Build it for real.** Minimal, but real — never a throwaway hack for a seam
  whose target is a skill worth having (see the project's VISION for the
  rent/build rule). A stub is only allowed for a rented commodity.
- **Run it. Get real feedback.** The running code is the source of truth, not
  your prediction of it. This is where the plan gets corrected.
- **Update the plan / log the decision.** If you learned something that changes
  direction, that's an ADR (§4). Then plan the next slice.

**The first slice is always the walking skeleton:** the thinnest end-to-end path
that proves the scariest unknown works at all. After that, order slices
**riskiest-first** — each slice should retire the biggest remaining unknown.

---

## 2. Vertical slices (tracer bullets)

Each slice is a **thin cut through every layer**, not a horizontal layer across
the whole system. A finished slice **runs and is demoable on its own.**

- Good: "one task → one command runs in the sandbox → output comes back" (thin,
  but end-to-end, and you can watch it work).
- Bad: "build the entire orchestrator layer" (a horizontal slab that demos
  nothing until everything else exists).

The `/to-issues` skill turns a plan into these slices. Detail only the next slice
or two — slices further out will change once the near ones teach you something,
so writing them in detail now is wasted work.

---

## 3. Epics — the top-level working unit

This repo's docs split directly into **epics** — self-contained chunks, each
with its own `PRD.md`/`ISSUES.md`, living in `docs/epics/<NNNN-slug>/`. There is
no project layer above the epic; the repo knows only itself. An epic ≠ a slice
(§2): the epic is the grouping (e.g. "sandbox lifecycle"), the slice is the thin
vertical cut you actually build next, inside that epic's `ISSUES.md`.

Number epics sequentially (`0001-`, `0002-`, ...) in the order they were
opened, not by priority — the roadmap or ISSUES already carries priority. The
`map.md` points at `epics/` (see §8).

---

## 4. How much to plan (reversibility calibration)

Planning depth is not a constant. Spend it where being wrong is expensive:

**planning depth ∝ (cost of being wrong) × (uncertainty)**

- **Two-way door** (cheap to reverse): decide fast, move, correct later from real
  feedback. Over-planning here is procrastination.
- **One-way door** (expensive to reverse): plan hard — but plan it *once*, shake
  the decision to confirm your certainty is real, then commit. Don't re-litigate
  a settled one-way door every session.
- **Swappable seams turn one-way doors into two-way doors.** Anything rented sits
  behind an interface you own, so "which vendor" stops being a one-way door.

The most expensive mistake is false certainty on a one-way door. The second most
expensive is treating every two-way door as if it were one-way — that's how
planning eats the time that should have gone to building.

---

## 5. When to write a decision down (ADRs)

An **ADR** (Architecture Decision Record) is a short, dated, **append-only** note
capturing a decision: the context, what was chosen, the alternatives, and the
consequence. They live in `docs/adr/NNNN-title.md`.

Write one when you **choose among real alternatives** or **pivot** (a prototype
failed, a direction changed). Do **not** edit old ADRs — if a decision is
reversed, write a *new* ADR that supersedes it and mark the old one Superseded.

Why this matters: ADRs are the **drain for churn.** Because they're append-only,
they can't rot, and they let the slow docs (VISION, the map) stay clean. The
current state of the project's thinking = VISION + the sum of the ADRs. Keep each
one short — a screen or less.

---

## 6. Keeping docs from rotting (the rate-of-change rule)

Sort every doc by how fast it changes, and **never let a slow doc hold fast
facts.** That single rule removes almost all maintenance:

| Tier | Changes | Examples | Rots? |
|---|---|---|---|
| Slow — why & rules | rarely | `VISION.md`, `references.md` | ~never |
| Slow — how we work | ~never | this doc | ~never |
| Slow — orientation | rarely | the project `map.md` | ~never — it's pointers |
| Append-only — ADRs | you add, never edit | `adr/*.md` | can't |
| Fast — live state | constantly | the code, `ISSUES.md`, git log | self-maintains |

Rot happens in exactly one place: hand-writing **fast state into a slow doc.** So
don't. The orientation doc *points* at code/issues/ADRs for current state; it
never restates them. State you'd be tempted to write in prose already lives, more
truthfully, in the code and the issue checkboxes.

---

## 7. How to help me (working style)

These bind the AI in the loop, not just the code:

- **Don't over-ask.** I get decision fatigue and dislike option-surveys.
  Recommend one path, state the sensible default you picked, and proceed. Ask
  only when the answer genuinely changes what you build.
- **Don't be a sycophant.** On a real decision, pressure-test my reasoning first,
  then endorse or push back with reasons. Agreement I didn't earn is worthless.
- **Apply the rent/build gate every time.** Before building anything, check the
  project's `VISION.md` and `references.md`: is this a commodity to rent, or the
  transferable skill to build? Renting the wrong thing wastes money; building the
  wrong thing wastes the scarce learning time.
- **Keep me moving.** If planning is circling, name it and push toward the next
  slice. Building finishes the plan.

---

## 8. The map for a fresh session

Opening a file means reading all of it — there's no partial read. So treat each
doc as an **open/skip decision**, not a fixed checklist to work through:

1. `CLAUDE.md` (loads automatically) — repo conventions + a pointer here.
2. This doc — how we work. Always worth opening, it's small.
3. The project's `map.md` — what the project is and where everything lives.
   Always open on a cold start.
4. From there, open only what the current task actually needs: `VISION.md`
   (decisions), `references.md` (facts), a specific ADR — pick by relevance,
   don't open all of them by default.
5. **`PRD.md` / `ISSUES.md` inside an `epics/<NNNN-slug>/` (§3) — usually skip.**
   They're fast-tier, disposable
   planning state, not orientation. Open one only if the task at hand *is* the
   specific problem that file tracks — e.g. you're actively working the
   feature it describes. Otherwise the code is the true current state; the
   file adds nothing and just burns context.
