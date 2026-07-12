# Platform backlog

Short list of things to investigate or build later. Entry format: see
`to-do-README.md`. Not live state — the code and each epic's `ISSUES.md` own that.

## GitHub authentication for repo fetching

Move repo fetching from public-clone-no-auth to the production standard: the
platform is a GitHub App that mints a short-lived, repo-scoped installation
token per run. The orchestrator holds the token and does the clone; the agent
sandbox never sees a credential. Later work: register the App, per-customer
install flow, token minting in the checkout step, and ephemeral-credential
handling.
