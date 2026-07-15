// Package orchestrator holds the run policy — the platform's core loop, kept out
// of package main so it is testable in isolation and is the natural home for a
// durable (Temporal) workflow later. It drives one task to a merge-ready PR but
// never merges: that stays a human's call for now.
package orchestrator

import (
	"context"
	"fmt"
	"log"
	"time"

	"orchestrator/internal/pod"
	"orchestrator/internal/prompt"
	"orchestrator/internal/repo"
)

// forge is the slice of the GitHub client this package needs, declared here so a
// fake can stand in for tests.
type forge interface {
	OpenPR(ctx context.Context, head, base, title, body string) (int, error)
}

// Deps are the collaborators a run needs, injected by the composition root.
type Deps struct {
	Runner    pod.Runner
	Forge     forge
	Workspace string
	BaseRef   string
}

// Run lets the agent edit once and opens a PR. The verdict is not ours to give:
// the project's own CI judges the PR (ADR-0010), which is why there is no retry
// loop here. The agent's status is only a self-report — it gates publishing a
// crashed run, nothing more.
func Run(ctx context.Context, deps Deps) error {
	result, err := deps.Runner.Run(ctx, prompt.Initial())
	if err != nil {
		return fmt.Errorf("agent pod: %w", err)
	}
	log.Printf("pod run status: %s", result.Status)
	if result.Status != "success" {
		return fmt.Errorf("agent reported failure: %s", result.Summary)
	}

	// The agent worked on its own copy, so our clone is still pristine.
	if err := repo.ApplyDiff(ctx, deps.Workspace, result.Diff); err != nil {
		return err
	}

	return publish(ctx, deps)
}

func publish(ctx context.Context, deps Deps) error {
	branch := fmt.Sprintf("agent/run-%d", time.Now().Unix())
	if err := repo.CreateBranch(ctx, deps.Workspace, branch); err != nil {
		return err
	}
	if err := repo.CommitAll(ctx, deps.Workspace, "Automated change by the orchestrator"); err != nil {
		return err
	}
	if err := repo.Push(ctx, deps.Workspace, branch); err != nil {
		return err
	}
	log.Printf("pushed %s", branch)

	prNumber, err := deps.Forge.OpenPR(ctx, branch, deps.BaseRef,
		"Automated change by the orchestrator",
		"Opened by the orchestrator. CI is the verdict (ADR-0010) — review it and merge manually.")
	if err != nil {
		return err
	}
	log.Printf("opened PR #%d — review its CI and merge it yourself", prNumber)
	return nil
}
