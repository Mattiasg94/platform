# 0008 — The project declares its own build/test environment

**Status:** Superseded by ADR-0009

**Context:** The agent edits code, and that code has to be built and tested
somewhere. The tempting shortcut is to bake each language's toolchain into the
harness image (we briefly put Go there). That doesn't scale: it grows one image
per language, and it breaks the moment a repo is polyglot — which real repos are.
It also entangles two things that should stay separate: the rented harness
(one image, language-agnostic) and each project's build environment (many,
project-specific). The industry already has the seam — a repo declares its
environment via a `devcontainer.json`, a `Dockerfile`, or its CI config — and the
same declaration is what verification needs anyway (ADR-0005).

**Decision:** The **project** declares its own build/test environment, in the
project repo (a `Dockerfile` today). The harness image stays language-agnostic —
it only edits files, carries no per-project toolchains. Verification runs the
project's tests in that declared environment, not in the harness. New
dependencies (e.g. added to `go.mod`) need no harness change; the project's
toolchain fetches them.

**Alternatives:** Bake every language into the harness image (rejected — one
image per language, fails on polyglot repos, couples harness to project). A
single fat image with all toolchains (rejected — unbounded, still can't match
arbitrary project versions).

**Consequences:**
- The agent pod loses the Go toolchain and the Bash tool; its task is pure edits.
  The task prompt no longer mentions running or verifying anything.
- `demo-project/` now carries its own `Dockerfile` + `ENVIRONMENT.md`, standing in
  for how any real customer repo declares its environment.
- Verification (ADR-0005) has a concrete home to run in: build the project's
  environment, run its suite there — a clean runtime the agent didn't set up.
- Interim honesty: the orchestrator does not yet run that verification; it's
  declared and documented, wired up in a later slice.

**Refines:** ADR-0004 (the sandbox is a plain container) and ADR-0005 (separate
verification runtime) — this says *where* the project's build/test environment
comes from: the project, not the harness.
