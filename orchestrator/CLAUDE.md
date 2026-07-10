# Orchestrator — working rules

Go module. All checks run through the `Makefile` — it is the single source of
truth; CI (later) just calls its targets.

## Checks

- **While editing (the AI's loop):** run `make check` — fast: format, vet, lint,
  test. This is the feedback loop, and it must stay green.
- **Advisory checks are human-run, not the AI's:** `make security`, `make vuln`,
  `make deadcode`. Do not run these as a gate or auto-"fix" their findings. They
  surface things Mattias reviews and may consciously accept (a prototype security
  finding, a vuln with no upstream fix, code that's dead only because it isn't
  wired up yet). Leave those calls to him.

## Suppressing a lint finding

If a linter is genuinely wrong, suppress narrowly: `//nolint:<linter> // <reason>`
(the form is enforced by `nolintlint`). Prefer fixing over suppressing.

## Committing a slice

Don't commit unless asked. When you do, write the message to capture intent —
the *why* and the context the diff can't show — not a restatement of what
changed. If the slice touches an architectural decision, reference the ADR
(e.g. "implements ADR-0004") instead of repeating it. `make precommit` must pass
first.

## Skills

- **Finishing a session or feature here:** use the `surface-lessons` skill.
