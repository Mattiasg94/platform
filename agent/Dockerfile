# Agent pod: the coding harness (Claude Agent SDK) layered on top of the project's
# own environment. BASE_IMAGE is that environment image (built from the project's
# own Dockerfile), so it already carries the project's toolchain; we add only the
# harness — the same set for every project, whatever its language. Supersedes the
# earlier "language-agnostic, edit-only" harness image (ADR-0008, now due a revisit).
ARG BASE_IMAGE
FROM ${BASE_IMAGE}

# The harness's own runtimes (not the project's): Python for our entrypoint and
# SDK client, Node + npm for the Claude Code CLI the SDK spawns, git for the diff.
# From Debian, not NodeSource: the external repo's DNS failed mid-build and
# silently left Node without npm. Debian's Node 18 runs the pinned CLI fine.
RUN apt-get update \
    && apt-get install -y --no-install-recommends python3 python3-pip git nodejs npm \
    && rm -rf /var/lib/apt/lists/*

# The harness itself: the Claude Code CLI (Node) that the Python SDK spawns.
# Pinned: the harness churns per release, so an unpinned build could pull a
# breaking version silently. Bump deliberately.
RUN npm install -g @anthropic-ai/claude-code@2.1.197

# The Python binding that drives that CLI.
# --break-system-packages: fine in a single-purpose container (PEP 668).
RUN pip3 install --no-cache-dir --break-system-packages claude-agent-sdk==0.2.111

WORKDIR /app
COPY main.py /app/main.py

# The repo is bind-mounted at /workspace and the API key arrives via env at run
# time — nothing secret is baked in. ENTRYPOINT (not CMD) also clears any CMD from
# the project base (e.g. the demo's `go test`), so the container runs the harness
# and the task arrives as the args appended after it.
ENTRYPOINT ["python3", "/app/main.py"]
