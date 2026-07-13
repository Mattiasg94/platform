# Issues: The agent runs on Cloud Run

Six slices. The ordering rests on one rule from the PRD: **change the agent's contract before
changing where the agent runs.** Issue 3 rewrites how the agent receives and returns work while
it still runs under local Docker, so it can be debugged in seconds. Issue 5 then moves it to
Cloud Run without touching the contract, so any failure there can only be about Cloud Run.

Issues 2 through 5 each leave the system fully working end to end — a task goes in, a pull
request comes out — which is what makes them safe places to stop.

---

## Issue 1 — Unblock the infrastructure pipeline

## What to build

Outstanding chores from the previous epic that are blocking or half-finished. All of these are
cloud writes and belong to Mattias, not to an agent.

The Terraform service account needs the service usage admin role before the branch declaring
the enabled APIs can be applied — without it the apply fails with a permission error, which is
the same class of failure that killed the first apply. Once granted, that branch can merge.

Separately, confirm that the Terraform check is actually *required* by the main branch ruleset.
It was added to the workflow but it was never confirmed that a red build can no longer be
merged, which makes the whole merge-is-apply pipeline advisory rather than binding.

## Acceptance criteria

- [x] `terraform-ci` holds the service usage admin role on the project
- [x] The branch declaring the enabled Google APIs is merged and applied cleanly
- [x] The `terraform` job is a required status check on the main branch ruleset
- [x] A pull request with a failing Terraform check cannot be merged

## Why this slice is cut here

It is not part of the agent's story, but it is unfinished business that will produce a
confusing failure at the worst possible moment if left. Grouped into one issue because it is
all the same five minutes of console and terminal work.

## Blocked by

None — can start immediately.

---

## Issue 2 — CI becomes the verdict; the verification container is retired

## What to build

Delete the separate verification runtime from the orchestrator. The orchestrator commits the
agent's work to a branch and opens a pull request, and the target project's own continuous
integration delivers the verdict on that pull request. A human reads the result.

This is a pure removal on our side. The target project's CI already runs on every pull request
and already runs lint *and* tests — a strict superset of what the verification container ran.
Nothing needs to be built for the verdict to appear; it appears the moment a pull request is
opened.

The principle of ADR-0005 — that the verdict must come from something the agent never touched —
is not weakened by this. CI satisfies it by definition: it runs on GitHub's runners, from the
pushed branch, with no agent involvement at all. So ADR-0005 must be **superseded** with the new
mechanism, not deleted. Its reasoning is what keeps the design honest.

## Acceptance criteria

- [x] The verification image, its build, and its invocation are gone from the orchestrator
- [x] A run still ends in a pull request, and the project's CI runs on that pull request
- [x] The CI result is visible as the pass/fail signal on the pull request
- [x] ADR-0005 is superseded by an ADR recording CI as the verdict mechanism (ADR-0010)
- [x] Nothing in the orchestrator depends on the target project's `verify` build stage

## Why this slice is cut here

Placed first because it is a pure deletion that depends on nothing, and because it makes every
slice after it smaller — every subsequent issue would otherwise have to carry the verification
container along and keep it working.

Deliberately does *not* include reading CI logs back and feeding them to the agent. That closes
the loop automatically, and it only makes sense once the orchestrator is a long-lived service
that can wait on a webhook. It belongs to the successor epic.

## Blocked by

None — can start immediately.

---

## Issue 3 — The agent receives and returns its work through the bucket

## What to build

Replace every laptop-shaped channel between the orchestrator and the agent with the bucket,
while the agent still runs under the local Docker daemon. This is the whole claim-check design,
proven with a fast local feedback loop.

The orchestrator writes the task and a tarball of the cloned workspace — git history included —
into the run's prefix in the bucket, then starts the agent carrying nothing but a run
identifier. The agent reads that identifier, pulls its own inputs down, unpacks the workspace,
does the work, and uploads its result. The orchestrator reads the result back from the bucket.

The bind mount is deleted, the task stops travelling as a command-line argument, and the result
stops being scraped from the container's stdout.

This has a consequence that must be handled in the same slice: **the agent no longer edits the
orchestrator's clone.** Today the bind mount makes the agent's edits appear on local disk,
which is what the branch push silently relies on. Once the workspace travels as a tarball, the
clone is untouched and the changes come back only as a diff. So the orchestrator must apply the
returned diff to its own clone before it can commit and open the pull request.

The agent will need credentials to reach the bucket while it is still a local container.
Mounting application-default credentials in read-only is the accepted temporary hack — it is
confined to the orchestrator's local invocation, and it is deleted in Issue 5.

## Acceptance criteria

- [x] The task is delivered as an object in the run's bucket prefix, not as a command-line argument
- [x] The workspace is delivered as a tarball in the run's bucket prefix, and includes git history
- [x] The agent unpacks the workspace and can run the project's tests inside it, unchanged
- [x] The agent writes its result as an object in the run's bucket prefix
- [x] The agent receives nothing but a run identifier from its invocation
- [x] The bind mount is gone from the container's configuration
- [x] The orchestrator applies the returned diff to its clone, commits it, and opens the pull request
- [x] A full run works end to end — task in, pull request out — and the project's CI runs on it
- [x] The agent's stdout carries only logs; nothing parses it

## Why this slice is cut here

This is the slice that freezes the agent's contract, and everything else in the epic depends on
it. It deliberately does *not* move the agent to Cloud Run, so that when the transport breaks —
and it will — the debugging loop is seconds long, not minutes, and the failure cannot be
confused with a cloud problem.

The diff-application work is folded in rather than split out because without it the slice
regresses: the pull request would silently stop containing the agent's changes. A slice that
ships an empty pull request is a worse checkpoint than a slightly fatter one.

## Blocked by

None strictly — but do Issue 2 first, or this slice has to keep a verification container
working against a clone it no longer owns.

---

## Issue 4 — The agent image is content-hashed, pushed, and cached

## What to build

Stop rebuilding the agent image on every run. Give it an identity, publish it, and reuse it.

Before a run, the orchestrator computes a hash of the files that define the environment — the
target project's Dockerfile and the agent's Dockerfile — and uses it as the image tag. It then
asks Artifact Registry whether an image with that tag already exists. On a hit it does nothing
at all. On a miss it builds the project's environment stage, layers the agent on top of it as
today, and pushes the result under that tag.

The image is still executed by the local Docker daemon. Only its provenance changes.

The hash covers exactly the inputs the environment stage reads, and emphatically **not** the
source code. Hashing the source would change the tag on every run, miss the cache every time,
and rebuild on every execution — strictly worse than today. Today the demo project's
environment stage copies nothing in, so two Dockerfiles are the entire input; if a project's
environment stage later pre-downloads dependencies, whatever manifest it copies in joins the
hash.

## Acceptance criteria

- [x] The agent image tag is derived from a hash of the environment-defining files only
- [x] Editing the target project's source code does not change the tag
- [x] Editing either Dockerfile does change the tag
- [x] On a cache hit, no image is built and nothing is pushed
- [x] On a cache miss, the image is built and pushed to Artifact Registry under the tag
- [x] A second consecutive run of the same task performs zero builds
- [x] Runs still work end to end, executed locally from the tagged image

## Why this slice is cut here

Split out from Issue 5 on purpose. This slice proves the cache key and the push while the
execution path is still boring, fast and local — so a bug in the hashing logic cannot be
mistaken for a Cloud Run problem, and vice versa. It is also the slice with the most immediate
payoff to daily work: local runs stop rebuilding two images every single time.

Kept separate from Issue 3 because the two changes are independent, and merging them would
create a slice where a failure has two possible causes.

## Blocked by

- Issue 3 — the image the cache serves must already be the cloud-shaped agent, or the cache
  would be populated with an image that is about to be replaced.

---

## Issue 5 — The agent executes as a Cloud Run Job

## What to build

Move the agent off the laptop. A new runner implementation ensures a Cloud Run job definition
exists and points at the image identified in Issue 4, creates an execution carrying only the
run identifier, waits for it to finish, and reads the result from the bucket.

The orchestrator continues to own the job definition, because it is the only component that
knows the current image tag. Terraform continues to own the static infrastructure it already
owns.

Two credentials change hands here. The Anthropic key is now injected by the platform from
Secret Manager rather than read from the host environment, and the agent's access to the bucket
now comes from the `agent-job` service account it already runs as — so the mounted local
credentials from Issue 3 are deleted. Both of those bindings already exist and were proven by
the blackboard spike.

Blocking on the execution to detect completion is accepted. A push-based completion signal is
deliberately deferred; a laptop process can afford to wait.

## Acceptance criteria

- [x] A Cloud Run job definition exists for the agent and points at the content-hashed image
- [x] The definition is created and updated by the orchestrator, not by Terraform
- [x] An execution is started carrying only the run identifier
- [x] The Anthropic key reaches the agent from Secret Manager, not from the host environment
- [x] The agent reaches the bucket using its own service account identity
- [x] The mounted local credentials from Issue 3 are gone
- [x] The orchestrator detects completion and reads the result from the bucket
- [x] A full run works end to end — task in, pull request out — with the agent running in the cloud
- [x] The local Docker runner is removed, or kept only if it still earns its place

## Why this slice is cut here

This is the payoff, and it is deliberately the *smallest* slice in the epic. Because Issues 3
and 4 already moved the contract and the image, nothing about how the agent receives or returns
work changes here. That is the entire point of the sequencing: any failure in this slice can
only be about Cloud Run itself.

## Blocked by

- Issue 4 — Cloud Run can only execute an image that has been pushed to a registry.

---

## Issue 6 — One test proves the two halves of the pod contract agree

**Deferred.** Not this week — picked up later. Left here so it is not forgotten.

## What to build

The pod's I/O contract (ADR-0007) is defined twice: once as the JSON the Python agent writes,
and once as the Go struct the orchestrator unmarshals into. Nothing checks that they still
agree, so a rename on one side is a silent runtime failure on the other.

Add a test that fails when they diverge.

## Acceptance criteria

- [ ] A test exists that fails if a field is renamed, removed, or retyped on either side
- [ ] The test runs as part of the normal check gate, not as an advisory extra
- [ ] The test does not require a real agent run, an API key, or the cloud

## Why this slice is cut here

Independent of the rest, and cheap. It is placed last only because the contract is still moving
until Issue 3 lands; pinning it before then would just mean writing it twice.

This is only affordable *because* the repository is a monorepo — the two definitions can be
compared in one test run. Worth stating, because it is one of the concrete dividends of that
choice.

## Blocked by

- Issue 3 — the contract it pins is rewritten there.
