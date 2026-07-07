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

// buildPrompt is the static task the orchestrator hands the pod. It exercises
// the whole code-and-test loop in one repeatable step: append one element to a
// Go function, which breaks a count test, then fix that test and prove it green
// with `go test`. Sourced here so the prompt travels orchestrator -> pod.
//
// Repeatable: each run appends a distinctly-timestamped element and bumps the
// expected count by one, so the suite is green again every time.
func buildPrompt() string {
	stamp := time.Now().UTC().Format("2006-01-02 15:04:05Z")
	return fmt.Sprintf(
		"The workspace is a Go module. In greeting.go, append exactly one new "+
			"element, the string \"ran at %s\", to the slice returned by the "+
			"Greetings function. That will break the count assertion in "+
			"greeting_test.go: update the expected count in that test to the new "+
			"number of elements. Then run `go test ./...` from the workspace root "+
			"and confirm it passes. Change nothing else.",
		stamp,
	)
}
