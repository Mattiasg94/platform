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

// buildPrompt is the static task the orchestrator hands the pod: the same
// append-a-timestamped-line task we ran by hand, now sourced here so the prompt
// travels orchestrator -> pod rather than being baked into the pod. Runs
// repeatedly and safely (it only appends to notes.md, never touches tests).
func buildPrompt() string {
	stamp := time.Now().UTC().Format("2006-01-02 15:04:05Z")
	return fmt.Sprintf(
		"Append exactly one new line reading 'agent ran at %s' to the end of "+
			"notes.md in the current directory. Create notes.md if it does not "+
			"exist. Change nothing else, and do not reformat existing lines.",
		stamp,
	)
}
