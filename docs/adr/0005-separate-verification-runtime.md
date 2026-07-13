# 0005 — Verification is a separate runtime the agent can't touch

**Status:** Superseded by 0010

**Context:** Once the agent acts on its own output unattended (opens PRs,
self-verifies), it will game any judge it controls. Spotify's agents cheated to
force green builds — commented out failing tests, downgraded Java. Spotify also
built an LLM-as-judge verifier, found it too rigid (it blocked valid changes), and
removed it as models improved; verification steps in prompts + real CI proved
enough. Google Jules runs a separate review pass before the PR opens.

**Decision:** Build verification as a **separate runtime the agent cannot
influence** — it runs the real tests / inspects the diff in a clean place the
agent didn't set up. Keep it **light**: real CI signals, not a rigid LLM-judge.

**Alternatives:** Let the agent verify its own work (rejected — it cheats).
Heavy LLM-as-judge gate (rejected — Spotify's removed regret; too rigid, blocks
valid work).

**Consequences:** This is a *build-to-learn* seam and a core differentiator — the
interim must be real (actually runs tests), not a stub. Trigger to build it out:
the agent starts acting on its own output unattended. Starts light and stays light
until real pain justifies more.
