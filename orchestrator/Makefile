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

.PHONY: check ci precommit fmt fmt-check vet lint test deadcode vuln security install-tools install-hooks

# Fast inner loop — what the AI runs while editing. Fail-fast, auto-formats.
check: fmt vet lint test

# The blocking CI gate. The inner-loop checks, but formatting is verified rather
# than applied — CI judges a finished artifact, it does not edit your code —
# plus dead-code detection. Anything here failing fails the build.
ci: fmt-check vet lint test deadcode

# Runs on every commit via .githooks/pre-commit (install once with
# `make install-hooks`). The inner-loop checks plus deadcode, so unreachable
# code is caught at commit time instead of sitting silently until a human spots
# it. vuln and security stay advisory — see below.
precommit: check deadcode


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

# Whole-program dead code: unreachable exported funcs/types. golangci-lint's
# `unused` misses these — it treats every exported symbol as a live entry point,
# so an exported func with zero callers slips through. This is the check that
# catches it, which is why it gates `ci` and `precommit`.
#
# deadcode always exits 0 (it's built to report, not gate), so we fail the build
# ourselves when it prints anything. -test treats functions reached only from
# tests as live, so we don't fail on production code that only tests exercise.
deadcode:
	@out="$$(deadcode -test ./...)"; \
	if [ -n "$$out" ]; then \
		echo "$$out"; \
		echo "deadcode: unreachable code found (wire it up or remove it)"; \
		exit 1; \
	fi

# --- Advisory checks (human-run) ----------------------------------------------
# Run these when you want to, not on every save and not as a hard gate. Each can
# report something you'll consciously choose not to fix — a prototype security
# finding, a vuln with no upstream fix — so they need your judgment rather than
# blocking. CI runs them too, in a separate non-blocking workflow.

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

# Point git at the version-controlled hooks dir (platform/.githooks) so
# `make precommit` runs on every commit. Run once per clone.
install-hooks:
	git config core.hooksPath .githooks
	chmod +x ../.githooks/*
