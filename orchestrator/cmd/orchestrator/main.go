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

	workspace, cleanup, err := repo.Checkout(ctx, cfg.ProjectRepoURL, cfg.ProjectRef, repo.Auth{Token: cfg.GitHubToken})
	if err != nil {
		return fmt.Errorf("checkout project repo: %w", err)
	}
	defer cleanup()

	runner, err := pod.NewDocker(workspace)
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
