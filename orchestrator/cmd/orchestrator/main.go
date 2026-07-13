package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"orchestrator/internal/config"
	"orchestrator/internal/gh"
	"orchestrator/internal/orchestrator"
	"orchestrator/internal/pod"
	"orchestrator/internal/repo"
	"orchestrator/internal/runs"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// The token authenticates origin so the branch we push later rides the same
	// remote (docs/to-do.md tracks the move to a GitHub App token).
	workspace, cleanup, err := repo.Checkout(ctx, cfg.ProjectRepoURL, cfg.ProjectRef, repo.Auth{Token: cfg.GitHubToken})
	if err != nil {
		return fmt.Errorf("checkout project repo: %w", err)
	}
	defer cleanup()

	blackboard, err := runs.NewStore(ctx, cfg.RunsBucket)
	if err != nil {
		return err
	}

	// Docker is still here, but demoted to the one job that genuinely needs a local
	// daemon: building the image. The agent itself runs in the cloud.
	builder := pod.NewBuilder(workspace, cfg.GCPProject)

	runner, err := pod.NewCloudRun(ctx, builder, cfg.RunsBucket, cfg.GCPProject, blackboard)
	if err != nil {
		return err
	}

	return orchestrator.Run(ctx, orchestrator.Deps{
		Runner:    runner,
		Forge:     gh.NewClient(cfg.GitHubToken, cfg.ProjectOwner, cfg.ProjectRepo),
		Workspace: workspace,
		BaseRef:   cfg.ProjectRef,
	})
}
