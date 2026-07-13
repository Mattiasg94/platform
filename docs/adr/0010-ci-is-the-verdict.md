# 0010 — The project's own CI is the verdict

**Status:** Accepted. Supersedes 0005.

**Context:** ADR-0005 required the verdict to come from a runtime the agent cannot
influence, and we built that as a separate `--target verify` container built and
run by the orchestrator. But every target project already has CI that runs on a
pull request — and the demo project's runs `make ci`, which is lint *and* tests,
where the verify container only ran tests. So the container was a second, weaker
copy of a check we already had, and one more thing to keep working as the agent
moves to Cloud Run (epic 0005).

**Decision:** The target project's own CI, running on the pull request the
orchestrator opens, is the verdict. Delete the separate verification runtime.

The principle of ADR-0005 is untouched: the verdict still comes from something the
agent never touched. CI satisfies that by construction — it runs on GitHub's
runners, from the pushed branch, with no agent involvement at all. Only the
mechanism changes. ADR-0005 itself pointed here: *"Keep it light: real CI signals,
not a rigid LLM-judge."*

**Alternatives:** Keep the verify container alongside CI (rejected — two judges
that can disagree, and the weaker one is the one we maintain). Port the verify
container to a second Cloud Run Job (rejected — pays cloud complexity to rebuild a
signal GitHub already gives us for free).

**Consequences:** The orchestrator's retry loop goes with it. That loop fed the
verify container's failing output back to the agent for another attempt; with no
verifier in-process, there is no signal to retry on, so a run is now a single
attempt that ends in a PR. The agent still iterates *internally* against the
project's real `make test` — that inner loop is unchanged, and it is untrusted by
design.

Closing the outer loop again — reading a failing CI run's logs and feeding them to
a fresh agent run — is deferred until the orchestrator is a long-lived service that
can wait on a webhook. A laptop process that blocks for minutes on a CI run is the
wrong shape. Until then a human reads the red build.
