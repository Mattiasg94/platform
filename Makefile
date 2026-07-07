# Platform dev tasks. The orchestrator launches the agent pod; these targets are
# the thin manual entry points around that. The commands live here (executable,
# self-verifying) rather than in prose that quietly goes stale.

IMAGE := agent-pod

.PHONY: agent-build run agent-run

## Build the agent pod image.
agent-build:
	docker build -t $(IMAGE) ./agent

## The full slice: the orchestrator launches the agent pod on demo-project,
## sends it a task, and prints the structured result (status + diff).
## DOCKER_HOST is resolved from the active docker context because the Go SDK
## reads DOCKER_HOST but not CLI contexts (Docker Desktop uses its own socket).
run: agent-build
	cd orchestrator && \
	  DOCKER_HOST="$$(docker context inspect -f '{{.Endpoints.docker.Host}}')" \
	  go run ./cmd/orchestrator

## Run the agent pod alone (no orchestrator), for debugging the harness itself.
## Encodes the run-time facts the orchestrator sets in its container spec:
## non-root user (the CLI refuses bypassPermissions as root, and matching the
## host uid keeps workspace files owned by you) and a writable HOME.
agent-run: agent-build
	docker run --rm \
	  --user "$$(id -u):$$(id -g)" \
	  -e HOME=/tmp \
	  -v "$$(pwd)/demo-project:/workspace" \
	  --env-file .env \
	  $(IMAGE) \
	  "Append exactly one new line reading 'manual run' to notes.md."
