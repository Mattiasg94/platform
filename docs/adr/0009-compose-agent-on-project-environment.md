# 0009 — Compose the agent pod on the project's own environment

**Status:** Accepted

**Context:** ADR-0008 kept the harness image language-agnostic and edit-only: it
carried no project toolchain, so the agent could not build or run anything. That
stripped the agent of a feedback loop — it edited blind and could not check its
own work — which is most of what makes an agentic coding loop effective. We still
want the project (not the platform) to own its environment, and we still don't
want to bake every language into one universal harness image. The open question
is only *how* the harness and the project's toolchain end up in one place.

**Decision:** Build the agent pod image *from* the project's own declared
environment — layer the fixed harness (Node, Python, the Claude CLI) on top of
the project's toolchain image, so one container has both. The project's Dockerfile
splits into a source-free `env` stage (just the toolchain) and a `verify` stage
(env + source + test command). The agent layers on `env`; because that stage
carries no source, editing the workspace does not invalidate the harness build
cache. The agent reads the source through the bind mount at run time, not a
baked-in copy, and gets a Bash tool so it can run the project's `make test` and
iterate — its own feedback loop.

**Alternatives:** Keep the agent edit-only (ADR-0008) — rejected: no feedback
loop, a much weaker agent. Bake every language into one harness image — rejected
by ADR-0008's own reasoning: unbounded, breaks on polyglot repos. Run the harness
and the project env as two separate containers — deferred: more isolation than we
need now, more machinery. Inject the harness at run time instead of baking it —
deferred: the natural optimization once composed-image rebuilds actually hurt.

**Consequences:**
- The agent pod carries the project's toolchain and runs `make test`; the task
  prompt now asks it to verify and iterate, not just edit.
- This is the agent's *untrusted* self-check. Trusted verification — a clean
  runtime the agent can't influence (ADR-0005) — is still a separate, later slice.
- A project now declares its environment in two stages; the agnostic contract a
  project must meet gains "expose a buildable, source-free `env` stage."
- Supersedes ADR-0008. Carried over from it: the project, not the platform, owns
  its build/test environment. Reversed: the harness is no longer toolchain-free
  and edit-only.

**Refines:** ADR-0004 (the sandbox is a plain container).
