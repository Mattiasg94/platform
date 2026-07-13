// Package prompt builds the task text handed to the agent. It is kept apart from
// the orchestration policy on purpose: wording changes here, flow changes in
// package orchestrator. Both are stand-ins until real task input exists.
package prompt

import (
	"fmt"
	"time"
)

func Initial() string {
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
