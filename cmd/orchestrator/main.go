package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"orchestrator/internal/config"
	"orchestrator/internal/sandbox"

	dockerclient "github.com/docker/docker/client"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	dockerCli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	defer dockerCli.Close()

	sb := sandbox.NewDockerSandbox(dockerCli)
	if err := sb.SweepOrphans(ctx); err != nil {
		return fmt.Errorf("sandbox sweep: %w", err)
	}
	if err := sb.Start(ctx); err != nil {
		return fmt.Errorf("sandbox start: %w", err)
	}
	defer func() {
		if err := sb.Destroy(ctx); err != nil {
			log.Printf("sandbox destroy: %v", err)
		}
	}()

	// No Brain is wired in yet — the hand-rolled loop was pruned (ADR-0006).
	// Sandbox lifecycle alone is proven here; issue 2 of the walking-skeleton
	// project wires a rented SDK back in behind the Brain interface.
	fmt.Println("Orchestrator - sandbox ready (Ctrl+C to exit)")
	<-ctx.Done()
	return nil
}
