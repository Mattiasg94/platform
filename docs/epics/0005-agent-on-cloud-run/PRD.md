# Epic 0005 — The agent runs on Cloud Run

## Business Goal & Objective

The platform's north star is employability in AI engineering: a single-user system that is
nonetheless production-shaped, where the substrate is rented and the policy is built. Today the
coding agent only exists on one laptop. It is bound to a local Docker daemon, a local
filesystem, and a local environment variable. That means it cannot be triggered by anything
except a human sitting at that machine, it cannot run more than one job at a time, and none of
it is defensible as a real system.

This epic cuts the *agent* free of the laptop. When it is done, the agent is a container image
in Artifact Registry, executed on demand as a Cloud Run Job, communicating only through a
bucket. Nothing about it will assume a human is nearby.

The orchestrator stays on the laptop, and moves to the cloud in a successor epic. That is the
sequencing choice this whole epic rests on, and it has a consequence worth stating plainly:

**The agent's contract must be cloud-correct now. The orchestrator is allowed to stay
temporarily laptop-shaped.** Short-lived hacks in the orchestrator are acceptable and expected
here — mounted local credentials, blocking waits, building images with the local Docker daemon.
They are acceptable *because they are confined to the orchestrator*, which is being rewritten
anyway. What is not acceptable is letting any of that leak into the agent's contract, because
the agent is the thing we are freezing.

## Problem Statement

The agent works, but every mechanism it uses to talk to the outside world is a local Docker
feature, and none of those features exist in the cloud.

It receives its workspace as a bind mount from a directory on disk. There is no shared
filesystem in Cloud Run.

It receives its task as a command-line argument. Prompts are growing, argv has a hard size
limit, and putting the content of the work into the invocation is the wrong shape regardless.

It returns its result on stdout, which the orchestrator reads back through the Docker daemon's
log pipe. There is no daemon in the cloud, and Cloud Logging is for humans, not for parsing.

It reads its API key from the host's environment.

And, most awkwardly, the agent image does not exist as a fixed artifact at all. It is rebuilt
from scratch on every single run, layered on top of the target project's environment image, so
the agent gets the project's real toolchain and its tests give real feedback (ADR-0008,
ADR-0009). That is a good decision and we intend to keep it. But Cloud Run runs images. It does
not build them.

## Proposed Solution

Two things move at completely different speeds, and the whole design falls out of separating
them.

The code changes on every run — that is what a run is for. The environment changes almost
never, only when someone edits a Dockerfile. Things moving at different speeds should travel by
different mechanisms.

So the environment gets baked into an image, once, and cached in Artifact Registry under a tag
that is a hash of the files that shape it. The code never enters that image. It travels
separately, at run time, as a tarball through the bucket. They meet at the workspace directory
inside the running container, and never before. This is already how it works — the environment
stage is source-free by construction — so moving to the cloud changes only the delivery truck,
not the fact.

Before each run the orchestrator computes the environment hash and asks Artifact Registry a
cheap question: does an image with this tag already exist? Almost always the answer is yes, and
the run starts immediately with no build at all. Occasionally someone changed a Dockerfile, the
answer is no, and the image is rebuilt and pushed automatically. Nobody has to remember to do
anything. The tag is the identity of the environment.

The orchestrator then clones the project, tars the tree up including its git history, writes it
and the task into the bucket, and starts a Cloud Run Job execution carrying nothing but a run
identifier. The agent pulls its own inputs, works, runs the project's real tests inside the
project's real toolchain, and writes its diff back to the bucket. The orchestrator applies that
diff to its own clone, commits it to a branch, and opens the pull request.

The verdict then comes from the target project's own continuous integration, which runs on that
pull request. The separate verification container is retired.

The agent holds no GitHub credential at any point. It is a pure function: workspace in, diff
out.

## User Stories

1. As the orchestrator, I want to hand the agent its workspace through a bucket rather than a
   bind mount, so that the agent can run on a machine I have never seen.
2. As the orchestrator, I want to hand the agent its task through a bucket rather than argv, so
   that a large prompt is never truncated by an operating system limit.
3. As the orchestrator, I want to read the agent's result from a bucket rather than from a log
   pipe, so that I do not depend on a Docker daemon being present.
4. As the orchestrator, I want to apply the agent's returned diff to my own clone, so that I can
   still open a pull request now that the agent no longer edits my disk.
5. As a developer, I want the target project's own CI to be the verdict on the agent's change,
   so that the judgement comes from something the agent never touched and I do not maintain a
   second, weaker copy of the test run.
6. As the agent, I want my Anthropic key injected by the platform from Secret Manager, so that
   no credential has to be handed to me by a human or stored on disk.
7. As the orchestrator, I want the agent image to be identified by a hash of the files that
   define its environment, so that I rebuild it exactly when it is stale and never otherwise.
8. As the orchestrator, I want to skip the build entirely when that image already exists, so
   that the common case costs nothing and a run starts immediately.
9. As the agent, I want to keep running inside the target project's real toolchain, so that
   when I run the project's tests I get true feedback rather than an approximation.
10. As the agent, I want to hold no GitHub credential, so that a model-generated shell command
    can never reach a repository.
11. As a developer, I want one test that proves the Go and Python halves of the pod contract
    still agree, so that a silent mismatch between them cannot ship.

## Technical Direction & Assumptions

### What is in the image, and what is not

The image is the target project's environment stage, plus the agent's own layer on top of it —
Python, Node, git, the Claude harness, and the entrypoint script. The application source is not
in the image and never will be. It arrives at run time.

### The cache key

The tag is a hash of the files that shape the environment: the target project's Dockerfile and
the agent's Dockerfile. Nothing else. In particular it is emphatically **not** a hash of the
source code — hashing the source would change the tag on every run, miss the cache every time,
and rebuild an image on every execution, which is strictly worse than what we have today.

Right now the demo project's environment stage is a bare Go base image with a working
directory; it copies nothing in. So today two Dockerfiles are the entire hash input. If a
project's environment stage later pre-downloads dependencies, then whatever manifest it copies
in — a `go.mod`, a lockfile — becomes part of the hash too. The rule is: hash exactly the
inputs the environment stage reads.

### Communication

The claim check pattern, from Enterprise Integration Patterns. The dispatch message carries a
pointer; the payload lives in blob storage. This is not a workaround for our smallness — every
message transport has a small size limit, so this pattern is what real systems do. The bucket
already exists, with a thirty-day lifecycle on objects.

Per run, the bucket holds the task, the workspace tarball, and the result.

### The agent no longer edits our disk

This is the non-obvious consequence of deleting the bind mount, and it must not be discovered
late. Today the agent's edits appear directly in the orchestrator's clone, which is what makes
the branch push work. Once the workspace travels as a tarball, the clone is untouched and the
changes come back only as a diff.

So the orchestrator must apply the returned diff to its own clone before it can commit and push
a branch. Everything downstream of that then works unchanged.

### Where things run, and what owns what

The orchestrator stays on the laptop for this epic. It has a Docker daemon, so it can build and
push images directly, and it can afford to block while waiting for a job to finish. Both of
those conveniences vanish in the successor epic and we will pay for them then, not now.

Terraform owns the static infrastructure — the bucket, the service account, the secret, the
registry, the enabled APIs. All of that already exists and is proven by the blackboard spike.
The Cloud Run **job definition**, however, points at an image whose tag changes with the
environment hash. That makes it a runtime artifact rather than static infrastructure, and the
orchestrator is the only thing that knows the hash. So the orchestrator creates and updates job
definitions; Terraform does not.

### The two feedback loops

The inner loop is the agent running the project's tests inside its own container. It is fast,
cheap, and untrustworthy — which is fine, because it is only the agent's own feedback while it
works, exactly as a human runs tests before pushing.

The outer loop is the authoritative verdict, and from this epic on it is the target project's
own continuous integration, running on the pull request. It takes minutes and it costs a git
push, so the agent enters it only when it believes it is finished.

Do not be tempted to make CI the agent's iteration loop. The agent iterates against the inner
loop; CI is the judge, not the feedback.

In this epic a human reads the CI result. Feeding a failing CI run's logs back into a fresh
agent run — closing the loop automatically — is deferred to the successor epic, because it only
makes sense once the orchestrator is a long-lived service that can wait on a webhook.

## Decisions & Rationale

**The image never contains the source code.** It was never going to; the environment stage is
source-free by construction, precisely so that editing source does not invalidate the agent
image layered on top. Recognising this is what collapses the problem: the code and the
environment were never coupled, so moving to the cloud changes only how the code is delivered.

**Cache the image ahead of time, keyed by a content hash of its own inputs.** Two alternatives
were considered and rejected. *One generic fat image for every project*, with dependencies
installed at run time, is by far the simplest thing that works — but the agent would no longer
be in the project's real environment, only an approximation of it, and the resulting class of
bug is genuinely nasty: the agent's tests pass and yours do not, and nothing tells you why.
That trades away the most interesting property of the design for a week of convenience.
*Building fresh on every run with Cloud Build* is always correct, and costs minutes of latency
and real money on every single run to rebuild something that is byte-identical ninety-nine
times out of a hundred. It is the "correct" answer nobody ships. Caching keyed by content gives
the speed of the first and the correctness of the second, and the thing that decides between
them is the content of the files, not a human's memory.

**Hash the environment-shaping files, not the source.** Hashing the source would invalidate the
cache on every run and make the mechanism pointless. This is stated explicitly because it is
the one place the design is easy to get backwards.

**The orchestrator owns all GitHub interaction; the agent holds no credential.** The agent is a
container executing model-generated shell with permissions bypassed. Handing it a GitHub token
— even a read-only one — would place a token to every repository inside an untrusted process.
It is the same category of mistake as letting the agent produce its own test verdict, which
ADR-0005 already forbids. The orchestrator clones, tars, and uploads instead. The cost is close
to zero: a tarball containing the git directory is still a real repository, so history survives
and `git diff` works unchanged.

**The workspace travels as a tarball through the bucket.** The alternative — having the agent
clone the repository itself — is superficially simpler and was rejected because it requires
giving the agent exactly the credential we just decided it must not have.

**Change the contract before changing the runtime.** Sequencing decision. If the transport and
the execution environment are flipped in one step and something breaks, it is not attributable
to one of them — and every debugging cycle then costs a cloud round-trip instead of a few
seconds. So the agent learns to speak through the bucket while still running under local
Docker, and only then moves.

**The orchestrator, not Terraform, owns Cloud Run job definitions.** A job definition points at
an image tag that varies with the environment hash, so it is a runtime artifact and the
orchestrator is the only component that knows the correct value. Terraform continues to own
everything static.

**The orchestrator stays local, and may take hacks that the successor epic will delete.**
Mounted application-default credentials, a blocking wait on the job execution, and image builds
against the local Docker daemon are all knowingly temporary. They are acceptable because they
are confined to the orchestrator, which is being rewritten in the successor epic anyway. The
constraint is that none of them may leak into the agent's contract.

**Continuous integration becomes the verdict, and the verification container is retired.** The
principle behind ADR-0005 is that the verdict must come from something the agent never touched,
so that it cannot be gamed. CI satisfies that by definition — it runs on GitHub's runners, from
the pushed branch, with no agent involvement whatsoever. And the target project's CI is already
a strict superset of the verification container: it runs lint *and* tests, where the container
only ran tests. So maintaining the container means maintaining a second, weaker copy of a check
we already have.

The principle of ADR-0005 therefore survives intact; only its mechanism is replaced. ADR-0005 is
superseded by ADR-0010, not deleted.

This takes the orchestrator's **retry loop** with it. That loop fed the verification container's
failing output back to the agent for another attempt; with no verifier in-process there is no
signal to retry on, so a run becomes a single attempt that ends in a pull request. The agent
still iterates internally against the project's real `make test` — that inner loop is unchanged,
and it is untrusted by design.

What is *deferred* is closing the loop automatically — reading a failing CI run's logs and
feeding them back into a fresh agent run. That only makes sense once the orchestrator is a
long-lived service that can wait on a webhook; a laptop process that blocks for minutes on a CI
run is the wrong shape and would be rebuilt immediately. Until then a human reads the result.

## Out of Scope

**The orchestrator moving into Cloud Run.** It stays on the laptop. This is the successor epic,
and it is the reason several deliberate hacks are permitted here.

**Reading CI logs back and feeding them to the agent.** Deferred with the orchestrator, for the
reason above. In this epic CI delivers the verdict on the pull request and a human reads it;
nothing is automatically fed back.

**Cloud Build.** Not needed while the orchestrator has a local Docker daemon. It becomes
necessary the moment the orchestrator loses one.

**A push-based completion signal.** Wiring bucket object-finalize events to Pub/Sub is the
right answer for an orchestrator that cannot sit and block, and this orchestrator can sit and
block. Blocking on the job execution is sufficient here.

**Presigned URLs.** They would let the agent hold no cloud credential at all, only a
time-limited URL to read one object and write one. It is the correct end state and it is not
needed yet.

**Durable orchestration** — the reconciler pattern, or Temporal (ADR-0003). This is the real
long-term gap: if the orchestrator dies mid-run, the run is lost. It is deliberately not
addressed here, because addressing it requires the agent to already be a remote, restartable
job — which is what this epic delivers.

**A message queue.** It would not replace the bucket, because the claim check is still needed,
and it would replace dispatch, which the Cloud Run Jobs API already does. The genuine trigger
for introducing one is backpressure against Anthropic's rate limits, and we are nowhere near
that.

**Concurrency.** One run at a time.

## Open Questions for Technical Discovery

How large is the workspace tarball in practice once the git history is included, and does
uploading it on every run introduce noticeable latency? If it does, a shallow clone is the
obvious mitigation, but it should be measured rather than pre-optimised.

Does the Cloud Run job definition need to be one per project, or can a single definition be
repointed at whichever image the current run needs? One per project is cleaner and costs
nothing, since a definition is inert and free — but it has not been decided.

Does `git apply` of the agent's returned diff handle every case we care about — new files,
deletions, renames, binary files? The agent already stages intent-to-add before diffing, which
covers new files, but this should be proven rather than assumed.
