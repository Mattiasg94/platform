# 0001 — Rent the inner agent loop (Claude Agent SDK), don't build it

**Status:** Accepted — refined by 0007 (the rented loop runs *inside* the
sandbox; the anti-lock-in seam is the pod I/O contract, not an in-process
`Brain` interface).

**Context:** The inner agent loop — file-edit parsing, tool-use turn loop,
context/token management — is tempting to build and I nearly did (I was going to
write my own Claude-Code-style SDK). It churns every model release, and
off-the-shelf now beats a hand-roll. Spotify built their own, then killed it for
the SDK: their loop needed rigid step-by-step prompts where the model did better
with "describe the end state." This is the "reinventing the wheel" trap.

**Decision:** Rent the inner loop (Claude Agent SDK or equivalent). Put it behind
a `Brain` interface I own so it never hardens into vendor lock-in.

**Alternatives:** Build my own loop (rejected — commodity that churns, teaches
little transferable skill, and Spotify's lived regret warns against it).

**Consequences:** The loop is a rented commodity behind `Brain`. My build effort
goes to the platform *around* it (orchestration, verification, feedback loops,
observability) — the part that transfers. Swapping SDKs later is a two-way door.
