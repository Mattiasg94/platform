# Layered on the project's env image, passed as BASE_IMAGE (ADR-0009).
ARG BASE_IMAGE
FROM ${BASE_IMAGE}

# Node + npm from Debian, not NodeSource: the external repo's DNS failed mid-build
# and silently left Node without npm. Debian's Node 18 runs the pinned CLI.
RUN apt-get update \
    && apt-get install -y --no-install-recommends python3 python3-pip git nodejs npm \
    && rm -rf /var/lib/apt/lists/*

RUN npm install -g @anthropic-ai/claude-code@2.1.197
RUN pip3 install --no-cache-dir --break-system-packages claude-agent-sdk==0.2.111

WORKDIR /app
COPY main.py /app/main.py

# ENTRYPOINT (not CMD) clears the base's CMD (e.g. the demo's `go test`); the task
# arrives as args appended here.
ENTRYPOINT ["python3", "/app/main.py"]
