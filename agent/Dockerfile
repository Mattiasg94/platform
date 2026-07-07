# Agent pod: the rented coding harness (Claude Agent SDK) baked into an image.
#
# Two runtimes on purpose. Python is our entrypoint language, but the Python
# SDK is only a thin client — the actual agent harness is the Claude Code CLI,
# a Node program the SDK launches as a subprocess. So the image needs Node
# (the harness) *and* Python (our entrypoint + the client binding). See
# agent/README.md for the full why.
FROM node:20-bookworm-slim

# Python runtime for our entrypoint and the SDK client; git so the pod can diff
# the workspace it edited (the result contract); curl + ca-certificates to fetch
# the Go toolchain below.
RUN apt-get update \
    && apt-get install -y --no-install-recommends python3 python3-pip git curl ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Go toolchain, so the pod can run the fixture's tests (go test) as part of the
# verify-in-pod loop. INTERIM: baking a language into the harness image is the
# per-runtime anti-pattern we flagged — it doesn't scale to polyglot repos. The
# durable design is the *project* declaring its own environment (devcontainer /
# its own Dockerfile) and the harness layering on top. Fine while there's one Go
# fixture and we're proving the loop; revisit before a second language appears.
ARG GO_VERSION=1.23.4
RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -o /tmp/go.tgz \
    && tar -C /usr/local -xzf /tmp/go.tgz \
    && rm /tmp/go.tgz
ENV PATH="/usr/local/go/bin:${PATH}"

# The harness itself: the Claude Code CLI (Node) that the Python SDK spawns.
# Pinned: the harness churns per release, so an unpinned build could pull a
# breaking version silently. Bump deliberately.
RUN npm install -g @anthropic-ai/claude-code@2.1.197

# The Python binding that drives that CLI.
# --break-system-packages: fine in a single-purpose container (PEP 668).
RUN pip3 install --no-cache-dir --break-system-packages claude-agent-sdk==0.2.111

WORKDIR /app
COPY main.py /app/main.py

# The repo to edit is bind-mounted at /workspace at run time; the API key
# arrives as ANTHROPIC_API_KEY in the environment. Nothing secret is baked in.
ENTRYPOINT ["python3", "/app/main.py"]
