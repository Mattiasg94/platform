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

const maxAttempts = 3

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

	task := buildPrompt()
	var lastFailure string
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		log.Printf("attempt %d/%d", attempt, maxAttempts)

		runStart := time.Now()
		result, err := runner.Run(ctx, task)
		if err != nil {
			return fmt.Errorf("agent pod: %w", err)
		}
		log.Printf("pod run — %s; status: %s", time.Since(runStart).Round(time.Millisecond), result.Status)

		// Trusted verdict: the agent's own make-test can be gamed; this re-runs in
		// a clean runtime it never touched (ADR-0005).
		verifyStart := time.Now()
		verification, err := runner.Verify(ctx)
		if err != nil {
			return fmt.Errorf("verify: %w", err)
		}
		log.Printf("verification — %s", time.Since(verifyStart).Round(time.Millisecond))

		if verification.Passed {
			log.Printf("trusted verification: PASSED on attempt %d", attempt)
			printDiff(result.Diff)
			return nil
		}

		log.Printf("trusted verification: FAILED on attempt %d", attempt)
		lastFailure = verification.Output
		// Feed the grounded failure back; the bind-mounted workspace still holds
		// the agent's edits, so the next attempt continues from there.
		task = retryPrompt(verification.Output)
	}

	return fmt.Errorf("verification failed after %d attempts:\n%s", maxAttempts, lastFailure)
}

func printDiff(diff string) {
	fmt.Println("--- workspace diff ---")
	if diff == "" {
		fmt.Println("(no changes)")
	} else {
		fmt.Print(diff)
	}
}

// buildPrompt is a stand-in hardcoded task until real task input exists.
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

// retryPrompt feeds a failed trusted verification back to the agent for another
// attempt.
func retryPrompt(failure string) string {
	return fmt.Sprintf(
		"Your previous change to the Go project at the workspace root did not pass "+
			"verification. Test output:\n\n%s\n\nFix the code so the suite passes, "+
			"then run `make test` to confirm. Do not delete, skip, or weaken tests.",
		failure,
	)
}
