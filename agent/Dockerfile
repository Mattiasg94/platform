# Layered on the project's env image, passed as BASE_IMAGE (ADR-0009).
ARG BASE_IMAGE
FROM ${BASE_IMAGE}

# Node + npm from Debian, not NodeSource: the external repo's DNS failed mid-build
# and silently left Node without npm. Debian's Node 18 runs the pinned CLI.
RUN apt-get update \
    && apt-get install -y --no-install-recommends python3 python3-pip git nodejs npm \
    && rm -rf /var/lib/apt/lists/*

RUN npm install -g @anthropic-ai/claude-code@2.1.197
RUN pip3 install --no-cache-dir --break-system-packages \
    claude-agent-sdk==0.2.111 \
    google-cloud-storage==2.19.0

# The harness refuses to bypass permissions as root, so the pod must not be root.
# Baking the user into the image rather than overriding it at `docker run` keeps
# this true wherever the image runs — Cloud Run does not let a caller pick a uid.
# uid 1000 also happens to match the host user, which is what lets the pod read
# the credentials file lent to it locally (see pod.execute).
RUN useradd --create-home --uid 1000 agent \
    && mkdir -p /workspace \
    && chown agent:agent /workspace
USER agent

WORKDIR /app
COPY main.py /app/main.py

# ENTRYPOINT (not CMD) clears the base's CMD (e.g. the demo's `go test`). It takes
# no arguments: the pod is told only its RUN_ID, through the environment, and
# fetches the task and the workspace itself.
ENTRYPOINT ["python3", "/app/main.py"]
