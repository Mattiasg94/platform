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

const maxAttempts = 3

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

// Run builds the images, then loops: the agent edits, a trusted runtime the
// agent never touched verifies (ADR-0005), and a failure is fed back for another
// attempt. On a pass it opens a PR and stops.
func Run(ctx context.Context, deps Deps) error {
	if err := deps.Runner.EnsureImage(ctx); err != nil {
		return err
	}

	task := prompt.Initial()
	var lastFailure string
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		log.Printf("attempt %d/%d", attempt, maxAttempts)

		result, err := deps.Runner.Run(ctx, task)
		if err != nil {
			return fmt.Errorf("agent pod: %w", err)
		}
		log.Printf("pod run status: %s", result.Status)

		verification, err := deps.Runner.Verify(ctx)
		if err != nil {
			return fmt.Errorf("verify: %w", err)
		}
		if verification.Passed {
			log.Printf("trusted verification: PASSED on attempt %d", attempt)
			return publish(ctx, deps)
		}

		log.Printf("trusted verification: FAILED on attempt %d", attempt)
		lastFailure = verification.Output
		task = prompt.Retry(verification.Output)
	}
	return fmt.Errorf("verification failed after %d attempts:\n%s", maxAttempts, lastFailure)
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
		"Opened by the orchestrator after passing trusted verification. Review its CI and merge manually.")
	if err != nil {
		return err
	}
	log.Printf("opened PR #%d — review its CI and merge it yourself", prNumber)
	return nil
}
