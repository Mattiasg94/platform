# Orchestrator — dev checks.
#
# `make check` runs the whole ladder locally, cheapest first:
#   format -> vet -> lint -> test
#
# Tool versions are pinned below so you, the AI, and CI all run identical
# checks. Bump the version here and re-run `make install-tools`.

# Put the pinned tools (installed under GOPATH/bin) on PATH for every recipe, so
# `make lint` works without you having to add GOPATH/bin to your shell.
export PATH := $(shell go env GOPATH)/bin:$(PATH)

GOLANGCI_LINT_VERSION := v2.12.2
DEADCODE_VERSION      := v0.48.0
GOVULNCHECK_VERSION   := v1.5.0
GOSEC_VERSION         := v2.27.1

.PHONY: check precommit fmt fmt-check vet lint test deadcode vuln security install-tools

# Fast inner loop — what the AI runs while editing. Fail-fast, auto-formats.
check: fmt vet lint test

# The deterministic gate to run before committing. Right now it is just `check`
# — there is no deterministic, always-actionable check that only matters at
# commit time yet. Kept as a stable named target so commit tooling has something
# to call and more can be added later. The advisory checks (deadcode, vuln,
# security) are deliberately NOT here — see below.
precommit: check

# Format code in place. Mechanical, no judgment — just fixes and moves on.
fmt:
	gofmt -w .

# Fail (don't rewrite) if anything is unformatted. This is the CI-friendly
# variant: CI judges a finished artifact, it does not edit your code.
fmt-check:
	@out="$$(gofmt -l .)"; \
	if [ -n "$$out" ]; then \
		echo "These files are not gofmt'ed:"; echo "$$out"; exit 1; \
	fi

# Built-in Go bug checker. Free, fast, almost no false positives.
vet:
	go vet ./...

# Aggregated third-party linters (config in .golangci.yml).
lint:
	golangci-lint run

# Tests with the race detector on — Orchestrator manages containers, so it's
# concurrent, and -race catches a whole class of otherwise-invisible bugs.
test:
	go test -race ./...

# --- Advisory checks (human-run) ----------------------------------------------
# Run these when you want to, not on every save and not as a hard gate. Each can
# report things you'll consciously choose not to fix — a prototype security
# finding, a vuln with no upstream fix, code that's dead only because it isn't
# wired up yet — so they need your judgment rather than blocking.

# Whole-program dead code (unreachable exported funcs/types).
deadcode:
	deadcode ./...

# Known vulnerabilities in dependencies we actually call (call-graph aware, so
# low-noise). Needs the vuln DB (network).
vuln:
	govulncheck ./...

# Security static analysis (subprocess/exec, credential handling, etc.). Kept
# out of `check` on purpose: in a prototype many findings are ones you'll review
# and accept rather than restructure the code to satisfy.
security:
	gosec ./...

# Install pinned dev tools into $(go env GOPATH)/bin.
install-tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	go install golang.org/x/tools/cmd/deadcode@$(DEADCODE_VERSION)
	go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
	go install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)
