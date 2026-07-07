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

	log.Printf("building agent image…")
	if err := runner.EnsureImage(ctx); err != nil {
		return err
	}

	prompt := buildPrompt()
	log.Printf("launching agent pod; task: %s", prompt)

	result, err := runner.Run(ctx, prompt)
	if err != nil {
		return fmt.Errorf("agent pod: %w", err)
	}

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

// buildPrompt is the static task the orchestrator hands the pod. It is two
// precise, deterministic file edits — no instruction to run or verify anything.
// The pod only edits; verification runs later in the project's own environment
// (ADR-0005, ADR-0008). It is fine for the suite to be red between the two edits.
//
// Repeatable: each run appends a distinctly-timestamped element and bumps the
// expected count by one, keeping code and test in step.
func buildPrompt() string {
	stamp := time.Now().UTC().Format("2006-01-02 15:04:05Z")
	return fmt.Sprintf(
		"Make exactly these two edits in the Go files at the workspace root, and "+
			"nothing else:\n"+
			"1. In greeting.go, add one line to the slice returned by the Greetings "+
			"function: the string \"ran at %s\".\n"+
			"2. In greeting_test.go, increase the `want` constant by exactly one so "+
			"it equals the new number of elements in that slice.",
		stamp,
	)
}
