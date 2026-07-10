package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"orchestrator/internal/config"
	"orchestrator/internal/pod"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	config.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	runner, err := pod.NewDocker()
	if err != nil {
		return err
	}

	if err := runner.EnsureImage(ctx); err != nil {
		return err
	}

	prompt := buildPrompt()
	log.Printf("launching agent pod; task: %s", prompt)

	runStart := time.Now()
	result, err := runner.Run(ctx, prompt)
	if err != nil {
		return fmt.Errorf("agent pod: %w", err)
	}
	log.Printf("pod run — %s", time.Since(runStart).Round(time.Millisecond))

	log.Printf("pod status: %s", result.Status)
	log.Printf("pod summary: %s", result.Summary)
	fmt.Println("--- workspace diff ---")
	if result.Diff == "" {
		fmt.Println("(no changes)")
	} else {
		fmt.Print(result.Diff)
	}
	return nil
}

// buildPrompt is the task the orchestrator hands the pod. It gives the agent a
// goal and lets it verify its own work with the project's `make test`, iterating
// on failures — the agent's feedback loop. Adding a greeting breaks the test's
// count assertion, so the agent must notice the red suite and fix it, which is
// exactly the loop we want to exercise. This is the agent's own *untrusted*
// check; trusted verification is a separate step the agent can't touch (a later
// slice, ADR-0005).
//
// Repeatable: each run appends a distinctly-timestamped greeting.
func buildPrompt() string {
	stamp := time.Now().UTC().Format("2006-01-02 15:04:05Z")
	return fmt.Sprintf(
		"In the Go project at the workspace root, add a new greeting to the slice "+
			"returned by the Greetings function in greeting.go: the string "+
			"\"ran at %s\". Then run `make test` to check your work; if the suite "+
			"fails, fix whatever legitimately needs fixing and re-run `make test` "+
			"until it passes. The suite must genuinely pass — do not delete, skip, "+
			"or weaken tests to force it.",
		stamp,
	)
}
